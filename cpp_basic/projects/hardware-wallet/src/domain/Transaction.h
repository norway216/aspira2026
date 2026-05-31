#pragma once

#include <QString>
#include <QByteArray>

struct Transaction {
    QString id;
    QString walletId;
    QString toPublicKey;           // Recipient's Ed25519 public key (hex)
    double amount = 0.0;
    double fee = 0.0;
    QByteArray signature;          // Ed25519 signature (64 bytes)
    QString nonce;                 // Unique nonce (hex)
    int status = 0;                // 0=pending, 1=confirmed, 2=failed
    QString memo;
    qint64 createdAt = 0;
    bool isIncoming = false;       // For display

    enum Status { Pending = 0, Confirmed = 1, Failed = 2 };

    QString statusString() const {
        switch (status) {
            case Pending: return "Pending";
            case Confirmed: return "Confirmed";
            case Failed: return "Failed";
            default: return "Unknown";
        }
    }

    bool isValid() const {
        return !id.isEmpty() && !walletId.isEmpty() && !toPublicKey.isEmpty()
               && amount > 0 && !nonce.isEmpty() && !signature.isEmpty();
    }
};
