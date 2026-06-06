package models

import "time"

type ExchangeRate struct {
	ID         int64     `json:"id" db:"id"`
	Source     string    `json:"source" db:"source"`
	Target     string    `json:"target" db:"target"`
	Rate       float64   `json:"rate" db:"rate"`
	Bid        float64   `json:"bid" db:"bid"`
	Ask        float64   `json:"ask" db:"ask"`
	SourceName string    `json:"source_name" db:"source_name"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type UpsertExchangeRateRequest struct {
	Source     string  `json:"source" binding:"required"`
	Target     string  `json:"target" binding:"required"`
	Rate       float64 `json:"rate" binding:"required"`
	Bid        float64 `json:"bid"`
	Ask        float64 `json:"ask"`
	SourceName string  `json:"source_name"`
}
