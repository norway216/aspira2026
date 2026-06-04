#include "TransactionRepository.h"
#include "Database.h"

#include <QSqlQuery>
#include <QSqlError>
#include <QVariant>
#include <QDebug>

TransactionRepository::TransactionRepository(Database* db) : m_db(db) {}

Result<void> TransactionRepository::insert(const Transaction& tx) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "INSERT INTO transactions (id, wallet_id, to_public_key, amount, fee, "
        "signature, nonce, status, memo, created_at) "
        "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
    ));

    query.addBindValue(tx.id);
    query.addBindValue(tx.walletId);
    query.addBindValue(tx.toPublicKey);
    query.addBindValue(tx.amount);
    query.addBindValue(tx.fee);
    query.addBindValue(tx.signature);
    query.addBindValue(tx.nonce);
    query.addBindValue(tx.status);
    query.addBindValue(tx.memo);
    query.addBindValue(tx.createdAt);

    if (!query.exec()) {
        QString err = query.lastError().text();
        if (err.contains("UNIQUE", Qt::CaseInsensitive) && err.contains("nonce", Qt::CaseInsensitive)) {
            return Result<void>::fail(ErrorCode::DuplicateNonce,
                QString("Duplicate nonce: %1").arg(tx.nonce));
        }
        return Result<void>::fail(ErrorCode::DatabaseInsertFailed,
            QString("Failed to insert transaction: %1").arg(err));
    }

    return Result<void>::ok();
}

Result<std::optional<Transaction>> TransactionRepository::findById(const QString& id) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("SELECT * FROM transactions WHERE id = ?"));
    query.addBindValue(id);

    if (!query.exec()) {
        return Result<std::optional<Transaction>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    if (query.next()) {
        return Result<std::optional<Transaction>>::ok(txFromQuery(query));
    }

    return Result<std::optional<Transaction>>::ok(std::nullopt);
}

Result<std::optional<Transaction>> TransactionRepository::findByNonce(const QString& nonce) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("SELECT * FROM transactions WHERE nonce = ?"));
    query.addBindValue(nonce);

    if (!query.exec()) {
        return Result<std::optional<Transaction>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    if (query.next()) {
        return Result<std::optional<Transaction>>::ok(txFromQuery(query));
    }

    return Result<std::optional<Transaction>>::ok(std::nullopt);
}

Result<std::vector<Transaction>> TransactionRepository::findByWalletId(
    const QString& walletId, int offset, int limit) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "SELECT * FROM transactions WHERE wallet_id = ? "
        "ORDER BY created_at DESC LIMIT ? OFFSET ?"
    ));
    query.addBindValue(walletId);
    query.addBindValue(limit);
    query.addBindValue(offset);

    if (!query.exec()) {
        return Result<std::vector<Transaction>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    std::vector<Transaction> txs;
    while (query.next()) {
        txs.push_back(txFromQuery(query));
    }
    return Result<std::vector<Transaction>>::ok(std::move(txs));
}

Result<std::vector<Transaction>> TransactionRepository::findByUserId(
    const QString& userId, int offset, int limit) {
    QSqlQuery query(m_db->sqlDatabase());
    // Join with wallets to find all transactions for a user
    query.prepare(QStringLiteral(
        "SELECT t.* FROM transactions t "
        "INNER JOIN wallets w ON t.wallet_id = w.id "
        "WHERE w.user_id = ? "
        "ORDER BY t.created_at DESC LIMIT ? OFFSET ?"
    ));
    query.addBindValue(userId);
    query.addBindValue(limit);
    query.addBindValue(offset);

    if (!query.exec()) {
        return Result<std::vector<Transaction>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    std::vector<Transaction> txs;
    while (query.next()) {
        txs.push_back(txFromQuery(query));
    }
    return Result<std::vector<Transaction>>::ok(std::move(txs));
}

Result<std::vector<Transaction>> TransactionRepository::findAll() {
    QSqlQuery query(m_db->sqlDatabase());
    if (!query.exec(QStringLiteral("SELECT * FROM transactions ORDER BY created_at DESC"))) {
        return Result<std::vector<Transaction>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    std::vector<Transaction> txs;
    while (query.next()) {
        txs.push_back(txFromQuery(query));
    }
    return Result<std::vector<Transaction>>::ok(std::move(txs));
}

Result<int> TransactionRepository::countByWalletId(const QString& walletId) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("SELECT COUNT(*) FROM transactions WHERE wallet_id = ?"));
    query.addBindValue(walletId);

    if (!query.exec()) {
        return Result<int>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }
    if (query.next()) {
        return Result<int>::ok(query.value(0).toInt());
    }
    return Result<int>::ok(0);
}

Result<void> TransactionRepository::removeByUserId(const QString& userId) {
    QSqlQuery query(m_db->sqlDatabase());
    // Delete transactions where wallet belongs to user (per architecture §6.4)
    query.prepare(QStringLiteral(
        "DELETE FROM transactions "
        "WHERE wallet_id IN (SELECT id FROM wallets WHERE user_id = ?)"));
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseDeleteFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> TransactionRepository::removeByWalletId(const QString& walletId) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("DELETE FROM transactions WHERE wallet_id = ?"));
    query.addBindValue(walletId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseDeleteFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> TransactionRepository::updateStatus(const QString& txId, int newStatus) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("UPDATE transactions SET status = ? WHERE id = ?"));
    query.addBindValue(newStatus);
    query.addBindValue(txId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseUpdateFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<std::vector<Transaction>> TransactionRepository::findReceived(
    const QString& publicKeyHex, int offset, int limit) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "SELECT * FROM transactions WHERE to_public_key = ? "
        "ORDER BY created_at DESC LIMIT ? OFFSET ?"
    ));
    query.addBindValue(publicKeyHex);
    query.addBindValue(limit);
    query.addBindValue(offset);

    if (!query.exec()) {
        return Result<std::vector<Transaction>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    std::vector<Transaction> txs;
    while (query.next()) {
        auto tx = txFromQuery(query);
        tx.isIncoming = true;
        txs.push_back(std::move(tx));
    }
    return Result<std::vector<Transaction>>::ok(std::move(txs));
}

Result<int> TransactionRepository::countReceived(const QString& publicKeyHex) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("SELECT COUNT(*) FROM transactions WHERE to_public_key = ?"));
    query.addBindValue(publicKeyHex);

    if (!query.exec()) {
        return Result<int>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }
    if (query.next()) {
        return Result<int>::ok(query.value(0).toInt());
    }
    return Result<int>::ok(0);
}

Transaction TransactionRepository::txFromQuery(QSqlQuery& query) const {
    Transaction tx;
    tx.id = query.value("id").toString();
    tx.walletId = query.value("wallet_id").toString();
    tx.toPublicKey = query.value("to_public_key").toString();
    tx.amount = query.value("amount").toDouble();
    tx.fee = query.value("fee").toDouble();
    tx.signature = query.value("signature").toByteArray();
    tx.nonce = query.value("nonce").toString();
    tx.status = query.value("status").toInt();
    tx.memo = query.value("memo").toString();
    tx.createdAt = query.value("created_at").toLongLong();
    return tx;
}
