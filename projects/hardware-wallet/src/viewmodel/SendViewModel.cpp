#include "SendViewModel.h"
#include "service/AuthService.h"
#include "service/WalletService.h"
#include "service/TransactionService.h"

// WalletListModel
WalletListModel::WalletListModel(QObject* parent) : QAbstractListModel(parent) {}

int WalletListModel::rowCount(const QModelIndex& parent) const {
    if (parent.isValid()) return 0;
    return m_wallets.size();
}

QVariant WalletListModel::data(const QModelIndex& index, int role) const {
    if (!index.isValid() || index.row() >= m_wallets.size()) return {};
    const auto& w = m_wallets.at(index.row());
    switch (role) {
        case UserIdRole: return w.userId;
        case PublicKeyRole: return w.publicKeyHex();
        case BalanceRole: return w.balance;
        default: return {};
    }
}

QHash<int, QByteArray> WalletListModel::roleNames() const {
    return {
        {UserIdRole, "userId"},
        {PublicKeyRole, "publicKey"},
        {BalanceRole, "balance"}
    };
}

void WalletListModel::setWallets(const std::vector<Wallet>& wallets) {
    beginResetModel();
    m_wallets.clear();
    for (const auto& w : wallets) m_wallets.append(w);
    endResetModel();
}

void WalletListModel::clear() {
    beginResetModel();
    m_wallets.clear();
    endResetModel();
}

Wallet WalletListModel::walletAt(int index) const {
    if (index >= 0 && index < m_wallets.size()) return m_wallets.at(index);
    return Wallet{};
}

// SendViewModel
SendViewModel::SendViewModel(AuthService* authService, WalletService* walletService,
                               TransactionService* txService, QObject* parent)
    : QObject(parent), m_authService(authService), m_walletService(walletService),
      m_txService(txService) {

    connect(m_walletService, &WalletService::walletLoaded, this,
        [this](Result<Wallet> result) {
            if (result.isOk()) {
                m_balance = result.value().balance;
                m_myWalletId = result.value().id;
                m_myPublicKey = result.value().publicKeyHex();
                emit balanceChanged();
                emit myWalletIdChanged();
            }
        });

    connect(m_walletService, &WalletService::allWalletsLoaded, this,
        [this](Result<std::vector<Wallet>> result) {
            if (result.isOk()) {
                // Filter out own wallet
                std::vector<Wallet> others;
                for (auto& w : result.value()) {
                    if (w.publicKeyHex() != m_myPublicKey) {
                        others.push_back(std::move(w));
                    }
                }
                m_walletListModel.setWallets(others);
                emit walletListChanged();
            }
        });

    connect(m_txService, &TransactionService::sendCompleted, this,
        [this](Result<Transaction> result) {
            setLoading(false);
            if (result.isOk()) {
                emit sendSucceeded(result.value().id);
                refresh();
            } else {
                setError(result.error().message);
                emit sendFailed(result.error().message);
            }
        });
}

void SendViewModel::setRecipientAddress(const QString& addr) {
    if (m_recipientAddress != addr) { m_recipientAddress = addr; emit recipientAddressChanged(); }
}

void SendViewModel::setAmount(double amount) {
    if (m_amount != amount) { m_amount = amount; emit amountChanged(); }
}

void SendViewModel::setMemo(const QString& memo) {
    if (m_memo != memo) { m_memo = memo; emit memoChanged(); }
}

void SendViewModel::refresh() {
    m_walletService->getWallet();
}

void SendViewModel::send() {
    if (m_recipientAddress.isEmpty()) { setError("Please select a recipient"); return; }
    if (m_amount <= 0) { setError("Amount must be positive"); return; }
    if (m_myWalletId.isEmpty()) { refresh(); return; }
    setLoading(true); setError("");
    m_txService->sendTransaction(m_myWalletId, m_recipientAddress, m_amount, m_memo);
}

void SendViewModel::loadWallets() {
    m_showingWalletList = true;
    emit showingWalletListChanged();
    m_walletService->listAllWallets();
}

void SendViewModel::selectWallet(int index) {
    Wallet w = m_walletListModel.walletAt(index);
    if (!w.publicKeyHex().isEmpty()) {
        setRecipientAddress(w.publicKeyHex());
    }
    m_showingWalletList = false;
    emit showingWalletListChanged();
}

void SendViewModel::clearError() { setError(""); }
void SendViewModel::setLoading(bool loading) { if (m_loading != loading) { m_loading = loading; emit loadingChanged(); } }
void SendViewModel::setError(const QString& error) { if (m_errorMessage != error) { m_errorMessage = error; emit errorMessageChanged(); } }
