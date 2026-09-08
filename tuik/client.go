package tuik

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the SDMX REST base URL published in the TÜİK
	// documentation.
	DefaultBaseURL = "https://nsiws.tuik.gov.tr/rest"

	defaultUserAgent = "tzercin-analytics-tuik-go/0"
	maxErrorBody     = 32 << 10
)

// TokenSource supplies an access token for an API request. Implementations may
// cache and refresh tokens. Token values must never be logged.
type TokenSource interface {
	Token(context.Context) (AccessToken, error)
}

// TokenSourceFunc adapts a function to TokenSource.
type TokenSourceFunc func(context.Context) (AccessToken, error)

// Token calls f(ctx).
func (f TokenSourceFunc) Token(ctx context.Context) (AccessToken, error) {
	return f(ctx)
}

// Client calls the TÜİK SDMX REST API.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	tokens     TokenSource
	userAgent  string
}

// ClientOption configures a Client.
type ClientOption func(*clientConfig) error

type clientConfig struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// WithBaseURL overrides the REST base URL. It is primarily useful for testing
// or for a compatible proxy. The URL must use HTTP or HTTPS and must not contain
// a query or fragment.
func WithBaseURL(rawURL string) ClientOption {
	return func(cfg *clientConfig) error {
		cfg.baseURL = rawURL
		return nil
	}
}

// WithHTTPClient supplies the HTTP client used for SDMX requests. The caller
// owns its transport and redirect policy.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(cfg *clientConfig) error {
		if client == nil {
			return errors.New("tuik: HTTP client is nil")
		}
		cfg.httpClient = client
		return nil
	}
}

// WithUserAgent overrides the default User-Agent header. An empty value
// disables the header.
func WithUserAgent(value string) ClientOption {
	return func(cfg *clientConfig) error {
		cfg.userAgent = value
		return nil
	}
}

// NewClient constructs an authenticated TÜİK client.
func NewClient(tokens TokenSource, options ...ClientOption) (*Client, error) {
	if tokens == nil {
		return nil, errors.New("tuik: token source is nil")
	}

	cfg := clientConfig{
		baseURL:   DefaultBaseURL,
		userAgent: defaultUserAgent,
	}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("tuik: client option is nil")
		}
		if err := option(&cfg); err != nil {
			return nil, err
		}
	}

	baseURL, err := parseBaseURL(cfg.baseURL)
	if err != nil {
		return nil, err
	}
	if cfg.httpClient == nil {
		cfg.httpClient = defaultHTTPClient(baseURL)
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: cfg.httpClient,
		tokens:     tokens,
		userAgent:  cfg.userAgent,
	}, nil
}

func parseBaseURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("tuik: parse base URL: %w", err)
	}
	if (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, errors.New("tuik: base URL must be an absolute HTTP(S) URL")
	}
	if u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("tuik: base URL must not contain user info, a query, or a fragment")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	return u, nil
}

func defaultHTTPClient(baseURL *url.URL) *http.Client {
	return sameOriginHTTPClient(baseURL, 60*time.Second)
}

func sameOriginHTTPClient(baseURL *url.URL, timeout time.Duration) *http.Client {
	origin := baseURL.Scheme + "://" + baseURL.Host
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if req.URL.Scheme+"://"+req.URL.Host != origin {
				return errors.New("tuik: refusing cross-origin redirect")
			}
			return nil
		},
	}
}

// Do authenticates and executes req. It sets Authorization and User-Agent on a
// clone, leaving req unchanged. A non-2xx response is returned as *HTTPError.
// The caller must close the body of a successful response.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("tuik: request is nil")
	}
	if ctx == nil {
		return nil, errors.New("tuik: context is nil")
	}
	if req.URL == nil || req.URL.Scheme != c.baseURL.Scheme || req.URL.Host != c.baseURL.Host {
		return nil, errors.New("tuik: request URL must use the configured base URL origin")
	}
	basePath := c.baseURL.Path + "/"
	if req.URL.Path != c.baseURL.Path && !strings.HasPrefix(req.URL.Path, basePath) {
		return nil, errors.New("tuik: request URL must be below the configured base URL path")
	}

	token, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("tuik: obtain access token: %w", err)
	}
	if token.Value == "" {
		return nil, errors.New("tuik: token source returned an empty access token")
	}

	request := req.Clone(ctx)
	request.Header = req.Header.Clone()
	request.Header.Set("Authorization", "Bearer "+token.Value)
	if request.Header.Get("User-Agent") == "" && c.userAgent != "" {
		request.Header.Set("User-Agent", c.userAgent)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("tuik: %s %s: %w", request.Method, safeURL(request.URL), err)
	}
	if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		return response, nil
	}

	return nil, newHTTPError(request, response)
}

func (c *Client) get(ctx context.Context, segments []string, query url.Values) (*http.Response, error) {
	u := *c.baseURL
	u.Path = c.baseURL.Path + "/" + strings.Join(segments, "/")
	u.RawQuery = query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("tuik: construct request: %w", err)
	}
	return c.Do(ctx, request)
}

func safeURL(u *url.URL) string {
	if u == nil {
		return "<nil URL>"
	}
	copy := *u
	copy.User = nil
	copy.RawQuery = ""
	copy.Fragment = ""
	return copy.Redacted()
}

// HTTPError describes a non-success response. Body contains at most 32 KiB and
// is intended for diagnostics; the response body has already been closed.
type HTTPError struct {
	Method     string
	URL        string
	Status     string
	StatusCode int
	Header     http.Header
	Body       []byte
}

func newHTTPError(request *http.Request, response *http.Response) *HTTPError {
	body, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorBody))
	_ = response.Body.Close()
	return &HTTPError{
		Method:     request.Method,
		URL:        safeURL(request.URL),
		Status:     response.Status,
		StatusCode: response.StatusCode,
		Header:     response.Header.Clone(),
		Body:       body,
	}
}

// Error implements error without including the response body, headers, or any
// credential value.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("tuik: %s %s: %s", e.Method, e.URL, e.Status)
}
