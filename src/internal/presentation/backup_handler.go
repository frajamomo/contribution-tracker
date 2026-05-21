package presentation

import (
	"encoding/json"
	"net/http"

	"contribution-tracker/internal/application"
	"contribution-tracker/internal/domain"
)

type BackupHandler struct {
	backupService application.BackupServicePort
}

func NewBackupHandler(backupService application.BackupServicePort) *BackupHandler {
	return &BackupHandler{backupService: backupService}
}

func (h *BackupHandler) Export(w http.ResponseWriter, r *http.Request) {
	backup, err := h.backupService.Export(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to export backup")
		return
	}

	writeJSON(w, http.StatusOK, backup)
}

func (h *BackupHandler) Restore(w http.ResponseWriter, r *http.Request) {
	var backup domain.BackupFile
	if err := json.NewDecoder(r.Body).Decode(&backup); err != nil {
		writeError(w, http.StatusBadRequest, "invalid backup data")
		return
	}

	if err := h.backupService.Restore(r.Context(), &backup); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to restore backup")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "restored"})
}
