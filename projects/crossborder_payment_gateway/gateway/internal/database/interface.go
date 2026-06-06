package database

import "github.com/aspira/crossborder-payment-gateway/internal/models"

type DB interface {
	// Transactions
	CreateTransaction(txn *models.Transaction) error
	GetTransaction(id string) (*models.Transaction, error)
	ListTransactions(query TransactionQuery) ([]models.Transaction, int64, error)
	UpdateTransactionStatus(id string, status models.TransactionStatus) error
	UpdateTransactionStatusValidated(id string, from, to models.TransactionStatus) error
	GetTransactionsByStatus(status models.TransactionStatus) ([]models.Transaction, error)
	GetLastTransactionHash() string

	// Accounts
	CreateAccount(acct *models.Account) error
	GetAccount(id string) (*models.Account, error)
	ListAccounts(merchantID string) ([]models.Account, error)
	UpdateAccountBalance(id string, balance, reserved, dailyUsed, monthlyUsed int64) error
	UpdateAccount(id string, acct *models.Account) error

	// Audit Logs
	CreateAuditLog(log *models.AuditLog) error
	ListAuditLogs(query AuditLogQuery) ([]models.AuditLog, int64, error)

	// Merchants
	CreateMerchant(m *models.Merchant) error
	GetMerchant(id string) (*models.Merchant, error)
	ListMerchants() ([]models.Merchant, error)
	UpdateMerchant(id string, m *models.Merchant) error
	DeleteMerchant(id string) error

	// Users
	GetUserByUsername(username string) (*models.User, error)
	CreateUser(user *models.User) error

	// Exchange Rates
	GetExchangeRates() ([]models.ExchangeRate, error)
	GetExchangeRate(source, target string) (*models.ExchangeRate, error)
	UpsertExchangeRate(rate *models.ExchangeRate) error

	// Dashboard
	GetDashboardStats() (*models.DashboardStats, error)
	GetRecentTransactions(limit int) ([]models.Transaction, error)
	GetTPSHistory(seconds int) ([]models.TPSDataPoint, error)
	GetVolumeHistory(hours int) ([]models.VolumeDataPoint, error)

	// Idempotency
	CreateIdempotencyKey(key string, responseBody string, responseStatus int) error
	GetIdempotencyKey(key string) (responseBody string, responseStatus int, err error)

	// API Keys
	GetMerchantByAPIKey(apiKey string) (*models.Merchant, error)

	// Quotes
	CreateQuote(q *models.Quote) error
	GetQuote(id string) (*models.Quote, error)
	ListQuotes(merchantID string, page, pageSize int) ([]models.Quote, int64, error)
	UpdateQuoteStatus(id string, status models.QuoteStatus) error
	AcceptQuote(id string, orderRef string) error

	// Reconciliation
	CreateReconciliationRecord(r *models.ReconciliationRecord) error
	GetReconciliationRecords(status string, page, pageSize int) ([]models.ReconciliationRecord, int64, error)
	UpdateReconciliationStatus(id int64, status models.ReconciliationStatus, discrepancy string) error
	GetReconciliationSummary() (matched, mismatched, pending, errors int64, err error)

	// Lifecycle
	RunMigrations() error
	Close() error
}

type TransactionQuery struct {
	Status     string
	MerchantID string
	StartDate  string
	EndDate    string
	Search     string
	Page       int
	PageSize   int
}

type AuditLogQuery struct {
	Action     string
	Resource   string
	UserID     string
	StartDate  string
	EndDate    string
	Page       int
	PageSize   int
}
