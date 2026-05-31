#pragma once

#include <QObject>
#include <QTimer>

/**
 * Monitors user activity and auto-locks the session after a configurable timeout.
 * Any user interaction should call reset() to restart the countdown.
 */
class SessionMonitor : public QObject {
    Q_OBJECT
    Q_PROPERTY(int timeoutSeconds READ timeoutSeconds WRITE setTimeoutSeconds NOTIFY timeoutChanged)
    Q_PROPERTY(int remainingSeconds READ remainingSeconds NOTIFY remainingChanged)

public:
    explicit SessionMonitor(QObject* parent = nullptr);
    ~SessionMonitor();

    int timeoutSeconds() const { return m_timeoutSeconds; }
    void setTimeoutSeconds(int seconds);

    int remainingSeconds() const { return m_remainingSeconds; }

    bool isActive() const { return m_active; }

public slots:
    // Reset the countdown (call on any user interaction)
    void reset();

    // Start monitoring
    void start();

    // Stop monitoring
    void stop();

    // Force immediate lock
    void forceLock();

signals:
    void timeoutChanged(int seconds);
    void remainingChanged(int seconds);
    void sessionExpired();
    void lockWarning(int secondsLeft);  // Emitted 30 seconds before expiry

private slots:
    void onTick();

private:
    QTimer m_timer;
    int m_timeoutSeconds = 300;  // 5 minutes default
    int m_remainingSeconds = 300;
    bool m_active = false;
    bool m_warningSent = false;
};
