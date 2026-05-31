#include "HistoryViewModel.h"
#include "service/WalletService.h"
#include "service/TransactionService.h"

HistoryViewModel::HistoryViewModel(WalletService* walletService, TransactionService* txService,
                                     QObject* parent)
    : QObject(parent), m_walletService(walletService), m_txService(txService) {

    connect(m_walletService, &WalletService::walletLoaded, this,
        [this](Result<Wallet> result) {
            if (result.isOk()) {
                m_walletId = result.value().id;
                m_publicKey = result.value().publicKeyHex();
                m_currentOffset = 0;
                m_txModel.clear();
                loadMore();
            }
        });

    connect(m_txService, &TransactionService::historyLoaded, this,
        [this](Result<std::vector<Transaction>> result) {
            m_loading = false;
            emit loadingChanged();
            if (result.isOk()) {
                const auto& txs = result.value();
                m_txModel.setTransactions(txs);
                m_currentOffset += static_cast<int>(txs.size());
                m_canLoadMore = (txs.size() >= m_pageSize);
                emit transactionsChanged();
                emit canLoadMoreChanged();
            }
        });
}

void HistoryViewModel::refresh() {
    m_currentOffset = 0;
    m_txModel.clear();
    m_walletService->getWallet();
}

void HistoryViewModel::loadMore() {
    if (m_loading || m_walletId.isEmpty()) return;
    m_loading = true;
    emit loadingChanged();
    m_txService->getFullHistory(m_walletId, m_publicKey, m_currentOffset, m_pageSize);
}
