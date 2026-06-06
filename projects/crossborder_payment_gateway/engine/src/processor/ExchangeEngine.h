#pragma once

#include <string>
#include <unordered_map>
#include <shared_mutex>
#include <optional>
#include <iostream>

namespace payment_engine {

struct ExchangeRate {
    std::string source;
    std::string target;
    double rate = 0.0;
    double bid = 0.0;
    double ask = 0.0;
};

/**
 * In-memory exchange rate engine.
 * Thread-safe with shared_mutex.
 */
class ExchangeEngine {
public:
    ExchangeEngine() {
        LoadDefaultRates();
    }

    void UpdateRate(const ExchangeRate& rate);
    std::optional<ExchangeRate> GetRate(const std::string& source, const std::string& target);
    void LoadDefaultRates();

private:
    static std::string MakeKey(const std::string& source, const std::string& target) {
        return source + "-" + target;
    }

    std::unordered_map<std::string, ExchangeRate> rates_;
    mutable std::shared_mutex mutex_;
};

} // namespace payment_engine
