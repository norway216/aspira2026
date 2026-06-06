#pragma once

#include <string>
#include <fstream>
#include <iostream>
#include "engine/Common.h"

namespace payment_engine {

struct EngineConfig {
    std::string listen_addr = "0.0.0.0";
    int listen_port = 9100;
    int worker_threads = 8;
    int max_connections = 1000;
    int session_timeout_sec = 30;
    int checkpoint_interval = 1000;
    std::string db_dsn;
    bool verbose = false;

    static EngineConfig LoadFromFile(const std::string& path) {
        EngineConfig cfg = Default();
        std::ifstream file(path);
        if (!file.is_open()) {
            std::cerr << "[EngineConfig] Warning: cannot open " << path
                      << ", using defaults." << std::endl;
            return cfg;
        }
        std::string content((std::istreambuf_iterator<char>(file)),
                             std::istreambuf_iterator<char>());
        SimpleJson doc(content);

        if (auto v = doc.GetString("listen_addr")) cfg.listen_addr = *v;
        if (auto v = doc.GetInt64("listen_port")) cfg.listen_port = static_cast<int>(*v);
        if (auto v = doc.GetInt64("worker_threads")) cfg.worker_threads = static_cast<int>(*v);
        if (auto v = doc.GetInt64("max_connections")) cfg.max_connections = static_cast<int>(*v);
        if (auto v = doc.GetInt64("session_timeout_sec")) cfg.session_timeout_sec = static_cast<int>(*v);
        if (auto v = doc.GetInt64("checkpoint_interval")) cfg.checkpoint_interval = static_cast<int>(*v);
        if (auto v = doc.GetString("db_dsn")) cfg.db_dsn = *v;
        if (auto v = doc.GetString("verbose")) cfg.verbose = (*v == "true");

        return cfg;
    }

    static EngineConfig Default() {
        return EngineConfig{};
    }
};

} // namespace payment_engine
