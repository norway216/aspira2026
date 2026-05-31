#pragma once

#include <QString>
#include <QDateTime>
#include <QByteArray>

struct User {
    QString id;
    QString username;
    QByteArray encryptedDEK;      // DEK encrypted with password-derived KEK
    QByteArray passwordVerifier;  // Argon2id hash of password (hex)
    QByteArray salt;              // Random salt for password hashing
    unsigned long long kdfOpsLimit = 4;
    size_t kdfMemLimit = 268435456; // 256 MiB
    qint64 createdAt = 0;
    qint64 lastLoginAt = 0;
    int failedAttempts = 0;
    qint64 lockedUntil = 0;

    // Security wipe fields (per architecture §5.1)
    int walletWiped = 0;
    qint64 wipedAt = 0;
    int backupCreated = 0;
    qint64 lastBackupAt = 0;

    bool isLocked() const {
        if (lockedUntil == 0) return false;
        return QDateTime::currentMSecsSinceEpoch() < lockedUntil;
    }

    bool isValid() const {
        return !id.isEmpty() && !username.isEmpty() && !passwordVerifier.isEmpty();
    }
};
