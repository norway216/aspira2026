#pragma once

#include <QObject>
#include <QString>
#include <QVector>
#include <QAbstractListModel>

#include "domain/Transaction.h"
#include "domain/Wallet.h"

class AuthService;
class WalletService;
class TransactionService;

// Model for displaying transactions in QML ListView
class TransactionListModel : public QAbstractListModel {
    Q_OBJECT
public:
    enum Roles {
        TypeRole = Qt::UserRole + 1,
        AmountRole,
        AddressRole,
        TimestampRole,
        StatusRole,
        NonceRole
    };

    explicit TransactionListModel(QObject* parent = nullptr);

    int rowCount(const QModelIndex& parent = QModelIndex()) const override;
    QVariant data(const QModelIndex& index, int role = Qt::DisplayRole) const override;
    QHash<int, QByteArray> roleNames() const override;

    void setTransactions(const std::vector<Transaction>& txs);
    void clear();

private:
    QVector<Transaction> m_transactions;
};

class DashboardViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString username READ username NOTIFY usernameChanged)
    Q_PROPERTY(double balance READ balance NOTIFY balanceChanged)
    Q_PROPERTY(QString address READ address NOTIFY addressChanged)
    Q_PROPERTY(QString publicKey READ publicKey NOTIFY publicKeyChanged)
    Q_PROPERTY(QObject* transactions READ transactions NOTIFY transactionsChanged)
    Q_PROPERTY(bool hasWallet READ hasWallet NOTIFY hasWalletChanged)
    Q_PROPERTY(QString errorMessage READ errorMessage NOTIFY errorMessageChanged)
    Q_PROPERTY(bool loading READ loading NOTIFY loadingChanged)

public:
    explicit DashboardViewModel(AuthService* authService, WalletService* walletService,
                                 TransactionService* txService, QObject* parent = nullptr);

    QString username() const { return m_username; }
    double balance() const { return m_balance; }
    QString address() const { return m_address; }
    QString publicKey() const { return m_publicKey; }
    QObject* transactions() { return &m_txModel; }
    bool hasWallet() const { return m_hasWallet; }
    QString errorMessage() const { return m_errorMessage; }
    bool loading() const { return m_loading; }

    Q_INVOKABLE void refresh();
    Q_INVOKABLE void createWallet();
    Q_INVOKABLE void deleteWallet();
    Q_INVOKABLE void loadTransactions();

signals:
    void usernameChanged();
    void balanceChanged();
    void addressChanged();
    void publicKeyChanged();
    void transactionsChanged();
    void hasWalletChanged();
    void errorMessageChanged();
    void loadingChanged();
    void walletCreated();
    void walletDeleted();

private:
    AuthService* m_authService;
    WalletService* m_walletService;
    TransactionService* m_txService;
    TransactionListModel m_txModel;

    QString m_username;
    double m_balance = 0.0;
    QString m_address;
    QString m_publicKey;
    QString m_walletId;
    bool m_hasWallet = false;
    QString m_errorMessage;
    bool m_loading = false;

    void setLoading(bool loading);
    void setError(const QString& error);
    void updateFromWallet(const Wallet& wallet);
};
