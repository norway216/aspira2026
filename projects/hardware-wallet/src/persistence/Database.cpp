#include "Database.h"

#include <QSqlQuery>
#include <QSqlError>
#include <QVariant>
#include <QDebug>

const QString Database::CONNECTION_NAME = QStringLiteral("wallet_main");

Database::Database(const QString& dbPath)
    : m_dbPath(dbPath) {
}

Database::~Database() {
    close();
}

Result<void> Database::open() {
    if (QSqlDatabase::contains(CONNECTION_NAME)) {
        m_db = QSqlDatabase::database(CONNECTION_NAME);
    } else {
        m_db = QSqlDatabase::addDatabase(QStringLiteral("QSQLITE"), CONNECTION_NAME);
    }

    m_db.setDatabaseName(m_dbPath);

    if (!m_db.open()) {
        return Result<void>::fail(ErrorCode::DatabaseOpenFailed,
            QString("Failed to open database: %1").arg(m_db.lastError().text()));
    }

    // Enable WAL mode
    QSqlQuery query(m_db);
    if (!query.exec(QStringLiteral("PRAGMA journal_mode=WAL"))) {
        qWarning() << "Failed to enable WAL mode:" << query.lastError().text();
    }

    // Enable foreign keys
    if (!query.exec(QStringLiteral("PRAGMA foreign_keys=ON"))) {
        qWarning() << "Failed to enable foreign keys:" << query.lastError().text();
    }

    // Enable secure_delete for crypto-sensitive data
    if (!query.exec(QStringLiteral("PRAGMA secure_delete=ON"))) {
        qWarning() << "Failed to enable secure_delete:" << query.lastError().text();
    }

    // Set synchronous mode
    if (!query.exec(QStringLiteral("PRAGMA synchronous=NORMAL"))) {
        qWarning() << "Failed to set synchronous mode:" << query.lastError().text();
    }

    // Set busy timeout (5 seconds)
    if (!query.exec(QStringLiteral("PRAGMA busy_timeout=5000"))) {
        qWarning() << "Failed to set busy timeout:" << query.lastError().text();
    }

    // Run migrations
    return migrate();
}

void Database::close() {
    if (m_db.isOpen()) {
        m_db.close();
    }
}

bool Database::isOpen() const {
    return m_db.isOpen();
}

Result<void> Database::beginTransaction() {
    if (!m_db.transaction()) {
        return Result<void>::fail(ErrorCode::DatabaseTransactionFailed,
            QString("Failed to begin transaction: %1").arg(m_db.lastError().text()));
    }
    return Result<void>::ok();
}

Result<void> Database::commit() {
    if (!m_db.commit()) {
        return Result<void>::fail(ErrorCode::DatabaseTransactionFailed,
            QString("Failed to commit transaction: %1").arg(m_db.lastError().text()));
    }
    return Result<void>::ok();
}

Result<void> Database::rollback() {
    if (!m_db.rollback()) {
        return Result<void>::fail(ErrorCode::DatabaseTransactionFailed,
            QString("Failed to rollback transaction: %1").arg(m_db.lastError().text()));
    }
    return Result<void>::ok();
}

Result<void> Database::execute(const QString& sql) {
    QSqlQuery query(m_db);
    if (!query.exec(sql)) {
        return Result<void>::fail(ErrorCode::DatabaseQueryFailed,
            QString("SQL error: %1").arg(query.lastError().text()));
    }
    return Result<void>::ok();
}

Result<bool> Database::checkIntegrity() {
    QSqlQuery query(m_db);
    if (!query.exec(QStringLiteral("PRAGMA integrity_check"))) {
        return Result<bool>::fail(ErrorCode::DatabaseIntegrityFailed,
            query.lastError().text());
    }
    if (query.next()) {
        QString result = query.value(0).toString();
        return Result<bool>::ok(result.toLower() == "ok");
    }
    return Result<bool>::fail(ErrorCode::DatabaseIntegrityFailed, "No result from integrity check");
}

Result<int> Database::schemaVersion() {
    QSqlQuery query(m_db);
    if (!query.exec(QStringLiteral("SELECT MAX(version) FROM schema_version"))) {
        return Result<int>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }
    if (query.next()) {
        return Result<int>::ok(query.value(0).toInt());
    }
    return Result<int>::ok(0);
}

