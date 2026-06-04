#pragma once

#include <vector>
#include <queue>
#include <thread>
#include <mutex>
#include <condition_variable>
#include <functional>
#include <future>
#include <memory>
#include <type_traits>
#include <stdexcept>

namespace efr {

/**
 * 固定大小线程池
 *
 * 用于流水线中人脸检测和识别的并行处理。
 * 支持提交任意可调用对象并返回 std::future。
 */
class ThreadPool {
public:
    explicit ThreadPool(size_t num_threads)
        : stop_(false)
        , active_tasks_(0)
    {
        if (num_threads == 0) {
            num_threads = std::thread::hardware_concurrency();
            if (num_threads == 0) num_threads = 2;
        }
        workers_.reserve(num_threads);
        for (size_t i = 0; i < num_threads; ++i) {
            workers_.emplace_back(&ThreadPool::workerLoop, this);
        }
    }

    ~ThreadPool() {
        shutdown();
    }

    ThreadPool(const ThreadPool&) = delete;
    ThreadPool& operator=(const ThreadPool&) = delete;

    /// 提交任务，返回 future
    template<typename F, typename... Args>
    auto submit(F&& f, Args&&... args)
        -> std::future<typename std::invoke_result_t<F, Args...>>
    {
        using ReturnType = typename std::invoke_result_t<F, Args...>;

        auto task = std::make_shared<std::packaged_task<ReturnType()>>(
            std::bind(std::forward<F>(f), std::forward<Args>(args)...)
        );

        std::future<ReturnType> result = task->get_future();
        {
            std::lock_guard<std::mutex> lock(mutex_);
            if (stop_) {
                throw std::runtime_error("ThreadPool: submit on stopped pool");
            }
            tasks_.emplace([task]() { (*task)(); });
            active_tasks_++;
        }
        cv_.notify_one();
        return result;
    }

    /// 等待所有已提交任务完成
    void waitAll() {
        std::unique_lock<std::mutex> lock(mutex_);
        cv_done_.wait(lock, [this] {
            return tasks_.empty() && active_tasks_ == 0;
        });
    }

    /// 获取线程数
    size_t size() const { return workers_.size(); }

    /// 获取待处理任务数
    size_t pendingTasks() const {
        std::lock_guard<std::mutex> lock(mutex_);
        return tasks_.size();
    }

    /// 优雅关闭：完成已提交任务后退出
    void shutdown() {
        {
            std::lock_guard<std::mutex> lock(mutex_);
            if (stop_) return;
            stop_ = true;
        }
        cv_.notify_all();
        for (auto& worker : workers_) {
            if (worker.joinable()) {
                worker.join();
            }
        }
        workers_.clear();
    }

private:
    void workerLoop() {
        while (true) {
            std::function<void()> task;
            {
                std::unique_lock<std::mutex> lock(mutex_);
                cv_.wait(lock, [this] {
                    return stop_ || !tasks_.empty();
                });

                if (stop_ && tasks_.empty()) {
                    return;
                }

                task = std::move(tasks_.front());
                tasks_.pop();
            }

            task();

            {
                std::lock_guard<std::mutex> lock(mutex_);
                active_tasks_--;
                if (active_tasks_ == 0 && tasks_.empty()) {
                    cv_done_.notify_all();
                }
            }
        }
    }

    std::vector<std::thread> workers_;
    std::queue<std::function<void()>> tasks_;

    mutable std::mutex mutex_;
    std::condition_variable cv_;
    std::condition_variable cv_done_;
    bool stop_;
    size_t active_tasks_;
};

} // namespace efr
