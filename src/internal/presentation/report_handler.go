package presentation

import (
	"net/http"
	"time"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
)

type ReportHandler struct {
	reportService application.ReportServicePort
}

func NewReportHandler(reportService application.ReportServicePort) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

func (h *ReportHandler) StreamReport(w http.ResponseWriter, r *http.Request) {
	authCtx := GetAuthContext(r)
	if authCtx == nil {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req ReportRequestDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	since, err := time.Parse("2006-01-02", req.Since)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid since date")
		return
	}

	until, err := time.Parse("2006-01-02", req.Until)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid until date")
		return
	}

	var types []domain.ActivityType
	for _, t := range req.Types {
		switch t {
		case "COMMIT":
			types = append(types, domain.ActivityTypeCommit)
		case "MERGED_PR":
			types = append(types, domain.ActivityTypeMergedPullRequest)
		case "ISSUE":
			types = append(types, domain.ActivityTypeIssue)
		case "PR_REVIEW":
			types = append(types, domain.ActivityTypePullRequestReview)
		}
	}

	query := application.ReportQuery{
		TeamID:      req.TeamID,
		MemberID:    req.MemberID,
		CallerID:    authCtx.UserID,
		CallerRoles: authCtx.Roles,
		Since:       since,
		Until:       until,
		Types:       types,
		ReportType:  domain.ReportType(req.ReportType),
	}

	sse, err := NewSSEWriter(w)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	events := make(chan application.ReportEvent, 16)
	go h.reportService.GenerateReport(r.Context(), query, events)

	for event := range events {
		switch e := event.(type) {
		case *application.UserReportEvent:
			dto := UserReportToDTO(e.Report)
			sse.WriteEvent("USER_REPORT", SSEEventData{
				Type:   "USER_REPORT",
				Report: &dto,
			})
		case *application.ReportCompleteEvent:
			sse.WriteEvent("COMPLETE", SSEEventData{Type: "COMPLETE"})
		case *application.ReportErrorEvent:
			sse.WriteEvent("ERROR", SSEEventData{
				Type:  "ERROR",
				Error: e.Message,
			})
		}
	}
}
