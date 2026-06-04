package api

import (
	"net/http"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/scheduler"
)

type SchedulerHandler struct {
	Engine *scheduler.Engine
}

func (h *SchedulerHandler) Schedule(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "anonymous"
	}

	node, err := h.Engine.SelectNode(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, common.APIResponse{
			Success: false,
			Error:   "no available node: " + err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, common.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"node_id":    node.NodeID,
			"ip":         node.IP,
			"proxy_port": node.ProxyPort,
			"strategy":   h.Engine.CurrentStrategy(),
		},
	})
}

func (h *SchedulerHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.Engine.ListEvents(r.Context(), 200)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: events})
}

func (h *SchedulerHandler) SetStrategy(w http.ResponseWriter, r *http.Request) {
	strategy := r.URL.Query().Get("strategy")
	if strategy == "" {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "strategy query parameter required"})
		return
	}
	if err := h.Engine.SetStrategy(strategy); err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Message: "strategy changed to " + strategy})
}
