#include "BackupViewModel.h"
#include "service/BackupService.h"

BackupViewModel::BackupViewModel(BackupService* backupService, QObject* parent)
    : QObject(parent), m_backupService(backupService) {

    connect(m_backupService, &BackupService::exportCompleted, this,
        [this](Result<void> result) {
            setLoading(false);
            if (result.isOk()) {
                setStatus("Backup created successfully!");
                emit exportSucceeded();
            } else {
                setError(result.error().message);
                emit exportFailed(result.error().message);
            }
        });
}

void BackupViewModel::setFilePath(const QString& path) {
    if (m_filePath != path) {
        m_filePath = path;
        emit filePathChanged();
    }
}

void BackupViewModel::setPassword(const QString& pw) {
    if (m_password != pw) {
        m_password = pw;
        emit passwordChanged();
    }
}

void BackupViewModel::setConfirmPassword(const QString& pw) {
    if (m_confirmPassword != pw) {
        m_confirmPassword = pw;
        emit confirmPasswordChanged();
    }
}

void BackupViewModel::exportBackup() {
    if (m_password.length() < 8) {
        setError("Password must be at least 8 characters");
        return;
    }

    if (m_password != m_confirmPassword) {
        setError("Passwords do not match");
        return;
    }

    if (m_filePath.isEmpty()) {
        m_filePath = "wallet_backup.dat";
    }

    setLoading(true);
    setError("");
    setStatus("Creating backup...");
    m_backupService->exportBackup(m_filePath, m_password);
}

void BackupViewModel::clearError() {
    setError("");
}

void BackupViewModel::setLoading(bool loading) {
    if (m_loading != loading) {
        m_loading = loading;
        emit loadingChanged();
    }
}

void BackupViewModel::setError(const QString& error) {
    if (m_errorMessage != error) {
        m_errorMessage = error;
        emit errorMessageChanged();
    }
}

void BackupViewModel::setStatus(const QString& status) {
    if (m_status != status) {
        m_status = status;
        emit statusChanged();
    }
}