Result<void> Database::createVersionTable() {
    QSqlQuery query(m_db);
    if (!query.exec(QStringLiteral(
            "CREATE TABLE IF NOT EXISTS schema_version ("
            "  version INTEGER PRIMARY KEY,"
            "  applied_at TEXT NOT NULL DEFAULT (datetime('now'))"
            ")"))) {
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
            QString("Failed to create schema_version: %1").arg(query.lastError().text()));
    }
    return Result<void>::ok();
}

Result<void> Database::migrate() {
    auto verResult = createVersionTable();
    if (verResult.isFail()) return verResult;

    int currentVersion = schemaVersion().valueOr(0);

    // Migrations are sequential and additive
    if (currentVersion < 1) {
        auto result = migrateV1();
        if (result.isFail()) return result;
    }

    if (currentVersion < 2) {
        auto result = migrateV2();
        if (result.isFail()) return result;
    }

    return Result<void>::ok();
}

Result<void> Database::migrateV1() {
    qDebug() << "Running migration V1: Initial schema";

    auto txResult = beginTransaction();
    if (txResult.isFail()) return txResult;

    QSqlQuery query(m_db);

    // Users table
    if (!query.exec(QStringLiteral(
            "CREATE TABLE IF NOT EXISTS users ("
            "  id TEXT PRIMARY KEY,"
            "  username TEXT UNIQUE NOT NULL,"
            "  encrypted_dek BLOB NOT NULL,"
            "  password_verifier TEXT NOT NULL,"
            "  salt BLOB NOT NULL,"
            "  kdf_ops_limit INTEGER NOT NULL DEFAULT 4,"
            "  kdf_mem_limit INTEGER NOT NULL DEFAULT 268435456,"
            "  created_at INTEGER NOT NULL,"
            "  last_login_at INTEGER DEFAULT 0,"
            "  failed_attempts INTEGER NOT NULL DEFAULT 0,"
            "  locked_until INTEGER NOT NULL DEFAULT 0"
            ")"))) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
            QString("Failed to create users table: %1").arg(query.lastError().text()));
    }

    // Wallets table
    if (!query.exec(QStringLiteral(
            "CREATE TABLE IF NOT EXISTS wallets ("
            "  id TEXT PRIMARY KEY,"
            "  user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,"
            "  encrypted_seed BLOB NOT NULL,"
            "  public_key BLOB NOT NULL,"
            "  balance REAL NOT NULL DEFAULT 0.0,"
            "  created_at INTEGER NOT NULL"
            ")"))) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
            QString("Failed to create wallets table: %1").arg(query.lastError().text()));
    }

    if (!query.exec(QStringLiteral(
            "CREATE INDEX IF NOT EXISTS idx_wallets_user_id ON wallets(user_id)"))) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
            QString("Failed to create wallets index: %1").arg(query.lastError().text()));
    }

    // Transactions table
    if (!query.exec(QStringLiteral(
            "CREATE TABLE IF NOT EXISTS transactions ("
            "  id TEXT PRIMARY KEY,"
            "  wallet_id TEXT NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,"
            "  to_public_key TEXT NOT NULL,"
            "  amount REAL NOT NULL,"
            "  fee REAL NOT NULL DEFAULT 0.0,"
            "  signature BLOB NOT NULL,"
            "  nonce TEXT NOT NULL UNIQUE,"
            "  status INTEGER NOT NULL DEFAULT 0,"
            "  memo TEXT,"
            "  created_at INTEGER NOT NULL"
            ")"))) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
            QString("Failed to create transactions table: %1").arg(query.lastError().text()));
    }

    if (!query.exec(QStringLiteral(
            "CREATE INDEX IF NOT EXISTS idx_tx_wallet_id ON transactions(wallet_id)"))) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed, "Failed to create tx index");
    }

    if (!query.exec(QStringLiteral(
            "CREATE INDEX IF NOT EXISTS idx_tx_nonce ON transactions(nonce)"))) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed, "Failed to create nonce index");
    }

    if (!query.exec(QStringLiteral(
            "CREATE INDEX IF NOT EXISTS idx_tx_created_at ON transactions(created_at)"))) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed, "Failed to create tx date index");
    }

    // Audit logs table (hash chain)
    if (!query.exec(QStringLiteral(
            "CREATE TABLE IF NOT EXISTS audit_logs ("
            "  id INTEGER PRIMARY KEY AUTOINCREMENT,"
            "  action TEXT NOT NULL,"
            "  user_id TEXT,"
            "  details TEXT,"
            "  previous_hash TEXT NOT NULL,"
            "  hash TEXT NOT NULL UNIQUE,"
            "  created_at INTEGER NOT NULL"
            ")"))) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
            QString("Failed to create audit_logs table: %1").arg(query.lastError().text()));
    }

    if (!query.exec(QStringLiteral(
            "CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_logs(created_at)"))) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed, "Failed to create audit index");
    }

    // Record migration
    if (!query.prepare(QStringLiteral("INSERT INTO schema_version (version) VALUES (1)"))) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed, "Failed to record migration");
    }

    if (!query.exec()) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
            QString("Failed to record migration V1: %1").arg(query.lastError().text()));
    }

    return commit();
}

