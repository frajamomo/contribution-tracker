package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// ApiClient handles authenticated HTTP communication with the GitHub API.
type ApiClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewApiClient creates a new GitHub API client.
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

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/vnd.github+json")

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

// GetAll performs paginated GET requests following GitHub's Link header with rel="next".
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

		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Accept", "application/vnd.github+json")

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
		currentURL = parseNextLink(resp.Header.Get("Link"))
	}

	return pages, nil
}

// linkNextRe matches the "next" relation in GitHub's Link header.
var linkNextRe = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// parseNextLink extracts the URL for the next page from a GitHub Link header.
func parseNextLink(header string) string {
	if header == "" {
		return ""
	}
	matches := linkNextRe.FindStringSubmatch(header)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}
