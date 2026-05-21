package presentation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
)

func TestExport_Success(t *testing.T) {
	svc := &mockBackupService{
		exportFn: func(ctx context.Context) (*domain.BackupFile, error) {
			return &domain.BackupFile{
				Metadata: domain.BackupMetadata{
					AppVersion: "1.0",
					ExportedAt: time.Now(),
				},
				Users: []domain.User{{ID: "u-1", Username: "alice"}},
			}, nil
		},
	}

	handler := NewBackupHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/backup", nil)
	rr := httptest.NewRecorder()

	handler.Export(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var backup domain.BackupFile
	json.NewDecoder(rr.Body).Decode(&backup)
	if len(backup.Users) != 1 {
		t.Errorf("expected 1 user, got %d", len(backup.Users))
	}
}

func TestExport_Error(t *testing.T) {
	svc := &mockBackupService{
		exportFn: func(ctx context.Context) (*domain.BackupFile, error) {
			return nil, application.NewInternalError("db error", nil)
		},
	}

	handler := NewBackupHandler(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/admin/backup", nil)
	rr := httptest.NewRecorder()

	handler.Export(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rr.Code)
	}
}

func TestRestore_Success(t *testing.T) {
	var restored bool
	svc := &mockBackupService{
		restoreFn: func(ctx context.Context, data *domain.BackupFile) error {
			restored = true
			return nil
		},
	}

	handler := NewBackupHandler(svc)
	body := strings.NewReader(`{"metadata":{"appVersion":"1.0"},"users":[{"id":"u-1","username":"alice"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/restore", body)
	rr := httptest.NewRecorder()

	handler.Restore(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !restored {
		t.Error("restore was not called")
	}
}

func TestRestore_InvalidBody(t *testing.T) {
	handler := NewBackupHandler(&mockBackupService{})
	body := strings.NewReader(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/restore", body)
	rr := httptest.NewRecorder()

	handler.Restore(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}
