package eurostat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDataURLAndFetchRetry(t *testing.T) {
	t.Parallel()
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts++
		if request.Header.Get("User-Agent") != "test-agent" {
			t.Errorf("wrong user agent: %q", request.Header.Get("User-Agent"))
		}
		if attempts == 1 {
			writer.Header().Set("Retry-After", "1")
			http.Error(writer, "busy", http.StatusTooManyRequests)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var delays []time.Duration
	client := NewClient()
	client.BaseURL = server.URL + "/data/"
	client.UserAgent = "test-agent"
	client.Now = func() time.Time { return time.Date(2026, 9, 9, 8, 0, 0, 0, time.UTC) }
	client.Sleep = func(_ context.Context, duration time.Duration) error {
		delays = append(delays, duration)
		return nil
	}
	requestURL, err := client.DataURL("TEST", url.Values{"time": {"2025", "2024"}, "geo": {"TR"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(requestURL, "/data/TEST?geo=TR&time=2025&time=2024") {
		t.Fatalf("unexpected deterministic URL: %s", requestURL)
	}
	response, err := client.Fetch(context.Background(), requestURL)
	if err != nil {
		t.Fatal(err)
	}
	if attempts != 2 || len(delays) != 1 || delays[0] != time.Second || string(response.Body) != `{"ok":true}` {
		t.Fatalf("attempts=%d delays=%v response=%q", attempts, delays, response.Body)
	}
}

func TestRetryAfterZeroIsHonoured(t *testing.T) {
	t.Parallel()
	delay, ok := retryAfter("0", time.Time{})
	if !ok || delay != 0 {
		t.Fatalf("retry-after zero = %s, %v", delay, ok)
	}
}
