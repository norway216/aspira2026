#include "ThreadPool.h"
#include <QMutexLocker>

ThreadPool *ThreadPool::s_instance = nullptr;

ThreadPool::ThreadPool(QObject *parent)
    : QObject(parent)
{
    s_instance = this;
    // Default: use all cores except one (or at least 1)
    int cores = std::max(1, QThread::idealThreadCount() - 1);
    m_pool.setMaxThreadCount(cores);
}

ThreadPool::~ThreadPool()
{
    m_pool.waitForDone();
    if (s_instance == this) s_instance = nullptr;
}

int ThreadPool::activeThreadCount() const { return m_pool.activeThreadCount(); }
int ThreadPool::maxThreadCount()    const { return m_pool.maxThreadCount(); }

void ThreadPool::setMaxThreadCount(int n)
{
    if (n != m_pool.maxThreadCount()) {
        m_pool.setMaxThreadCount(std::max(1, n));
        emit maxThreadCountChanged();
    }
}

bool ThreadPool::submit(std::function<void()> task, int priority)
{
    auto *runnable = new FunctionRunnable(std::move(task));
    runnable->setAutoDelete(true);
    m_pool.start(runnable, priority);
    return true;
}

bool ThreadPool::waitForDone(int timeoutMs) { return m_pool.waitForDone(timeoutMs); }
void ThreadPool::clear()                     { m_pool.clear(); }

ThreadPool *ThreadPool::instance() { return s_instance; }
