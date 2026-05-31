#pragma once

#include <QObject>
#include <QThread>
#include <QRunnable>
#include <QThreadPool>
#include <QMutex>
#include <functional>
#include <memory>

/// Thin wrapper around QThreadPool with priority-based scheduling
/// and hardware-resource-aware allocation.
class ThreadPool : public QObject
{
    Q_OBJECT
    Q_PROPERTY(int activeThreadCount READ activeThreadCount NOTIFY activeThreadCountChanged)
    Q_PROPERTY(int maxThreadCount READ maxThreadCount WRITE setMaxThreadCount NOTIFY maxThreadCountChanged)

public:
    explicit ThreadPool(QObject *parent = nullptr);
    ~ThreadPool() override;

    int  activeThreadCount() const;
    int  maxThreadCount()    const;
    void setMaxThreadCount(int n);

    /// Submit a task (returns true if queued).
    /// Priority: 0 = highest, larger = lower.
    bool submit(std::function<void()> task, int priority = 0);

    /// Wait for all pending tasks to complete (timeout in ms, -1 = forever).
    bool waitForDone(int timeoutMs = -1);

    /// Clear the pending queue.
    void clear();

    /// Return the global instance (singleton).
    static ThreadPool *instance();

signals:
    void activeThreadCountChanged();
    void maxThreadCountChanged();

private:
    QThreadPool m_pool;
    static ThreadPool *s_instance;
};

/// A QRunnable that wraps a std::function for use with QThreadPool.
class FunctionRunnable : public QRunnable
{
public:
    explicit FunctionRunnable(std::function<void()> fn) : m_fn(std::move(fn)) {}
    void run() override { if (m_fn) m_fn(); }
private:
    std::function<void()> m_fn;
};
