#include "AppCore.h"
#include <QJsonDocument>
#include <QJsonArray>
#include <QJsonObject>
#include <span>

namespace ehw {

AppCore::AppCore(QObject* parent)
    : QObject(parent)
    , m_walletCore(std::make_unique<WalletCore>())
    , m_txManager(std::make_unique<TransactionManager>())
    , m_backup(std::make_unique<BackupManager>())
    , m_network(std::make_unique<NetworkManager>())
    , m_security(std::make_unique<SecurityManager>())
    , m_threadPool(std::make_unique<ThreadPool>())
{
    log(LogLevel::Info, "AppCore initialized");
    m_security->initialize();

    connect(m_network.get(), &NetworkManager::connectedChanged,
            this, &AppCore::connectedChanged);
    connect(m_network.get(), &NetworkManager::networkErrorOccurred,
            this, [this](const QString& err) { setErrorStr(err); });
}

AppCore::~AppCore() {
    log(LogLevel::Info, "AppCore destroyed");
}

// ---- Property Helpers ----
QString AppCore::getBalance() const {
    double btc = static_cast<double>(m_walletState.balance) / 100000000.0;
    return QString::number(btc, 'f', 8);
}

QString AppCore::getSecurityLevel() const {
    switch (m_security->securityLevel()) {
        case SecurityManager::SecurityLevel::Standard: return "Standard";
        case SecurityManager::SecurityLevel::Enhanced: return "Enhanced";
        case SecurityManager::SecurityLevel::Maximum:  return "Maximum";
    }
    return "Unknown";
}

void AppCore::runAsync(std::function<void()> task) {
    m_threadPool->enqueue(ThreadPool::Priority::Normal, std::move(task));
}

// ---- Wallet Creation ----
void AppCore::createWallet(const QString& passphrase) {
    Q_EMIT operationStarted("Creating wallet...");
    m_passphrase = passphrase;

    runAsync([this, passphrase]() {
        auto result = m_walletCore->createWallet(passphrase.toStdString());

        if (!result.ok()) {
            setError(result.error);
            Q_EMIT operationCompleted("Wallet creation failed");
            return;
        }

        auto& wdata = result.value;
        m_currentMnemonic = QString::fromStdString(wdata.mnemonic);
        m_currentPrivateKey = ByteVector(
            wdata.masterPrivateKey.begin(), wdata.masterPrivateKey.end());
        m_currentPublicKey = std::move(wdata.masterPublicKey);

        auto encrypted = m_walletCore->encryptPrivateKey(
            m_currentPrivateKey, passphrase.toStdString());
        if (encrypted.ok()) {
            m_encryptedPrivateKey = std::move(encrypted.value);
        }

        m_walletState.initialized = true;
        m_walletState.locked = false;
        m_walletState.address = wdata.address;
        m_walletState.balance = 0;

        updateWalletState();
        Q_EMIT mnemonicGenerated(m_currentMnemonic);
        Q_EMIT operationCompleted("Wallet created successfully");
        log(LogLevel::Info, "Wallet created: {}", wdata.address);
    });
}

// ---- Wallet Recovery ----
void AppCore::recoverWalletFromMnemonic(const QString& mnemonic,
                                         const QString& passphrase) {
    Q_EMIT operationStarted("Recovering wallet...");
    m_passphrase = passphrase;

    runAsync([this, mnemonic, passphrase]() {
        auto result = m_walletCore->recoverWallet(
            mnemonic.toStdString(), passphrase.toStdString());

        if (!result.ok()) {
            setError(result.error);
            Q_EMIT operationCompleted("Wallet recovery failed");
            return;
        }

        auto& wdata = result.value;
        m_currentMnemonic = QString::fromStdString(wdata.mnemonic);
        m_currentPrivateKey = ByteVector(
            wdata.masterPrivateKey.begin(), wdata.masterPrivateKey.end());
        m_currentPublicKey = std::move(wdata.masterPublicKey);

        auto encrypted = m_walletCore->encryptPrivateKey(
            m_currentPrivateKey, passphrase.toStdString());
        if (encrypted.ok()) {
            m_encryptedPrivateKey = std::move(encrypted.value);
        }

        m_walletState.initialized = true;
        m_walletState.locked = false;
        m_walletState.address = wdata.address;

        updateWalletState();
        Q_EMIT operationCompleted("Wallet recovered successfully");
        log(LogLevel::Info, "Wallet recovered: {}", wdata.address);
    });
}

// ---- Lock / Unlock ----
void AppCore::lockWallet() {
    m_walletState.locked = true;
    m_currentPrivateKey.clear();
    Q_EMIT walletStateChanged();
    log(LogLevel::Info, "Wallet locked");
}

void AppCore::unlockWallet(const QString& passphrase) {
    Q_EMIT operationStarted("Unlocking wallet...");

    runAsync([this, passphrase]() {
        auto decrypted = m_walletCore->decryptPrivateKey(
            m_encryptedPrivateKey, passphrase.toStdString());
        if (!decrypted.ok()) {
            setErrorStr("Invalid passphrase");
            Q_EMIT operationCompleted("Unlock failed");
            return;
        }
        m_currentPrivateKey = std::move(decrypted.value);
        m_walletState.locked = false;
        Q_EMIT walletStateChanged();
        Q_EMIT operationCompleted("Wallet unlocked");
    });
}

// ---- Transaction Signing ----
void AppCore::signTransaction(const QString& toAddress, const QString& amountSatoshis) {
    if (!m_walletState.initialized || m_walletState.locked) {
        setErrorStr("Wallet not initialized or locked");
        return;
    }

    Q_EMIT operationStarted("Signing transaction...");

    runAsync([this, toAddress, amountSatoshis]() {
        uint64_t amount = amountSatoshis.toULongLong();

        TransactionInput input;
        input.txHash = ByteVector(32, 0xAA);
        input.outputIndex = 0;

        TransactionOutput output;
        output.amount = amount;
        output.address = toAddress.toStdString();
        output.scriptPubKey = {0x76, 0xA9, 0x14};

        auto addrData = toAddress.toLatin1();
        auto addrHash = crypto().sha256(
            std::span<const uint8_t>(
                reinterpret_cast<const uint8_t*>(addrData.constData()),
                static_cast<size_t>(addrData.size())));
        if (addrHash.ok()) {
            output.scriptPubKey.insert(output.scriptPubKey.end(),
                                        addrHash.value.begin(),
                                        addrHash.value.begin() + 20);
        } else {
            output.scriptPubKey.resize(23, 0x00);
        }
        output.scriptPubKey.push_back(0x88);
        output.scriptPubKey.push_back(0xAC);

        auto txResult = m_txManager->createTransaction({input}, {output});
        if (!txResult.ok()) {
            setError(txResult.error);
            Q_EMIT operationCompleted("Transaction creation failed");
            return;
        }

        // Sign with private key (pass vector directly — span converts implicitly)
        auto signedResult = m_txManager->signTransaction(
            txResult.value,
            std::span<const uint8_t>(m_currentPrivateKey.data(), m_currentPrivateKey.size()),
            std::span<const uint8_t>(m_currentPublicKey.data(), m_currentPublicKey.size()));

        if (!signedResult.ok()) {
            setError(signedResult.error);
            Q_EMIT operationCompleted("Signing failed");
            return;
        }

        m_txManager->addToHistory(signedResult.value);

        m_network->broadcastTransaction(
            signedResult.value.signedTx,
            [this](Result<QString> result) {
                if (result.ok()) {
                    Q_EMIT transactionSigned(result.value);
                    Q_EMIT transactionCountChanged();
                }
            });

        Q_EMIT transactionSigned(
            QString::fromStdString(bytesToHex(signedResult.value.txHash)));
        Q_EMIT operationCompleted("Transaction signed");
    });
}

// ---- Backup ----
void AppCore::createBackup(int totalShares, int threshold,
                            const QString& passphrase) {
    if (m_currentPrivateKey.empty()) {
        setErrorStr("No wallet to backup");
        return;
    }

    Q_EMIT operationStarted("Creating backup...");

    runAsync([this, totalShares, threshold, passphrase]() {
        auto seedResult = m_walletCore->mnemonicToSeed(
            m_currentMnemonic.toStdString(), passphrase.toStdString());
        if (!seedResult.ok()) {
            setError(seedResult.error);
            Q_EMIT operationCompleted("Backup failed");
            return;
        }

        auto shares = m_backup->createBackup(
            seedResult.value, passphrase.toStdString(),
            static_cast<uint8_t>(totalShares), static_cast<uint8_t>(threshold));

        if (!shares.ok()) {
            setError(shares.error);
            Q_EMIT operationCompleted("Backup failed");
            return;
        }

        m_walletState.backupShareCount = totalShares;
        m_walletState.backupThreshold = threshold;

        QStringList shareList;
        for (const auto& s : shares.value) {
            shareList.append(QString::fromStdString(s));
        }

        Q_EMIT backupCreated(shareList);
        Q_EMIT operationCompleted("Backup created");
        log(LogLevel::Info, "Backup created: {}/{} shares", totalShares, threshold);
    });
}

void AppCore::recoverBackup(const QStringList& shares, int threshold,
                             const QString& passphrase) {
    Q_EMIT operationStarted("Recovering from backup...");

    runAsync([this, shares, threshold, passphrase]() {
        std::vector<std::string> shareVec;
        shareVec.reserve(static_cast<size_t>(shares.size()));
        for (const auto& s : shares) {
            shareVec.push_back(s.toStdString());
        }

        auto recovered = m_backup->recoverFromBackup(
            shareVec, passphrase.toStdString(),
            static_cast<uint8_t>(threshold));

        if (!recovered.ok()) {
            setError(recovered.error);
            Q_EMIT operationCompleted("Recovery failed");
            return;
        }

        log(LogLevel::Info, "Backup recovered: {} bytes", recovered.value.size());
        Q_EMIT backupRecovered();
        Q_EMIT operationCompleted("Backup recovered");
    });
}

// ---- Balance ----
void AppCore::refreshBalance() {
    if (m_walletState.address.empty()) return;

    m_network->queryBalance(
        QString::fromStdString(m_walletState.address),
        [this](Result<uint64_t> result) {
            if (result.ok()) {
                m_walletState.balance = result.value;
                Q_EMIT balanceChanged(getBalance());
            }
        });
}

// ---- Broadcast ----
void AppCore::broadcastSampleTransaction() {
    Q_EMIT operationStarted("Broadcasting transaction...");
    if (!m_txManager->getHistory().empty()) {
        const auto& lastTx = m_txManager->getHistory().back();
        if (!lastTx.signedTx.empty()) {
            m_network->broadcastTransaction(
                lastTx.signedTx,
                [this](Result<QString> result) {
                    if (result.ok()) {
                        Q_EMIT operationCompleted("Transaction broadcasted");
                    } else {
                        setError(result.error);
                    }
                });
            return;
        }
    }
    setErrorStr("No signed transaction to broadcast");
}

// ---- Network ----
void AppCore::checkNetworkStatus() {
    m_network->checkConnection([](Result<bool>) {});
}

// ---- Error Handling ----
void AppCore::clearError() {
    m_lastError.clear();
}

void AppCore::setError(WalletError error) {
    m_lastError = QString::fromStdString(walletErrorStr(error));
    if (error != WalletError::Success) {
        Q_EMIT errorOccurred(m_lastError);
    }
}

void AppCore::setErrorStr(const QString& error) {
    m_lastError = error;
    Q_EMIT errorOccurred(error);
}

// ---- Backup History ----
QString AppCore::getBackupHistoryJson() {
    QJsonArray arr;
    for (const auto& v : m_backup->getBackupHistory()) {
        QJsonObject obj;
        obj["version"] = static_cast<int>(v.version);
        obj["timestamp"] = QString::fromStdString(v.timestamp);
        obj["totalShares"] = static_cast<int>(v.totalShares);
        obj["threshold"] = static_cast<int>(v.threshold);
        arr.append(obj);
    }
    return QString::fromUtf8(QJsonDocument(arr).toJson(QJsonDocument::Compact));
}

// ---- Transaction History ----
QVariantList AppCore::getTransactionHistory() {
    QVariantList list;
    for (const auto& tx : m_txManager->getHistory()) {
        QVariantMap txMap;
        txMap["txid"] = QString::fromStdString(bytesToHex(tx.txHash));
        txMap["inputs"] = static_cast<int>(tx.inputs.size());
        txMap["outputs"] = static_cast<int>(tx.outputs.size());
        txMap["signed"] = tx.isSigned;
        if (!tx.outputs.empty()) {
            txMap["amount"] = QVariant::fromValue(tx.outputs[0].amount);
        }
        list.append(txMap);
    }
    return list;
}

// ---- State Management ----
void AppCore::updateWalletState() {
    Q_EMIT walletStateChanged();
    Q_EMIT balanceChanged(getBalance());
    Q_EMIT transactionCountChanged();
    Q_EMIT backupStateChanged();
}

} // namespace ehw
