#include "RegisterViewModel.h"
#include "service/AuthService.h"

RegisterViewModel::RegisterViewModel(AuthService* authService, QObject* parent)
    : QObject(parent), m_authService(authService) {

    connect(m_authService, &AuthService::registerCompleted, this,
        [this](bool success, QString errorMessage) {
            setLoading(false);
            if (success) {
                emit registerSucceeded();
            } else {
                setError(errorMessage);
                emit registerFailed(errorMessage);
            }
        });
}

void RegisterViewModel::setUsername(const QString& username) {
    if (m_username != username) { m_username = username; emit usernameChanged(); }
}
void RegisterViewModel::setPassword(const QString& password) {
    if (m_password != password) { m_password = password; emit passwordChanged(); }
}
void RegisterViewModel::setConfirmPassword(const QString& password) {
    if (m_confirmPassword != password) { m_confirmPassword = password; emit confirmPasswordChanged(); }
}

void RegisterViewModel::registerUser() {
    if (m_username.length() < 3) { setError("Username must be at least 3 characters"); return; }
    if (m_password.length() < 8) { setError("Password must be at least 8 characters"); return; }
    if (m_password != m_confirmPassword) { setError("Passwords do not match"); return; }
    setLoading(true); setError("");
    m_authService->registerUser(m_username, m_password);
}

void RegisterViewModel::clearError() { setError(""); }
void RegisterViewModel::setLoading(bool loading) { if (m_loading != loading) { m_loading = loading; emit loadingChanged(); } }
void RegisterViewModel::setError(const QString& error) { if (m_errorMessage != error) { m_errorMessage = error; emit errorMessageChanged(); } }
