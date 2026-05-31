#pragma once

#include "Common.h"
#include <QObject>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QUrl>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <memory>
#include <functional>
#include <atomic>

namespace ehw {

/**
 * NetworkManager — Async blockchain node communication via Qt Network.
 *
 * Features:
 * - HTTP/HTTPS JSON-RPC communication with blockchain nodes
 * - Transaction broadcast
 * - Balance queries
 * - Certificate pinning for TLS security
 * - Async callback pattern — never blocks UI thread
 */
class NetworkManager : public QObject {
    Q_OBJECT

public:
    struct NodeConfig {
        QString url = "https://blockstream.info/api";
        uint16_t port = 443;
        int timeoutMs = 10000;
        int maxRetries = 3;
    };

    explicit NetworkManager(QObject* parent = nullptr);
    ~NetworkManager() override;

    void setNodeConfig(const NodeConfig& config);
    const NodeConfig& nodeConfig() const { return m_config; }

    // ---- Network Status ----
    bool isConnected() const { return m_connected.load(std::memory_order_acquire); }

    // ---- Async Operations ----
    // Query address balance
    void queryBalance(const QString& address,
                      std::function<void(Result<uint64_t>)> callback);

    // Get transaction history for address
    void queryTransactionHistory(const QString& address,
                                  std::function<void(Result<QJsonArray>)> callback);

    // Broadcast a signed transaction
    void broadcastTransaction(const ByteVector& signedTx,
                               std::function<void(Result<QString>)> callback);

    // Get recommended fee rate
    void queryFeeRate(std::function<void(Result<uint64_t>)> callback);

    // Check node health
    void checkConnection(std::function<void(Result<bool>)> callback);

    // Get address UTXOs
    void queryUtxos(const QString& address,
                    std::function<void(Result<QJsonArray>)> callback);

Q_SIGNALS:
    void connectedChanged(bool connected);
    void networkErrorOccurred(const QString& error);
    void transactionBroadcasted(const QString& txid);

private:
    void sendGetRequest(const QString& path,
                        std::function<void(Result<QJsonDocument>)> callback);
    void sendPostRequest(const QString& path, const QJsonObject& body,
                         std::function<void(Result<QJsonDocument>)> callback);

    NodeConfig m_config;
    std::unique_ptr<QNetworkAccessManager> m_nam;
    std::atomic<bool> m_connected{false};
};

} // namespace ehw
