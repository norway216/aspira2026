package api

import (
	"encoding/json"
	"net/http"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/db"
)

type PolicyHandler struct {
	DB db.DB
}

func (h *PolicyHandler) List(w http.ResponseWriter, r *http.Request) {
	policies, err := h.DB.ListPolicies(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: policies})
}

func (h *PolicyHandler) Upsert(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "user_id is required"})
		return
	}

	var policy common.Policy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	policy.UserID = userID
	if policy.Status == "" {
		policy.Status = common.PolicyStatusActive
	}
	if policy.MaxRateMbps <= 0 {
		policy.MaxRateMbps = 10
	}
	if policy.BurstMbps <= 0 {
		policy.BurstMbps = policy.MaxRateMbps * 2
	}
	if policy.MaxConnections <= 0 {
		policy.MaxConnections = 5
	}

	if err := h.DB.UpsertPolicy(r.Context(), &policy); err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Message: "policy saved", Data: policy})
}

func (h *PolicyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "user_id is required"})
		return
	}

	if err := h.DB.DeletePolicy(r.Context(), userID); err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Message: "policy deleted"})
}
