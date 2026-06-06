#include <iostream>
#include <csignal>
#include <atomic>
#include <thread>
#include <chrono>
#include <cstdlib>
#include <cstring>

#include "config/EngineConfig.h"
#include "network/TCPServer.h"

namespace pe = payment_engine;

std::atomic<bool> g_running{true};

void SignalHandler(int sig) {
    if (sig == SIGINT || sig == SIGTERM) {
        g_running = false;
    }
}

void PrintBanner() {
    std::cout << "=================================================" << std::endl;
    std::cout << "  Aspira Cross-Border Payment Engine v1.0" << std::endl;
    std::cout << "  High-Performance C++ Transaction Processor" << std::endl;
    std::cout << "=================================================" << std::endl;
}

int main(int argc, char* argv[]) {
    PrintBanner();

    // Parse command line for config file path
    std::string config_path = "config/engine.json";
    if (argc > 1) {
        config_path = argv[1];
    }

    // Load configuration
    pe::EngineConfig config;
    if (config_path == "default") {
        config = pe::EngineConfig::Default();
        std::cout << "Using default configuration" << std::endl;
    } else {
        config = pe::EngineConfig::LoadFromFile(config_path);
        std::cout << "Loaded configuration from: " << config_path << std::endl;
    }

    std::cout << "Listening on " << config.listen_addr << ":" << config.listen_port << std::endl;
    std::cout << "Worker threads: " << config.worker_threads << std::endl;
    std::cout << "Max connections: " << config.max_connections << std::endl;
    std::cout << "Session timeout: " << config.session_timeout_sec << "s" << std::endl;
    std::cout << "Hash checkpoint interval: " << config.checkpoint_interval << std::endl;
    std::cout << std::endl;

    // Set up signal handlers
    signal(SIGINT, SignalHandler);
    signal(SIGTERM, SignalHandler);
    signal(SIGPIPE, SIG_IGN); // Ignore SIGPIPE to handle EPIPE via write errors

    // Create and start the server
    pe::TCPServer server(config);
    if (!server.Start()) {
        std::cerr << "Failed to start engine!" << std::endl;
        return 1;
    }

    std::cout << "Engine started successfully." << std::endl;
    std::cout << "Press Ctrl+C to stop." << std::endl;
    std::cout << std::endl;

    // Main loop: print stats every 5 seconds
    int stats_interval = 5;
    while (g_running) {
        std::this_thread::sleep_for(std::chrono::seconds(stats_interval));

        if (!g_running) break;

        auto connections = server.GetActiveConnections();
        auto queue_depth = server.GetQueueDepth();
        auto tps = server.GetTPS();

        std::cout << "[STATS] Connections: " << connections
                  << " | Queue depth: " << queue_depth
                  << " | TPS: " << tps
                  << std::endl;
    }

    std::cout << std::endl;
    std::cout << "Shutting down..." << std::endl;
    server.Stop();
    std::cout << "Engine stopped." << std::endl;

    return 0;
}
