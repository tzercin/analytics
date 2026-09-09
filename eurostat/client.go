package eurostat

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultBaseURL = "https://ec.europa.eu/eurostat/api/dissemination/statistics/1.0/data/"

type Client struct {
	HTTPClient  *http.Client
	BaseURL     string
	UserAgent   string
	MaxAttempts int
	BaseDelay   time.Duration
	Now         func() time.Time
	Sleep       func(context.Context, time.Duration) error
}

type Response struct {
	Body        []byte
	URL         string
	RetrievedAt time.Time
	Date        string
	ContentType string
}

func NewClient() *Client {
	return &Client{
		HTTPClient:  &http.Client{Timeout: 90 * time.Second},
		BaseURL:     DefaultBaseURL,
		UserAgent:   "tzercin-analytics/1.0 (+https://github.com/tzercin/analytics; reproducible public-data analysis)",
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		Now:         time.Now,
		Sleep: func(ctx context.Context, duration time.Duration) error {
			timer := time.NewTimer(duration)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	}
}

func (client *Client) DataURL(datasetCode string, filters url.Values) (string, error) {
	if datasetCode == "" || strings.ContainsAny(datasetCode, "/?#") {
		return "", fmt.Errorf("invalid dataset code %q", datasetCode)
	}
	base, err := url.Parse(client.BaseURL)
	if err != nil {
		return "", fmt.Errorf("parse Eurostat base URL: %w", err)
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/" + datasetCode
	base.RawQuery = filters.Encode()
	return base.String(), nil
}

// Fetch performs one request at a time and retries only 429 and transient 5xx
// responses. Retry-After is honoured; otherwise exponential delay plus up to
// 250 ms of cryptographic jitter is used.
func (client *Client) Fetch(ctx context.Context, requestURL string) (*Response, error) {
	if client.HTTPClient == nil || client.Now == nil || client.Sleep == nil || client.MaxAttempts < 1 {
		return nil, fmt.Errorf("invalid Eurostat client configuration")
	}
	var lastStatus string
	for attempt := 0; attempt < client.MaxAttempts; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return nil, fmt.Errorf("build Eurostat request: %w", err)
		}
		request.Header.Set("Accept", "application/json")
		request.Header.Set("User-Agent", client.UserAgent)
		response, err := client.HTTPClient.Do(request)
		if err != nil {
			return nil, fmt.Errorf("Eurostat request: %w", err)
		}
		body, readErr := io.ReadAll(io.LimitReader(response.Body, 32<<20))
		closeErr := response.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read Eurostat response: %w", readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close Eurostat response: %w", closeErr)
		}
		if response.StatusCode == http.StatusOK {
			return &Response{
				Body:        body,
				URL:         response.Request.URL.String(),
				RetrievedAt: client.Now(),
				Date:        response.Header.Get("Date"),
				ContentType: response.Header.Get("Content-Type"),
			}, nil
		}
		lastStatus = response.Status
		retryable := response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
		if !retryable || attempt+1 == client.MaxAttempts {
			return nil, fmt.Errorf("Eurostat response %s: %s", response.Status, strings.TrimSpace(string(body)))
		}
		delay, hasRetryAfter := retryAfter(response.Header.Get("Retry-After"), client.Now())
		if !hasRetryAfter {
			delay = client.BaseDelay * time.Duration(1<<attempt)
			delay += jitter(250 * time.Millisecond)
		}
		if err := client.Sleep(ctx, delay); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("Eurostat request failed after %d attempts: %s", client.MaxAttempts, lastStatus)
}

func retryAfter(value string, now time.Time) (time.Duration, bool) {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}
	if when, err := http.ParseTime(value); err == nil && when.After(now) {
		return when.Sub(now), true
	}
	return 0, false
}

func jitter(maximum time.Duration) time.Duration {
	if maximum <= 0 {
		return 0
	}
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return 0
	}
	return time.Duration(binary.LittleEndian.Uint64(buffer[:]) % uint64(maximum))
}
