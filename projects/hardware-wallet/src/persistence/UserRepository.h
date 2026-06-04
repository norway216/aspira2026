#pragma once

#include "util/Result.h"
#include "domain/User.h"

#include <optional>
#include <vector>

class Database;

class UserRepository {
public:
    explicit UserRepository(Database* db);

    // Create tables
    Result<void> initSchema();

    // CRUD operations
    Result<void> insert(const User& user);
    Result<std::optional<User>> findById(const QString& id);
    Result<std::optional<User>> findByUsername(const QString& username);
    Result<std::vector<User>> findAll();

    // Update operations
    Result<void> updateLoginTime(const QString& userId, qint64 timestamp);
    Result<void> incrementFailedAttempts(const QString& userId);
    Result<void> resetFailedAttempts(const QString& userId);
    Result<void> lockUser(const QString& userId, qint64 lockedUntil);
    Result<void> updatePasswordVerifier(const QString& userId, const QByteArray& newVerifier,
                                        const QByteArray& newEncryptedDEK);

    // Security wipe operations (per architecture §6.1)
    Result<void> markWalletWiped(const QString& userId, qint64 wipedAt);
    Result<void> destroyEncryptedDEK(const QString& userId);
    Result<void> markBackupCreated(const QString& userId, qint64 backupAt);

    // Delete
    Result<void> remove(const QString& userId);

    // Count
    Result<int> count();

private:
    Database* m_db;

    // Serialize user to/from DB columns
    User userFromQuery(class QSqlQuery& query) const;
};
