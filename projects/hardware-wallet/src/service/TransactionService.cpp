#include "TransactionService.h"
#include "AuthService.h"
#include "persistence/Database.h"
#include "persistence/WalletRepository.h"
#include "persistence/TransactionRepository.h"
#include "persistence/AuditLogRepository.h"
#include "domain/AuditLogEntry.h"
#include "worker/CryptoTask.h"
#include "crypto/SignatureHelper.h"
#include "crypto/KeyManager.h"
#include "crypto/AEADHelper.h"
#include "crypto/HashChain.h"
#include "util/CryptoUtils.h"

#include <QDebug>

TransactionService::TransactionService(Database* db, AuthService* authService,
                                         QThreadPool* cryptoPool, QObject* parent)
    : QObject(parent), m_db(db), m_authService(authService), m_cryptoPool(cryptoPool) {
    m_walletRepo = new WalletRepository(db);
    m_txRepo = new TransactionRepository(db);
    m_auditRepo = new AuditLogRepository(db);
}

void TransactionService::sendTransaction(const QString& fromWalletId,
                                          const QString& toPublicKeyHex,
                                          double amount, const QString& memo) {
    if (!m_authService->isLoggedIn()) {
        emit sendCompleted(Result<Transaction>::fail(ErrorCode::NotLoggedIn, "Not logged in"));
        return;
    }

    if (amount <= 0) {
        emit sendCompleted(Result<Transaction>::fail(ErrorCode::InvalidAmount, "Amount must be positive"));
        return;
    }

    // Get sender wallet
    auto senderResult = m_walletRepo->findById(fromWalletId);
    if (senderResult.isFail() || !senderResult.value().has_value()) {
        emit sendCompleted(Result<Transaction>::fail(ErrorCode::WalletNotFound, "Sender wallet not found"));
        return;
    }

    Wallet senderWallet = std::move(senderResult.value().value());

    // Balance check
    double fee = 0.0; // No fee for simulated transactions
    if (senderWallet.balance < amount + fee) {
        emit sendCompleted(Result<Transaction>::fail(ErrorCode::InsufficientBalance,
            QString("Insufficient balance. Have: %1, need: %2")
                .arg(senderWallet.balance).arg(amount + fee)));
        return;
    }

    // Find recipient wallet
    auto recipientResult = m_walletRepo->findByPublicKey(toPublicKeyHex);
    if (recipientResult.isFail()) {
        emit sendCompleted(Result<Transaction>::fail(recipientResult.error().code,
            recipientResult.error().message));
        return;
    }

    auto&& optRecipient = recipientResult.value();
    if (!optRecipient.has_value()) {
        emit sendCompleted(Result<Transaction>::fail(ErrorCode::RecipientNotFound,
            "Recipient not found. Check the public key."));
        return;
    }

    Wallet recipientWallet = std::move(optRecipient.value());

    // Prevent sending to self
    if (senderWallet.id == recipientWallet.id) {
        emit sendCompleted(Result<Transaction>::fail(ErrorCode::InvalidRecipient,
            "Cannot send to yourself"));
        return;
    }

    // Generate nonce
    auto nonceResult = KeyManager::createNonce();
    if (nonceResult.isFail()) {
        emit sendCompleted(Result<Transaction>::fail(nonceResult.error().code, nonceResult.error().message));
        return;
    }

    // Create transaction draft
    Transaction tx;
    tx.id = CryptoUtils::generateId();
    tx.walletId = senderWallet.id;
    tx.toPublicKey = toPublicKeyHex;
    tx.amount = amount;
    tx.fee = fee;
    tx.nonce = nonceResult.value().toHex();
    tx.status = Transaction::Pending;
    tx.memo = memo;
    tx.createdAt = CryptoUtils::currentTimestampMs();

    // Build canonical message for signing
    // tx_message = SHA256(version || from || to || amount || fee || nonce || timestamp)
    QByteArray msgData;
    msgData.append("1"); // version
    msgData.append(senderWallet.publicKeyHex().toUtf8());
    msgData.append(toPublicKeyHex.toUtf8());
    msgData.append(QByteArray::number(amount, 'f', 8));
    msgData.append(QByteArray::number(fee, 'f', 8));
    msgData.append(tx.nonce.toUtf8());
    msgData.append(QByteArray::number(tx.createdAt));

    // Hash the message
    auto hashResult = HashChain::blake2b(
        reinterpret_cast<const unsigned char*>(msgData.constData()), msgData.size());
    if (hashResult.isFail()) {
        emit sendCompleted(Result<Transaction>::fail(hashResult.error().code, hashResult.error().message));
        return;
    }

    QString toUserId = recipientWallet.userId;

    // Need to decrypt the wallet seed to get the secret key
    // The seed was encrypted with DEK
    auto seedResult = AEADHelper::decrypt(
        SecureBuffer::fromQByteArray(senderWallet.encryptedSeed), m_authService->currentDEK());
    if (seedResult.isFail()) {
        emit sendCompleted(Result<Transaction>::fail(ErrorCode::DecryptionFailed, "Failed to access wallet seed"));
        return;
    }

    // Reconstruct key pair from seed
    auto keyPairResult = SignatureHelper::generateFromSeed(seedResult.value());
    if (keyPairResult.isFail()) {
        emit sendCompleted(Result<Transaction>::fail(keyPairResult.error().code, keyPairResult.error().message));
        return;
    }

    // Sign the transaction
    auto sigResult = SignatureHelper::sign(hashResult.value(), keyPairResult.value().secretKey);
    if (sigResult.isFail()) {
        emit sendCompleted(Result<Transaction>::fail(sigResult.error().code, sigResult.error().message));
        return;
    }

    tx.signature = sigResult.value().toQByteArray();
    tx.status = Transaction::Confirmed;

    // Begin database transaction for atomic update
    auto dbTxResult = m_db->beginTransaction();
    if (dbTxResult.isFail()) {
        emit sendCompleted(Result<Transaction>::fail(dbTxResult.error().code, dbTxResult.error().message));
        return;
    }

    // Update sender balance
    double newSenderBalance = senderWallet.balance - amount - fee;
    auto updateResult = m_walletRepo->updateBalance(senderWallet.id, newSenderBalance);
    if (updateResult.isFail()) {
        m_db->rollback();
        emit sendCompleted(Result<Transaction>::fail(updateResult.error().code, updateResult.error().message));
        return;
    }

    // Update recipient balance
    double newRecipientBalance = recipientWallet.balance + amount;
    auto updateRecipientResult = m_walletRepo->updateBalance(recipientWallet.id, newRecipientBalance);
    if (updateRecipientResult.isFail()) {
        m_db->rollback();
        emit sendCompleted(Result<Transaction>::fail(updateRecipientResult.error().code, updateRecipientResult.error().message));
        return;
    }

    // Insert transaction
    auto insertResult = m_txRepo->insert(tx);
    if (insertResult.isFail()) {
        m_db->rollback();
        emit sendCompleted(Result<Transaction>::fail(insertResult.error().code, insertResult.error().message));
        return;
    }

    // Commit
    auto commitResult = m_db->commit();
    if (commitResult.isFail()) {
        m_db->rollback();
        emit sendCompleted(Result<Transaction>::fail(commitResult.error().code, commitResult.error().message));
        return;
    }

    // Log audit (outside transaction - audit log is independent)
    m_auditRepo->append(AuditLogEntry::ACTION_SEND_TX, m_authService->currentUserId(),
        QString("{\"tx_id\":\"%1\",\"amount\":%2,\"to\":\"%3\"}")
            .arg(tx.id).arg(amount).arg(recipientWallet.userId));

    qDebug() << "Transaction completed:" << tx.id << "amount:" << amount
             << "from:" << senderWallet.id << "to:" << recipientWallet.id;

    emit sendCompleted(Result<Transaction>::ok(std::move(tx)));
}

