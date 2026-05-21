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

func TestStreamReport_Success(t *testing.T) {
	svc := &mockReportService{
		generateFn: func(ctx context.Context, query application.ReportQuery, out chan<- application.ReportEvent) {
			defer close(out)
			out <- &application.UserReportEvent{
				Report: application.UserReport{
					User: domain.User{ID: "u-1", Username: "alice", DisplayName: "Alice"},
					Counts: []application.ActivityCount{
						{Type: domain.ActivityTypeCommit, Count: 5},
					},
				},
			}
			out <- &application.ReportCompleteEvent{}
		},
	}

	handler := NewReportHandler(svc)
	body := strings.NewReader(`{"teamId":"t-1","since":"2024-01-01","until":"2024-01-31","types":["COMMIT"],"reportType":"ACTIVITY_LOG"}`)
	req := newAuthenticatedRequest(http.MethodPost, "/api/reports/stream", body, &application.AuthContext{
		AccountID: "acc-1",
		UserID:    "u-1",
		Roles:     map[domain.Role]bool{domain.RoleTeamMember: true},
	})

	rr := httptest.NewRecorder()
	handler.StreamReport(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	if ct := rr.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("expected Content-Type text/event-stream, got %s", ct)
	}

	respBody := rr.Body.String()
	if !strings.Contains(respBody, "event: USER_REPORT") {
		t.Error("expected USER_REPORT event in response")
	}
	if !strings.Contains(respBody, "event: COMPLETE") {
		t.Error("expected COMPLETE event in response")
	}
}

func TestStreamReport_InvalidDates(t *testing.T) {
	handler := NewReportHandler(&mockReportService{})
	body := strings.NewReader(`{"teamId":"t-1","since":"not-a-date","until":"2024-01-31","types":[],"reportType":"ACTIVITY_LOG"}`)
	req := newAuthenticatedRequest(http.MethodPost, "/api/reports/stream", body, &application.AuthContext{
		UserID: "u-1",
		Roles:  map[domain.Role]bool{domain.RoleTeamMember: true},
	})

	rr := httptest.NewRecorder()
	handler.StreamReport(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

func TestStreamReport_ErrorEvent(t *testing.T) {
	svc := &mockReportService{
		generateFn: func(ctx context.Context, query application.ReportQuery, out chan<- application.ReportEvent) {
			defer close(out)
			out <- &application.ReportErrorEvent{Message: "fetch failed"}
		},
	}

	handler := NewReportHandler(svc)
	body := strings.NewReader(`{"teamId":"t-1","since":"2024-01-01","until":"2024-01-31","types":[],"reportType":"ACTIVITY_LOG"}`)
	req := newAuthenticatedRequest(http.MethodPost, "/api/reports/stream", body, &application.AuthContext{
		UserID: "u-1",
		Roles:  map[domain.Role]bool{domain.RoleTeamMember: true},
	})

	rr := httptest.NewRecorder()
	handler.StreamReport(rr, req)

	if !strings.Contains(rr.Body.String(), "fetch failed") {
		t.Error("expected error message in response")
	}
}

func TestStreamReport_QueryFields(t *testing.T) {
	var capturedQuery application.ReportQuery
	svc := &mockReportService{
		generateFn: func(ctx context.Context, query application.ReportQuery, out chan<- application.ReportEvent) {
			capturedQuery = query
			close(out)
		},
	}

	handler := NewReportHandler(svc)
	body := strings.NewReader(`{"teamId":"t-1","since":"2024-03-01","until":"2024-03-31","types":["COMMIT","MERGED_PR"],"reportType":"ACTIVITY_LOG"}`)
	req := newAuthenticatedRequest(http.MethodPost, "/api/reports/stream", body, &application.AuthContext{
		UserID: "u-1",
		Roles:  map[domain.Role]bool{domain.RoleTeamLeader: true},
	})

	rr := httptest.NewRecorder()
	handler.StreamReport(rr, req)

	_ = rr
	_ = json.NewEncoder

	if capturedQuery.TeamID != "t-1" {
		t.Errorf("expected team ID t-1, got %s", capturedQuery.TeamID)
	}
	if capturedQuery.CallerID != "u-1" {
		t.Errorf("expected caller ID u-1, got %s", capturedQuery.CallerID)
	}
	expected := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	if !capturedQuery.Since.Equal(expected) {
		t.Errorf("expected since %v, got %v", expected, capturedQuery.Since)
	}
	if len(capturedQuery.Types) != 2 {
		t.Errorf("expected 2 types, got %d", len(capturedQuery.Types))
	}
}
