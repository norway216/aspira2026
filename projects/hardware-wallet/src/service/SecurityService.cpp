#include "SecurityService.h"

#include "persistence/Database.h"
#include "persistence/UserRepository.h"
#include "persistence/WalletRepository.h"
#include "persistence/TransactionRepository.h"
#include "persistence/AuditLogRepository.h"
#include "domain/AuditLogEntry.h"
#include "util/CryptoUtils.h"
#include "app/Constants.h"

#include <QJsonDocument>
#include <QJsonObject>
#include <QDebug>

SecurityService::SecurityService(Database* db,
                                 UserRepository* userRepo,
                                 WalletRepository* walletRepo,
                                 TransactionRepository* txRepo,
                                 AuditLogRepository* auditRepo,
                                 QObject* parent)
    : QObject(parent)
    , m_db(db)
    , m_userRepo(userRepo)
    , m_walletRepo(walletRepo)
    , m_txRepo(txRepo)
    , m_auditRepo(auditRepo)
{
}

Result<void> SecurityService::wipeUserWalletData(const QString& userId,
                                                  const QString& reason)
{
    qWarning() << "Wiping wallet data for user:" << userId << "reason:" << reason;

    // 1. Begin transaction
    auto txRes = m_db->beginTransaction();
    if (txRes.isFail()) {
        qCritical() << "Failed to begin transaction for wallet wipe:" << txRes.error().message;
        return Result<void>::fail(ErrorCode::WalletWipeFailed,
            "Transaction start failed: " + txRes.error().message);
    }

    // 2. Write SECURITY_SELF_DESTRUCT audit log BEFORE deletion
    //    (per architecture §7.2: audit log must survive the wipe)
    QJsonObject details;
    details["reason"] = reason;
    details["policy"] = "password_failed_3_times";
    details["timestamp"] = QString::number(CryptoUtils::currentTimestampMs());

    auto auditRes = m_auditRepo->append(
        AuditLogEntry::ACTION_SECURITY_SELF_DESTRUCT,
        userId,
        QString::fromUtf8(QJsonDocument(details).toJson(QJsonDocument::Compact))
    );

    if (auditRes.isFail()) {
        m_db->rollback();
        return Result<void>::fail(ErrorCode::WalletWipeFailed,
            "Audit log write failed: " + auditRes.error().message);
    }

    // 3. Destroy encrypted DEK (crypto-erasure with random garbage)
    auto destroyDekRes = m_userRepo->destroyEncryptedDEK(userId);
    if (destroyDekRes.isFail()) {
        m_db->rollback();
        return Result<void>::fail(ErrorCode::WalletWipeFailed,
            "DEK destruction failed: " + destroyDekRes.error().message);
    }

    // 4. Delete transactions (MUST precede wallet deletion per architecture §6.4)
    auto removeTxRes = m_txRepo->removeByUserId(userId);
    if (removeTxRes.isFail()) {
        m_db->rollback();
        return Result<void>::fail(ErrorCode::WalletWipeFailed,
            "Transaction deletion failed: " + removeTxRes.error().message);
    }

    // 5. Delete wallet
    auto removeWalletRes = m_walletRepo->removeByUserId(userId);
    if (removeWalletRes.isFail()) {
        m_db->rollback();
        return Result<void>::fail(ErrorCode::WalletWipeFailed,
            "Wallet deletion failed: " + removeWalletRes.error().message);
    }

    // 6. Mark wallet_wiped (user record already updated by destroyEncryptedDEK
    //    but we also want a clean audit entry)
    auto markRes = m_userRepo->markWalletWiped(userId, CryptoUtils::currentTimestampMs());
    if (markRes.isFail()) {
        m_db->rollback();
        return Result<void>::fail(ErrorCode::WalletWipeFailed,
            "Mark wallet wiped failed: " + markRes.error().message);
    }

    // 7. Also write WALLET_WIPED audit entry
    QJsonObject wipedDetails;
    wipedDetails["reason"] = reason;
    auto wipedAuditRes = m_auditRepo->append(
        AuditLogEntry::ACTION_WALLET_WIPED,
        userId,
        QString::fromUtf8(QJsonDocument(wipedDetails).toJson(QJsonDocument::Compact))
    );
    if (wipedAuditRes.isFail()) {
        m_db->rollback();
        return Result<void>::fail(ErrorCode::WalletWipeFailed,
            "Wiped audit log write failed: " + wipedAuditRes.error().message);
    }

    // 8. Commit transaction
    auto commitRes = m_db->commit();
    if (commitRes.isFail()) {
        m_db->rollback();
        return Result<void>::fail(ErrorCode::WalletWipeFailed,
            "Commit failed: " + commitRes.error().message);
    }

    // 9. SQLite secure cleanup (OUTSIDE transaction — per architecture §10)
    if constexpr (AppConstants::ENABLE_VACUUM_AFTER_WIPE) {
        auto cleanupRes = cleanupSQLite();
        if (cleanupRes.isFail()) {
            qWarning() << "SQLite secure cleanup failed:" << cleanupRes.error().message;
            // Non-fatal: wallet data is already gone
        }
    }

    qInfo() << "Wallet wipe completed for user:" << userId;
    return Result<void>::ok();
}

Result<bool> SecurityService::isWalletWiped(const QString& userId)
{
    auto userRes = m_userRepo->findById(userId);
    if (userRes.isFail() || !userRes.value().has_value()) {
        return Result<bool>::fail(ErrorCode::UserNotFound, "User not found");
    }
    return Result<bool>::ok(userRes.value()->walletWiped != 0);
}

Result<void> SecurityService::cleanupSQLite()
{
    if constexpr (AppConstants::ENABLE_SQLITE_SECURE_DELETE) {
        return m_db->secureCleanup();
    }
    return Result<void>::ok();
}
