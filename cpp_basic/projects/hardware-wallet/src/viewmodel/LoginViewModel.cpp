#include "LoginViewModel.h"
#include "service/AuthService.h"
#include "app/Constants.h"

#include <QRegularExpression>

LoginViewModel::LoginViewModel(AuthService* authService, QObject* parent)
    : QObject(parent), m_authService(authService) {

    m_remainingAttempts = AppConstants::MAX_FAILED_ATTEMPTS;

    connect(m_authService, &AuthService::loginCompleted, this,
        [this](bool success, QString errorMessage) {
            setLoading(false);
            if (success) {
                m_loggedIn = true;
                m_walletWiped = false;
                m_failedAttempts = 0;
                m_remainingAttempts = AppConstants::MAX_FAILED_ATTEMPTS;
                m_securityWarning.clear();
                emit loggedInChanged();
                emit walletWipedChanged();
                emit loginSucceeded();
            } else {
                setError(errorMessage);
                parseSecurityInfo(errorMessage);
                emit loginFailed(errorMessage);
            }
        });
}

void LoginViewModel::parseSecurityInfo(const QString& errorMessage)
{
    // Detect wallet-wiped message
    if (errorMessage.contains("wiped", Qt::CaseInsensitive) ||
        errorMessage.contains("Wallet data has been")) {
        if (!m_walletWiped) {
            m_walletWiped = true;
            emit walletWipedChanged();
        }
    }

    // Parse remaining attempts from error message patterns
    // e.g., "1 attempt remaining" or "2 attempts remaining"
    QRegularExpression remainingRe("(\\d+)\\s*attempt.*remaining",
                                    QRegularExpression::CaseInsensitiveOption);
    auto match = remainingRe.match(errorMessage);
    if (match.hasMatch()) {
        int remaining = match.captured(1).toInt();
        if (m_remainingAttempts != remaining) {
            m_remainingAttempts = remaining;
            m_failedAttempts = AppConstants::MAX_FAILED_ATTEMPTS - remaining;
            emit remainingAttemptsChanged();
            emit failedAttemptsChanged();
        }

        // Set security warning when 1 attempt remains
        if (remaining == 1) {
            QString warning = "WARNING: Next failed attempt will permanently delete your wallet data!";
            if (m_securityWarning != warning) {
                m_securityWarning = warning;
                emit securityWarningChanged();
            }
        }
    }

    // Detect wipe outcome
    if (errorMessage.contains("securely wiped", Qt::CaseInsensitive) ||
        errorMessage.contains("wallet data has been securely", Qt::CaseInsensitive)) {
        m_walletWiped = true;
        m_remainingAttempts = 0;
        m_failedAttempts = AppConstants::MAX_FAILED_ATTEMPTS;
        emit walletWipedChanged();
        emit remainingAttemptsChanged();
        emit failedAttemptsChanged();
    }
}

void LoginViewModel::setUsername(const QString& username) {
    if (m_username != username) {
        m_username = username;
        emit usernameChanged();
    }
}

void LoginViewModel::setPassword(const QString& password) {
    if (m_password != password) {
        m_password = password;
        emit passwordChanged();
    }
}

void LoginViewModel::login() {
    if (m_username.isEmpty() || m_password.isEmpty()) {
        setError("Please enter username and password");
        return;
    }
    setLoading(true);
    setError("");
    m_securityWarning.clear();
    emit securityWarningChanged();
    m_authService->login(m_username, m_password);
}

void LoginViewModel::clearError() { setError(""); }
void LoginViewModel::setLoading(bool loading) {
    if (m_loading != loading) { m_loading = loading; emit loadingChanged(); }
}
void LoginViewModel::setError(const QString& error) {
    if (m_errorMessage != error) { m_errorMessage = error; emit errorMessageChanged(); }
}