void TransactionService::getHistory(const QString& walletId, int offset, int limit) {
    auto result = m_txRepo->findByWalletId(walletId, offset, limit);
    if (result.isFail()) {
        emit historyLoaded(Result<std::vector<Transaction>>::fail(result.error().code, result.error().message));
        return;
    }
    emit historyLoaded(Result<std::vector<Transaction>>::ok(std::move(result.value())));
}

void TransactionService::getReceivedHistory(const QString& publicKeyHex, int offset, int limit) {
    auto result = m_txRepo->findReceived(publicKeyHex, offset, limit);
    if (result.isFail()) {
        emit historyLoaded(Result<std::vector<Transaction>>::fail(result.error().code, result.error().message));
        return;
    }
    // Re-emit through same signal since DashboardViewModel connects to historyLoaded
    emit historyLoaded(Result<std::vector<Transaction>>::ok(std::move(result.value())));
}

void TransactionService::getFullHistory(const QString& walletId, const QString& publicKeyHex,
                                         int offset, int limit) {
    // Get both sent and received, then merge and sort
    auto sentResult = m_txRepo->findByWalletId(walletId, 0, 1000);
    auto receivedResult = m_txRepo->findReceived(publicKeyHex, 0, 1000);

    std::vector<Transaction> allTxs;

    if (sentResult.isOk()) {
        for (auto& tx : sentResult.value()) {
            tx.isIncoming = false;
            allTxs.push_back(std::move(tx));
        }
    }

    if (receivedResult.isOk()) {
        for (auto& tx : receivedResult.value()) {
            tx.isIncoming = true;
            allTxs.push_back(std::move(tx));
        }
    }

    // Sort by creation time (newest first)
    std::sort(allTxs.begin(), allTxs.end(),
              [](const Transaction& a, const Transaction& b) {
                  return a.createdAt > b.createdAt;
              });

    // Apply pagination
    int start = offset;
    int end = std::min(offset + limit, static_cast<int>(allTxs.size()));
    std::vector<Transaction> page;
    for (int i = start; i < end; ++i) {
        page.push_back(std::move(allTxs[i]));
    }

    emit historyLoaded(Result<std::vector<Transaction>>::ok(std::move(page)));
}

void TransactionService::getTransactionCount(const QString& walletId) {
    auto result = m_txRepo->countByWalletId(walletId);
    if (result.isFail()) {
        emit transactionCountLoaded(Result<int>::fail(result.error().code, result.error().message));
        return;
    }
    emit transactionCountLoaded(Result<int>::ok(result.value()));
}

void TransactionService::getBalance(const QString& walletId) {
    auto result = m_walletRepo->findById(walletId);
    if (result.isFail() || !result.value().has_value()) {
        emit balanceLoaded(Result<double>::fail(ErrorCode::WalletNotFound, "Wallet not found"));
        return;
    }
    emit balanceLoaded(Result<double>::ok(result.value().value().balance));
}
