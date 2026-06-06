package models

import "time"

type Merchant struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	APIKey       string    `json:"api_key" db:"api_key"`
	APISecret    string    `json:"-" db:"api_secret"`
	Status       string    `json:"status" db:"status"`
	DailyLimit   int64     `json:"daily_limit" db:"daily_limit"`
	MonthlyLimit int64     `json:"monthly_limit" db:"monthly_limit"`
	CallbackURL  string    `json:"callback_url" db:"callback_url"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type CreateMerchantRequest struct {
	Name         string `json:"name" binding:"required"`
	DailyLimit   int64  `json:"daily_limit"`
	MonthlyLimit int64  `json:"monthly_limit"`
	CallbackURL  string `json:"callback_url"`
}

type UpdateMerchantRequest struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	DailyLimit   int64  `json:"daily_limit"`
	MonthlyLimit int64  `json:"monthly_limit"`
	CallbackURL  string `json:"callback_url"`
}
