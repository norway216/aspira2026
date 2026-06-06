#pragma once

#include <atomic>
#include <array>
#include <new>
#include <thread>

namespace payment_engine {

/**
 * Lock-free MPMC (Multi-Producer Multi-Consumer) bounded queue.
 *
 * Uses Dmitry Vyukov's turn-based sequence locking algorithm,
 * which avoids ABA problems without the need for DCAS or tagged pointers.
 *
 * Each slot has a sequence number that cycles through the states:
 *   seq == i       : slot i is ready for push (producer turn)
 *   seq == i+1     : slot i is ready for pop (consumer turn)
 *   seq == i+1+N   : slot i is ready for next push cycle
 *   (where N = Capacity)
 */
template <typename T, size_t Capacity>
class MPMCQueue {
    static_assert((Capacity & (Capacity - 1)) == 0, "Capacity must be power of 2");
    static constexpr size_t MASK = Capacity - 1;

    struct Slot {
        std::atomic<size_t> sequence;
        T data;
    };

    std::array<Slot, Capacity> slots_;
    alignas(64) std::atomic<size_t> head_{0};
    alignas(64) std::atomic<size_t> tail_{0};

public:
    MPMCQueue() {
        for (size_t i = 0; i < Capacity; ++i) {
            slots_[i].sequence.store(i, std::memory_order_relaxed);
        }
    }

    MPMCQueue(const MPMCQueue&) = delete;
    MPMCQueue& operator=(const MPMCQueue&) = delete;
    MPMCQueue(MPMCQueue&&) = delete;
    MPMCQueue& operator=(MPMCQueue&&) = delete;

    /**
     * Try to push an item into the queue.
     * Returns false if the queue is full.
     */
    bool tryPush(T&& item) {
        size_t h;
        size_t seq;
        Slot* slot;

        while (true) {
            h = head_.load(std::memory_order_relaxed);
            slot = &slots_[h & MASK];
            seq = slot->sequence.load(std::memory_order_acquire);
            int64_t diff = static_cast<int64_t>(seq) - static_cast<int64_t>(h);

            if (diff == 0) {
                // Slot is ours to claim
                if (head_.compare_exchange_weak(h, h + 1, std::memory_order_relaxed)) {
                    break; // We own this slot
                }
                // CAS failed, retry
            } else if (diff < 0) {
                // Queue is full (consumer hasn't advanced far enough)
                return false;
            } else {
                // diff > 0: another producer is ahead of us; spin a bit then retry
                std::this_thread::yield();
            }
        }

        // Write the data
        new (&slot->data) T(std::move(item));
        // Make available for consumer
        slot->sequence.store(h + 1, std::memory_order_release);
        return true;
    }

    /**
     * Try to pop an item from the queue.
     * Returns false if the queue is empty.
     */
    bool tryPop(T& item) {
        size_t t;
        size_t seq;
        Slot* slot;

        while (true) {
            t = tail_.load(std::memory_order_relaxed);
            slot = &slots_[t & MASK];
            seq = slot->sequence.load(std::memory_order_acquire);
            int64_t diff = static_cast<int64_t>(seq) - static_cast<int64_t>(t + 1);

            if (diff == 0) {
                // Slot is ours to claim
                if (tail_.compare_exchange_weak(t, t + 1, std::memory_order_relaxed)) {
                    break; // We own this slot
                }
                // CAS failed, retry
            } else if (diff < 0) {
                // Queue is empty
                return false;
            } else {
                // diff > 0: another consumer is ahead of us; spin a bit then retry
                std::this_thread::yield();
            }
        }

        // Read the data
        item = std::move(slot->data);
        slot->data.~T();
        // Make available for next producer (advance by Capacity to maintain ordering)
        slot->sequence.store(t + MASK + 1, std::memory_order_release);
        return true;
    }

    size_t size() const {
        size_t h = head_.load(std::memory_order_acquire);
        size_t t = tail_.load(std::memory_order_acquire);
        return h - t;
    }

    bool empty() const {
        return head_.load(std::memory_order_acquire) ==
               tail_.load(std::memory_order_acquire);
    }

    size_t capacity() const { return Capacity; }
};

} // namespace payment_engine
