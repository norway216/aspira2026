#pragma once

#include <thread>
#include <atomic>
#include <memory>
#include <string>
#include <vector>
#include <functional>
#include "engine/Common.h"
#include "pipeline/RingBuffer.h"

namespace payment_engine {

// Forward declaration
class TCPServer;

/**
 * Per-connection TCP session.
 *
 * Each session has:
 * - Read thread: reads length-prefixed JSON from TCP, parses to EngineRequest,
 *   pushes to the shared MPMCQueue
 * - Write thread: pops from the SPSC output RingBuffer, writes to TCP
 */
class Session {
public:
    Session(int fd, TCPServer* server);
    ~Session();

    Session(const Session&) = delete;
    Session& operator=(const Session&) = delete;
    Session(Session&&) = delete;
    Session& operator=(Session&&) = delete;

    void Start();
    void Stop();

    /**
     * Send a response to this session's output queue.
     * Called by worker threads.
     */
    bool SendResponse(const EngineResponse& resp);

    int GetFD() const { return fd_; }
    bool IsActive() const { return active_.load(std::memory_order_acquire); }

private:
    int fd_;
    std::atomic<bool> active_{true};
    RingBuffer<std::string, 1024> output_queue_; // SPSC: worker -> session write thread
    std::thread read_thread_;
    std::thread write_thread_;
    TCPServer* server_;

    void readLoop();
    void writeLoop();

    /**
     * Read exactly n bytes from fd (handling partial reads).
     * Returns true on success, false on error/EOF.
     */
    bool readExact(uint8_t* buf, size_t n);

    /**
     * Write exactly n bytes to fd (handling partial writes).
     * Returns true on success, false on error.
     */
    bool writeExact(const uint8_t* buf, size_t n);
};

} // namespace payment_engine
