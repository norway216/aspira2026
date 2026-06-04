#pragma once

#include "util/Result.h"
#include "crypto/SecureBuffer.h"

#include <QString>
#include <QJsonObject>

class Database;

/**
 * Manages encrypted backup export and restore.
 *
 * Backup format (binary blob):
 * [SALT(16)][OPS_LIMIT(4)][MEM_LIMIT(4)][NONCE(24)][TAG(16)][ENCRYPTED_JSON]
 *
 * The JSON payload contains all wallet data encrypted with a backup password.
 */
class BackupManager {
public:
    explicit BackupManager(Database* db);

    // Export encrypted backup to file
    Result<void> exportBackup(const QString& filePath, const QString& backupPassword);

    // Import encrypted backup from file
    // Returns the number of users restored
    Result<int> importBackup(const QString& filePath, const QString& backupPassword);

    // Verify backup file (check format, decrypt, validate structure)
    Result<QJsonObject> verifyBackup(const QString& filePath, const QString& backupPassword);

private:
    Database* m_db;

    // Build the JSON payload from current database state
    Result<QJsonObject> buildPayload();

    // Restore database from JSON payload
    Result<int> restoreFromPayload(const QJsonObject& payload);
};
