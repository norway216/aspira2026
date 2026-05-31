#pragma once

#include <QObject>

#include "util/Result.h"

class Database;
class BackupManager;
class AuditLogRepository;
class AuthService;

/**
 * BackupService - orchestrates encrypted backup export and restore.
 */
class BackupService : public QObject {
    Q_OBJECT
public:
    explicit BackupService(Database* db, AuthService* authService, QObject* parent = nullptr);

    // Export encrypted backup to file path
    void exportBackup(const QString& filePath, const QString& backupPassword);

    // Import encrypted backup from file path
    void importBackup(const QString& filePath, const QString& backupPassword);

signals:
    void exportCompleted(Result<void> result);
    void importCompleted(Result<int> result);

private:
    Database* m_db;
    BackupManager* m_backupManager;
    AuditLogRepository* m_auditRepo;
    AuthService* m_authService;
};
