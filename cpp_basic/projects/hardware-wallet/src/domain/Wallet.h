#pragma once

#include <QString>
#include <QByteArray>

struct Wallet {
    QString id;
    QString userId;
    QByteArray encryptedSeed;     // Encrypted wallet seed
    QByteArray publicKey;         // Ed25519 public key (32 bytes raw)
    double balance = 0.0;
    qint64 createdAt = 0;

    QString publicKeyHex() const {
        return publicKey.toHex();
    }

    QString shortAddress() const {
        if (publicKeyHex().length() > 40) {
            return publicKeyHex().left(20) + "..." + publicKeyHex().right(10);
        }
        return publicKeyHex();
    }

    bool isValid() const {
        return !id.isEmpty() && !userId.isEmpty() && !publicKey.isEmpty();
    }
};
