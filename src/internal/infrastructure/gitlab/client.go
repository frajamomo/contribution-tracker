package gitlab

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ApiClient handles authenticated HTTP communication with the GitLab API.
type ApiClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewApiClient creates a new GitLab API client.
func NewApiClient(baseURL, apiKey string) *ApiClient {
	return &ApiClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

// Get performs an authenticated GET request and returns the response body.
func (c *ApiClient) Get(ctx context.Context, url string) ([]byte, error) {
	fullURL := url
	if !strings.HasPrefix(url, "http") {
		fullURL = c.baseURL + url
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// GetAll performs paginated GET requests following GitLab's X-Next-Page header.
func (c *ApiClient) GetAll(ctx context.Context, url string) ([][]byte, error) {
	var pages [][]byte
	currentURL := url
	if !strings.HasPrefix(currentURL, "http") {
		currentURL = c.baseURL + currentURL
	}

	for currentURL != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, currentURL, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("PRIVATE-TOKEN", c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("executing request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response body: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
		}

		pages = append(pages, body)
		currentURL = nextPageURL(req.URL, resp.Header.Get("X-Next-Page"))
	}

	return pages, nil
}

// nextPageURL constructs the URL for the next page from the X-Next-Page header.
// If the header is empty or "0", there is no next page.
func nextPageURL(currentURL interface{ String() string }, nextPage string) string {
	if nextPage == "" || nextPage == "0" {
		return ""
	}

	// Parse the current URL to replace/add the page parameter.
	raw := currentURL.String()

	hasPageParam := false
	if qIdx := strings.Index(raw, "?"); qIdx >= 0 {
		params := strings.Split(raw[qIdx+1:], "&")
		for _, p := range params {
			if p == "page" || strings.HasPrefix(p, "page=") {
				hasPageParam = true
				break
			}
		}
	}

	if hasPageParam {
		parts := strings.SplitN(raw, "?", 2)
		if len(parts) != 2 {
			return ""
		}
		params := strings.Split(parts[1], "&")
		for i, p := range params {
			if p == "page" || strings.HasPrefix(p, "page=") {
				params[i] = "page=" + nextPage
			}
		}
		return parts[0] + "?" + strings.Join(params, "&")
	}

	// Append page parameter.
	if strings.Contains(raw, "?") {
		return raw + "&page=" + nextPage
	}
	return raw + "?page=" + nextPage
}
