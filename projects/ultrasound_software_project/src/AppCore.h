#pragma once

#include <QObject>
#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <memory>

class ParamManager;
class FrameGenerator;
class ThreadPool;
class ConfigManager;
class UltrasoundImageProvider;

/// Central application controller that wires C++ back-end and QML front-end.
/// Owns all core services and is exposed to QML as a root context property.
class AppCore : public QObject
{
    Q_OBJECT

    // ---- Application state exposed to QML ----
    Q_PROPERTY(bool   running      READ running      NOTIFY runningChanged)
    Q_PROPERTY(QString statusText  READ statusText   NOTIFY statusTextChanged)
    Q_PROPERTY(int    frameCount   READ frameCount   NOTIFY frameCountChanged)

public:
    explicit AppCore(QObject *parent = nullptr);
    ~AppCore() override;

    /// Initialize all subsystems and return true on success.
    bool initialize(QQmlApplicationEngine &engine);

    /// Start acquisition / simulation.
    Q_INVOKABLE void startAcquisition();
    /// Stop acquisition.
    Q_INVOKABLE void stopAcquisition();
    /// Toggle freeze.
    Q_INVOKABLE void toggleFreeze();

    // Accessors
    bool    running()     const;
    QString statusText()  const;
    int     frameCount()  const;

    // Sub-system accessors
    ParamManager           *paramManager()  const;
    ConfigManager          *configManager() const;
    ThreadPool             *threadPool()    const;

public slots:
    /// Save current parameters as a preset.
    void savePreset(const QString &name);
    /// Load a preset.
    void loadPreset(const QString &name);
    /// Save all settings on exit.
    void saveSettingsOnExit();

signals:
    void runningChanged();
    void statusTextChanged();
    void frameCountChanged();
    void errorOccurred(const QString &message);

private slots:
    void onFrameReady();
    void onParamsChanged();

private:
    void loadSettings();

    std::unique_ptr<ParamManager>            m_paramManager;
    std::unique_ptr<FrameGenerator>          m_frameGenerator;
    std::unique_ptr<ThreadPool>              m_threadPool;
    std::unique_ptr<ConfigManager>           m_configManager;
    std::unique_ptr<UltrasoundImageProvider> m_imageProvider;

    QQmlApplicationEngine *m_engine = nullptr;
    bool     m_running    = false;
    QString  m_statusText;
    int      m_frameCount = 0;
};
