#pragma once

#include <QObject>
#include <QString>

class AuthService;

class RegisterViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString username READ username WRITE setUsername NOTIFY usernameChanged)
    Q_PROPERTY(QString password READ password WRITE setPassword NOTIFY passwordChanged)
    Q_PROPERTY(QString confirmPassword READ confirmPassword WRITE setConfirmPassword NOTIFY confirmPasswordChanged)
    Q_PROPERTY(QString errorMessage READ errorMessage NOTIFY errorMessageChanged)
    Q_PROPERTY(bool loading READ loading NOTIFY loadingChanged)

public:
    explicit RegisterViewModel(AuthService* authService, QObject* parent = nullptr);

    QString username() const { return m_username; }
    void setUsername(const QString& username);

    QString password() const { return m_password; }
    void setPassword(const QString& password);

    QString confirmPassword() const { return m_confirmPassword; }
    void setConfirmPassword(const QString& password);

    QString errorMessage() const { return m_errorMessage; }
    bool loading() const { return m_loading; }

    Q_INVOKABLE void registerUser();
    Q_INVOKABLE void clearError();

signals:
    void usernameChanged();
    void passwordChanged();
    void confirmPasswordChanged();
    void errorMessageChanged();
    void loadingChanged();
    void registerSucceeded();
    void registerFailed(const QString& error);

private:
    AuthService* m_authService;
    QString m_username;
    QString m_password;
    QString m_confirmPassword;
    QString m_errorMessage;
    bool m_loading = false;

    void setLoading(bool loading);
    void setError(const QString& error);
};
