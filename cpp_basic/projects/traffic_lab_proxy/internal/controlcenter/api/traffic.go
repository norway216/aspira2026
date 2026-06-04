package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aspira2026/traffic_lab_proxy/internal/common"
	"github.com/aspira2026/traffic_lab_proxy/internal/controlcenter/db"
)

type TrafficHandler struct {
	DB db.DB
}

func (h *TrafficHandler) Report(w http.ResponseWriter, r *http.Request) {
	var req common.TrafficReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "invalid request body"})
		return
	}

	if req.NodeID == "" {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "node_id is required"})
		return
	}

	var logs []*common.TrafficRecord
	for _, rec := range req.Records {
		logs = append(logs, &common.TrafficRecord{
			NodeID:          req.NodeID,
			UserID:          rec.UserID,
			UploadBytes:     rec.UploadBytes,
			DownloadBytes:   rec.DownloadBytes,
			ConnectionCount: rec.ConnectionCount,
		})
	}

	if err := h.DB.BatchInsertTrafficLogs(r.Context(), logs); err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	// Update user traffic used
	for _, rec := range req.Records {
		_ = h.DB.AddUserTraffic(r.Context(), rec.UserID, rec.UploadBytes+rec.DownloadBytes)
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Message: "traffic reported"})
}

func (h *TrafficHandler) UserTraffic(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "user_id query parameter required"})
		return
	}

	summary, err := h.DB.GetUserTrafficSummary(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: summary})
}

func (h *TrafficHandler) NodeTraffic(w http.ResponseWriter, r *http.Request) {
	nodeID := r.URL.Query().Get("node_id")
	if nodeID == "" {
		writeJSON(w, http.StatusBadRequest, common.APIResponse{Success: false, Error: "node_id query parameter required"})
		return
	}

	summary, err := h.DB.GetNodeTrafficSummary(r.Context(), nodeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: summary})
}

func (h *TrafficHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	logs, err := h.DB.ListTrafficLogs(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: logs})
}

func (h *TrafficHandler) UserAllTraffic(w http.ResponseWriter, r *http.Request) {
	users, err := h.DB.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	type UserTrafficRow struct {
		Username       string `json:"username"`
		TotalUpload    int64  `json:"total_upload"`
		TotalDownload  int64  `json:"total_download"`
		TrafficUsed    int64  `json:"traffic_used"`
		TrafficTotal   int64  `json:"traffic_total"`
		LastSeen       string `json:"last_seen,omitempty"`
	}

	var rows []UserTrafficRow
	for _, u := range users {
		summary, _ := h.DB.GetUserTrafficSummary(r.Context(), u.Username)
		row := UserTrafficRow{
			Username:      u.Username,
			TrafficUsed:   u.TrafficUsed,
			TrafficTotal:  u.TrafficTotal,
		}
		if summary != nil {
			row.TotalUpload = summary.TotalUpload
			row.TotalDownload = summary.TotalDownload
		}
		rows = append(rows, row)
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: rows})
}

func (h *TrafficHandler) RecentTraffic(w http.ResponseWriter, r *http.Request) {
	logs, err := h.DB.ListTrafficLogs(r.Context(), 200)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, common.APIResponse{Success: false, Error: err.Error()})
		return
	}

	// Group by hour for chart data
	type TimePoint struct {
		Time           string `json:"time"`
		TotalUpload    int64  `json:"total_upload"`
		TotalDownload  int64  `json:"total_download"`
	}

	hourMap := make(map[string]*TimePoint)
	for _, l := range logs {
		hourKey := l.CreatedAt.Truncate(time.Hour).Format(time.RFC3339)
		if _, ok := hourMap[hourKey]; !ok {
			hourMap[hourKey] = &TimePoint{Time: hourKey}
		}
		hourMap[hourKey].TotalUpload += l.UploadBytes
		hourMap[hourKey].TotalDownload += l.DownloadBytes
	}

	var points []TimePoint
	for _, p := range hourMap {
		points = append(points, *p)
	}

	writeJSON(w, http.StatusOK, common.APIResponse{Success: true, Data: points})
}
