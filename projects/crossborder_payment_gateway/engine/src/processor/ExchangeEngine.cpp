#include "processor/ExchangeEngine.h"

namespace payment_engine {

void ExchangeEngine::UpdateRate(const ExchangeRate& rate) {
    std::unique_lock lock(mutex_);
    rates_[MakeKey(rate.source, rate.target)] = rate;
    // Also store reverse rate if not explicitly set
    std::string rev_key = MakeKey(rate.target, rate.source);
    if (rates_.find(rev_key) == rates_.end()) {
        ExchangeRate rev;
        rev.source = rate.target;
        rev.target = rate.source;
        rev.rate = (rate.rate != 0.0) ? (1.0 / rate.rate) : 0.0;
        rev.bid = rev.rate;
        rev.ask = rev.rate;
        rates_[rev_key] = rev;
    }
}

std::optional<ExchangeRate> ExchangeEngine::GetRate(const std::string& source,
                                                     const std::string& target) {
    std::shared_lock lock(mutex_);
    std::string key = MakeKey(source, target);
    auto it = rates_.find(key);
    if (it == rates_.end()) {
        return std::nullopt;
    }
    return it->second;
}

void ExchangeEngine::LoadDefaultRates() {
    std::unique_lock lock(mutex_);

    auto add = [&](const std::string& src, const std::string& tgt, double rate,
                   double bid, double ask) {
        ExchangeRate r;
        r.source = src;
        r.target = tgt;
        r.rate = rate;
        r.bid = bid;
        r.ask = ask;
        rates_[MakeKey(src, tgt)] = r;
    };

    add("USD", "CNY", 7.25, 7.24, 7.26);
    add("USD", "EUR", 0.92, 0.91, 0.93);
    add("USD", "JPY", 155.0, 154.5, 155.5);
    add("USD", "GBP", 0.79, 0.78, 0.80);
    add("EUR", "CNY", 7.88, 7.86, 7.90);

    // Add reverse rates
    add("CNY", "USD", 1.0 / 7.25, 1.0 / 7.26, 1.0 / 7.24);
    add("EUR", "USD", 1.0 / 0.92, 1.0 / 0.93, 1.0 / 0.91);
    add("JPY", "USD", 1.0 / 155.0, 1.0 / 155.5, 1.0 / 154.5);
    add("GBP", "USD", 1.0 / 0.79, 1.0 / 0.80, 1.0 / 0.78);
    add("CNY", "EUR", 1.0 / 7.88, 1.0 / 7.90, 1.0 / 7.86);

    std::cout << "[ExchangeEngine] Loaded default exchange rates" << std::endl;
}

} // namespace payment_engine
