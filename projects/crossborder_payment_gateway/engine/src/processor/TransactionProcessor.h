#pragma once

#include <string>
#include <atomic>
#include <memory>
#include "engine/Common.h"
#include "processor/ValidationEngine.h"
#include "processor/ExchangeEngine.h"
#include "storage/AccountStore.h"
#include "storage/TransactionStore.h"
#include "crypto/HashChain.h"

namespace payment_engine {

/**
 * Core transaction processing engine.
 *
 * Process() performs:
 * 1. Validate transaction (accounts, balance, limits)
 * 2. Look up exchange rate
 * 3. Calculate target amount
 * 4. Generate hash chain entry
 * 5. Reserve/settle funds
 * 6. Store transaction record
 * 7. Update TPS metrics
 */
class TransactionProcessor {
public:
    TransactionProcessor(ValidationEngine* validator, ExchangeEngine* exchanger,
                         AccountStore* accounts, TransactionStore* txn_store,
                         HashChain* hash_chain);

    /**
     * Process a single transaction request.
     */
    TransactionResult Process(const TransactionRequest& req);

    /**
     * Handle an incoming EngineRequest (JSON parsing + processing + response).
     */
    EngineResponse HandleRequest(const EngineRequest& req);

    int64_t GetTPS() const { return tps_.load(std::memory_order_acquire); }

    void UpdateTPS();

    std::string GetHealthInfo() const;

private:
    ValidationEngine* validator_;
    ExchangeEngine* exchanger_;
    AccountStore* accounts_;
    TransactionStore* txn_store_;
    HashChain* hash_chain_;

    std::atomic<int64_t> txn_counter_{0};
    std::atomic<int64_t> tps_{0};
    Clock::time_point tps_window_start_;
    std::atomic<int64_t> tps_window_count_{0};
};

} // namespace payment_engine
