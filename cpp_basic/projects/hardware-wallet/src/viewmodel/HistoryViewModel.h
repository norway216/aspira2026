#pragma once

#include <QObject>
#include <QString>
#include "viewmodel/DashboardViewModel.h" // for TransactionListModel

class TransactionService;
class WalletService;

class HistoryViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(QObject* transactions READ transactions NOTIFY transactionsChanged)
    Q_PROPERTY(int totalCount READ totalCount NOTIFY totalCountChanged)
    Q_PROPERTY(bool canLoadMore READ canLoadMore NOTIFY canLoadMoreChanged)
    Q_PROPERTY(bool loading READ loading NOTIFY loadingChanged)

public:
    explicit HistoryViewModel(WalletService* walletService, TransactionService* txService,
                               QObject* parent = nullptr);

    QObject* transactions() { return &m_txModel; }
    int totalCount() const { return m_totalCount; }
    bool canLoadMore() const { return m_canLoadMore; }
    bool loading() const { return m_loading; }

    Q_INVOKABLE void refresh();
    Q_INVOKABLE void loadMore();

signals:
    void transactionsChanged();
    void totalCountChanged();
    void canLoadMoreChanged();
    void loadingChanged();

private:
    WalletService* m_walletService;
    TransactionService* m_txService;
    TransactionListModel m_txModel;

    int m_totalCount = 0;
    int m_currentOffset = 0;
    int m_pageSize = 20;
    bool m_canLoadMore = false;
    bool m_loading = false;
    QString m_walletId;
    QString m_publicKey;
};
