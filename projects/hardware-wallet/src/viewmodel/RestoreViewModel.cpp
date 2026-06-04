#include "RestoreViewModel.h"
#include "service/BackupService.h"

RestoreViewModel::RestoreViewModel(BackupService* backupService, QObject* parent)
    : QObject(parent), m_backupService(backupService) {

    connect(m_backupService, &BackupService::importCompleted, this,
        [this](Result<int> result) {
            setLoading(false);
            if (result.isOk()) {
                setStatus(QString("Restored %1 users successfully!").arg(result.value()));
                emit importSucceeded(result.value());
            } else {
                setError(result.error().message);
                emit importFailed(result.error().message);
            }
        });
}

void RestoreViewModel::setFilePath(const QString& path) {
    if (m_filePath != path) {
        m_filePath = path;
        emit filePathChanged();
    }
}

void RestoreViewModel::setPassword(const QString& pw) {
    if (m_password != pw) {
        m_password = pw;
        emit passwordChanged();
    }
}

void RestoreViewModel::importBackup() {
    if (m_filePath.isEmpty()) {
        setError("Please select a backup file");
        return;
    }

    if (m_password.isEmpty()) {
        setError("Please enter the backup password");
        return;
    }

    setLoading(true);
    setError("");
    setStatus("Restoring from backup...");
    m_backupService->importBackup(m_filePath, m_password);
}

void RestoreViewModel::clearError() {
    setError("");
}

void RestoreViewModel::setLoading(bool loading) {
    if (m_loading != loading) {
        m_loading = loading;
        emit loadingChanged();
    }
}

void RestoreViewModel::setError(const QString& error) {
    if (m_errorMessage != error) {
        m_errorMessage = error;
        emit errorMessageChanged();
    }
}

void RestoreViewModel::setStatus(const QString& status) {
    if (m_status != status) {
        m_status = status;
        emit statusChanged();
    }
}
