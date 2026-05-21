package gitlab

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiClient_Get_SetsAuthHeader(t *testing.T) {
	var receivedToken string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedToken = r.Header.Get("PRIVATE-TOKEN")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "glpat-abc123")
	body, err := client.Get(context.Background(), "/test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("unexpected body: %s", body)
	}
	if receivedToken != "glpat-abc123" {
		t.Errorf("expected PRIVATE-TOKEN 'glpat-abc123', got %q", receivedToken)
	}
}

func TestApiClient_Get_ErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "bad-key")
	_, err := client.Get(context.Background(), "/api/v4/projects")

	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestApiClient_GetAll_FollowsPagination(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		page := r.URL.Query().Get("page")

		switch page {
		case "", "1":
			w.Header().Set("X-Next-Page", "2")
			w.Write([]byte(`[1,2]`))
		case "2":
			w.Header().Set("X-Next-Page", "3")
			w.Write([]byte(`[3,4]`))
		case "3":
			// No X-Next-Page header means last page.
			w.Write([]byte(`[5]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "key")
	pages, err := client.GetAll(context.Background(), fmt.Sprintf("%s/api/v4/projects?page=1", server.URL))

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(pages))
	}
	if requestCount != 3 {
		t.Errorf("expected 3 requests, got %d", requestCount)
	}
	if string(pages[0]) != `[1,2]` {
		t.Errorf("unexpected page 0: %s", pages[0])
	}
	if string(pages[2]) != `[5]` {
		t.Errorf("unexpected page 2: %s", pages[2])
	}
}

func TestApiClient_GetAll_SinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`["only"]`))
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "key")
	pages, err := client.GetAll(context.Background(), "/api/v4/projects")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
}

func TestNextPageURL(t *testing.T) {
	tests := []struct {
		name     string
		inputURL string
		nextPage string
		want     string
	}{
		{
			name:     "replaces existing page param",
			inputURL: "https://gitlab.com/api/v4/projects?page=1&per_page=100",
			nextPage: "2",
			want:     "https://gitlab.com/api/v4/projects?page=2&per_page=100",
		},
		{
			name:     "empty next page",
			inputURL: "https://gitlab.com/api/v4/projects?page=3",
			nextPage: "",
			want:     "",
		},
		{
			name:     "zero next page",
			inputURL: "https://gitlab.com/api/v4/projects?page=3",
			nextPage: "0",
			want:     "",
		},
		{
			name:     "appends page when no page param",
			inputURL: "https://gitlab.com/api/v4/projects?per_page=100",
			nextPage: "2",
			want:     "https://gitlab.com/api/v4/projects?per_page=100&page=2",
		},
		{
			name:     "appends page when no query at all",
			inputURL: "https://gitlab.com/api/v4/projects",
			nextPage: "2",
			want:     "https://gitlab.com/api/v4/projects?page=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextPageURL(testStringer{tt.inputURL}, tt.nextPage)
			if got != tt.want {
				t.Errorf("nextPageURL(%q, %q) = %q, want %q", tt.inputURL, tt.nextPage, got, tt.want)
			}
		})
	}
}

// testStringer implements the interface needed by nextPageURL for testing.
type testStringer struct{ s string }

func (s testStringer) String() string { return s.s }
