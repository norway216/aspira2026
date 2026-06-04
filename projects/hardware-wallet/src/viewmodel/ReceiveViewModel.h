#pragma once

#include <QObject>
#include <QString>

class WalletService;

class ReceiveViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString publicKey READ publicKey NOTIFY publicKeyChanged)
    Q_PROPERTY(QString qrCodeData READ qrCodeData NOTIFY qrCodeDataChanged)

public:
    explicit ReceiveViewModel(WalletService* walletService, QObject* parent = nullptr);

    QString publicKey() const { return m_publicKey; }
    QString qrCodeData() const { return m_publicKey; }

    Q_INVOKABLE void refresh();

signals:
    void publicKeyChanged();
    void qrCodeDataChanged();

private:
    WalletService* m_walletService;
    QString m_publicKey;
};
