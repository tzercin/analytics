package tuik

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestAPIKeyTokenSourcePostsDocumentedFormAndCaches(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"grant_type": "password",
			"client_id":  DefaultAPIKeyClientID,
			"api_key":    "api-secret",
		}
		for key, value := range want {
			if got := r.PostForm.Get(key); got != value {
				t.Errorf("%s = %q, want %q", key, got, value)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"access-secret","expires_in":3600}`)
	}))
	defer server.Close()

	source, err := NewAPIKeyTokenSource("api-secret", WithTokenURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		token, err := source.Token(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if token.Value != "access-secret" || token.Expiry.IsZero() {
			t.Fatalf("token = %#v", token)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("token calls = %d, want 1", got)
	}
}

func TestClientCredentialsTokenSourcePostsDocumentedForm(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"grant_type":    "client_credentials",
			"client_id":     "client-id",
			"client_secret": "client-secret",
		}
		for key, value := range want {
			if got := r.PostForm.Get(key); got != value {
				t.Errorf("%s = %q, want %q", key, got, value)
			}
		}
		_, _ = io.WriteString(w, `{"access_token":"access-secret","expires_in":"60"}`)
	}))
	defer server.Close()

	source, err := NewClientCredentialsTokenSource(
		"client-id",
		"client-secret",
		WithTokenURL(server.URL),
		WithTokenExpirySkew(0),
	)
	if err != nil {
		t.Fatal(err)
	}
	token, err := source.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if token.Value != "access-secret" {
		t.Fatalf("token value = %q", token.Value)
	}
}

func TestTokenErrorDoesNotExposeSecretsOrBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, "server-secret-detail")
	}))
	defer server.Close()

	source, err := NewAPIKeyTokenSource("api-secret", WithTokenURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = source.Token(context.Background())
	if err == nil {
		t.Fatal("expected token error")
	}
	if strings.Contains(err.Error(), "api-secret") || strings.Contains(err.Error(), "server-secret-detail") {
		t.Fatalf("error leaks sensitive detail: %q", err)
	}
}

func TestDefaultTokenClientRejectsCrossOriginRedirect(t *testing.T) {
	t.Parallel()

	targetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetCalled = true
	}))
	defer target.Close()

	sourceServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer sourceServer.Close()

	source, err := NewAPIKeyTokenSource("api-secret", WithTokenURL(sourceServer.URL))
	if err != nil {
		t.Fatal(err)
	}
	_, err = source.Token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cross-origin redirect") {
		t.Fatalf("unexpected error: %v", err)
	}
	if targetCalled {
		t.Fatal("redirect target received a credential-bearing request")
	}
}

func TestTokenSourceRefreshesNearExpiry(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, `{"access_token":"access-secret","expires_in":20}`)
	}))
	defer server.Close()

	source, err := NewAPIKeyTokenSource("api-secret", WithTokenURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}
	first, err := source.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := source.Token(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Value != second.Value {
		t.Fatal("unexpected token value")
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("token calls = %d, want 2", got)
	}
}

func TestParseExpiresIn(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want time.Duration
		ok   bool
	}{
		{raw: `60`, want: time.Minute, ok: true},
		{raw: `"60"`, want: time.Minute, ok: true},
		{raw: `null`, ok: true},
		{raw: `-1`},
		{raw: `"1 trailing"`},
		{raw: `{}`},
	}
	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			t.Parallel()
			got, err := parseExpiresIn([]byte(test.raw))
			if (err == nil) != test.ok {
				t.Fatalf("err = %v, want success %v", err, test.ok)
			}
			if got != test.want {
				t.Fatalf("duration = %v, want %v", got, test.want)
			}
		})
	}
}
