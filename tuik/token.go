package tuik

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultTokenURL is the token endpoint published in the TÜİK
	// documentation.
	DefaultTokenURL = "https://giris.tuik.gov.tr/realms/web/protocol/openid-connect/token"

	// DefaultAPIKeyClientID is the client identifier published for the
	// SMS-verified API-key flow.
	DefaultAPIKeyClientID = "nsi-ws-consumer"

	maxTokenResponse = 1 << 20
)

// AccessToken is an OAuth access token. Value is sensitive and must not be
// logged or persisted in the repository.
type AccessToken struct {
	Value  string
	Expiry time.Time
}

// StaticTokenSource returns a TokenSource for an already-obtained token. It is
// useful when token lifecycle management happens outside this package.
func StaticTokenSource(value string) TokenSource {
	return TokenSourceFunc(func(context.Context) (AccessToken, error) {
		return AccessToken{Value: value}, nil
	})
}

// TokenOption configures a TÜİK token source.
type TokenOption func(*tokenConfig) error

type tokenConfig struct {
	tokenURL   string
	httpClient *http.Client
	expirySkew time.Duration
	now        func() time.Time
}

// WithTokenURL overrides the TÜİK token endpoint. It is primarily intended for
// tests or a compatible proxy.
func WithTokenURL(rawURL string) TokenOption {
	return func(cfg *tokenConfig) error {
		cfg.tokenURL = rawURL
		return nil
	}
}

// WithTokenHTTPClient supplies the HTTP client used for token requests. The
// caller owns its transport and redirect policy; redirects that preserve POST
// bodies can expose credentials if allowed to cross origins.
func WithTokenHTTPClient(client *http.Client) TokenOption {
	return func(cfg *tokenConfig) error {
		if client == nil {
			return errors.New("tuik: token HTTP client is nil")
		}
		cfg.httpClient = client
		return nil
	}
}

// WithTokenExpirySkew refreshes a cached token this long before its reported
// expiry. The default is 30 seconds.
func WithTokenExpirySkew(skew time.Duration) TokenOption {
	return func(cfg *tokenConfig) error {
		if skew < 0 {
			return errors.New("tuik: token expiry skew must not be negative")
		}
		cfg.expirySkew = skew
		return nil
	}
}

type oauthTokenSource struct {
	tokenURL   *url.URL
	httpClient *http.Client
	form       url.Values
	expirySkew time.Duration
	now        func() time.Time

	mu     sync.Mutex
	cached AccessToken
}

// NewAPIKeyTokenSource creates the API-key token flow documented for verified
// TÜİK Data Portal users.
func NewAPIKeyTokenSource(apiKey string, options ...TokenOption) (TokenSource, error) {
	if apiKey == "" {
		return nil, errors.New("tuik: API key is empty")
	}
	return newOAuthTokenSource(url.Values{
		"grant_type": {"password"},
		"client_id":  {DefaultAPIKeyClientID},
		"api_key":    {apiKey},
	}, options...)
}

// NewClientCredentialsTokenSource creates the client-credentials flow
// documented for users provisioned directly by TÜİK.
func NewClientCredentialsTokenSource(clientID, clientSecret string, options ...TokenOption) (TokenSource, error) {
	if clientID == "" {
		return nil, errors.New("tuik: client ID is empty")
	}
	if clientSecret == "" {
		return nil, errors.New("tuik: client secret is empty")
	}
	return newOAuthTokenSource(url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}, options...)
}

func newOAuthTokenSource(form url.Values, options ...TokenOption) (TokenSource, error) {
	cfg := tokenConfig{
		tokenURL:   DefaultTokenURL,
		expirySkew: 30 * time.Second,
		now:        time.Now,
	}
	for _, option := range options {
		if option == nil {
			return nil, errors.New("tuik: token option is nil")
		}
		if err := option(&cfg); err != nil {
			return nil, err
		}
	}

	tokenURL, err := parseTokenURL(cfg.tokenURL)
	if err != nil {
		return nil, err
	}
	if cfg.httpClient == nil {
		cfg.httpClient = sameOriginHTTPClient(tokenURL, 30*time.Second)
	}

	return &oauthTokenSource{
		tokenURL:   tokenURL,
		httpClient: cfg.httpClient,
		form:       cloneValues(form),
		expirySkew: cfg.expirySkew,
		now:        cfg.now,
	}, nil
}

func parseTokenURL(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("tuik: parse token URL: %w", err)
	}
	if (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
		return nil, errors.New("tuik: token URL must be an absolute HTTP(S) URL")
	}
	if u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return nil, errors.New("tuik: token URL must not contain user info, a query, or a fragment")
	}
	return u, nil
}

func cloneValues(values url.Values) url.Values {
	copy := make(url.Values, len(values))
	for key, value := range values {
		copy[key] = append([]string(nil), value...)
	}
	return copy
}

func (s *oauthTokenSource) Token(ctx context.Context) (AccessToken, error) {
	if ctx == nil {
		return AccessToken{}, errors.New("tuik: context is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if s.cached.Value != "" && (s.cached.Expiry.IsZero() || now.Add(s.expirySkew).Before(s.cached.Expiry)) {
		return s.cached, nil
	}

	token, err := s.requestToken(ctx, now)
	if err != nil {
		return AccessToken{}, err
	}
	s.cached = token
	return token, nil
}

type tokenResponse struct {
	AccessToken string          `json:"access_token"`
	ExpiresIn   json.RawMessage `json:"expires_in"`
}

func (s *oauthTokenSource) requestToken(ctx context.Context, now time.Time) (AccessToken, error) {
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		s.tokenURL.String(),
		strings.NewReader(s.form.Encode()),
	)
	if err != nil {
		return AccessToken{}, fmt.Errorf("tuik: construct token request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return AccessToken{}, fmt.Errorf("tuik: token request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return AccessToken{}, &TokenError{
			Status:     response.Status,
			StatusCode: response.StatusCode,
		}
	}

	var payload tokenResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxTokenResponse))
	if err := decoder.Decode(&payload); err != nil {
		return AccessToken{}, fmt.Errorf("tuik: decode token response: %w", err)
	}
	if payload.AccessToken == "" {
		return AccessToken{}, errors.New("tuik: token response did not contain access_token")
	}

	expiresIn, err := parseExpiresIn(payload.ExpiresIn)
	if err != nil {
		return AccessToken{}, err
	}
	token := AccessToken{Value: payload.AccessToken}
	if expiresIn > 0 {
		token.Expiry = now.Add(expiresIn)
	}
	return token, nil
}

func parseExpiresIn(raw json.RawMessage) (time.Duration, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, nil
	}

	var seconds int64
	if err := json.Unmarshal(raw, &seconds); err == nil {
		if seconds < 0 {
			return 0, errors.New("tuik: token expires_in is negative")
		}
		return secondsDuration(seconds)
	}

	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, errors.New("tuik: token expires_in is not a number")
	}
	seconds, err := strconv.ParseInt(text, 10, 64)
	if err != nil || seconds < 0 {
		return 0, errors.New("tuik: token expires_in is not a non-negative integer")
	}
	return secondsDuration(seconds)
}

func secondsDuration(seconds int64) (time.Duration, error) {
	if seconds > math.MaxInt64/int64(time.Second) {
		return 0, errors.New("tuik: token expires_in is too large")
	}
	return time.Duration(seconds) * time.Second, nil
}

// TokenError describes a rejected token request. Credential values and the
// response body are intentionally excluded.
type TokenError struct {
	Status     string
	StatusCode int
}

func (e *TokenError) Error() string {
	return fmt.Sprintf("tuik: token request: %s", e.Status)
}
