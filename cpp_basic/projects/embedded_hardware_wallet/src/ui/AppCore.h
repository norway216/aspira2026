#pragma once

#include <QObject>
#include <QString>
#include <QStringList>
#include <QVariantList>
#include <QVariantMap>
#include <QTimer>
#include <memory>

#include "Common.h"
#include "core/WalletCore.h"
#include "core/CryptoProvider.h"
#include "core/SecureMemory.h"
#include "transaction/TransactionManager.h"
#include "backup/BackupManager.h"
#include "network/NetworkManager.h"
#include "security/SecurityManager.h"
#include "pipeline/ThreadPool.h"

namespace ehw {

/**
 * AppCore — Central C++/QML bridge for the hardware wallet.
 *
 * Exposes properties and invokable methods to QML.
 * Owns all subsystem instances and orchestrates the wallet lifecycle.
 */
class AppCore : public QObject {
    Q_OBJECT

    // ---- Wallet State Properties ----
    Q_PROPERTY(bool walletInitialized READ isWalletInitialized NOTIFY walletStateChanged)
    Q_PROPERTY(bool walletLocked READ isWalletLocked NOTIFY walletStateChanged)
    Q_PROPERTY(QString address READ getAddress NOTIFY walletStateChanged)
    Q_PROPERTY(QString balance READ getBalance NOTIFY balanceChanged)
    Q_PROPERTY(QString mnemonic READ getMnemonic NOTIFY mnemonicGenerated)
    Q_PROPERTY(int transactionCount READ getTransactionCount NOTIFY transactionCountChanged)
    Q_PROPERTY(int backupShareCount READ getBackupShareCount NOTIFY backupStateChanged)
    Q_PROPERTY(bool networkConnected READ isNetworkConnected NOTIFY connectedChanged)
    Q_PROPERTY(int threadCount READ getThreadCount CONSTANT)
    Q_PROPERTY(QString securityLevel READ getSecurityLevel NOTIFY securityLevelChanged)
    Q_PROPERTY(bool securityEvents READ hasSecurityEvents NOTIFY securityStateChanged)
    Q_PROPERTY(QString lastError READ getLastError NOTIFY errorOccurred)

public:
    explicit AppCore(QObject* parent = nullptr);
    ~AppCore() override;

    // ---- Property Getters ----
    bool isWalletInitialized() const { return m_walletState.initialized; }
    bool isWalletLocked() const { return m_walletState.locked; }
    QString getAddress() const { return QString::fromStdString(m_walletState.address); }
    QString getBalance() const;
    QString getMnemonic() const { return m_currentMnemonic; }
    int getTransactionCount() const { return static_cast<int>(m_txManager->getHistoryCount()); }
    int getBackupShareCount() const { return m_walletState.backupShareCount; }
    bool isNetworkConnected() const { return m_network->isConnected(); }
    int getThreadCount() const { return static_cast<int>(m_threadPool->threadCount()); }
    QString getSecurityLevel() const;
    bool hasSecurityEvents() const { return m_security->securityEventCount() > 0; }
    QString getLastError() const { return m_lastError; }

    // ---- Q_INVOKABLE Methods ----
    Q_INVOKABLE void createWallet(const QString& passphrase = QString());
    Q_INVOKABLE void recoverWalletFromMnemonic(const QString& mnemonic,
                                                const QString& passphrase = QString());
    Q_INVOKABLE void lockWallet();
    Q_INVOKABLE void unlockWallet(const QString& passphrase);
    Q_INVOKABLE void signTransaction(const QString& toAddress, const QString& amountSatoshis);
    Q_INVOKABLE void createBackup(int totalShares, int threshold,
                                   const QString& passphrase);
    Q_INVOKABLE void recoverBackup(const QStringList& shares, int threshold,
                                    const QString& passphrase);
    Q_INVOKABLE void refreshBalance();
    Q_INVOKABLE void broadcastSampleTransaction();
    Q_INVOKABLE void checkNetworkStatus();
    Q_INVOKABLE void clearError();
    Q_INVOKABLE QString getBackupHistoryJson();

    // ---- Transaction History ----
    Q_INVOKABLE QVariantList getTransactionHistory();

Q_SIGNALS:
    void walletStateChanged();
    void mnemonicGenerated(const QString& mnemonic);
    void balanceChanged(const QString& balance);
    void transactionCountChanged();
    void backupStateChanged();
    void connectedChanged(bool connected);
    void securityLevelChanged();
    void securityStateChanged();
    void errorOccurred(const QString& error);
    void operationStarted(const QString& operation);
    void operationCompleted(const QString& operation);
    void backupCreated(const QStringList& shares);
    void backupRecovered();
    void transactionSigned(const QString& txid);

private:
    // Subsystems
    std::unique_ptr<WalletCore> m_walletCore;
    std::unique_ptr<TransactionManager> m_txManager;
    std::unique_ptr<BackupManager> m_backup;
    std::unique_ptr<NetworkManager> m_network;
    std::unique_ptr<SecurityManager> m_security;
    std::unique_ptr<ThreadPool> m_threadPool;

    // Wallet state
    WalletState m_walletState;
    QString m_currentMnemonic;
    SecureByteVector m_currentPrivateKey;
    ByteVector m_currentPublicKey;
    QString m_lastError;
    QString m_passphrase;

    // Current wallet data for encryption/decryption
    ByteVector m_encryptedPrivateKey;

    void updateWalletState();
    void setError(WalletError error);
    void setErrorStr(const QString& error);
    void runAsync(std::function<void()> task);
};

} // namespace ehw
