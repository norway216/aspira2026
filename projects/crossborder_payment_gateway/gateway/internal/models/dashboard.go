package models

type DashboardStats struct {
	TotalTransactions int64   `json:"total_transactions"`
	TodayVolume       int64   `json:"today_volume"`
	TodayCount        int64   `json:"today_count"`
	SuccessRate       float64 `json:"success_rate"`
	ActiveAccounts    int64   `json:"active_accounts"`
	PendingCount      int64   `json:"pending_count"`
	EngineConnected   bool    `json:"engine_connected"`
	CurrentTPS        float64 `json:"current_tps"`
}

type TPSDataPoint struct {
	Timestamp int64   `json:"timestamp"`
	TPS       float64 `json:"tps"`
}
