#pragma once

#include <optional>
#include <string>
#include "engine/Common.h"
#include "storage/AccountStore.h"

namespace payment_engine {

/**
 * Transaction validation engine.
 * Checks account existence, status, balance sufficiency, and limits.
 */
class ValidationEngine {
public:
    explicit ValidationEngine(AccountStore* accounts);

    /**
     * Validate a transaction request.
     * Returns std::nullopt if valid, or an EngineError describing the failure.
     */
    std::optional<EngineError> Validate(const TransactionRequest& req);

    bool AccountExists(const std::string& account_id);
    bool HasSufficientFunds(const std::string& account_id, int64_t amount);
    bool IsWithinLimits(const std::string& account_id, int64_t amount);

private:
    AccountStore* accounts_;
};

} // namespace payment_engine
