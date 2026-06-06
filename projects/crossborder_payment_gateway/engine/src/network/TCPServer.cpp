#include "network/TCPServer.h"
#include "network/Session.h"
#include "processor/TransactionProcessor.h"
#include "processor/ValidationEngine.h"
#include "processor/ExchangeEngine.h"
#include "storage/AccountStore.h"
#include "storage/TransactionStore.h"
#include "crypto/HashChain.h"

#include <unistd.h>
#include <fcntl.h>
#include <sys/socket.h>
#include <sys/epoll.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <cerrno>
#include <cstring>
#include <iostream>
#include <sstream>
#include <algorithm>

namespace payment_engine {

TCPServer::TCPServer(const EngineConfig& config)
    : config_(config)
{
    request_queue_ = std::make_unique<MPMCQueue<EngineRequest, 65536>>();

    // Initialize processing components
    accounts_ = std::make_unique<AccountStore>();
    txn_store_ = std::make_unique<TransactionStore>();
    exchanger_ = std::make_unique<ExchangeEngine>();
    hash_chain_ = std::make_unique<HashChain>(config_.checkpoint_interval);
    validator_ = std::make_unique<ValidationEngine>(accounts_.get());
    processor_ = std::make_unique<TransactionProcessor>(
        validator_.get(), exchanger_.get(),
        accounts_.get(), txn_store_.get(), hash_chain_.get());

    // Create thread pool with worker threads from config
    int num_workers = config_.worker_threads > 0 ? config_.worker_threads : 8;
    thread_pool_ = std::make_unique<ThreadPool>(num_workers);
}

TCPServer::~TCPServer() {
    Stop();
}

bool TCPServer::Start() {
    if (running_.load(std::memory_order_acquire)) {
        return true;
    }

    // Create socket
    listen_fd_ = ::socket(AF_INET, SOCK_STREAM | SOCK_NONBLOCK, 0);
    if (listen_fd_ < 0) {
        std::cerr << "[TCPServer] Failed to create socket: "
                  << std::strerror(errno) << std::endl;
        return false;
    }

    // Allow address reuse
    int optval = 1;
    if (::setsockopt(listen_fd_, SOL_SOCKET, SO_REUSEADDR,
                     &optval, sizeof(optval)) < 0) {
        std::cerr << "[TCPServer] Failed to set SO_REUSEADDR: "
                  << std::strerror(errno) << std::endl;
        ::close(listen_fd_);
        listen_fd_ = -1;
        return false;
    }

    // Bind
    struct sockaddr_in addr;
    std::memset(&addr, 0, sizeof(addr));
    addr.sin_family = AF_INET;
    addr.sin_port = htons(static_cast<uint16_t>(config_.listen_port));
    if (::inet_pton(AF_INET, config_.listen_addr.c_str(), &addr.sin_addr) != 1) {
        std::cerr << "[TCPServer] Invalid listen address: "
                  << config_.listen_addr << std::endl;
        ::close(listen_fd_);
        listen_fd_ = -1;
        return false;
    }

    if (::bind(listen_fd_, reinterpret_cast<struct sockaddr*>(&addr),
               sizeof(addr)) < 0) {
        std::cerr << "[TCPServer] Failed to bind to " << config_.listen_addr
                  << ":" << config_.listen_port << " - "
                  << std::strerror(errno) << std::endl;
        ::close(listen_fd_);
        listen_fd_ = -1;
        return false;
    }

    // Listen
    if (::listen(listen_fd_, config_.max_connections) < 0) {
        std::cerr << "[TCPServer] Failed to listen: "
                  << std::strerror(errno) << std::endl;
        ::close(listen_fd_);
        listen_fd_ = -1;
        return false;
    }

    running_.store(true, std::memory_order_release);

    // Start accept thread
    accept_thread_ = std::thread([this]() { acceptLoop(); });

    // Start worker processing: enqueue a persistent worker that processes
    // requests from the MPMC queue in a loop
    for (int i = 0; i < config_.worker_threads; ++i) {
        thread_pool_->enqueue(ThreadPool::Priority::Normal,
            [this]() {
                while (this->running_.load(std::memory_order_acquire)) {
                    EngineRequest req;
                    if (this->request_queue_->tryPop(req)) {
                        this->workerCallback(std::move(req));
                    } else {
                        // Queue empty, yield briefly
                        std::this_thread::yield();
                    }
                }
            });
    }

    std::cout << "[TCPServer] Listening on " << config_.listen_addr
              << ":" << config_.listen_port
              << " with " << config_.worker_threads << " workers" << std::endl;
    return true;
}

void TCPServer::acceptLoop() {
    // Create epoll fd for scalable event-driven accept
    int epoll_fd = ::epoll_create1(0);
    if (epoll_fd < 0) {
        std::cerr << "[TCPServer] Failed to create epoll fd: "
                  << std::strerror(errno) << std::endl;
        return;
    }

    struct epoll_event ev;
    ev.events = EPOLLIN;
    ev.data.fd = listen_fd_;
    if (::epoll_ctl(epoll_fd, EPOLL_CTL_ADD, listen_fd_, &ev) < 0) {
        std::cerr << "[TCPServer] Failed to add listen fd to epoll: "
                  << std::strerror(errno) << std::endl;
        ::close(epoll_fd);
        return;
    }

    struct epoll_event events[16];

    while (running_.load(std::memory_order_acquire)) {
        int nfds = ::epoll_wait(epoll_fd, events, 16, 1000); // 1s timeout for re-checking running_
        if (nfds < 0) {
            if (errno == EINTR) continue;
            std::cerr << "[TCPServer] epoll_wait error: "
                      << std::strerror(errno) << std::endl;
            break;
        }

        for (int i = 0; i < nfds; ++i) {
            if (events[i].data.fd == listen_fd_ && (events[i].events & EPOLLIN)) {
                // Accept all pending connections (non-blocking)
                while (true) {
                    struct sockaddr_in client_addr;
                    socklen_t addrlen = sizeof(client_addr);

                    int client_fd = ::accept4(listen_fd_,
                        reinterpret_cast<struct sockaddr*>(&client_addr),
                        &addrlen, SOCK_NONBLOCK);
                    if (client_fd < 0) {
                        if (errno == EINTR) continue;
                        break; // No more connections to accept (EAGAIN/EWOULDBLOCK)
                    }

                    // Check max connections
                    size_t current_sessions;
                    {
                        std::lock_guard lock(sessions_mutex_);
                        current_sessions = sessions_.size();
                    }
                    if (current_sessions >= static_cast<size_t>(config_.max_connections)) {
                        std::cerr << "[TCPServer] Max connections ("
                                  << config_.max_connections
                                  << ") reached, rejecting new connection" << std::endl;
                        ::close(client_fd);
                        continue;
                    }

                    char client_ip[INET_ADDRSTRLEN];
                    ::inet_ntop(AF_INET, &client_addr.sin_addr,
                                client_ip, sizeof(client_ip));
                    int client_port = ntohs(client_addr.sin_port);

                    // Create and start session
                    auto session = std::make_unique<Session>(client_fd, this);
                    {
                        std::lock_guard lock(sessions_mutex_);
                        sessions_.push_back(std::move(session));
                        sessions_.back()->Start();
                    }

                    if (config_.verbose) {
                        std::cout << "[TCPServer] New connection from "
                                  << client_ip << ":" << client_port
                                  << " (fd=" << client_fd << ")" << std::endl;
                    }
                }
            }
        }
    }

    ::close(epoll_fd);
}

void TCPServer::workerCallback(EngineRequest req) {
    // Process the request
    EngineResponse resp = processor_->HandleRequest(req);

    // Route the response to the correct session
    if (resp.session_fd >= 0) {
        RouteResponse(resp);
    }
}

void TCPServer::RouteResponse(const EngineResponse& resp) {
    std::lock_guard lock(sessions_mutex_);
    for (auto& session : sessions_) {
        if (session->GetFD() == resp.session_fd && session->IsActive()) {
            session->SendResponse(resp);
            return;
        }
    }
}

bool TCPServer::PushRequest(EngineRequest&& req) {
    return request_queue_->tryPush(std::move(req));
}

int TCPServer::GetActiveConnections() const {
    std::lock_guard lock(sessions_mutex_);
    return static_cast<int>(sessions_.size());
}

size_t TCPServer::GetQueueDepth() const {
    return request_queue_->size();
}

int64_t TCPServer::GetTPS() const {
    return processor_->GetTPS();
}

void TCPServer::removeSession(int fd) {
    std::lock_guard lock(sessions_mutex_);
    auto it = std::remove_if(sessions_.begin(), sessions_.end(),
        [fd](const std::unique_ptr<Session>& s) {
            return s->GetFD() == fd;
        });
    if (it != sessions_.end()) {
        sessions_.erase(it, sessions_.end());
    }
}

void TCPServer::Stop() {
    bool expected = true;
    if (!running_.compare_exchange_strong(expected, false,
                                          std::memory_order_acq_rel)) {
        return; // Already stopped
    }

    std::cout << "[TCPServer] Stopping..." << std::endl;

    // Stop accept thread
    if (accept_thread_.joinable()) {
        accept_thread_.join();
    }

    // Close listen socket
    if (listen_fd_ >= 0) {
        ::close(listen_fd_);
        listen_fd_ = -1;
    }

    // Stop all sessions
    {
        std::lock_guard lock(sessions_mutex_);
        for (auto& session : sessions_) {
            session->Stop();
        }
        sessions_.clear();
    }

    // Stop thread pool
    thread_pool_->shutdown();

    std::cout << "[TCPServer] Stopped" << std::endl;
}

void TCPServer::Run() {
    if (!Start()) {
        std::cerr << "[TCPServer] Failed to start" << std::endl;
        return;
    }

    // Main loop - the accept thread and worker threads do the actual work
    // This method can be used as a blocking run, or the caller can manage
    // the lifecycle with Start/Stop.
    while (running_.load(std::memory_order_acquire)) {
        std::this_thread::sleep_for(std::chrono::seconds(1));
    }
}

} // namespace payment_engine
