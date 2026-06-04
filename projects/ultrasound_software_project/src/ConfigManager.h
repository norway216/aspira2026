#pragma once

#include <QObject>
#include <QString>
#include <QJsonObject>
#include <QMutex>

/// Reads/writes JSON configuration files for presets and system settings.
class ConfigManager : public QObject
{
    Q_OBJECT
public:
    explicit ConfigManager(QObject *parent = nullptr);

    /// Load a JSON config file. Returns true on success.
    Q_INVOKABLE bool loadConfig(const QString &filePath);
    /// Save current config to a JSON file.
    Q_INVOKABLE bool saveConfig(const QString &filePath);

    /// Set/get a nested JSON value by key path (e.g. "params.gain").
    Q_INVOKABLE QVariant value(const QString &keyPath, const QVariant &defaultVal = QVariant()) const;
    Q_INVOKABLE void setValue(const QString &keyPath, const QVariant &val);

    /// Return the entire config as a JSON object.
    QJsonObject rootObject() const;
    void setRootObject(const QJsonObject &obj);

    /// Return the default config directory.
    static QString defaultConfigDir();
    static QString defaultConfigPath();

signals:
    void configLoaded(const QString &path);
    void configSaved(const QString &path);

private:
    QJsonObject resolveKeyPath(const QString &keyPath, bool createMissing) const;
    QStringList splitKeyPath(const QString &keyPath) const;

    mutable QMutex m_mutex;
    QJsonObject m_config;
};
