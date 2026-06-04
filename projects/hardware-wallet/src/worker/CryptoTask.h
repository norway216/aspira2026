#pragma once

#include <QRunnable>
#include <QObject>
#include <QThreadPool>
#include <functional>

/**
 * CryptoTask - QRunnable for dispatching crypto operations to QThreadPool.
 *
 * The work function runs on a worker thread.
 * The callback runs on the main thread via QMetaObject::invokeMethod.
 */
class CryptoTask : public QObject, public QRunnable {
    Q_OBJECT
public:
    using WorkFunc = std::function<void()>;
    using CallbackFunc = std::function<void()>;

    CryptoTask(WorkFunc work, CallbackFunc onComplete, QObject* parent = nullptr);

    void run() override;

    static CryptoTask* submit(WorkFunc work, CallbackFunc onComplete,
                              QObject* parent = nullptr,
                              QThreadPool* pool = nullptr);

signals:
    void finished();

private:
    WorkFunc m_work;
    CallbackFunc m_onComplete;
};
