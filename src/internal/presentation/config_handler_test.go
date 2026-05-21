package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConfigGetAll_Success(t *testing.T) {
	repo := &mockConfigRepo{
		findAllFn: func(ctx context.Context) (map[string]string, error) {
			return map[string]string{"github_api_key": "ghp_xxx", "gitlab_api_key": "glpat-xxx"}, nil
		},
	}

	handler := NewConfigHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/config", nil)
	rr := httptest.NewRecorder()

	handler.GetAll(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var config map[string]string
	json.NewDecoder(rr.Body).Decode(&config)
	if config["github_api_key"] != "ghp_xxx" {
		t.Errorf("expected github_api_key ghp_xxx, got %s", config["github_api_key"])
	}
}

func TestConfigSet_Success(t *testing.T) {
	var setKey, setValue string
	repo := &mockConfigRepo{
		setFn: func(ctx context.Context, key, value string) error {
			setKey = key
			setValue = value
			return nil
		},
	}

	handler := NewConfigHandler(repo)
	body := strings.NewReader(`{"key":"github_api_key","value":"ghp_new"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/config", body)
	rr := httptest.NewRecorder()

	handler.Set(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if setKey != "github_api_key" || setValue != "ghp_new" {
		t.Errorf("expected key=github_api_key value=ghp_new, got key=%s value=%s", setKey, setValue)
	}
}

func TestConfigSet_MissingKey(t *testing.T) {
	handler := NewConfigHandler(&mockConfigRepo{})
	body := strings.NewReader(`{"key":"","value":"something"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/config", body)
	rr := httptest.NewRecorder()

	handler.Set(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
