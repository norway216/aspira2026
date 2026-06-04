package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/db"
)

type NodeHandler struct {
	DB db.DB
}

func (h *NodeHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req common.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	if req.NodeID == "" || req.NodeSecret == "" {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "node_id and node_secret are required"})
		return
	}

	// Check if already exists
	existing, _ := h.DB.GetNode(r.Context(), req.NodeID)

	now := time.Now().UTC()
	node := &common.Node{
		NodeID:           req.NodeID,
		Name:             req.Name,
		IP:               req.IP,
		ProxyPort:        req.ProxyPort,
		MaxBandwidthMbps: req.MaxBandwidthMbps,
		MaxConnections:   req.MaxConnections,
		NodeSecret:       req.NodeSecret,
		Weight:           1,
		Status:           common.NodeStatusOnline,
		LastHeartbeat:    &now,
	}

	if existing != nil {
		// Update existing
		node.CreatedAt = existing.CreatedAt
		node.Weight = existing.Weight
		if err := h.DB.UpsertNode(r.Context(), node); err != nil {
			writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
			return
		}
	} else {
		if err := h.DB.RegisterNode(r.Context(), node); err != nil {
			writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
			return
		}
	}

	writeJSON(w, http.StatusOK, common.APIResponse{
		Success: true,
		Message: "node registered",
		Data:    map[string]string{"node_id": req.NodeID},
	})
}

func (h *NodeHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req common.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	if req.NodeID == "" {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "node_id is required"})
		return
	}

	// Evaluate status based on thresholds
	status := req.Status
	if status == "" || status == common.NodeStatusOnline {
		if req.CPUUsage > 85 || req.MemoryUsage > 90 {
			status = common.NodeStatusDegraded
		} else {
			bandwidthRatio := (req.RxMbps + req.TxMbps) / 100.0 // ratio relative to 100 Mbps default
			if bandwidthRatio > 0.9 || req.CurrentConnections > 900 {
				status = common.NodeStatusDegraded
			} else {
				status = common.NodeStatusOnline
			}
		}
	}

	if err := h.DB.UpdateNodeHeartbeat(r.Context(), req.NodeID, req.CPUUsage, req.MemoryUsage,
		req.CurrentConnections, req.RxMbps, req.TxMbps, status); err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Message: "heartbeat received"})
}

func (h *NodeHandler) List(w http.ResponseWriter, r *http.Request) {
	nodes, err := h.DB.ListNodes(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	// Strip secrets
	for _, n := range nodes {
		n.NodeSecret = ""
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: nodes})
}

func (h *NodeHandler) Get(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	node, err := h.DB.GetNode(r.Context(), nodeID)
	if err != nil {
		if ae, ok := err.(*common.AppError); ok {
			writeJSON(w, ae.Code, common.APIResponse{Success: false, Error: ae.Message})
			return
		}
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	node.NodeSecret = ""
	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: node})
}

func (h *NodeHandler) Disable(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	if err := h.DB.UpdateNodeStatus(r.Context(), nodeID, common.NodeStatusDisabled); err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Message: "node disabled"})
}

func (h *NodeHandler) Enable(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	if err := h.DB.UpdateNodeStatus(r.Context(), nodeID, common.NodeStatusOnline); err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Message: "node enabled"})
}

func (h *NodeHandler) GetPolicies(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")

	// Get node info
	node, err := h.DB.GetNode(r.Context(), nodeID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, common.APIResponse{Success: false, Error: "node not found"})
		return
	}

	// Get all active users and their policies
	users, err := h.DB.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	var userPolicies []common.UserPolicySummary
	for _, user := range users {
		if user.Status != common.UserStatusActive {
			continue
		}

		ups := common.UserPolicySummary{
			UserID:         user.Username,
			Token:          user.Token,
			MaxRateMbps:    user.MaxRateMbps,
			MaxConnections: user.MaxConnections,
			Status:         user.Status,
		}

		// Try to get explicit policy (overrides user defaults)
		policy, _ := h.DB.GetUserPolicy(r.Context(), user.Username)
		if policy != nil {
			ups.MaxRateMbps = policy.MaxRateMbps
			ups.BurstMbps = policy.BurstMbps
			ups.MaxConnections = policy.MaxConnections
			ups.Status = policy.Status
		}

		userPolicies = append(userPolicies, ups)
	}

	writeJSON(w, http.StatusOK, common.PolicyResponse{
		NodeID: nodeID,
		GlobalPolicy: common.GlobalPolicy{
			MaxNodeBandwidthMbps: node.MaxBandwidthMbps,
			MaxNodeConnections:   node.MaxConnections,
		},
		UserPolicies: userPolicies,
	})
}
