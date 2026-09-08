package tuik

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStructureBuildsDocumentedQuery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.EscapedPath(), "/rest/dataflow/TR/all/latest"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if got, want := r.URL.RawQuery, "detail=full&references=children"; got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("User-Agent"); got != defaultUserAgent {
			t.Errorf("User-Agent = %q", got)
		}
		_, _ = io.WriteString(w, "metadata")
	}))
	defer server.Close()

	client, err := NewClient(StaticTokenSource("test-token"), WithBaseURL(server.URL+"/rest"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Structure(context.Background(), StructureQuery{
		Resource:   Dataflows,
		Agency:     "TR",
		ID:         "all",
		Version:    "latest",
		Detail:     "full",
		References: "children",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if got, err := io.ReadAll(response.Body); err != nil || string(got) != "metadata" {
		t.Fatalf("body = %q, err = %v", got, err)
	}
}

func TestDataBuildsDocumentedFilters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.EscapedPath(), "/rest/data/TR,example,1.1/A.TR+OTHER...3"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if got, want := r.URL.RawQuery, "endPeriod=2010&format=SDMX-CSV&startPeriod=2008"; got != want {
			t.Errorf("query = %q, want %q", got, want)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewClient(StaticTokenSource("test-token"), WithBaseURL(server.URL+"/rest"))
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Data(context.Background(), DataQuery{
		Agency:      "TR",
		Dataflow:    "example",
		Version:     "1.1",
		Key:         "A.TR+OTHER...3",
		StartPeriod: "2008",
		EndPeriod:   "2010",
		Format:      FormatSDMXCSV,
	})
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
}

func TestQueriesRejectPathDelimiters(t *testing.T) {
	t.Parallel()

	client, err := NewClient(StaticTokenSource("test-token"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		call func() error
	}{
		{
			name: "structure ID",
			call: func() error {
				_, err := client.Structure(context.Background(), StructureQuery{Resource: Dataflows, Agency: "TR", ID: "../secret"})
				return err
			},
		},
		{
			name: "series key",
			call: func() error {
				_, err := client.Data(context.Background(), DataQuery{Agency: "TR", Dataflow: "flow", Version: "1.0", Key: "x?format=JSON"})
				return err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.call(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
