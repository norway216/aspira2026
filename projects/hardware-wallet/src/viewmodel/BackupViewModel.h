#pragma once

#include <QObject>
#include <QString>

class BackupService;

class BackupViewModel : public QObject {
    Q_OBJECT
    Q_PROPERTY(QString status READ status NOTIFY statusChanged)
    Q_PROPERTY(bool loading READ loading NOTIFY loadingChanged)
    Q_PROPERTY(QString errorMessage READ errorMessage NOTIFY errorMessageChanged)
    Q_PROPERTY(QString filePath READ filePath WRITE setFilePath NOTIFY filePathChanged)
    Q_PROPERTY(QString password READ password WRITE setPassword NOTIFY passwordChanged)
    Q_PROPERTY(QString confirmPassword READ confirmPassword WRITE setConfirmPassword NOTIFY confirmPasswordChanged)

public:
    explicit BackupViewModel(BackupService* backupService, QObject* parent = nullptr);

    QString status() const { return m_status; }
    bool loading() const { return m_loading; }
    QString errorMessage() const { return m_errorMessage; }
    QString filePath() const { return m_filePath; }
    void setFilePath(const QString& path);
    QString password() const { return m_password; }
    void setPassword(const QString& pw);
    QString confirmPassword() const { return m_confirmPassword; }
    void setConfirmPassword(const QString& pw);

    Q_INVOKABLE void exportBackup();
    Q_INVOKABLE void clearError();

signals:
    void statusChanged();
    void loadingChanged();
    void errorMessageChanged();
    void filePathChanged();
    void passwordChanged();
    void confirmPasswordChanged();
    void exportSucceeded();
    void exportFailed(const QString& error);

private:
    BackupService* m_backupService;
    QString m_status;
    bool m_loading = false;
    QString m_errorMessage;
    QString m_filePath;
    QString m_password;
    QString m_confirmPassword;

    void setLoading(bool loading);
    void setError(const QString& error);
    void setStatus(const QString& status);
};
