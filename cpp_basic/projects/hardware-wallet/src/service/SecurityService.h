#pragma once

#include <QObject>

#include "util/Result.h"

class Database;
class UserRepository;
class WalletRepository;
class TransactionRepository;
class AuditLogRepository;

/**
 * SecurityService — orchestrates secure wallet data destruction.
 *
 * Per architecture §7:
 * - Destroys encrypted DEK (crypto-erasure)
 * - Deletes wallet + transaction records
 * - Records audit trail
 * - Runs SQLite secure cleanup (checkpoint + VACUUM)
 *
 * This is the single entry point for the "3-strikes" self-destruct flow.
 */
class SecurityService : public QObject {
    Q_OBJECT
public:
    explicit SecurityService(Database* db,
                            UserRepository* userRepo,
                            WalletRepository* walletRepo,
                            TransactionRepository* txRepo,
                            AuditLogRepository* auditRepo,
                            QObject* parent = nullptr);

    /// Wipe all wallet data for a user (DEK destruction + wallet delete + tx delete).
    /// Returns the audit log entry ID on success (for verification).
    Result<void> wipeUserWalletData(const QString& userId, const QString& reason);

    /// Check if user's wallet has been wiped
    Result<bool> isWalletWiped(const QString& userId);

private:
    Database* m_db;
    UserRepository* m_userRepo;
    WalletRepository* m_walletRepo;
    TransactionRepository* m_txRepo;
    AuditLogRepository* m_auditRepo;

    /// SQLite secure cleanup — must be called outside any transaction.
    Result<void> cleanupSQLite();
};