Result<void> Database::migrateV2() {
    qDebug() << "Running migration V2: Add security wipe columns to users";

    auto txResult = beginTransaction();
    if (txResult.isFail()) return txResult;

    QSqlQuery query(m_db);

    // Add wallet_wiped column if not exists (SQLite ALTER TABLE ADD COLUMN)
    query.prepare(QStringLiteral(
        "ALTER TABLE users ADD COLUMN wallet_wiped INTEGER NOT NULL DEFAULT 0"));
    if (!query.exec()) {
        // Column may already exist — ignore duplicate column error
        QString err = query.lastError().text();
        if (!err.contains("duplicate column", Qt::CaseInsensitive)) {
            rollback();
            return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
                QString("V2: Failed to add wallet_wiped: %1").arg(err));
        }
    }

    query.prepare(QStringLiteral(
        "ALTER TABLE users ADD COLUMN wiped_at INTEGER NOT NULL DEFAULT 0"));
    if (!query.exec()) {
        QString err = query.lastError().text();
        if (!err.contains("duplicate column", Qt::CaseInsensitive)) {
            rollback();
            return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
                QString("V2: Failed to add wiped_at: %1").arg(err));
        }
    }

    query.prepare(QStringLiteral(
        "ALTER TABLE users ADD COLUMN backup_created INTEGER NOT NULL DEFAULT 0"));
    if (!query.exec()) {
        QString err = query.lastError().text();
        if (!err.contains("duplicate column", Qt::CaseInsensitive)) {
            rollback();
            return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
                QString("V2: Failed to add backup_created: %1").arg(err));
        }
    }

    query.prepare(QStringLiteral(
        "ALTER TABLE users ADD COLUMN last_backup_at INTEGER NOT NULL DEFAULT 0"));
    if (!query.exec()) {
        QString err = query.lastError().text();
        if (!err.contains("duplicate column", Qt::CaseInsensitive)) {
            rollback();
            return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
                QString("V2: Failed to add last_backup_at: %1").arg(err));
        }
    }

    // Record migration
    query.prepare(QStringLiteral("INSERT INTO schema_version (version) VALUES (2)"));
    if (!query.exec()) {
        rollback();
        return Result<void>::fail(ErrorCode::DatabaseMigrationFailed,
            QString("V2: Failed to record migration: %1").arg(query.lastError().text()));
    }

    return commit();
}

Result<void> Database::secureCleanup() {
    qDebug() << "Running SQLite secure cleanup...";

    QSqlQuery query(m_db);

    // Ensure secure_delete is ON
    if (!query.exec(QStringLiteral("PRAGMA secure_delete=ON"))) {
        return Result<void>::fail(ErrorCode::SecureCleanupFailed,
            QString("secure_delete: %1").arg(query.lastError().text()));
    }

    // WAL checkpoint (TRUNCATE) — must be outside a transaction
    if (!query.exec(QStringLiteral("PRAGMA wal_checkpoint(TRUNCATE)"))) {
        return Result<void>::fail(ErrorCode::SecureCleanupFailed,
            QString("wal_checkpoint: %1").arg(query.lastError().text()));
    }

    // VACUUM to reclaim space and overwrite deleted pages
    if (!query.exec(QStringLiteral("VACUUM"))) {
        return Result<void>::fail(ErrorCode::SecureCleanupFailed,
            QString("VACUUM: %1").arg(query.lastError().text()));
    }

    qDebug() << "SQLite secure cleanup completed";
    return Result<void>::ok();
}
