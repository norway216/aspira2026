#pragma once

#include <atomic>
#include <vector>
#include <memory>
#include <optional>
#include <new>

namespace ehw {

/**
 * Lock-free SPSC (Single Producer Single Consumer) ring buffer.
 *
 * Features:
 * - Cache-line padding to prevent false sharing
 * - Power-of-2 capacity for efficient modulo via bitmask
 * - Wait-free for both producer and consumer
 * - Supports move-only and copyable types
 */
template <typename T, size_t Capacity = 1024>
class RingBuffer {
    static_assert((Capacity & (Capacity - 1)) == 0, "Capacity must be a power of 2");
    static constexpr size_t kMask = Capacity - 1;
    static constexpr size_t kCacheLine = 64;

public:
    RingBuffer() {
        m_buffer.resize(Capacity);
    }

    ~RingBuffer() = default;

    RingBuffer(const RingBuffer&) = delete;
    RingBuffer& operator=(const RingBuffer&) = delete;
    RingBuffer(RingBuffer&&) = delete;
    RingBuffer& operator=(RingBuffer&&) = delete;

    /**
     * Try to push an item. Returns false if buffer is full.
     */
    bool tryPush(const T& item) {
        size_t head = m_head.load(std::memory_order_relaxed);
        size_t next = (head + 1) & kMask;
        if (next == m_tailCache) {
            m_tailCache = m_tail.load(std::memory_order_acquire);
            if (next == m_tailCache) {
                return false; // Full
            }
        }
        new (&m_buffer[head]) T(item);
        m_head.store(next, std::memory_order_release);
        return true;
    }

    bool tryPush(T&& item) {
        size_t head = m_head.load(std::memory_order_relaxed);
        size_t next = (head + 1) & kMask;
        if (next == m_tailCache) {
            m_tailCache = m_tail.load(std::memory_order_acquire);
            if (next == m_tailCache) {
                return false; // Full
            }
        }
        new (&m_buffer[head]) T(std::move(item));
        m_head.store(next, std::memory_order_release);
        return true;
    }

    /**
     * Try to pop an item. Returns std::nullopt if buffer is empty.
     */
    std::optional<T> tryPop() {
        size_t tail = m_tail.load(std::memory_order_relaxed);
        if (tail == m_headCache) {
            m_headCache = m_head.load(std::memory_order_acquire);
            if (tail == m_headCache) {
                return std::nullopt; // Empty
            }
        }
        T item = std::move(m_buffer[tail]);
        m_buffer[tail].~T();
        m_tail.store((tail + 1) & kMask, std::memory_order_release);
        return item;
    }

    bool empty() const {
        return m_tail.load(std::memory_order_acquire) ==
               m_head.load(std::memory_order_acquire);
    }

    size_t size() const {
        size_t head = m_head.load(std::memory_order_acquire);
        size_t tail = m_tail.load(std::memory_order_acquire);
        if (head >= tail) return head - tail;
        return Capacity - tail + head;
    }

    size_t capacity() const { return Capacity; }

private:
    // Cache-line aligned to prevent false sharing
    alignas(kCacheLine) std::atomic<size_t> m_head{0};
    alignas(kCacheLine) std::atomic<size_t> m_tail{0};
    alignas(kCacheLine) size_t m_headCache{0};   // Producer's cached tail
    alignas(kCacheLine) size_t m_tailCache{0};   // Consumer's cached head

    std::vector<T> m_buffer;
};

} // namespace ehw
