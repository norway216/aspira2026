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

	// Enable WAL mode for concurrent reads + writes (massive TPS improvement)
	// WAL allows multiple readers and one writer to coexist without blocking each other
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	sqliteDB := &SQLiteDB{db: db}

	// Apply performance pragmas
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-65536",   // 64MB cache
		"PRAGMA foreign_keys=ON",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA mmap_size=268435456", // 256MB memory map
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return nil, fmt.Errorf("pragma error (%s): %w", p, err)
		}
	}

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

	// Seed accounts (multi-currency accounts for each merchant)
	accounts := []struct {
		id, merchantID, currency string
		balance                  int64
	}{
		// Merchant 1
		{"acct-001", "merchant-001", "USD", 100000000},   // 1M USD
		{"acct-002", "merchant-001", "CNY", 10000000000}, // 100M CNY
		{"acct-005", "merchant-001", "EUR", 50000000},    // 500K EUR
		{"acct-006", "merchant-001", "JPY", 10000000000}, // 100M JPY
		{"acct-007", "merchant-001", "GBP", 30000000},    // 300K GBP
		{"acct-008", "merchant-001", "CHF", 20000000},    // 200K CHF
		{"acct-009", "merchant-001", "CAD", 40000000},    // 400K CAD
		{"acct-010", "merchant-001", "AUD", 40000000},    // 400K AUD
		{"acct-011", "merchant-001", "NZD", 30000000},    // 300K NZD
		{"acct-012", "merchant-001", "SGD", 30000000},    // 300K SGD
		{"acct-013", "merchant-001", "HKD", 50000000},    // 500K HKD
		// Merchant 2
		{"acct-003", "merchant-002", "USD", 50000000},    // 500K USD
		{"acct-004", "merchant-002", "CNY", 5000000000},  // 50M CNY
		{"acct-014", "merchant-002", "EUR", 30000000},    // 300K EUR
		{"acct-015", "merchant-002", "JPY", 5000000000},  // 50M JPY
		{"acct-016", "merchant-002", "GBP", 20000000},    // 200K GBP
		{"acct-017", "merchant-002", "CHF", 15000000},    // 150K CHF
		{"acct-018", "merchant-002", "AUD", 30000000},    // 300K AUD
		{"acct-019", "merchant-002", "SGD", 25000000},    // 250K SGD
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
		{"USD", "CHF", 0.8985, 0.8970, 0.9000, "SNB"},
		{"USD", "CAD", 1.3670, 1.3650, 1.3690, "BOC"},
		{"USD", "AUD", 1.5210, 1.5190, 1.5230, "RBA"},
		{"USD", "NZD", 1.6390, 1.6370, 1.6410, "RBNZ"},
		{"USD", "SGD", 1.3485, 1.3470, 1.3500, "MAS"},
		{"USD", "HKD", 7.8120, 7.8100, 7.8140, "HKMA"},
		{"EUR", "CNY", 7.8740, 7.8680, 7.8800, "CFETS"},
		{"EUR", "CHF", 0.9750, 0.9730, 0.9770, "SNB"},
		{"EUR", "JPY", 169.05, 168.80, 169.30, "BOJ"},
		{"EUR", "GBP", 0.8585, 0.8570, 0.8600, "BOE"},
		{"CNY", "JPY", 21.470, 21.420, 21.520, "CFETS"},
		{"CNY", "EUR", 0.1271, 0.1268, 0.1274, "CFETS"},
		{"AUD", "NZD", 1.0775, 1.0760, 1.0790, "RBA"},
		{"SGD", "HKD", 5.7950, 5.7900, 5.8000, "MAS"},
		// Reverse rates (1/rate) for common pairs
		{"CNY", "USD", 0.1379, 0.1377, 0.1380, "CFETS"},
		{"EUR", "USD", 1.0852, 1.0834, 1.0870, "ECB"},
		{"JPY", "USD", 0.006420, 0.006410, 0.006430, "BOJ"},
		{"GBP", "USD", 1.2642, 1.2616, 1.2668, "BOE"},
		{"CHF", "USD", 1.1130, 1.1111, 1.1149, "SNB"},
		{"CAD", "USD", 0.7315, 0.7305, 0.7326, "BOC"},
		{"AUD", "USD", 0.6575, 0.6566, 0.6584, "RBA"},
		{"NZD", "USD", 0.6101, 0.6094, 0.6109, "RBNZ"},
		{"SGD", "USD", 0.7416, 0.7407, 0.7425, "MAS"},
		{"HKD", "USD", 0.1280, 0.1279, 0.1281, "HKMA"},
		// Cross-rates via CNY
		{"CNY", "HKD", 1.0771, 1.0760, 1.0782, "CFETS"},
		{"CNY", "SGD", 0.1859, 0.1855, 0.1863, "CFETS"},
		{"CNY", "AUD", 0.2097, 0.2093, 0.2101, "CFETS"},
		{"CNY", "GBP", 0.1091, 0.1089, 0.1093, "CFETS"},
		{"JPY", "CNY", 0.04658, 0.04645, 0.04671, "CFETS"},
		{"GBP", "CNY", 9.1701, 9.1580, 9.1822, "CFETS"},
		{"AUD", "CNY", 4.7693, 4.7610, 4.7776, "CFETS"},
		{"SGD", "CNY", 5.3789, 5.3700, 5.3878, "CFETS"},
		// Cross-rates via EUR
		{"EUR", "AUD", 1.6506, 1.6470, 1.6542, "ECB"},
		{"EUR", "CAD", 1.4835, 1.4800, 1.4870, "ECB"},
		{"EUR", "NZD", 1.7785, 1.7750, 1.7820, "ECB"},
		// Cross-rates between Asian currencies
		{"JPY", "SGD", 0.008660, 0.008640, 0.008680, "MAS"},
		{"JPY", "HKD", 0.05015, 0.05000, 0.05030, "HKMA"},
	}
	for _, r := range rates {
		if _, err := s.db.Exec(
			"INSERT OR IGNORE INTO exchange_rates (source, target, rate, bid, ask, source_name) VALUES (?, ?, ?, ?, ?, ?)",
			r.source, r.target, r.rate, r.bid, r.ask, r.sourceName,
		); err != nil {
			return err
		}
	}

	// Seed sample transactions (multi-currency demo)
	type sampleTxn struct {
		status                        models.TransactionStatus
		payerAcct, payeeAcct          string
		srcCurrency, tgtCurrency      string
		srcAmt                        int64
		rate                          float64
		fee                           int64
		desc                          string
		hoursAgo                      int
	}
	samples := []sampleTxn{
		{models.TxnCompleted, "acct-002", "acct-001", "CNY", "USD", 50000, 7.2530, 150, "跨境电商货款", 20},
		{models.TxnCompleted, "acct-005", "acct-002", "EUR", "CNY", 12000, 7.8740, 85, "进口商品结算", 18},
		{models.TxnCompleted, "acct-006", "acct-001", "JPY", "USD", 250000, 0.00642, 320, "软件服务费", 16},
		{models.TxnCompleted, "acct-001", "acct-008", "USD", "CHF", 8000, 0.8985, 60, "瑞士银行转账", 14},
		{models.TxnFailed, "acct-009", "acct-003", "CAD", "USD", 15000, 0.7315, 110, "跨境贸易结算", 12},
		{models.TxnCompleted, "acct-007", "acct-005", "GBP", "EUR", 9500, 1.1635, 75, "英国电商收款", 10},
		{models.TxnRefunded, "acct-010", "acct-018", "AUD", "AUD", 20000, 1.0000, 40, "退款-商户争议", 8},
		{models.TxnCompleted, "acct-003", "acct-012", "USD", "SGD", 18000, 1.3485, 130, "新加坡汇款", 6},
		{models.TxnPending, "acct-013", "acct-002", "HKD", "CNY", 50000, 0.9285, 95, "香港贸易结算", 4},
		{models.TxnCompleted, "acct-011", "acct-010", "NZD", "AUD", 12000, 0.9280, 70, "跨塔斯曼汇款", 2},
	}
	prevHash := "0000000000000000000000000000000000000000000000000000000000000000"

	for i, tx := range samples {
		txnID := fmt.Sprintf("txn-sample-%03d", i+1)
		tgtAmt := int64(float64(tx.srcAmt-tx.fee) * tx.rate)
		ts := time.Now().Add(-time.Duration(tx.hoursAgo) * time.Hour)

		hashInput := fmt.Sprintf("%s|%s|%d|%s|%d|%d", prevHash, txnID, tx.srcAmt, tx.tgtCurrency, tgtAmt, ts.UnixNano())
		hash := sha256.Sum256([]byte(hashInput))
		currHash := hex.EncodeToString(hash[:])
		prevHashCopy := prevHash

		if _, err := s.db.Exec(
			`INSERT INTO transactions (id, merchant_id, payer_account_id, payee_account_id,
			 source_currency, target_currency, source_amount, target_amount, exchange_rate, fee,
			 status, description, reference_id, hash_chain_prev, hash_chain_curr,
			 created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			txnID, "merchant-001", tx.payerAcct, tx.payeeAcct,
			tx.srcCurrency, tx.tgtCurrency, tx.srcAmt, tgtAmt, tx.rate, tx.fee,
			string(tx.status), tx.desc, fmt.Sprintf("REF-%03d", i+1),
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

func (s *SQLiteDB) GetLastTransactionHash() string {
	var hash string
	err := s.db.QueryRow(
		"SELECT hash_chain_curr FROM transactions ORDER BY created_at DESC LIMIT 1",
	).Scan(&hash)
	if err != nil {
		return "0000000000000000000000000000000000000000000000000000000000000000"
	}
	return hash
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

	// Compute time boundaries in Go to avoid SQLite timezone issues.
	// SQLite datetime('now') returns UTC, but created_at stores local time,
	// and Go's time.Now() format is not parsable by SQLite's date() function.
	now := time.Now()
	oneSecAgo := now.Add(-1 * time.Second).Format("2006-01-02 15:04:05")
	fiveSecAgo := now.Add(-5 * time.Second).Format("2006-01-02 15:04:05")
	oneDayAgo := now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05")
	todayStart := now.Format("2006-01-02") + " 00:00:00"
	tomorrowStart := now.Add(24 * time.Hour).Format("2006-01-02") + " 00:00:00"

	// Compute TPS: count transactions in the last 1 second
	var tpsCount int64
	s.db.QueryRow(
		"SELECT COUNT(*) FROM transactions WHERE created_at >= ?",
		oneSecAgo,
	).Scan(&tpsCount)
	stats.CurrentTPS = float64(tpsCount)

	// Also compute average TPS over last 5 seconds for smoother display
	var tps5s int64
	s.db.QueryRow(
		"SELECT COUNT(*) FROM transactions WHERE created_at >= ?",
		fiveSecAgo,
	).Scan(&tps5s)
	avgTPS5s := float64(tps5s) / 5.0
	if avgTPS5s > stats.CurrentTPS {
		stats.CurrentTPS = avgTPS5s
	}

	// Total transactions
	s.db.QueryRow("SELECT COUNT(*) FROM transactions").Scan(&stats.TotalTransactions)

	// Today's stats — use string prefix matching against created_at
	s.db.QueryRow(
		"SELECT COUNT(*), COALESCE(SUM(target_amount), 0) FROM transactions WHERE created_at >= ? AND created_at < ?",
		todayStart, tomorrowStart,
	).Scan(&stats.TodayCount, &stats.TodayVolume)

	// Success rate (last 24 hours)
	var total24h, success24h int64
	s.db.QueryRow("SELECT COUNT(*) FROM transactions WHERE created_at >= ?", oneDayAgo).Scan(&total24h)
	s.db.QueryRow("SELECT COUNT(*) FROM transactions WHERE created_at >= ? AND status = 'completed'", oneDayAgo).Scan(&success24h)
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

	// Compute time boundary in Go to avoid SQLite datetime() timezone issues.
	// created_at stores local time strings; we compare against a local time prefix.
	since := time.Now().Add(-time.Duration(seconds) * time.Second).Format("2006-01-02 15:04:05")

	// Group by the second-precision prefix of created_at (first 19 chars = "YYYY-MM-DD HH:MM:SS")
	rows, err := s.db.Query(
		`SELECT SUBSTR(created_at, 1, 19) as ts, COUNT(*) as cnt
		 FROM transactions
		 WHERE created_at >= ?
		 GROUP BY ts
		 ORDER BY ts ASC`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]models.TPSDataPoint, 0, seconds)
	for rows.Next() {
		var tsStr string
		var cnt int64
		if err := rows.Scan(&tsStr, &cnt); err != nil {
			continue
		}
		// Parse as local time, then convert to unix timestamp for the API
		t, err := time.ParseInLocation("2006-01-02 15:04:05", tsStr, time.Local)
		if err != nil {
			continue
		}
		points = append(points, models.TPSDataPoint{
			Timestamp: t.Unix(),
			TPS:       float64(cnt),
		})
	}

	return points, nil
}

func (s *SQLiteDB) GetVolumeHistory(hours int) ([]models.VolumeDataPoint, error) {
	if hours <= 0 {
		hours = 24
	}

	// Compute time boundary in Go to avoid SQLite datetime() timezone issues
	since := time.Now().Add(-time.Duration(hours) * time.Hour).Format("2006-01-02 15:04:05")

	// Group by hour — extract "YYYY-MM-DD HH:00" prefix
	rows, err := s.db.Query(
		`SELECT SUBSTR(created_at, 1, 13) || ':00' as hour_label,
		        COALESCE(SUM(target_amount), 0) as vol,
		        COUNT(*) as cnt
		 FROM transactions
		 WHERE created_at >= ?
		 GROUP BY hour_label
		 ORDER BY hour_label ASC`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]models.VolumeDataPoint, 0, hours)
	for rows.Next() {
		var dp models.VolumeDataPoint
		if err := rows.Scan(&dp.Label, &dp.Volume, &dp.Count); err != nil {
			continue
		}
		// Parse label to get unix timestamp
		t, err := time.ParseInLocation("2006-01-02 15:04:05", dp.Label, time.Local)
		if err == nil {
			dp.Timestamp = t.Unix()
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
