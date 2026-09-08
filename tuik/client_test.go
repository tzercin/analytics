package tuik

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClientRejectsRequestOutsideBaseURLBeforeGettingToken(t *testing.T) {
	t.Parallel()

	called := false
	client, err := NewClient(TokenSourceFunc(func(context.Context) (AccessToken, error) {
		called = true
		return AccessToken{Value: "secret"}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodGet, "https://example.com/collect", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.Do(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "configured base URL origin") {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Fatal("token source called for rejected URL")
	}
}

func TestClientReturnsSanitizedHTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer access-secret" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("X-Request-ID", "test-id")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "diagnostic-body")
	}))
	defer server.Close()

	client, err := NewClient(
		StaticTokenSource("access-secret"),
		WithBaseURL(server.URL+"/rest"),
	)
	if err != nil {
		t.Fatal(err)
	}

	request, err := http.NewRequest(http.MethodGet, server.URL+"/rest/dataflow/TR/all?private=query", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(context.Background(), request)
	if response != nil {
		t.Fatal("expected nil response")
	}
	var httpError *HTTPError
	if !errors.As(err, &httpError) {
		t.Fatalf("error type = %T, want *HTTPError", err)
	}
	if httpError.StatusCode != http.StatusBadRequest {
		t.Fatalf("status code = %d", httpError.StatusCode)
	}
	if got := string(httpError.Body); got != "diagnostic-body" {
		t.Fatalf("body = %q", got)
	}
	if got := httpError.Header.Get("X-Request-ID"); got != "test-id" {
		t.Fatalf("request ID = %q", got)
	}
	if strings.Contains(err.Error(), "private") || strings.Contains(err.Error(), "diagnostic-body") || strings.Contains(err.Error(), "access-secret") {
		t.Fatalf("error leaks request or response detail: %q", err)
	}
}

func TestDefaultClientRejectsCrossOriginRedirect(t *testing.T) {
	t.Parallel()

	targetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetCalled = true
	}))
	defer target.Close()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	client, err := NewClient(
		StaticTokenSource("secret"),
		WithBaseURL(source.URL+"/rest"),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Structure(context.Background(), StructureQuery{
		Resource: Dataflows,
		Agency:   "TR",
		ID:       "all",
	})
	if err == nil || !strings.Contains(err.Error(), "cross-origin redirect") {
		t.Fatalf("unexpected error: %v", err)
	}
	if targetCalled {
		t.Fatal("redirect target received a credential-bearing request")
	}
}
