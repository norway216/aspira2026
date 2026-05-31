#pragma once

#include <QObject>
#include <QThreadPool>
#include <memory>

#include "util/Result.h"
#include "domain/Wallet.h"
#include "crypto/SecureBuffer.h"

class Database;
class WalletRepository;
class AuditLogRepository;
class AuthService;

/**
 * WalletService - manages wallet creation, balance queries, and key management.
 */
class WalletService : public QObject {
    Q_OBJECT
public:
    explicit WalletService(Database* db, AuthService* authService,
                           QThreadPool* cryptoPool, QObject* parent = nullptr);

    // Create a new wallet for the current user
    void createWallet();

    // Delete the current user's wallet
    void deleteWallet();

    // Get wallet info for current user
    void getWallet();

    // Get balance for current user
    void getBalance();

    // Get wallet by user ID (for cross-user transfers)
    void getWalletByUserId(const QString& userId);

    // List all wallets (for recipient selection)
    void listAllWallets();

signals:
    void walletCreated(Result<Wallet> result);
    void walletDeleted(Result<void> result);
    void walletLoaded(Result<Wallet> result);
    void balanceLoaded(Result<double> result);
    void walletByUserIdLoaded(Result<Wallet> result);
    void allWalletsLoaded(Result<std::vector<Wallet>> result);

private:
    Database* m_db;
    WalletRepository* m_walletRepo;
    AuditLogRepository* m_auditRepo;
    AuthService* m_authService;
    QThreadPool* m_cryptoPool;
};
