#include "AppCore.h"
#include "ParamManager.h"
#include "FrameGenerator.h"
#include "ThreadPool.h"
#include "ConfigManager.h"
#include "UltrasoundImageProvider.h"

#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QTimer>

AppCore::AppCore(QObject *parent)
    : QObject(parent)
{
}

AppCore::~AppCore()
{
    saveSettingsOnExit();
    if (m_frameGenerator)
        m_frameGenerator->stop();
}

bool AppCore::initialize(QQmlApplicationEngine &engine)
{
    m_engine = &engine;

    // ---- Create services ----
    m_paramManager  = std::make_unique<ParamManager>(this);
    m_threadPool    = std::make_unique<ThreadPool>(this);
    m_configManager = std::make_unique<ConfigManager>(this);
    m_frameGenerator = std::make_unique<FrameGenerator>(m_paramManager.get(), this);
    m_imageProvider  = std::make_unique<UltrasoundImageProvider>();

    // ---- Wire signals ----
    connect(m_frameGenerator.get(), &FrameGenerator::frameReady,
            this, &AppCore::onFrameReady);
    connect(m_paramManager.get(), &ParamManager::paramsChanged,
            this, &AppCore::onParamsChanged);

    // ---- Expose to QML ----
    QQmlContext *ctx = engine.rootContext();
    ctx->setContextProperty("appCore",      this);
    ctx->setContextProperty("paramManager", m_paramManager.get());
    ctx->setContextProperty("configManager",m_configManager.get());
    ctx->setContextProperty("threadPool",   m_threadPool.get());

    // Register image provider
    engine.addImageProvider("ultrasound", m_imageProvider.get());

    // ---- Load saved settings ----
    loadSettings();

    // ---- Start simulation ----
    m_frameGenerator->start();
    m_running = true;
    m_statusText = QStringLiteral("Running — B-mode");
    emit runningChanged();
    emit statusTextChanged();

    return true;
}

void AppCore::startAcquisition()
{
    if (m_frameGenerator) {
        m_paramManager->setFreeze(false);
        m_frameGenerator->start();
        m_running = true;
        m_statusText = QStringLiteral("Running — ") + m_paramManager->mode();
        emit runningChanged();
        emit statusTextChanged();
    }
}

void AppCore::stopAcquisition()
{
    if (m_frameGenerator) {
        m_frameGenerator->stop();
        m_running = false;
        m_statusText = QStringLiteral("Stopped");
        emit runningChanged();
        emit statusTextChanged();
    }
}

void AppCore::toggleFreeze()
{
    bool frozen = !m_paramManager->freeze();
    m_paramManager->setFreeze(frozen);
    m_statusText = frozen ? QStringLiteral("Frozen") : QStringLiteral("Running — ") + m_paramManager->mode();
    emit statusTextChanged();
}

// ---------------------------------------------------------------------------
// Accessors
// ---------------------------------------------------------------------------
bool    AppCore::running()     const { return m_running; }
QString AppCore::statusText()  const { return m_statusText; }
int     AppCore::frameCount()  const { return m_frameCount; }

ParamManager  *AppCore::paramManager()  const { return m_paramManager.get(); }
ConfigManager *AppCore::configManager() const { return m_configManager.get(); }
ThreadPool    *AppCore::threadPool()    const { return m_threadPool.get(); }

// ---------------------------------------------------------------------------
// Slots
// ---------------------------------------------------------------------------
void AppCore::savePreset(const QString &name)
{
    QJsonObject preset = m_paramManager->toJson();
    m_configManager->setValue("presets." + name, QJsonValue(preset));
    m_configManager->saveConfig(ConfigManager::defaultConfigPath());
}

void AppCore::loadPreset(const QString &name)
{
    QVariant presetVar = m_configManager->value("presets." + name);
    if (presetVar.isValid() && presetVar.canConvert<QJsonObject>()) {
        m_paramManager->fromJson(presetVar.toJsonObject());
    }
}

void AppCore::saveSettingsOnExit()
{
    QJsonObject settings;
    settings["params"] = m_paramManager->toJson();
    m_configManager->setRootObject(settings);
    m_configManager->saveConfig(ConfigManager::defaultConfigPath());
}

void AppCore::loadSettings()
{
    QString path = ConfigManager::defaultConfigPath();
    if (m_configManager->loadConfig(path)) {
        QVariant paramsVar = m_configManager->value("params");
        if (paramsVar.isValid() && paramsVar.canConvert<QJsonObject>()) {
            m_paramManager->fromJson(paramsVar.toJsonObject());
        }
    }
}

// ---------------------------------------------------------------------------
// Internal slots
// ---------------------------------------------------------------------------
void AppCore::onFrameReady()
{
    if (m_frameGenerator && m_imageProvider) {
        m_imageProvider->updateFrame(m_frameGenerator->latestFrame());
        m_frameCount++;
        emit frameCountChanged();
    }
}

void AppCore::onParamsChanged()
{
    // Could trigger additional processing when parameters change
}
