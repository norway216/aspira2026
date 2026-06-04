#pragma once

#include <QObject>
#include <QString>

class AuthService;

class LoginViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString username READ username WRITE setUsername NOTIFY usernameChanged)
    Q_PROPERTY(QString password READ password WRITE setPassword NOTIFY passwordChanged)
    Q_PROPERTY(QString errorMessage READ errorMessage NOTIFY errorMessageChanged)
    Q_PROPERTY(bool loading READ loading NOTIFY loadingChanged)
    Q_PROPERTY(bool loggedIn READ loggedIn NOTIFY loggedInChanged)
    // Security indicators (per architecture §11.2)
    Q_PROPERTY(int failedAttempts READ failedAttempts NOTIFY failedAttemptsChanged)
    Q_PROPERTY(int remainingAttempts READ remainingAttempts NOTIFY remainingAttemptsChanged)
    Q_PROPERTY(bool walletWiped READ walletWiped NOTIFY walletWipedChanged)
    Q_PROPERTY(QString securityWarning READ securityWarning NOTIFY securityWarningChanged)

public:
    explicit LoginViewModel(AuthService* authService, QObject* parent = nullptr);

    QString username() const { return m_username; }
    void setUsername(const QString& username);

    QString password() const { return m_password; }
    void setPassword(const QString& password);

    QString errorMessage() const { return m_errorMessage; }
    bool loading() const { return m_loading; }
    bool loggedIn() const { return m_loggedIn; }
    int failedAttempts() const { return m_failedAttempts; }
    int remainingAttempts() const { return m_remainingAttempts; }
    bool walletWiped() const { return m_walletWiped; }
    QString securityWarning() const { return m_securityWarning; }

    Q_INVOKABLE void login();
    Q_INVOKABLE void clearError();

signals:
    void usernameChanged();
    void passwordChanged();
    void errorMessageChanged();
    void loadingChanged();
    void loggedInChanged();
    void loginSucceeded();
    void loginFailed(const QString& error);
    void failedAttemptsChanged();
    void remainingAttemptsChanged();
    void walletWipedChanged();
    void securityWarningChanged();

private:
    AuthService* m_authService;
    QString m_username;
    QString m_password;
    QString m_errorMessage;
    bool m_loading = false;
    bool m_loggedIn = false;
    int m_failedAttempts = 0;
    int m_remainingAttempts = 3;
    bool m_walletWiped = false;
    QString m_securityWarning;

    void setLoading(bool loading);
    void setError(const QString& error);
    void parseSecurityInfo(const QString& errorMessage);
};
