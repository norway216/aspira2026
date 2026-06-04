#pragma once

#include "util/Result.h"
#include "domain/Transaction.h"

#include <optional>
#include <vector>

class Database;

class TransactionRepository {
public:
    explicit TransactionRepository(Database* db);

    // CRUD
    Result<void> insert(const Transaction& tx);
    Result<std::optional<Transaction>> findById(const QString& id);
    Result<std::optional<Transaction>> findByNonce(const QString& nonce);

    // Paginated queries
    Result<std::vector<Transaction>> findByWalletId(const QString& walletId,
                                                     int offset = 0, int limit = 20);
    Result<std::vector<Transaction>> findByUserId(const QString& userId,
                                                   int offset = 0, int limit = 20);

    // Find received transactions (where to_public_key matches)
    Result<std::vector<Transaction>> findReceived(const QString& publicKeyHex,
                                                   int offset = 0, int limit = 20);

    // Get all transactions (for backup)
    Result<std::vector<Transaction>> findAll();

    // Count
    Result<int> countByWalletId(const QString& walletId);
    Result<int> countReceived(const QString& publicKeyHex);

    // Update status
    Result<void> updateStatus(const QString& txId, int newStatus);

    // Security wipe deletion (per architecture §6.4)
    Result<void> removeByUserId(const QString& userId);
    Result<void> removeByWalletId(const QString& walletId);

private:
    Database* m_db;
    Transaction txFromQuery(class QSqlQuery& query) const;
};
