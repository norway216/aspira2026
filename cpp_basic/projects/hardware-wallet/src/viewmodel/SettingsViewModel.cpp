#include "SettingsViewModel.h"
#include "service/AuthService.h"
#include "service/AuditService.h"
#include "worker/SessionMonitor.h"

SettingsViewModel::SettingsViewModel(AuthService* authService, AuditService* auditService,
                                       QObject* parent)
    : QObject(parent), m_authService(authService), m_auditService(auditService) {
    m_sessionMonitor = authService->sessionMonitor();

    connect(m_authService, &AuthService::passwordChangeCompleted, this,
        [this](bool success, QString errorMessage) {
            if (success) {
                emit passwordChanged(true, "Password changed successfully");
            } else {
                emit passwordChanged(false, errorMessage);
            }
        });

    connect(m_authService, &AuthService::logoutCompleted, this,
        [this]() {
            emit logoutCompleted();
        });

    connect(m_auditService, &AuditService::integrityVerified, this,
        [this](Result<bool> result) {
            if (result.isOk()) {
                m_auditIntegrityOk = result.value();
                emit auditIntegrityOkChanged();
                emit integrityCheckCompleted(result.value());
            }
        });
}

void SettingsViewModel::setAutoLockTimeout(int seconds) {
    if (m_autoLockTimeout != seconds) {
        m_autoLockTimeout = seconds;
        if (m_sessionMonitor) {
            m_sessionMonitor->setTimeoutSeconds(seconds);
        }
        emit autoLockTimeoutChanged();
    }
}

void SettingsViewModel::refresh() {
    m_currentUsername = m_authService->currentUsername();
    emit currentUsernameChanged();
    verifyAuditIntegrity();
}

void SettingsViewModel::changePassword(const QString& oldPassword, const QString& newPassword) {
    m_authService->changePassword(oldPassword, newPassword);
}

void SettingsViewModel::verifyAuditIntegrity() {
    m_auditService->verifyIntegrity();
}

void SettingsViewModel::logout() {
    m_authService->logout();
}
