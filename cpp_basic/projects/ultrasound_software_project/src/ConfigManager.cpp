#include "ConfigManager.h"

#include <QFile>
#include <QJsonDocument>
#include <QJsonArray>
#include <QStandardPaths>
#include <QDir>
#include <QMutexLocker>

// Helper: recursively set a nested value in a QJsonObject
static void setJsonNested(QJsonObject &obj, const QStringList &keys, int idx, const QVariant &val)
{
    if (idx >= keys.size()) return;
    if (idx == keys.size() - 1) {
        obj[keys[idx]] = QJsonValue::fromVariant(val);
        return;
    }
    QJsonObject inner = obj.value(keys[idx]).toObject();
    setJsonNested(inner, keys, idx + 1, val);
    obj[keys[idx]] = inner;
}

ConfigManager::ConfigManager(QObject *parent)
    : QObject(parent)
{
}

// ---------------------------------------------------------------------------
QStringList ConfigManager::splitKeyPath(const QString &keyPath) const
{
    return keyPath.split('.', Qt::SkipEmptyParts);
}

QJsonObject ConfigManager::resolveKeyPath(const QString &keyPath, bool /*createMissing*/) const
{
    // For simplicity, only support one-level nesting: "parent.child"
    QStringList parts = splitKeyPath(keyPath);
    if (parts.isEmpty()) return m_config;

    QJsonObject current = m_config;
    for (int i = 0; i < parts.size() - 1; ++i) {
        if (current.contains(parts[i]) && current[parts[i]].isObject())
            current = current[parts[i]].toObject();
        else
            return QJsonObject();  // path not found
    }
    // Return the parent object so caller can access the leaf
    return current;
}

QVariant ConfigManager::value(const QString &keyPath, const QVariant &defaultVal) const
{
    QMutexLocker lock(&m_mutex);
    QStringList parts = splitKeyPath(keyPath);
    if (parts.isEmpty()) return defaultVal;

    QJsonObject current = m_config;
    for (int i = 0; i < parts.size() - 1; ++i) {
        if (current.contains(parts[i]) && current[parts[i]].isObject())
            current = current[parts[i]].toObject();
        else
            return defaultVal;
    }

    QString leaf = parts.last();
    if (current.contains(leaf))
        return current[leaf].toVariant();
    return defaultVal;
}

void ConfigManager::setValue(const QString &keyPath, const QVariant &val)
{
    QMutexLocker lock(&m_mutex);
    QStringList parts = splitKeyPath(keyPath);
    if (parts.isEmpty()) return;

    setJsonNested(m_config, parts, 0, val);
}

// ---------------------------------------------------------------------------
QJsonObject ConfigManager::rootObject() const
{
    QMutexLocker lock(&m_mutex);
    return m_config;
}

void ConfigManager::setRootObject(const QJsonObject &obj)
{
    QMutexLocker lock(&m_mutex);
    m_config = obj;
}

// ---------------------------------------------------------------------------
bool ConfigManager::loadConfig(const QString &filePath)
{
    QFile file(filePath);
    if (!file.open(QIODevice::ReadOnly)) {
        qWarning("ConfigManager: Cannot open %s for reading", qPrintable(filePath));
        return false;
    }
    QByteArray data = file.readAll();
    file.close();

    QJsonParseError err;
    QJsonDocument doc = QJsonDocument::fromJson(data, &err);
    if (err.error != QJsonParseError::NoError) {
        qWarning("ConfigManager: JSON parse error: %s", qPrintable(err.errorString()));
        return false;
    }
    if (!doc.isObject()) {
        qWarning("ConfigManager: Root element is not an object");
        return false;
    }

    setRootObject(doc.object());
    emit configLoaded(filePath);
    return true;
}

bool ConfigManager::saveConfig(const QString &filePath)
{
    QMutexLocker lock(&m_mutex);
    QJsonDocument doc(m_config);

    QFile file(filePath);
    if (!file.open(QIODevice::WriteOnly)) {
        qWarning("ConfigManager: Cannot open %s for writing", qPrintable(filePath));
        return false;
    }
    file.write(doc.toJson(QJsonDocument::Indented));
    file.close();

    emit configSaved(filePath);
    return true;
}

QString ConfigManager::defaultConfigDir()
{
    QString dir = QStandardPaths::writableLocation(QStandardPaths::AppConfigLocation);
    QDir().mkpath(dir);
    return dir;
}

QString ConfigManager::defaultConfigPath()
{
    return defaultConfigDir() + "/ultrasound_config.json";
}
