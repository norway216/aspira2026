package database

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/aspira/crossborder-payment-gateway/internal/models"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type SQLiteDB struct {
	db *sql.DB
}

func NewSQLite(dsn string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1) // SQLite single-writer mode
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	sqliteDB := &SQLiteDB{db: db}
	if err := sqliteDB.RunMigrations(); err != nil {
		return nil, fmt.Errorf("migrations failed: %w", err)
	}

	return sqliteDB, nil
}

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

func (s *SQLiteDB) RunMigrations() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL DEFAULT 'merchant',
			email TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS merchants (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			api_key TEXT NOT NULL,
			api_secret TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			daily_limit INTEGER DEFAULT 100000000,
			monthly_limit INTEGER DEFAULT 1000000000,
			callback_url TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			merchant_id TEXT NOT NULL,
			currency TEXT NOT NULL,
			balance INTEGER DEFAULT 0,
			reserved_balance INTEGER DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'active',
			daily_limit INTEGER DEFAULT 100000000,
			daily_used INTEGER DEFAULT 0,
			monthly_limit INTEGER DEFAULT 1000000000,
			monthly_used INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS transactions (
			id TEXT PRIMARY KEY,
			merchant_id TEXT NOT NULL,
			payer_account_id TEXT NOT NULL,
			payee_account_id TEXT NOT NULL,
			source_currency TEXT NOT NULL,
			target_currency TEXT NOT NULL,
			source_amount INTEGER NOT NULL,
			target_amount INTEGER DEFAULT 0,
			exchange_rate REAL DEFAULT 0,
			fee INTEGER DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'pending',
			description TEXT DEFAULT '',
			reference_id TEXT DEFAULT '',
			callback_url TEXT DEFAULT '',
			hash_chain_prev TEXT DEFAULT '',
			hash_chain_curr TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT DEFAULT '',
			action TEXT NOT NULL,
			resource TEXT NOT NULL,
			resource_id TEXT DEFAULT '',
			ip_addr TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			detail TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS exchange_rates (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			target TEXT NOT NULL,
			rate REAL NOT NULL,
			bid REAL DEFAULT 0,
			ask REAL DEFAULT 0,
			source_name TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source, target)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_merchant ON transactions(merchant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_transactions_created ON transactions(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_merchant ON accounts(merchant_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("migration error: %w\nSQL: %s", err, m)
		}
	}

	// Seed data
	return s.seed()
}

func (s *SQLiteDB) seed() error {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// Admin user (password: admin123)
	adminPassHash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}
	adminID := "usr-admin-001"
	if _, err := s.db.Exec(
		"INSERT INTO users (id, username, password_hash, role, email) VALUES (?, ?, ?, ?, ?)",
		adminID, "admin", string(adminPassHash), "admin", "admin@aspira.com",
	); err != nil {
		return err
	}

	// Seed merchants
	merchants := []struct {
		id, name, apiKey, apiSecret, callbackURL string
		dailyLimit, monthlyLimit                 int64
	}{
		{"merchant-001", "环球贸易", "ak-merchant001-test", "sk-secret-001", "https://merchant1.example.com/callback", 100000000, 1000000000},
		{"merchant-002", "跨境通", "ak-merchant002-test", "sk-secret-002", "https://merchant2.example.com/callback", 50000000, 500000000},
	}
	for _, m := range merchants {
		if _, err := s.db.Exec(
			"INSERT INTO merchants (id, name, api_key, api_secret, status, daily_limit, monthly_limit, callback_url) VALUES (?, ?, ?, ?, 'active', ?, ?, ?)",
			m.id, m.name, m.apiKey, m.apiSecret, m.dailyLimit, m.monthlyLimit, m.callbackURL,
		); err != nil {
			return err
		}
	}

	// Seed accounts (USD and CNY for each merchant)
	accounts := []struct {
		id, merchantID, currency string
		balance                  int64
	}{
		{"acct-001", "merchant-001", "USD", 100000000}, // 1M USD
		{"acct-002", "merchant-001", "CNY", 10000000000}, // 100M CNY
		{"acct-003", "merchant-002", "USD", 50000000},  // 500K USD
		{"acct-004", "merchant-002", "CNY", 5000000000}, // 50M CNY
	}
	for _, a := range accounts {
		if _, err := s.db.Exec(
			"INSERT INTO accounts (id, merchant_id, currency, balance, reserved_balance, status) VALUES (?, ?, ?, ?, 0, 'active')",
			a.id, a.merchantID, a.currency, a.balance,
		); err != nil {
			return err
		}
	}

	// Seed exchange rates
	rates := []struct {
		source, target string
		rate, bid, ask float64
		sourceName     string
	}{
		{"USD", "CNY", 7.2530, 7.2480, 7.2580, "CFETS"},
		{"USD", "EUR", 0.9215, 0.9200, 0.9230, "ECB"},
		{"USD", "JPY", 155.75, 155.50, 156.00, "BOJ"},
		{"USD", "GBP", 0.7910, 0.7895, 0.7925, "BOE"},
		{"EUR", "CNY", 7.8740, 7.8680, 7.8800, "CFETS"},
	}
	for _, r := range rates {
		if _, err := s.db.Exec(
			"INSERT OR IGNORE INTO exchange_rates (source, target, rate, bid, ask, source_name) VALUES (?, ?, ?, ?, ?, ?)",
			r.source, r.target, r.rate, r.bid, r.ask, r.sourceName,
		); err != nil {
			return err
		}
	}

	// Seed sample transactions
	statuses := []models.TransactionStatus{
		models.TxnCompleted, models.TxnCompleted, models.TxnCompleted,
		models.TxnCompleted, models.TxnFailed, models.TxnCompleted,
		models.TxnRefunded, models.TxnCompleted, models.TxnPending,
		models.TxnCompleted,
	}
	prevHash := "0000000000000000000000000000000000000000000000000000000000000000" // genesis hash

	for i, status := range statuses {
		txnID := fmt.Sprintf("txn-sample-%03d", i+1)
		srcAmt := int64(10000 + i*5000)
		rate := 7.25
		fee := int64(150 + i*10)
		tgtAmt := int64(float64(srcAmt-fee) * rate)
		ts := time.Now().Add(-time.Duration(len(statuses)-i) * 2 * time.Hour)

		hashInput := fmt.Sprintf("%s|%s|%d|CNY|%d|%d", prevHash, txnID, srcAmt, tgtAmt, ts.UnixNano())
		hash := sha256.Sum256([]byte(hashInput))
		currHash := hex.EncodeToString(hash[:])
		prevHashCopy := prevHash

		if _, err := s.db.Exec(
			`INSERT INTO transactions (id, merchant_id, payer_account_id, payee_account_id,
			 source_currency, target_currency, source_amount, target_amount, exchange_rate, fee,
			 status, description, reference_id, hash_chain_prev, hash_chain_curr,
			 created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			txnID, "merchant-001", "acct-002", "acct-001",
			"CNY", "USD", srcAmt, tgtAmt, rate, fee,
			string(status), "跨境支付测试", fmt.Sprintf("REF-%03d", i+1),
			prevHashCopy, currHash,
			ts, ts,
		); err != nil {
			return err
		}
		prevHash = currHash
	}

	// Seed audit logs
	for i := 0; i < 5; i++ {
		txnID := fmt.Sprintf("txn-sample-%03d", i+1)
		ts := time.Now().Add(-time.Duration(5-i) * 3 * time.Hour)
		if _, err := s.db.Exec(
			"INSERT INTO audit_logs (user_id, action, resource, resource_id, ip_addr, user_agent, detail, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			adminID, "txn.create", "transaction", txnID, "127.0.0.1", "Gateway/1.0", fmt.Sprintf(`{"amount":%d}`, 10000+i*5000), ts,
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *SQLiteDB) CreateTransaction(txn *models.Transaction) error {
	_, err := s.db.Exec(
		`INSERT INTO transactions (id, merchant_id, payer_account_id, payee_account_id,
		 source_currency, target_currency, source_amount, target_amount, exchange_rate, fee,
		 status, description, reference_id, callback_url, hash_chain_prev, hash_chain_curr,
		 created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		txn.ID, txn.MerchantID, txn.PayerAccountID, txn.PayeeAccountID,
		txn.SourceCurrency, txn.TargetCurrency, txn.SourceAmount, txn.TargetAmount,
		txn.ExchangeRate, txn.Fee, string(txn.Status), txn.Description, txn.ReferenceID,
		txn.CallbackURL, txn.HashChainPrev, txn.HashChainCurr,
		txn.CreatedAt, txn.UpdatedAt,
	)
	return err
}

func (s *SQLiteDB) GetTransaction(id string) (*models.Transaction, error) {
	txn := &models.Transaction{}
	var status string
	err := s.db.QueryRow(
		`SELECT id, merchant_id, payer_account_id, payee_account_id,
		 source_currency, target_currency, source_amount, target_amount, exchange_rate, fee,
		 status, description, reference_id, callback_url, hash_chain_prev, hash_chain_curr,
		 created_at, updated_at FROM transactions WHERE id = ?`, id,
	).Scan(&txn.ID, &txn.MerchantID, &txn.PayerAccountID, &txn.PayeeAccountID,
		&txn.SourceCurrency, &txn.TargetCurrency, &txn.SourceAmount, &txn.TargetAmount,
		&txn.ExchangeRate, &txn.Fee, &status, &txn.Description, &txn.ReferenceID,
		&txn.CallbackURL, &txn.HashChainPrev, &txn.HashChainCurr,
		&txn.CreatedAt, &txn.UpdatedAt)
	if err != nil {
		return nil, err
	}
	txn.Status = models.TransactionStatus(status)
	return txn, nil
}

func (s *SQLiteDB) ListTransactions(query TransactionQuery) ([]models.Transaction, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}

	if query.Status != "" {
		where += " AND status = ?"
		args = append(args, query.Status)
	}
	if query.MerchantID != "" {
		where += " AND merchant_id = ?"
		args = append(args, query.MerchantID)
	}
	if query.StartDate != "" {
		where += " AND created_at >= ?"
		args = append(args, query.StartDate)
	}
	if query.EndDate != "" {
		where += " AND created_at <= ?"
		args = append(args, query.EndDate)
	}
	if query.Search != "" {
		where += " AND (id LIKE ? OR reference_id LIKE ? OR description LIKE ?)"
		s := "%" + query.Search + "%"
		args = append(args, s, s, s)
	}

	var total int64
	if err := s.db.QueryRow("SELECT COUNT(*) FROM transactions "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	offset := (query.Page - 1) * query.PageSize

	rows, err := s.db.Query(
		`SELECT id, merchant_id, payer_account_id, payee_account_id,
		 source_currency, target_currency, source_amount, target_amount, exchange_rate, fee,
		 status, description, reference_id, callback_url, hash_chain_prev, hash_chain_curr,
		 created_at, updated_at FROM transactions `+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		append(args, query.PageSize, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var txns []models.Transaction
	for rows.Next() {
		var txn models.Transaction
		var status string
		if err := rows.Scan(&txn.ID, &txn.MerchantID, &txn.PayerAccountID, &txn.PayeeAccountID,
			&txn.SourceCurrency, &txn.TargetCurrency, &txn.SourceAmount, &txn.TargetAmount,
			&txn.ExchangeRate, &txn.Fee, &status, &txn.Description, &txn.ReferenceID,
			&txn.CallbackURL, &txn.HashChainPrev, &txn.HashChainCurr,
			&txn.CreatedAt, &txn.UpdatedAt); err != nil {
			return nil, 0, err
		}
		txn.Status = models.TransactionStatus(status)
		txns = append(txns, txn)
	}
	return txns, total, nil
}

func (s *SQLiteDB) UpdateTransactionStatus(id string, status models.TransactionStatus) error {
	_, err := s.db.Exec("UPDATE transactions SET status = ?, updated_at = ? WHERE id = ?",
		string(status), time.Now(), id)
	return err
}

func (s *SQLiteDB) GetTransactionsByStatus(status models.TransactionStatus) ([]models.Transaction, error) {
	txns, _, err := s.ListTransactions(TransactionQuery{Status: string(status), Page: 1, PageSize: 1000})
	return txns, err
}

func (s *SQLiteDB) CreateAccount(acct *models.Account) error {
	_, err := s.db.Exec(
		`INSERT INTO accounts (id, merchant_id, currency, balance, reserved_balance, status, daily_limit, daily_used, monthly_limit, monthly_used, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		acct.ID, acct.MerchantID, acct.Currency, acct.Balance, acct.ReservedBalance,
		acct.Status, acct.DailyLimit, acct.DailyUsed, acct.MonthlyLimit, acct.MonthlyUsed,
		acct.CreatedAt, acct.UpdatedAt,
	)
	return err
}

func (s *SQLiteDB) GetAccount(id string) (*models.Account, error) {
	acct := &models.Account{}
	err := s.db.QueryRow(
		`SELECT id, merchant_id, currency, balance, reserved_balance, status,
		 daily_limit, daily_used, monthly_limit, monthly_used, created_at, updated_at
		 FROM accounts WHERE id = ?`, id,
	).Scan(&acct.ID, &acct.MerchantID, &acct.Currency, &acct.Balance, &acct.ReservedBalance,
		&acct.Status, &acct.DailyLimit, &acct.DailyUsed, &acct.MonthlyLimit, &acct.MonthlyUsed,
		&acct.CreatedAt, &acct.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return acct, nil
}

func (s *SQLiteDB) ListAccounts(merchantID string) ([]models.Account, error) {
	var rows *sql.Rows
	var err error
	if merchantID != "" {
		rows, err = s.db.Query(
			`SELECT id, merchant_id, currency, balance, reserved_balance, status,
			 daily_limit, daily_used, monthly_limit, monthly_used, created_at, updated_at
			 FROM accounts WHERE merchant_id = ? ORDER BY currency`, merchantID)
	} else {
		rows, err = s.db.Query(
			`SELECT id, merchant_id, currency, balance, reserved_balance, status,
			 daily_limit, daily_used, monthly_limit, monthly_used, created_at, updated_at
			 FROM accounts ORDER BY merchant_id, currency`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []models.Account
	for rows.Next() {
		var a models.Account
		if err := rows.Scan(&a.ID, &a.MerchantID, &a.Currency, &a.Balance, &a.ReservedBalance,
			&a.Status, &a.DailyLimit, &a.DailyUsed, &a.MonthlyLimit, &a.MonthlyUsed,
			&a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (s *SQLiteDB) UpdateAccountBalance(id string, balance, reserved, dailyUsed, monthlyUsed int64) error {
	_, err := s.db.Exec(
		`UPDATE accounts SET balance = ?, reserved_balance = ?, daily_used = ?, monthly_used = ?, updated_at = ? WHERE id = ?`,
		balance, reserved, dailyUsed, monthlyUsed, time.Now(), id)
	return err
}

func (s *SQLiteDB) UpdateAccount(id string, acct *models.Account) error {
	_, err := s.db.Exec(
		`UPDATE accounts SET status = ?, daily_limit = ?, monthly_limit = ?, updated_at = ? WHERE id = ?`,
		acct.Status, acct.DailyLimit, acct.MonthlyLimit, time.Now(), id)
	return err
}

func (s *SQLiteDB) CreateAuditLog(log *models.AuditLog) error {
	_, err := s.db.Exec(
		`INSERT INTO audit_logs (user_id, action, resource, resource_id, ip_addr, user_agent, detail, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		log.UserID, log.Action, log.Resource, log.ResourceID, log.IPAddr, log.UserAgent, log.Detail, log.CreatedAt,
	)
	return err
}

func (s *SQLiteDB) ListAuditLogs(query AuditLogQuery) ([]models.AuditLog, int64, error) {
	where := "WHERE 1=1"
	args := []interface{}{}

	if query.Action != "" {
		where += " AND action = ?"
		args = append(args, query.Action)
	}
	if query.Resource != "" {
		where += " AND resource = ?"
		args = append(args, query.Resource)
	}
	if query.UserID != "" {
		where += " AND user_id = ?"
		args = append(args, query.UserID)
	}
	if query.StartDate != "" {
		where += " AND created_at >= ?"
		args = append(args, query.StartDate)
	}
	if query.EndDate != "" {
		where += " AND created_at <= ?"
		args = append(args, query.EndDate)
	}

	var total int64
	if err := s.db.QueryRow("SELECT COUNT(*) FROM audit_logs "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	offset := (query.Page - 1) * query.PageSize

	rows, err := s.db.Query(
		`SELECT id, user_id, action, resource, resource_id, ip_addr, user_agent, detail, created_at
		 FROM audit_logs `+where+` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		append(args, query.PageSize, offset)...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []models.AuditLog
	for rows.Next() {
		var l models.AuditLog
		if err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.Resource, &l.ResourceID,
			&l.IPAddr, &l.UserAgent, &l.Detail, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

func (s *SQLiteDB) CreateMerchant(m *models.Merchant) error {
	apiKey := generateToken(24)
	apiSecret := generateToken(32)
	m.APIKey = apiKey
	m.APISecret = apiSecret

	_, err := s.db.Exec(
		`INSERT INTO merchants (id, name, api_key, api_secret, status, daily_limit, monthly_limit, callback_url, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.Name, m.APIKey, m.APISecret, m.Status, m.DailyLimit, m.MonthlyLimit, m.CallbackURL, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

func (s *SQLiteDB) GetMerchant(id string) (*models.Merchant, error) {
	m := &models.Merchant{}
	err := s.db.QueryRow(
		`SELECT id, name, api_key, api_secret, status, daily_limit, monthly_limit, callback_url, created_at, updated_at
		 FROM merchants WHERE id = ?`, id,
	).Scan(&m.ID, &m.Name, &m.APIKey, &m.APISecret, &m.Status, &m.DailyLimit, &m.MonthlyLimit, &m.CallbackURL, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (s *SQLiteDB) ListMerchants() ([]models.Merchant, error) {
	rows, err := s.db.Query(
		`SELECT id, name, api_key, api_secret, status, daily_limit, monthly_limit, callback_url, created_at, updated_at
		 FROM merchants ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var merchants []models.Merchant
	for rows.Next() {
		var m models.Merchant
		if err := rows.Scan(&m.ID, &m.Name, &m.APIKey, &m.APISecret, &m.Status,
			&m.DailyLimit, &m.MonthlyLimit, &m.CallbackURL, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		merchants = append(merchants, m)
	}
	return merchants, nil
}

func (s *SQLiteDB) UpdateMerchant(id string, m *models.Merchant) error {
	_, err := s.db.Exec(
		`UPDATE merchants SET name = ?, status = ?, daily_limit = ?, monthly_limit = ?, callback_url = ?, updated_at = ? WHERE id = ?`,
		m.Name, m.Status, m.DailyLimit, m.MonthlyLimit, m.CallbackURL, time.Now(), id)
	return err
}

func (s *SQLiteDB) DeleteMerchant(id string) error {
	_, err := s.db.Exec("UPDATE merchants SET status = 'inactive', updated_at = ? WHERE id = ?", time.Now(), id)
	return err
}

func (s *SQLiteDB) GetUserByUsername(username string) (*models.User, error) {
	u := &models.User{}
	err := s.db.QueryRow(
		`SELECT id, username, password_hash, role, email, created_at, updated_at FROM users WHERE username = ?`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *SQLiteDB) CreateUser(user *models.User) error {
	_, err := s.db.Exec(
		`INSERT INTO users (id, username, password_hash, role, email, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash, user.Role, user.Email, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (s *SQLiteDB) GetExchangeRates() ([]models.ExchangeRate, error) {
	rows, err := s.db.Query(
		`SELECT id, source, target, rate, bid, ask, source_name, created_at FROM exchange_rates ORDER BY source, target`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rates []models.ExchangeRate
	for rows.Next() {
		var r models.ExchangeRate
		if err := rows.Scan(&r.ID, &r.Source, &r.Target, &r.Rate, &r.Bid, &r.Ask, &r.SourceName, &r.CreatedAt); err != nil {
			return nil, err
		}
		rates = append(rates, r)
	}
	return rates, nil
}

func (s *SQLiteDB) GetExchangeRate(source, target string) (*models.ExchangeRate, error) {
	// Try direct pair first
	r := &models.ExchangeRate{}
	err := s.db.QueryRow(
		`SELECT id, source, target, rate, bid, ask, source_name, created_at FROM exchange_rates WHERE source = ? AND target = ?`,
		source, target,
	).Scan(&r.ID, &r.Source, &r.Target, &r.Rate, &r.Bid, &r.Ask, &r.SourceName, &r.CreatedAt)
	if err == nil {
		return r, nil
	}

	// Try reverse pair and invert
	err = s.db.QueryRow(
		`SELECT id, source, target, rate, bid, ask, source_name, created_at FROM exchange_rates WHERE source = ? AND target = ?`,
		target, source,
	).Scan(&r.ID, &r.Source, &r.Target, &r.Rate, &r.Bid, &r.Ask, &r.SourceName, &r.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("exchange rate not found: %s->%s", source, target)
	}

	// Invert the rate for reverse pair
	r.Rate = 1.0 / r.Rate
	r.Bid = 1.0 / r.Ask
	r.Ask = 1.0 / r.Bid
	r.Source = source
	r.Target = target
	return r, nil
}

func (s *SQLiteDB) UpsertExchangeRate(rate *models.ExchangeRate) error {
	_, err := s.db.Exec(
		`INSERT INTO exchange_rates (source, target, rate, bid, ask, source_name, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(source, target) DO UPDATE SET rate = ?, bid = ?, ask = ?, source_name = ?, created_at = ?`,
		rate.Source, rate.Target, rate.Rate, rate.Bid, rate.Ask, rate.SourceName, rate.CreatedAt,
		rate.Rate, rate.Bid, rate.Ask, rate.SourceName, rate.CreatedAt,
	)
	return err
}

func (s *SQLiteDB) GetDashboardStats() (*models.DashboardStats, error) {
	stats := &models.DashboardStats{}

	// Total transactions
	s.db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&stats.TotalTransactions)

	// Today's stats
	today := time.Now().Format("2006-01-02")
	s.db.QueryRow("SELECT COUNT(*), COALESCE(SUM(target_amount), 0) FROM transactions WHERE date(created_at) = ?",
		today).Scan(&stats.TodayCount, &stats.TodayVolume)

	// Success rate (last 24 hours)
	var total24h, success24h int64
	s.db.QueryRow("SELECT COUNT(*) FROM transactions WHERE created_at >= datetime('now', '-1 day')").Scan(&total24h)
	s.db.QueryRow("SELECT COUNT(*) FROM transactions WHERE created_at >= datetime('now', '-1 day') AND status = 'completed'").Scan(&success24h)
	if total24h > 0 {
		stats.SuccessRate = float64(success24h) / float64(total24h) * 100
	}

	// Active accounts
	s.db.QueryRow("SELECT COUNT(*) FROM accounts WHERE status = 'active'").Scan(&stats.ActiveAccounts)

	// Pending count
	s.db.QueryRow("SELECT COUNT(*) FROM transactions WHERE status = 'pending' OR status = 'processing'").Scan(&stats.PendingCount)

	return stats, nil
}

func (s *SQLiteDB) GetRecentTransactions(limit int) ([]models.Transaction, error) {
	txns, _, err := s.ListTransactions(TransactionQuery{Page: 1, PageSize: limit})
	return txns, err
}

func (s *SQLiteDB) GetTPSHistory(seconds int) ([]models.TPSDataPoint, error) {
	if seconds <= 0 {
		seconds = 60
	}

	// Group transactions by second for the last N seconds
	rows, err := s.db.Query(
		`SELECT strftime('%s', created_at) as ts, COUNT(*) as cnt
		 FROM transactions
		 WHERE created_at >= datetime('now', '-' || ? || ' seconds')
		 GROUP BY strftime('%s', created_at)
		 ORDER BY ts ASC`,
		fmt.Sprintf("%d", seconds),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]models.TPSDataPoint, 0, seconds)
	for rows.Next() {
		var dp models.TPSDataPoint
		if err := rows.Scan(&dp.Timestamp, &dp.TPS); err != nil {
			continue
		}
		points = append(points, dp)
	}

	return points, nil
}

func generateToken(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)[:length]
}
