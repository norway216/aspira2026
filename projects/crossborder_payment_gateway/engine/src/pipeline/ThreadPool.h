#pragma once

#include <vector>
#include <queue>
#include <thread>
#include <mutex>
#include <condition_variable>
#include <functional>
#include <future>
#include <memory>
#include <atomic>
#include <algorithm>
#include <type_traits>
#include <chrono>

namespace payment_engine {

/**
 * High-performance work-stealing thread pool with priority support.
 *
 * Features:
 * - Priority levels: HIGH, NORMAL, LOW
 * - Work-stealing between queues for load balancing
 * - std::jthread with stop_token for clean shutdown
 * - submit() returns std::future for result retrieval
 * - enqueue() for fire-and-forget
 */
class ThreadPool {
public:
    enum class Priority : uint8_t {
        High = 0,
        Normal = 1,
        Low = 2,
        Count = 3
    };

    struct Task {
        std::function<void()> func;
        Priority priority;
        uint64_t id;

        bool operator<(const Task& other) const {
            // Higher priority = lower numeric value = should be at top of queue
            if (priority != other.priority)
                return static_cast<uint8_t>(priority) > static_cast<uint8_t>(other.priority);
            return id > other.id; // FIFO within same priority
        }
    };

    struct WorkerQueue {
        std::priority_queue<Task> tasks;
        mutable std::mutex mutex;
        std::condition_variable_any cv;
    };

    explicit ThreadPool(size_t numThreads = 0)
        : m_running(true)
    {
        if (numThreads == 0) {
            numThreads = std::max(1u, std::thread::hardware_concurrency());
        }
        m_queues.reserve(numThreads);
        for (size_t i = 0; i < numThreads; ++i) {
            m_queues.emplace_back(std::make_unique<WorkerQueue>());
        }
        m_workers.reserve(numThreads);
        for (size_t i = 0; i < numThreads; ++i) {
            m_workers.emplace_back([this, i](std::stop_token stoken) {
                workerLoop(i, stoken);
            });
        }
    }

    ~ThreadPool() {
        shutdown();
    }

    ThreadPool(const ThreadPool&) = delete;
    ThreadPool& operator=(const ThreadPool&) = delete;
    ThreadPool(ThreadPool&&) = delete;
    ThreadPool& operator=(ThreadPool&&) = delete;

    /**
     * Submit a task with a given priority and return a std::future.
     */
    template <typename F, typename... Args>
    auto submit(Priority priority, F&& f, Args&&... args)
        -> std::future<std::invoke_result_t<F, Args...>>
    {
        using ReturnType = std::invoke_result_t<F, Args...>;
        auto task = std::make_shared<std::packaged_task<ReturnType()>>(
            std::bind(std::forward<F>(f), std::forward<Args>(args)...)
        );
        std::future<ReturnType> future = task->get_future();

        uint64_t id = m_taskCounter.fetch_add(1, std::memory_order_relaxed);
        size_t queueIdx = id % m_queues.size();

        {
            std::lock_guard lock(m_queues[queueIdx]->mutex);
            m_queues[queueIdx]->tasks.emplace(Task{
                [task]() { (*task)(); },
                priority,
                id
            });
        }
        m_queues[queueIdx]->cv.notify_one();
        return future;
    }

    /**
     * Enqueue a fire-and-forget task with a given priority.
     */
    void enqueue(Priority priority, std::function<void()> func) {
        uint64_t id = m_taskCounter.fetch_add(1, std::memory_order_relaxed);
        size_t queueIdx = id % m_queues.size();
        {
            std::lock_guard lock(m_queues[queueIdx]->mutex);
            m_queues[queueIdx]->tasks.emplace(Task{std::move(func), priority, id});
        }
        m_queues[queueIdx]->cv.notify_one();
    }

    size_t threadCount() const { return m_workers.size(); }

    size_t pendingTasks() const {
        size_t count = 0;
        for (const auto& q : m_queues) {
            std::lock_guard lock(q->mutex);
            count += q->tasks.size();
        }
        return count;
    }

    void shutdown() {
        m_running.store(false, std::memory_order_release);
        for (auto& q : m_queues) {
            q->cv.notify_all();
        }
        for (auto& w : m_workers) {
            if (w.joinable()) {
                w.request_stop();
                w.join();
            }
        }
        m_workers.clear();
        m_queues.clear();
    }

private:
    std::vector<std::unique_ptr<WorkerQueue>> m_queues;
    std::vector<std::jthread> m_workers;
    std::atomic<bool> m_running;
    std::atomic<uint64_t> m_taskCounter{0};

    void workerLoop(size_t homeIdx, std::stop_token stoken) {
        while (!stoken.stop_requested()) {
            Task task;
            bool found = false;

            // Try own queue first
            {
                auto& queue = m_queues[homeIdx];
                std::unique_lock lock(queue->mutex);
                queue->cv.wait_for(lock, std::chrono::milliseconds(10),
                    [&] { return !queue->tasks.empty() || stoken.stop_requested(); });

                if (stoken.stop_requested()) break;

                if (!queue->tasks.empty()) {
                    task = std::move(const_cast<Task&>(queue->tasks.top()));
                    queue->tasks.pop();
                    found = true;
                }
            }

            // Work stealing: try other queues in round-robin order
            if (!found) {
                for (size_t offset = 1; offset < m_queues.size(); ++offset) {
                    size_t idx = (homeIdx + offset) % m_queues.size();
                    auto& queue = m_queues[idx];
                    std::unique_lock lock(queue->mutex, std::try_to_lock);
                    if (lock.owns_lock() && !queue->tasks.empty()) {
                        task = std::move(const_cast<Task&>(queue->tasks.top()));
                        queue->tasks.pop();
                        found = true;
                        break;
                    }
                }
            }

            if (found && task.func) {
                task.func();
            }
        }
    }
};

} // namespace payment_engine
