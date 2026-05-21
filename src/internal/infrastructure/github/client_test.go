package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiClient_Get_SetsAuthHeader(t *testing.T) {
	var receivedAuth string
	var receivedAccept string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedAccept = r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "test-token-123")
	body, err := client.Get(context.Background(), "/test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("unexpected body: %s", body)
	}
	if receivedAuth != "Bearer test-token-123" {
		t.Errorf("expected Authorization 'Bearer test-token-123', got %q", receivedAuth)
	}
	if receivedAccept != "application/vnd.github+json" {
		t.Errorf("expected Accept 'application/vnd.github+json', got %q", receivedAccept)
	}
}

func TestApiClient_Get_ErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Not Found"}`))
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "key")
	_, err := client.Get(context.Background(), "/missing")

	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestApiClient_GetAll_FollowsPagination(t *testing.T) {
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		switch r.URL.Path {
		case "/page1":
			nextURL := fmt.Sprintf("http://%s/page2", r.Host)
			w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="next"`, nextURL))
			w.Write([]byte(`[1,2]`))
		case "/page2":
			nextURL := fmt.Sprintf("http://%s/page3", r.Host)
			w.Header().Set("Link", fmt.Sprintf(`<%s>; rel="next"`, nextURL))
			w.Write([]byte(`[3,4]`))
		case "/page3":
			w.Write([]byte(`[5]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewApiClient(server.URL, "key")
	pages, err := client.GetAll(context.Background(), "/page1")

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
	pages, err := client.GetAll(context.Background(), "/single")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
}

func TestParseNextLink(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{
			name:   "with next",
			header: `<https://api.github.com/repos?page=2>; rel="next", <https://api.github.com/repos?page=5>; rel="last"`,
			want:   "https://api.github.com/repos?page=2",
		},
		{
			name:   "no next",
			header: `<https://api.github.com/repos?page=5>; rel="last"`,
			want:   "",
		},
		{
			name:   "empty header",
			header: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseNextLink(tt.header)
			if got != tt.want {
				t.Errorf("parseNextLink(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}
