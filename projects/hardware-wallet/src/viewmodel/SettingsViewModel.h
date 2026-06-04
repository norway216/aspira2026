#pragma once

#include <QObject>
#include <QString>

class AuthService;
class AuditService;
class SessionMonitor;

class SettingsViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString currentUsername READ currentUsername NOTIFY currentUsernameChanged)
    Q_PROPERTY(int autoLockTimeout READ autoLockTimeout WRITE setAutoLockTimeout NOTIFY autoLockTimeoutChanged)
    Q_PROPERTY(bool auditIntegrityOk READ auditIntegrityOk NOTIFY auditIntegrityOkChanged)

public:
    explicit SettingsViewModel(AuthService* authService, AuditService* auditService,
                                QObject* parent = nullptr);

    QString currentUsername() const { return m_currentUsername; }
    int autoLockTimeout() const { return m_autoLockTimeout; }
    void setAutoLockTimeout(int seconds);
    bool auditIntegrityOk() const { return m_auditIntegrityOk; }

    Q_INVOKABLE void refresh();
    Q_INVOKABLE void changePassword(const QString& oldPassword, const QString& newPassword);
    Q_INVOKABLE void verifyAuditIntegrity();
    Q_INVOKABLE void logout();

signals:
    void currentUsernameChanged();
    void autoLockTimeoutChanged();
    void auditIntegrityOkChanged();
    void passwordChanged(bool success, const QString& message);
    void logoutCompleted();
    void integrityCheckCompleted(bool ok);

private:
    AuthService* m_authService;
    AuditService* m_auditService;
    SessionMonitor* m_sessionMonitor;

    QString m_currentUsername;
    int m_autoLockTimeout = 300;
    bool m_auditIntegrityOk = true;
};
