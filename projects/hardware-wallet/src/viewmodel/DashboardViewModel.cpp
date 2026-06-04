#include "DashboardViewModel.h"
#include "service/AuthService.h"
#include "service/WalletService.h"
#include "service/TransactionService.h"

#include <QDebug>
#include <QDateTime>
#include <algorithm>

// TransactionListModel implementation
TransactionListModel::TransactionListModel(QObject* parent) : QAbstractListModel(parent) {}

int TransactionListModel::rowCount(const QModelIndex& parent) const {
    if (parent.isValid()) return 0;
    return m_transactions.size();
}

QVariant TransactionListModel::data(const QModelIndex& index, int role) const {
    if (!index.isValid() || index.row() >= m_transactions.size()) return {};

    const auto& tx = m_transactions.at(index.row());
    switch (role) {
        case TypeRole: return tx.isIncoming ? "received" : "sent";
        case AmountRole: return tx.amount;
        case AddressRole: return tx.isIncoming ? tx.walletId : tx.toPublicKey;
        case TimestampRole: return QDateTime::fromMSecsSinceEpoch(tx.createdAt).toString("MM-dd hh:mm:ss");
        case StatusRole: return tx.statusString();
        case NonceRole: return tx.nonce;
        default: return {};
    }
}

QHash<int, QByteArray> TransactionListModel::roleNames() const {
    return {
        {TypeRole, "type"},
        {AmountRole, "amount"},
        {AddressRole, "address"},
        {TimestampRole, "timestamp"},
        {StatusRole, "status"},
        {NonceRole, "nonce"}
    };
}

void TransactionListModel::setTransactions(const std::vector<Transaction>& txs) {
    beginResetModel();
    m_transactions.clear();
    m_transactions.reserve(txs.size());
    for (const auto& tx : txs) {
        m_transactions.append(tx);
    }
    endResetModel();
}

void TransactionListModel::clear() {
    beginResetModel();
    m_transactions.clear();
    endResetModel();
}

// DashboardViewModel implementation
DashboardViewModel::DashboardViewModel(AuthService* authService,
                                         WalletService* walletService,
                                         TransactionService* txService,
                                         QObject* parent)
    : QObject(parent), m_authService(authService), m_walletService(walletService),
      m_txService(txService) {

    // Wallet loaded (from getWallet call)
    connect(m_walletService, &WalletService::walletLoaded, this,
        [this](Result<Wallet> result) {
            setLoading(false);
            if (result.isOk()) {
                updateFromWallet(result.value());
                // Load full transaction history
                m_txService->getFullHistory(m_walletId, m_publicKey, 0, 20);
            } else if (result.error().code == ErrorCode::WalletNotFound) {
                m_hasWallet = false;
                m_walletId.clear();
                m_publicKey.clear();
                m_balance = 0;
                m_txModel.clear();
                emit hasWalletChanged();
                emit balanceChanged();
                emit transactionsChanged();
            }
        });

    // Wallet created
    connect(m_walletService, &WalletService::walletCreated, this,
        [this](Result<Wallet> result) {
            setLoading(false);
            if (result.isOk()) {
                updateFromWallet(result.value());
                m_txModel.clear();
                emit transactionsChanged();
                emit walletCreated();
            } else {
                setError(result.error().message);
            }
        });

    // Wallet deleted
    connect(m_walletService, &WalletService::walletDeleted, this,
        [this](Result<void> result) {
            setLoading(false);
            if (result.isOk()) {
                m_hasWallet = false;
                m_walletId.clear();
                m_publicKey.clear();
                m_balance = 0;
                m_txModel.clear();
                emit hasWalletChanged();
                emit balanceChanged();
                emit transactionsChanged();
                emit walletDeleted();
            } else {
                setError(result.error().message);
            }
        });

    // Transaction history loaded
    connect(m_txService, &TransactionService::historyLoaded, this,
        [this](Result<std::vector<Transaction>> result) {
            if (result.isOk()) {
                m_txModel.setTransactions(result.value());
                emit transactionsChanged();
            }
        });
}

void DashboardViewModel::refresh() {
    m_username = m_authService->currentUsername();
    emit usernameChanged();
    m_walletService->getWallet();
}

void DashboardViewModel::createWallet() {
    setLoading(true);
    setError("");
    m_walletService->createWallet();
}

void DashboardViewModel::deleteWallet() {
    setLoading(true);
    setError("");
    m_walletService->deleteWallet();
}

void DashboardViewModel::loadTransactions() {
    if (!m_walletId.isEmpty() && !m_publicKey.isEmpty()) {
        m_txService->getFullHistory(m_walletId, m_publicKey, 0, 20);
    }
}

void DashboardViewModel::updateFromWallet(const Wallet& wallet) {
    m_hasWallet = true;
    m_balance = wallet.balance;
    m_address = wallet.shortAddress();
    m_publicKey = wallet.publicKeyHex();
    m_walletId = wallet.id;
    emit hasWalletChanged();
    emit balanceChanged();
    emit addressChanged();
    emit publicKeyChanged();
}

void DashboardViewModel::setLoading(bool loading) {
    if (m_loading != loading) { m_loading = loading; emit loadingChanged(); }
}

void DashboardViewModel::setError(const QString& error) {
    if (m_errorMessage != error) { m_errorMessage = error; emit errorMessageChanged(); }
}
