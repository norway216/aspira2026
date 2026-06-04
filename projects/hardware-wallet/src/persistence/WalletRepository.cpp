#include "WalletRepository.h"
#include "Database.h"

#include <QSqlQuery>
#include <QSqlError>
#include <QVariant>
#include <QDebug>

WalletRepository::WalletRepository(Database* db) : m_db(db) {}

Result<void> WalletRepository::insert(const Wallet& wallet) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "INSERT INTO wallets (id, user_id, encrypted_seed, public_key, balance, created_at) "
        "VALUES (?, ?, ?, ?, ?, ?)"
    ));

    query.addBindValue(wallet.id);
    query.addBindValue(wallet.userId);
    query.addBindValue(wallet.encryptedSeed);
    query.addBindValue(wallet.publicKey);
    query.addBindValue(wallet.balance);
    query.addBindValue(wallet.createdAt);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseInsertFailed,
            QString("Failed to insert wallet: %1").arg(query.lastError().text()));
    }

    return Result<void>::ok();
}

Result<std::optional<Wallet>> WalletRepository::findById(const QString& id) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("SELECT * FROM wallets WHERE id = ?"));
    query.addBindValue(id);

    if (!query.exec()) {
        return Result<std::optional<Wallet>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    if (query.next()) {
        return Result<std::optional<Wallet>>::ok(walletFromQuery(query));
    }

    return Result<std::optional<Wallet>>::ok(std::nullopt);
}

Result<std::optional<Wallet>> WalletRepository::findByUserId(const QString& userId) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("SELECT * FROM wallets WHERE user_id = ?"));
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<std::optional<Wallet>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    if (query.next()) {
        return Result<std::optional<Wallet>>::ok(walletFromQuery(query));
    }

    return Result<std::optional<Wallet>>::ok(std::nullopt);
}

Result<std::optional<Wallet>> WalletRepository::findByPublicKey(const QString& publicKeyHex) {
    // Find wallet by public key (hex). We need to match against stored BLOB.
    QSqlQuery query(m_db->sqlDatabase());
    // Get all wallets and compare public keys
    if (!query.exec(QStringLiteral("SELECT * FROM wallets"))) {
        return Result<std::optional<Wallet>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    while (query.next()) {
        Wallet wallet = walletFromQuery(query);
        if (wallet.publicKeyHex() == publicKeyHex) {
            return Result<std::optional<Wallet>>::ok(std::move(wallet));
        }
    }

    return Result<std::optional<Wallet>>::ok(std::nullopt);
}

Result<std::vector<Wallet>> WalletRepository::findAll() {
    QSqlQuery query(m_db->sqlDatabase());
    if (!query.exec(QStringLiteral("SELECT * FROM wallets ORDER BY created_at DESC"))) {
        return Result<std::vector<Wallet>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    std::vector<Wallet> wallets;
    while (query.next()) {
        wallets.push_back(walletFromQuery(query));
    }
    return Result<std::vector<Wallet>>::ok(std::move(wallets));
}

Result<void> WalletRepository::updateBalance(const QString& walletId, double newBalance) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("UPDATE wallets SET balance = ? WHERE id = ?"));
    query.addBindValue(newBalance);
    query.addBindValue(walletId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseUpdateFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> WalletRepository::remove(const QString& walletId) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("DELETE FROM wallets WHERE id = ?"));
    query.addBindValue(walletId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseDeleteFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> WalletRepository::removeByUserId(const QString& userId) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("DELETE FROM wallets WHERE user_id = ?"));
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseDeleteFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<int> WalletRepository::count() {
    QSqlQuery query(m_db->sqlDatabase());
    if (!query.exec(QStringLiteral("SELECT COUNT(*) FROM wallets"))) {
        return Result<int>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }
    if (query.next()) {
        return Result<int>::ok(query.value(0).toInt());
    }
    return Result<int>::ok(0);
}

Wallet WalletRepository::walletFromQuery(QSqlQuery& query) const {
    Wallet wallet;
    wallet.id = query.value("id").toString();
    wallet.userId = query.value("user_id").toString();
    wallet.encryptedSeed = (query.value("encrypted_seed").toByteArray());
    wallet.publicKey = (query.value("public_key").toByteArray());
    wallet.balance = query.value("balance").toDouble();
    wallet.createdAt = query.value("created_at").toLongLong();
    return wallet;
}
