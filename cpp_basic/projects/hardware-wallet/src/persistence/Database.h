#pragma once

#include "util/Result.h"
#include "util/Error.h"

#include <QSqlDatabase>
#include <QString>
#include <QStringList>

/**
 * Manages SQLite database connection with WAL mode and schema migrations.
 */
class Database {
public:
    explicit Database(const QString& dbPath);
    ~Database();

    // Open database and run migrations
    Result<void> open();

    // Close database
    void close();

    // Check if database is open
    bool isOpen() const;

    // Get the underlying QSqlDatabase
    QSqlDatabase& sqlDatabase() { return m_db; }

    // Transaction management
    Result<void> beginTransaction();
    Result<void> commit();
    Result<void> rollback();

    // Execute a raw SQL statement
    Result<void> execute(const QString& sql);

    // Get the database path
    QString path() const { return m_dbPath; }

    // Verify database integrity
    Result<bool> checkIntegrity();

    // Get current schema version
    Result<int> schemaVersion();

    // SQLite secure cleanup after wallet wipe (per architecture §10)
    // Must be called OUTSIDE any transaction.
    Result<void> secureCleanup();

private:
    QString m_dbPath;
    QSqlDatabase m_db;

    // Run all pending migrations
    Result<void> migrate();

    // Individual migrations
    Result<void> migrateV1();  // Initial schema
    Result<void> migrateV2();  // Add security wipe columns to users

    // Create schema_version table
    Result<void> createVersionTable();

    static const QString CONNECTION_NAME;
};
