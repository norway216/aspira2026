#pragma once

#include <vector>
#include <mutex>
#include <condition_variable>
#include <chrono>
#include <optional>

namespace efr {

/**
 * 线程安全环形缓冲区 (RingBuffer)
 *
 * 用于流水线各阶段之间的数据传递，避免频繁内存分配。
 * 支持单生产者 / 单消费者（SPSC）模式，以及多生产者 / 多消费者（MPMC）模式。
 *
 * @tparam T  存储的元素类型
 */
template<typename T>
class RingBuffer {
public:
    explicit RingBuffer(size_t capacity)
        : buffer_(capacity)
        , capacity_(capacity)
        , read_idx_(0)
        , write_idx_(0)
        , count_(0)
    {}

    RingBuffer(const RingBuffer&) = delete;
    RingBuffer& operator=(const RingBuffer&) = delete;
    RingBuffer(RingBuffer&&) = delete;
    RingBuffer& operator=(RingBuffer&&) = delete;

    // ---- 生产者接口 ----

    /// 尝试非阻塞推入，成功返回 true
    bool tryPush(T item) {
        std::lock_guard<std::mutex> lock(mutex_);
        if (count_ >= capacity_) {
            return false;
        }
        buffer_[write_idx_] = std::move(item);
        write_idx_ = (write_idx_ + 1) % capacity_;
        count_++;
        cv_.notify_one();
        return true;
    }

    /// 阻塞推入，超时返回 false（默认无限等待）
    bool push(T item, std::chrono::milliseconds timeout = std::chrono::milliseconds(5000)) {
        std::unique_lock<std::mutex> lock(mutex_);
        if (!cv_full_.wait_for(lock, timeout, [this] { return count_ < capacity_; })) {
            return false; // 超时
        }
        buffer_[write_idx_] = std::move(item);
        write_idx_ = (write_idx_ + 1) % capacity_;
        count_++;
        cv_.notify_one();
        return true;
    }

    // ---- 消费者接口 ----

    /// 尝试非阻塞弹出，无数据时返回 std::nullopt
    std::optional<T> tryPop() {
        std::lock_guard<std::mutex> lock(mutex_);
        if (count_ == 0) {
            return std::nullopt;
        }
        T item = std::move(buffer_[read_idx_]);
        read_idx_ = (read_idx_ + 1) % capacity_;
        count_--;
        cv_full_.notify_one();
        return item;
    }

    /// 阻塞弹出，超时返回 std::nullopt
    std::optional<T> pop(std::chrono::milliseconds timeout = std::chrono::milliseconds(5000)) {
        std::unique_lock<std::mutex> lock(mutex_);
        if (!cv_.wait_for(lock, timeout, [this] { return count_ > 0; })) {
            return std::nullopt; // 超时
        }
        T item = std::move(buffer_[read_idx_]);
        read_idx_ = (read_idx_ + 1) % capacity_;
        count_--;
        cv_full_.notify_one();
        return item;
    }

    // ---- 状态查询 ----

    size_t size() const {
        std::lock_guard<std::mutex> lock(mutex_);
        return count_;
    }

    size_t capacity() const { return capacity_; }

    bool empty() const {
        std::lock_guard<std::mutex> lock(mutex_);
        return count_ == 0;
    }

    bool full() const {
        std::lock_guard<std::mutex> lock(mutex_);
        return count_ >= capacity_;
    }

    /// 唤醒所有等待线程（用于关闭流水线）
    void notifyAll() {
        cv_.notify_all();
        cv_full_.notify_all();
    }

    /// 清空缓冲区
    void clear() {
        std::lock_guard<std::mutex> lock(mutex_);
        read_idx_ = 0;
        write_idx_ = 0;
        count_ = 0;
    }

private:
    std::vector<T> buffer_;
    const size_t capacity_;
    size_t read_idx_;
    size_t write_idx_;
    size_t count_;

    mutable std::mutex mutex_;
    std::condition_variable cv_;       // 消费者等待数据
    std::condition_variable cv_full_;  // 生产者等待空间
};

} // namespace efr
