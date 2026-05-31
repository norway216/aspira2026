#include "AuditLogRepository.h"
#include "Database.h"
#include "crypto/HashChain.h"
#include "util/CryptoUtils.h"

#include <QSqlQuery>
#include <QSqlError>
#include <QVariant>
#include <QDebug>

AuditLogRepository::AuditLogRepository(Database* db) : m_db(db) {}

Result<AuditLogEntry> AuditLogRepository::append(const QString& action, const QString& userId,
                                                   const QString& details) {
    // Get previous hash
    auto lastResult = lastEntry();
    SecureBuffer prevHash;
    if (lastResult.isOk() && lastResult.value().has_value()) {
        prevHash = SecureBuffer::fromHex(lastResult.value()->hash);
    } else {
        prevHash = HashChain::genesisHash();
    }

    qint64 now = CryptoUtils::currentTimestampMs();

    // Compute current hash
    auto hashResult = HashChain::computeHash(prevHash, action, userId, details, now);
    if (hashResult.isFail()) {
        return Result<AuditLogEntry>::fail(hashResult.error().code, hashResult.error().message);
    }

    AuditLogEntry entry;
    entry.action = action;
    entry.userId = userId;
    entry.details = details;
    entry.previousHash = prevHash.toHex();
    entry.hash = hashResult.value().toHex();
    entry.createdAt = now;

    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "INSERT INTO audit_logs (action, user_id, details, previous_hash, hash, created_at) "
        "VALUES (?, ?, ?, ?, ?, ?)"
    ));

    query.addBindValue(entry.action);
    query.addBindValue(entry.userId);
    query.addBindValue(entry.details);
    query.addBindValue(entry.previousHash);
    query.addBindValue(entry.hash);
    query.addBindValue(entry.createdAt);

    if (!query.exec()) {
        return Result<AuditLogEntry>::fail(ErrorCode::DatabaseInsertFailed,
            QString("Failed to insert audit log: %1").arg(query.lastError().text()));
    }

    entry.id = query.lastInsertId().toLongLong();
    return Result<AuditLogEntry>::ok(std::move(entry));
}

Result<std::vector<AuditLogEntry>> AuditLogRepository::findAll(int offset, int limit) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT ? OFFSET ?"
    ));
    query.addBindValue(limit);
    query.addBindValue(offset);

    if (!query.exec()) {
        return Result<std::vector<AuditLogEntry>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    std::vector<AuditLogEntry> entries;
    while (query.next()) {
        entries.push_back(entryFromQuery(query));
    }
    return Result<std::vector<AuditLogEntry>>::ok(std::move(entries));
}

Result<std::optional<AuditLogEntry>> AuditLogRepository::lastEntry() {
    QSqlQuery query(m_db->sqlDatabase());
    if (!query.exec(QStringLiteral(
            "SELECT * FROM audit_logs ORDER BY id DESC LIMIT 1"))) {
        return Result<std::optional<AuditLogEntry>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    if (query.next()) {
        return Result<std::optional<AuditLogEntry>>::ok(entryFromQuery(query));
    }

    return Result<std::optional<AuditLogEntry>>::ok(std::nullopt);
}

Result<bool> AuditLogRepository::verifyChain() {
    QSqlQuery query(m_db->sqlDatabase());
    if (!query.exec(QStringLiteral("SELECT * FROM audit_logs ORDER BY id ASC"))) {
        return Result<bool>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    SecureBuffer expectedPrevHash = HashChain::genesisHash();
    int entryCount = 0;

    while (query.next()) {
        AuditLogEntry entry = entryFromQuery(query);
        entryCount++;

        // Verify previous hash matches expected
        SecureBuffer actualPrevHash = SecureBuffer::fromHex(entry.previousHash);
        if (actualPrevHash.size() != expectedPrevHash.size()) {
            qWarning() << "Audit chain broken at entry" << entry.id << ": hash size mismatch";
            return Result<bool>::ok(false);
        }

        if (!CryptoUtils::secureCompare(actualPrevHash, expectedPrevHash)) {
            qWarning() << "Audit chain broken at entry" << entry.id << ": previous hash mismatch";
            return Result<bool>::ok(false);
        }

        // Verify current hash
        auto verifyResult = HashChain::verifyLink(
            actualPrevHash, entry.action, entry.userId, entry.details,
            entry.createdAt, SecureBuffer::fromHex(entry.hash));

        if (verifyResult.isFail() || !verifyResult.value()) {
            qWarning() << "Audit chain broken at entry" << entry.id << ": hash verification failed";
            return Result<bool>::ok(false);
        }

        expectedPrevHash = SecureBuffer::fromHex(entry.hash);
    }

    qDebug() << "Audit chain verified:" << entryCount << "entries, chain intact";
    return Result<bool>::ok(true);
}

Result<int> AuditLogRepository::count() {
    QSqlQuery query(m_db->sqlDatabase());
    if (!query.exec(QStringLiteral("SELECT COUNT(*) FROM audit_logs"))) {
        return Result<int>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }
    if (query.next()) {
        return Result<int>::ok(query.value(0).toInt());
    }
    return Result<int>::ok(0);
}

Result<std::vector<AuditLogEntry>> AuditLogRepository::getAllEntries() {
    return findAll(0, 1000000); // effectively unlimited for backup
}

AuditLogEntry AuditLogRepository::entryFromQuery(QSqlQuery& query) const {
    AuditLogEntry entry;
    entry.id = query.value("id").toLongLong();
    entry.action = query.value("action").toString();
    entry.userId = query.value("user_id").toString();
    entry.details = query.value("details").toString();
    entry.previousHash = query.value("previous_hash").toString();
    entry.hash = query.value("hash").toString();
    entry.createdAt = query.value("created_at").toLongLong();
    return entry;
}
