#include "SessionMonitor.h"
#include "app/Constants.h"

#include <QDebug>

SessionMonitor::SessionMonitor(QObject* parent)
    : QObject(parent) {
    m_timer.setInterval(1000); // Tick every second
    connect(&m_timer, &QTimer::timeout, this, &SessionMonitor::onTick);
}

SessionMonitor::~SessionMonitor() {
    stop();
}

void SessionMonitor::setTimeoutSeconds(int seconds) {
    if (seconds < 10) seconds = 10; // Minimum 10 seconds
    m_timeoutSeconds = seconds;
    m_remainingSeconds = seconds;
    m_warningSent = false;
    emit timeoutChanged(seconds);
}

void SessionMonitor::reset() {
    if (!m_active) return;
    m_remainingSeconds = m_timeoutSeconds;
    m_warningSent = false;
    emit remainingChanged(m_remainingSeconds);
}

void SessionMonitor::start() {
    m_remainingSeconds = m_timeoutSeconds;
    m_warningSent = false;
    m_active = true;
    m_timer.start();
    qDebug() << "Session monitor started, timeout:" << m_timeoutSeconds << "s";
}

void SessionMonitor::stop() {
    m_active = false;
    m_timer.stop();
    qDebug() << "Session monitor stopped";
}

void SessionMonitor::forceLock() {
    stop();
    emit sessionExpired();
}

void SessionMonitor::onTick() {
    if (!m_active) return;

    m_remainingSeconds--;
    emit remainingChanged(m_remainingSeconds);

    // Warning 30 seconds before expiry
    if (m_remainingSeconds <= AppConstants::AUTO_LOCK_WARNING_SECONDS && !m_warningSent) {
        m_warningSent = true;
        emit lockWarning(m_remainingSeconds);
    }

    // Session expired
    if (m_remainingSeconds <= 0) {
        stop();
        emit sessionExpired();
    }
}
