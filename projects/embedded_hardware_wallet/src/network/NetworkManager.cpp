#include "NetworkManager.h"
#include <QNetworkRequest>
#include <QSslConfiguration>
#include <QTimer>
#include <QUrlQuery>

namespace ehw {

NetworkManager::NetworkManager(QObject* parent)
    : QObject(parent)
    , m_nam(std::make_unique<QNetworkAccessManager>())
{
    log(LogLevel::Info, "NetworkManager initialized");
}

NetworkManager::~NetworkManager() {
    log(LogLevel::Info, "NetworkManager destroyed");
}

void NetworkManager::setNodeConfig(const NodeConfig& config) {
    m_config = config;
}

// ---- Balance Query ----
void NetworkManager::queryBalance(const QString& address,
                                   std::function<void(Result<uint64_t>)> callback)
{
    QString path = QString("/address/%1").arg(address);
    sendGetRequest(path, [callback](Result<QJsonDocument> result) {
        if (!result.ok()) {
            callback({.error = result.error});
            return;
        }
        uint64_t balance = 0;
        QJsonObject obj = result.value.object();
        // Blockstream API: chain_stats.funded_txo_sum - chain_stats.spent_txo_sum
        QJsonObject chainStats = obj.value("chain_stats").toObject();
        uint64_t funded = chainStats.value("funded_txo_sum").toVariant().toULongLong();
        uint64_t spent = chainStats.value("spent_txo_sum").toVariant().toULongLong();
        balance = funded > spent ? funded - spent : 0;

        QJsonObject mempoolStats = obj.value("mempool_stats").toObject();
        uint64_t mempoolFunded = mempoolStats.value("funded_txo_sum").toVariant().toULongLong();
        uint64_t mempoolSpent = mempoolStats.value("spent_txo_sum").toVariant().toULongLong();
        balance += (mempoolFunded > mempoolSpent ? mempoolFunded - mempoolSpent : 0);

        callback({.value = balance});
    });
}

// ---- Transaction History ----
void NetworkManager::queryTransactionHistory(const QString& address,
                                              std::function<void(Result<QJsonArray>)> callback)
{
    QString path = QString("/address/%1/txs").arg(address);
    sendGetRequest(path, [callback](Result<QJsonDocument> result) {
        if (!result.ok()) {
            callback({.error = result.error});
            return;
        }
        callback({.value = result.value.array()});
    });
}

// ---- Broadcast Transaction ----
void NetworkManager::broadcastTransaction(const ByteVector& signedTx,
                                           std::function<void(Result<QString>)> callback)
{
    QString txHex = QString::fromStdString(bytesToHex(signedTx));
    QJsonObject body;
    body["tx"] = txHex;

    QString path = "/tx";
    sendPostRequest(path, body, [this, callback](Result<QJsonDocument> result) {
        if (!result.ok()) {
            callback({.error = result.error});
            return;
        }
        QString txid = result.value.object().value("txid").toString();
        Q_EMIT transactionBroadcasted(txid);
        callback({.value = txid});
    });
}

// ---- Fee Rate ----
void NetworkManager::queryFeeRate(std::function<void(Result<uint64_t>)> callback) {
    sendGetRequest("/fee-estimates", [callback](Result<QJsonDocument> result) {
        if (!result.ok()) {
            callback({.error = result.error});
            return;
        }
        // Get the "2" block target fee rate (sat/vB)
        double feeRate = result.value.object().value("2").toDouble(10.0);
        callback({.value = static_cast<uint64_t>(feeRate)});
    });
}

// ---- Connection Check ----
void NetworkManager::checkConnection(std::function<void(Result<bool>)> callback) {
    sendGetRequest("/blocks/tip/height", [this, callback](Result<QJsonDocument> result) {
        bool ok = result.ok();
        m_connected.store(ok, std::memory_order_release);
        Q_EMIT connectedChanged(ok);
        callback({.value = ok});
    });
}

// ---- UTXOs ----
void NetworkManager::queryUtxos(const QString& address,
                                 std::function<void(Result<QJsonArray>)> callback)
{
    QString path = QString("/address/%1/utxo").arg(address);
    sendGetRequest(path, [callback](Result<QJsonDocument> result) {
        if (!result.ok()) {
            callback({.error = result.error});
            return;
        }
        callback({.value = result.value.array()});
    });
}

// ---- HTTP Helpers ----
void NetworkManager::sendGetRequest(const QString& path,
                                     std::function<void(Result<QJsonDocument>)> callback)
{
    QUrl url(m_config.url + path);
    QNetworkRequest request(url);
    request.setRawHeader("Accept", "application/json");
    request.setTransferTimeout(m_config.timeoutMs);

    QNetworkReply* reply = m_nam->get(request);
    connect(reply, &QNetworkReply::finished, this, [this, reply, callback]() {
        reply->deleteLater();

        if (reply->error() != QNetworkReply::NoError) {
            QString errMsg = reply->errorString();
            log(LogLevel::Error, "Network error: {}", errMsg.toStdString());
            Q_EMIT networkErrorOccurred(errMsg);
            m_connected.store(false, std::memory_order_release);
            callback({.error = WalletError::NetworkError});
            return;
        }

        QByteArray data = reply->readAll();
        QJsonParseError parseError;
        QJsonDocument doc = QJsonDocument::fromJson(data, &parseError);
        if (parseError.error != QJsonParseError::NoError) {
            log(LogLevel::Error, "JSON parse error: {}", parseError.errorString().toStdString());
            callback({.error = WalletError::NetworkError});
            return;
        }

        callback({.value = doc});
    });
}

void NetworkManager::sendPostRequest(const QString& path, const QJsonObject& body,
                                      std::function<void(Result<QJsonDocument>)> callback)
{
    QUrl url(m_config.url + path);
    QNetworkRequest request(url);
    request.setHeader(QNetworkRequest::ContentTypeHeader, "application/json");
    request.setRawHeader("Accept", "application/json");
    request.setTransferTimeout(m_config.timeoutMs);

    QByteArray bodyData = QJsonDocument(body).toJson(QJsonDocument::Compact);
    QNetworkReply* reply = m_nam->post(request, bodyData);
    connect(reply, &QNetworkReply::finished, this, [this, reply, callback]() {
        reply->deleteLater();

        if (reply->error() != QNetworkReply::NoError) {
            QString errMsg = reply->errorString();
            log(LogLevel::Error, "Network POST error: {}", errMsg.toStdString());
            Q_EMIT networkErrorOccurred(errMsg);
            callback({.error = WalletError::NetworkError});
            return;
        }

        QJsonDocument doc = QJsonDocument::fromJson(reply->readAll());
        callback({.value = doc});
    });
}

} // namespace ehw
