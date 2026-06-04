#include "UserRepository.h"
#include "Database.h"
#include "crypto/CryptoProvider.h"

#include <QSqlQuery>
#include <QSqlError>
#include <QVariant>
#include <QDebug>
#include <QDateTime>

UserRepository::UserRepository(Database* db) : m_db(db) {}

Result<void> UserRepository::initSchema() {
    return Result<void>::ok();
}

Result<void> UserRepository::insert(const User& user) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "INSERT INTO users (id, username, encrypted_dek, password_verifier, salt, "
        "kdf_ops_limit, kdf_mem_limit, created_at, last_login_at, failed_attempts, locked_until) "
        "VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
    ));

    query.addBindValue(user.id);
    query.addBindValue(user.username);
    query.addBindValue(user.encryptedDEK);
    query.addBindValue(user.passwordVerifier);
    query.addBindValue(user.salt);
    query.addBindValue(static_cast<qlonglong>(user.kdfOpsLimit));
    query.addBindValue(static_cast<qlonglong>(user.kdfMemLimit));
    query.addBindValue(user.createdAt);
    query.addBindValue(user.lastLoginAt);
    query.addBindValue(user.failedAttempts);
    query.addBindValue(user.lockedUntil);

    if (!query.exec()) {
        QString err = query.lastError().text();
        if (err.contains("UNIQUE", Qt::CaseInsensitive)) {
            return Result<void>::fail(ErrorCode::UserAlreadyExists,
                QString("User '%1' already exists").arg(user.username));
        }
        return Result<void>::fail(ErrorCode::DatabaseInsertFailed,
            QString("Failed to insert user: %1").arg(err));
    }

    return Result<void>::ok();
}

Result<std::optional<User>> UserRepository::findById(const QString& id) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("SELECT * FROM users WHERE id = ?"));
    query.addBindValue(id);

    if (!query.exec()) {
        return Result<std::optional<User>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    if (query.next()) {
        return Result<std::optional<User>>::ok(userFromQuery(query));
    }

    return Result<std::optional<User>>::ok(std::nullopt);
}

Result<std::optional<User>> UserRepository::findByUsername(const QString& username) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("SELECT * FROM users WHERE username = ?"));
    query.addBindValue(username);

    if (!query.exec()) {
        return Result<std::optional<User>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    if (query.next()) {
        return Result<std::optional<User>>::ok(userFromQuery(query));
    }

    return Result<std::optional<User>>::ok(std::nullopt);
}

Result<std::vector<User>> UserRepository::findAll() {
    QSqlQuery query(m_db->sqlDatabase());
    if (!query.exec(QStringLiteral("SELECT * FROM users ORDER BY created_at DESC"))) {
        return Result<std::vector<User>>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }

    std::vector<User> users;
    while (query.next()) {
        users.push_back(userFromQuery(query));
    }
    return Result<std::vector<User>>::ok(std::move(users));
}

Result<void> UserRepository::updateLoginTime(const QString& userId, qint64 timestamp) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("UPDATE users SET last_login_at = ? WHERE id = ?"));
    query.addBindValue(timestamp);
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseUpdateFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> UserRepository::incrementFailedAttempts(const QString& userId) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "UPDATE users SET failed_attempts = failed_attempts + 1 WHERE id = ?"));
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseUpdateFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> UserRepository::resetFailedAttempts(const QString& userId) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "UPDATE users SET failed_attempts = 0, locked_until = 0 WHERE id = ?"));
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseUpdateFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> UserRepository::lockUser(const QString& userId, qint64 lockedUntil) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("UPDATE users SET locked_until = ? WHERE id = ?"));
    query.addBindValue(lockedUntil);
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseUpdateFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> UserRepository::updatePasswordVerifier(const QString& userId,
                                                     const QByteArray& newVerifier,
                                                     const QByteArray& newEncryptedDEK) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "UPDATE users SET password_verifier = ?, encrypted_dek = ? WHERE id = ?"));
    query.addBindValue(newVerifier);
    query.addBindValue(newEncryptedDEK);
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseUpdateFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> UserRepository::markWalletWiped(const QString& userId, qint64 wipedAt) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "UPDATE users SET wallet_wiped = 1, wiped_at = ? WHERE id = ?"));
    query.addBindValue(wipedAt);
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseUpdateFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> UserRepository::destroyEncryptedDEK(const QString& userId) {
    // Overwrite encrypted_dek with random garbage before delete
    // (per architecture §6.2 — prevents seed recovery even if encryptedSeed remains)
    QByteArray randomGarbage(64, 0);
    CryptoProvider::instance().randomBytes(
        reinterpret_cast<unsigned char*>(randomGarbage.data()),
        randomGarbage.size());

    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "UPDATE users SET encrypted_dek = :encrypted_dek, "
        "wallet_wiped = 1, wiped_at = :wiped_at "
        "WHERE id = :user_id"));
    query.bindValue(":encrypted_dek", randomGarbage);
    query.bindValue(":wiped_at", QDateTime::currentMSecsSinceEpoch());
    query.bindValue(":user_id", userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseUpdateFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> UserRepository::markBackupCreated(const QString& userId, qint64 backupAt) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral(
        "UPDATE users SET backup_created = 1, last_backup_at = ? WHERE id = ?"));
    query.addBindValue(backupAt);
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseUpdateFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<void> UserRepository::remove(const QString& userId) {
    QSqlQuery query(m_db->sqlDatabase());
    query.prepare(QStringLiteral("DELETE FROM users WHERE id = ?"));
    query.addBindValue(userId);

    if (!query.exec()) {
        return Result<void>::fail(ErrorCode::DatabaseDeleteFailed, query.lastError().text());
    }
    return Result<void>::ok();
}

Result<int> UserRepository::count() {
    QSqlQuery query(m_db->sqlDatabase());
    if (!query.exec(QStringLiteral("SELECT COUNT(*) FROM users"))) {
        return Result<int>::fail(ErrorCode::DatabaseQueryFailed, query.lastError().text());
    }
    if (query.next()) {
        return Result<int>::ok(query.value(0).toInt());
    }
    return Result<int>::ok(0);
}

User UserRepository::userFromQuery(QSqlQuery& query) const {
    User user;
    user.id = query.value("id").toString();
    user.username = query.value("username").toString();
    user.encryptedDEK = query.value("encrypted_dek").toByteArray();
    user.passwordVerifier = query.value("password_verifier").toByteArray();
    user.salt = query.value("salt").toByteArray();
    user.kdfOpsLimit = query.value("kdf_ops_limit").toLongLong();
    user.kdfMemLimit = static_cast<size_t>(query.value("kdf_mem_limit").toLongLong());
    user.createdAt = query.value("created_at").toLongLong();
    user.lastLoginAt = query.value("last_login_at").toLongLong();
    user.failedAttempts = query.value("failed_attempts").toInt();
    user.lockedUntil = query.value("locked_until").toLongLong();
    // Security wipe fields (V2 migration)
    user.walletWiped = query.value("wallet_wiped").toInt();
    user.wipedAt = query.value("wiped_at").toLongLong();
    user.backupCreated = query.value("backup_created").toInt();
    user.lastBackupAt = query.value("last_backup_at").toLongLong();
    return user;
}
