#include "CryptoTask.h"

#include <QCoreApplication>
#include <QDebug>

CryptoTask::CryptoTask(WorkFunc work, CallbackFunc onComplete, QObject* parent)
    : QObject(parent), QRunnable(), m_work(std::move(work)), m_onComplete(std::move(onComplete)) {
    setAutoDelete(true);
}

void CryptoTask::run() {
    // Execute the work on the thread pool
    if (m_work) {
        m_work();
    }

    // Schedule callback on main thread.
    // IMPORTANT: Use QCoreApplication::instance() as context, NOT 'this',
    // because this CryptoTask will be auto-deleted when run() returns.
    if (m_onComplete) {
        QMetaObject::invokeMethod(QCoreApplication::instance(),
            [cb = std::move(m_onComplete)]() { cb(); },
            Qt::QueuedConnection);
    }
}

CryptoTask* CryptoTask::submit(WorkFunc work, CallbackFunc onComplete,
                                QObject* parent, QThreadPool* pool) {
    auto* task = new CryptoTask(std::move(work), std::move(onComplete), parent);

    if (pool) {
        pool->start(task);
    } else {
        QThreadPool::globalInstance()->start(task);
    }

    return task;
}
