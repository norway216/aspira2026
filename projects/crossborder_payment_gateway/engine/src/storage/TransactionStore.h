#pragma once

#include <string>
#include <vector>
#include <shared_mutex>
#include <optional>
#include <atomic>
#include <algorithm>
#include "engine/Common.h"

namespace payment_engine {

class TransactionStore {
public:
    struct TxnRecord {
        std::string id;
        std::string merchant_id;
        std::string source_currency;
        std::string target_currency;
        int64_t source_amount = 0;
        int64_t target_amount = 0;
        int64_t fee = 0;
        double exchange_rate = 0.0;
        TxnStatus status = TxnStatus::Pending;
        std::string hash_chain_prev;
        std::string hash_chain_curr;
        int64_t timestamp_ns = 0;
    };

    TransactionStore() = default;

    void Append(const TxnRecord& txn);
    std::optional<TxnRecord> Get(const std::string& id) const;
    std::vector<TxnRecord> GetRecent(int count) const;

    int64_t GetTotalCount() const { return total_count_.load(std::memory_order_acquire); }
    int64_t GetTotalVolume() const { return total_volume_.load(std::memory_order_acquire); }

private:
    std::vector<TxnRecord> transactions_;
    mutable std::shared_mutex mutex_;
    std::atomic<int64_t> total_count_{0};
    std::atomic<int64_t> total_volume_{0};
};

} // namespace payment_engine
