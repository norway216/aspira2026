#include "BackupService.h"
#include "AuthService.h"
#include "persistence/Database.h"
#include "persistence/BackupManager.h"
#include "persistence/AuditLogRepository.h"
#include "domain/AuditLogEntry.h"

#include <QDebug>

BackupService::BackupService(Database* db, AuthService* authService, QObject* parent)
    : QObject(parent), m_db(db), m_authService(authService) {
    m_backupManager = new BackupManager(db);
    m_auditRepo = new AuditLogRepository(db);
}

void BackupService::exportBackup(const QString& filePath, const QString& backupPassword) {
    if (!m_authService->isLoggedIn()) {
        emit exportCompleted(Result<void>::fail(ErrorCode::NotLoggedIn, "Not logged in"));
        return;
    }

    if (backupPassword.length() < 8) {
        emit exportCompleted(Result<void>::fail(ErrorCode::PasswordTooShort,
            "Backup password must be at least 8 characters"));
        return;
    }

    auto result = m_backupManager->exportBackup(filePath, backupPassword);
    if (result.isOk()) {
        m_auditRepo->append(AuditLogEntry::ACTION_BACKUP_EXPORT, m_authService->currentUserId(),
            QString("{\"file\":\"%1\"}").arg(filePath));
        qDebug() << "Backup exported successfully to:" << filePath;
    }

    emit exportCompleted(result);
}

void BackupService::importBackup(const QString& filePath, const QString& backupPassword) {
    // Import doesn't require login (it might be a fresh install)
    auto result = m_backupManager->importBackup(filePath, backupPassword);
    if (result.isOk()) {
        m_auditRepo->append(AuditLogEntry::ACTION_BACKUP_IMPORT, m_authService->isLoggedIn() ? m_authService->currentUserId() : "",
            QString("{\"file\":\"%1\",\"users\":%2}").arg(filePath).arg(result.value()));
        qDebug() << "Backup restored successfully:" << result.value() << "users";
    }

    emit importCompleted(result);
}
