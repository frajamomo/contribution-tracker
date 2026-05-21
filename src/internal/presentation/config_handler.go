package presentation

import (
	"net/http"

	"contribution-tracker/internal/application"
)

type ConfigHandler struct {
	configRepo application.ConfigRepository
}

func NewConfigHandler(configRepo application.ConfigRepository) *ConfigHandler {
	return &ConfigHandler{configRepo: configRepo}
}

func (h *ConfigHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	config, err := h.configRepo.FindAll(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load config")
		return
	}

	writeJSON(w, http.StatusOK, config)
}

func (h *ConfigHandler) Set(w http.ResponseWriter, r *http.Request) {
	var req ConfigSetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Key == "" {
		writeError(w, http.StatusBadRequest, "key is required")
		return
	}

	if err := h.configRepo.Set(r.Context(), req.Key, req.Value); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update config")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}
