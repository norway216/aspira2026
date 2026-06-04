#include "WalletService.h"
#include "AuthService.h"
#include "persistence/Database.h"
#include "persistence/WalletRepository.h"
#include "persistence/AuditLogRepository.h"
#include "domain/AuditLogEntry.h"
#include "worker/CryptoTask.h"
#include "crypto/SignatureHelper.h"
#include "crypto/KeyManager.h"
#include "crypto/AEADHelper.h"
#include "util/CryptoUtils.h"

#include <QDebug>

WalletService::WalletService(Database* db, AuthService* authService,
                               QThreadPool* cryptoPool, QObject* parent)
    : QObject(parent), m_db(db), m_authService(authService), m_cryptoPool(cryptoPool) {
    m_walletRepo = new WalletRepository(db);
    m_auditRepo = new AuditLogRepository(db);
}

void WalletService::createWallet() {
    if (!m_authService->isLoggedIn()) {
        emit walletCreated(Result<Wallet>::fail(ErrorCode::NotLoggedIn, "Not logged in"));
        return;
    }

    // Check if wallet already exists
    auto existing = m_walletRepo->findByUserId(m_authService->currentUserId());
    if (existing.isOk() && existing.value().has_value()) {
        emit walletCreated(Result<Wallet>::fail(ErrorCode::WalletAlreadyExists, "Wallet already exists"));
        return;
    }

    QString userId = m_authService->currentUserId();

    // Generate Ed25519 key pair
    auto keyPairResult = SignatureHelper::generateKeyPair();
    if (keyPairResult.isFail()) {
        emit walletCreated(Result<Wallet>::fail(keyPairResult.error().code,
            keyPairResult.error().message));
        return;
    }
    auto& keyPair = keyPairResult.value();

    // Generate wallet seed
    auto seedResult = KeyManager::createWalletSeed();
    if (seedResult.isFail()) {
        emit walletCreated(Result<Wallet>::fail(seedResult.error().code, seedResult.error().message));
        return;
    }

    // Encrypt seed with DEK
    auto encryptedSeedResult = AEADHelper::encrypt(
        seedResult.value(), m_authService->currentDEK());
    if (encryptedSeedResult.isFail()) {
        emit walletCreated(Result<Wallet>::fail(encryptedSeedResult.error().code,
            encryptedSeedResult.error().message));
        return;
    }

    // Create wallet
    Wallet wallet;
    wallet.id = CryptoUtils::generateId();
    wallet.userId = userId;
    wallet.encryptedSeed = encryptedSeedResult.value().toQByteArray();
    wallet.publicKey = keyPair.publicKey.toQByteArray();
    wallet.balance = 10000.0;
    wallet.createdAt = CryptoUtils::currentTimestampMs();

    auto insertResult = m_walletRepo->insert(wallet);
    if (insertResult.isFail()) {
        emit walletCreated(Result<Wallet>::fail(insertResult.error().code,
            insertResult.error().message));
        return;
    }

    m_auditRepo->append(AuditLogEntry::ACTION_WALLET_CREATE, userId,
        QString("{\"wallet_id\":\"%1\"}").arg(wallet.id));

    qDebug() << "Wallet created for user:" << userId << "wallet id:" << wallet.id;
    emit walletCreated(Result<Wallet>::ok(std::move(wallet)));
}

void WalletService::getWallet() {
    if (!m_authService->isLoggedIn()) {
        emit walletLoaded(Result<Wallet>::fail(ErrorCode::NotLoggedIn, "Not logged in"));
        return;
    }

    auto result = m_walletRepo->findByUserId(m_authService->currentUserId());
    if (result.isFail()) {
        emit walletLoaded(Result<Wallet>::fail(result.error().code, result.error().message));
        return;
    }

    auto optWallet = result.value();
    if (!optWallet.has_value()) {
        emit walletLoaded(Result<Wallet>::fail(ErrorCode::WalletNotFound, "No wallet found"));
        return;
    }

    emit walletLoaded(Result<Wallet>::ok(std::move(optWallet.value())));
}

void WalletService::getBalance() {
    if (!m_authService->isLoggedIn()) {
        emit balanceLoaded(Result<double>::fail(ErrorCode::NotLoggedIn, "Not logged in"));
        return;
    }

    auto result = m_walletRepo->findByUserId(m_authService->currentUserId());
    if (result.isFail()) {
        emit balanceLoaded(Result<double>::fail(result.error().code, result.error().message));
        return;
    }

    auto optWallet = result.value();
    if (!optWallet.has_value()) {
        emit balanceLoaded(Result<double>::fail(ErrorCode::WalletNotFound, "No wallet found"));
        return;
    }

    emit balanceLoaded(Result<double>::ok(optWallet.value().balance));
}

void WalletService::deleteWallet() {
    if (!m_authService->isLoggedIn()) {
        emit walletDeleted(Result<void>::fail(ErrorCode::NotLoggedIn, "Not logged in"));
        return;
    }

    auto result = m_walletRepo->findByUserId(m_authService->currentUserId());
    if (result.isFail() || !result.value().has_value()) {
        emit walletDeleted(Result<void>::fail(ErrorCode::WalletNotFound, "No wallet to delete"));
        return;
    }

    QString walletId = result.value()->id;
    QString userId = m_authService->currentUserId();
    auto removeResult = m_walletRepo->remove(walletId);
    if (removeResult.isFail()) {
        emit walletDeleted(Result<void>::fail(removeResult.error().code, removeResult.error().message));
        return;
    }

    m_auditRepo->append(AuditLogEntry::ACTION_WALLET_DELETE, userId,
        QString("{\"wallet_id\":\"%1\"}").arg(walletId));
    qDebug() << "Wallet deleted for user:" << userId;
    emit walletDeleted(Result<void>::ok());
}

void WalletService::listAllWallets() {
    auto result = m_walletRepo->findAll();
    if (result.isFail()) {
        emit allWalletsLoaded(Result<std::vector<Wallet>>::fail(result.error().code, result.error().message));
        return;
    }
    emit allWalletsLoaded(Result<std::vector<Wallet>>::ok(std::move(result.value())));
}

void WalletService::getWalletByUserId(const QString& userId) {
    auto result = m_walletRepo->findByUserId(userId);
    if (result.isFail()) {
        emit walletByUserIdLoaded(Result<Wallet>::fail(result.error().code, result.error().message));
        return;
    }

    auto optWallet = result.value();
    if (!optWallet.has_value()) {
        emit walletByUserIdLoaded(Result<Wallet>::fail(ErrorCode::WalletNotFound,
            "Wallet not found for user"));
        return;
    }

    emit walletByUserIdLoaded(Result<Wallet>::ok(std::move(optWallet.value())));
}
