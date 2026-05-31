#pragma once

#include <QObject>
#include <QString>
#include <QVector>
#include <QAbstractListModel>

#include "domain/Wallet.h"

class TransactionService;
class WalletService;
class AuthService;

// Simple model for recipient wallet list
class WalletListModel : public QAbstractListModel {
    Q_OBJECT
public:
    enum Roles { UserIdRole = Qt::UserRole + 1, PublicKeyRole, BalanceRole };

    explicit WalletListModel(QObject* parent = nullptr);
    int rowCount(const QModelIndex& parent = QModelIndex()) const override;
    QVariant data(const QModelIndex& index, int role = Qt::DisplayRole) const override;
    QHash<int, QByteArray> roleNames() const override;
    void setWallets(const std::vector<Wallet>& wallets);
    void clear();
    Wallet walletAt(int index) const;

private:
    QVector<Wallet> m_wallets;
};

class SendViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString recipientAddress READ recipientAddress WRITE setRecipientAddress NOTIFY recipientAddressChanged)
    Q_PROPERTY(double amount READ amount WRITE setAmount NOTIFY amountChanged)
    Q_PROPERTY(QString memo READ memo WRITE setMemo NOTIFY memoChanged)
    Q_PROPERTY(double balance READ balance NOTIFY balanceChanged)
    Q_PROPERTY(QString errorMessage READ errorMessage NOTIFY errorMessageChanged)
    Q_PROPERTY(bool loading READ loading NOTIFY loadingChanged)
    Q_PROPERTY(QString myWalletId READ myWalletId NOTIFY myWalletIdChanged)
    Q_PROPERTY(QObject* walletList READ walletList NOTIFY walletListChanged)
    Q_PROPERTY(bool showingWalletList READ showingWalletList NOTIFY showingWalletListChanged)

public:
    explicit SendViewModel(AuthService* authService, WalletService* walletService,
                           TransactionService* txService, QObject* parent = nullptr);

    QString recipientAddress() const { return m_recipientAddress; }
    void setRecipientAddress(const QString& addr);

    double amount() const { return m_amount; }
    void setAmount(double amount);

    QString memo() const { return m_memo; }
    void setMemo(const QString& memo);

    double balance() const { return m_balance; }
    QString errorMessage() const { return m_errorMessage; }
    bool loading() const { return m_loading; }
    QString myWalletId() const { return m_myWalletId; }
    QObject* walletList() { return &m_walletListModel; }
    bool showingWalletList() const { return m_showingWalletList; }

    Q_INVOKABLE void refresh();
    Q_INVOKABLE void send();
    Q_INVOKABLE void loadWallets();
    Q_INVOKABLE void selectWallet(int index);
    Q_INVOKABLE void clearError();

signals:
    void recipientAddressChanged();
    void amountChanged();
    void memoChanged();
    void balanceChanged();
    void errorMessageChanged();
    void loadingChanged();
    void myWalletIdChanged();
    void walletListChanged();
    void showingWalletListChanged();
    void sendSucceeded(const QString& txId);
    void sendFailed(const QString& error);

private:
    AuthService* m_authService;
    WalletService* m_walletService;
    TransactionService* m_txService;
    WalletListModel m_walletListModel;

    QString m_recipientAddress;
    double m_amount = 0.0;
    QString m_memo;
    double m_balance = 0.0;
    QString m_errorMessage;
    bool m_loading = false;
    bool m_showingWalletList = false;
    QString m_myWalletId;
    QString m_myPublicKey;

    void setLoading(bool loading);
    void setError(const QString& error);
};
