#pragma once

#include <atomic>
#include <memory>
#include <thread>
#include <vector>
#include <mutex>
#include <functional>
#include "config/EngineConfig.h"
#include "pipeline/MPMCQueue.h"
#include "pipeline/ThreadPool.h"
#include "engine/Common.h"

namespace payment_engine {

// Forward declarations
class Session;
class TransactionProcessor;
class ValidationEngine;
class ExchangeEngine;
class AccountStore;
class TransactionStore;
class HashChain;

/**
 * Main TCP server for the payment engine.
 *
 * Architecture:
 * - Non-blocking TCP socket with accept loop
 * - Each accepted connection gets a Session (read thread + write thread)
 * - Shared MPMCQueue<EngineRequest> (lock-free) for incoming requests
 * - ThreadPool with work-stealing for processing
 * - Workers pop from MPMCQueue, process via TransactionProcessor,
 *   and push responses to the corresponding Session's SPSC output queue
 */
class TCPServer {
public:
    explicit TCPServer(const EngineConfig& config);
    ~TCPServer();

    TCPServer(const TCPServer&) = delete;
    TCPServer& operator=(const TCPServer&) = delete;
    TCPServer(TCPServer&&) = delete;
    TCPServer& operator=(TCPServer&&) = delete;

    bool Start();
    void Stop();
    void Run();

    int GetActiveConnections() const;
    size_t GetQueueDepth() const;
    int64_t GetTPS() const;

    /**
     * Push a request into the shared processing queue.
     * Called by Session read threads.
     */
    bool PushRequest(EngineRequest&& req);

    /**
     * Route a response to the correct session by fd.
     * Called by worker threads.
     */
    void RouteResponse(const EngineResponse& resp);

private:
    EngineConfig config_;
    int listen_fd_ = -1;
    std::atomic<bool> running_{false};

    // Core components
    std::unique_ptr<MPMCQueue<EngineRequest, 65536>> request_queue_;
    std::unique_ptr<ThreadPool> thread_pool_;

    // Processing components
    std::unique_ptr<AccountStore> accounts_;
    std::unique_ptr<TransactionStore> txn_store_;
    std::unique_ptr<ExchangeEngine> exchanger_;
    std::unique_ptr<ValidationEngine> validator_;
    std::unique_ptr<HashChain> hash_chain_;
    std::unique_ptr<TransactionProcessor> processor_;

    // Session management
    std::vector<std::unique_ptr<Session>> sessions_;
    mutable std::mutex sessions_mutex_;
    std::thread accept_thread_;

    // Accept loop
    void acceptLoop();

    // Worker callback - processes requests from the queue
    void workerCallback(EngineRequest req);

    // Remove a session (called from Session destructor)
    friend class Session;
    void removeSession(int fd);
};

} // namespace payment_engine
