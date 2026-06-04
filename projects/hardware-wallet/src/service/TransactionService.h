#pragma once

#include <QObject>
#include <QThreadPool>

#include "util/Result.h"
#include "domain/Transaction.h"
#include "domain/Wallet.h"

#include <vector>

class Database;
class WalletRepository;
class TransactionRepository;
class AuditLogRepository;
class AuthService;

/**
 * TransactionService - constructs, signs, verifies, and submits transactions.
 * Manages balance checks, nonce generation, and atomic DB updates.
 */
class TransactionService : public QObject {
    Q_OBJECT
public:
    explicit TransactionService(Database* db, AuthService* authService,
                                 QThreadPool* cryptoPool, QObject* parent = nullptr);

    // Send a transaction
    // fromWalletId: sender's wallet ID
    // toPublicKeyHex: recipient's Ed25519 public key as hex
    // amount: amount to send
    // memo: optional memo
    void sendTransaction(const QString& fromWalletId, const QString& toPublicKeyHex,
                         double amount, const QString& memo = {});

    // Get sent transaction history for a wallet
    void getHistory(const QString& walletId, int offset = 0, int limit = 20);

    // Get received transaction history
    void getReceivedHistory(const QString& publicKeyHex, int offset = 0, int limit = 20);

    // Get combined history (sent + received, sorted by time)
    void getFullHistory(const QString& walletId, const QString& publicKeyHex,
                        int offset = 0, int limit = 20);

    // Get transaction count for a wallet
    void getTransactionCount(const QString& walletId);

    // Get balance
    void getBalance(const QString& walletId);

signals:
    void sendCompleted(Result<Transaction> result);
    void historyLoaded(Result<std::vector<Transaction>> result);
    void receivedHistoryLoaded(Result<std::vector<Transaction>> result);
    void transactionCountLoaded(Result<int> result);
    void balanceLoaded(Result<double> result);

private:
    Database* m_db;
    WalletRepository* m_walletRepo;
    TransactionRepository* m_txRepo;
    AuditLogRepository* m_auditRepo;
    AuthService* m_authService;
    QThreadPool* m_cryptoPool;

    // Internal: complete the send after signing
    void completeSend(Transaction tx, QString toUserId);
};
