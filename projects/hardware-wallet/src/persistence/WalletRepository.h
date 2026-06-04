#pragma once

#include "util/Result.h"
#include "domain/Wallet.h"

#include <optional>
#include <vector>

class Database;

class WalletRepository {
public:
    explicit WalletRepository(Database* db);

    // CRUD
    Result<void> insert(const Wallet& wallet);
    Result<std::optional<Wallet>> findById(const QString& id);
    Result<std::optional<Wallet>> findByUserId(const QString& userId);
    Result<std::optional<Wallet>> findByPublicKey(const QString& publicKeyHex);
    Result<std::vector<Wallet>> findAll();

    // Update balance
    Result<void> updateBalance(const QString& walletId, double newBalance);

    // Delete
    Result<void> remove(const QString& walletId);
    Result<void> removeByUserId(const QString& userId);

    // Count
    Result<int> count();

private:
    Database* m_db;
    Wallet walletFromQuery(class QSqlQuery& query) const;
};
