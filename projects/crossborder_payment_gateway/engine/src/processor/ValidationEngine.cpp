#include "processor/ValidationEngine.h"

namespace payment_engine {

ValidationEngine::ValidationEngine(AccountStore* accounts)
    : accounts_(accounts)
{
}

bool ValidationEngine::AccountExists(const std::string& account_id) {
    return accounts_->GetAccountCopy(account_id).has_value();
}

bool ValidationEngine::HasSufficientFunds(const std::string& account_id, int64_t amount) {
    auto acct = accounts_->GetAccountCopy(account_id);
    if (!acct) return false;
    int64_t available = acct->balance.load(std::memory_order_acquire) -
                       acct->reserved_balance.load(std::memory_order_acquire);
    return available >= amount;
}

bool ValidationEngine::IsWithinLimits(const std::string& account_id, int64_t amount) {
    auto acct = accounts_->GetAccountCopy(account_id);
    if (!acct) return false;
    // Check daily limit
    if (acct->daily_used + amount > acct->daily_limit) return false;
    // Check monthly limit
    if (acct->monthly_used + amount > acct->monthly_limit) return false;
    return true;
}

std::optional<EngineError> ValidationEngine::Validate(const TransactionRequest& req) {
    // Check payer account exists
    if (!AccountExists(req.payer_account_id)) {
        return EngineError{
            ErrorCode::AccountNotFound,
            "Payer account not found: " + req.payer_account_id,
            false
        };
    }

    // Check payee account exists
    if (!AccountExists(req.payee_account_id)) {
        return EngineError{
            ErrorCode::AccountNotFound,
            "Payee account not found: " + req.payee_account_id,
            false
        };
    }

    // Check payer account status
    auto payer = accounts_->GetAccountCopy(req.payer_account_id);
    if (payer && payer->status != "active") {
        return EngineError{
            ErrorCode::AccountFrozen,
            "Payer account is " + payer->status,
            false
        };
    }

    // Check source currency matches account currency
    if (payer && payer->currency != req.source_currency) {
        return EngineError{
            ErrorCode::InvalidCurrency,
            "Payer account currency mismatch: expected " + payer->currency +
                ", got " + req.source_currency,
            false
        };
    }

    // Check sufficient funds (amount + fee)
    int64_t total_required = req.source_amount + req.fee;
    if (!HasSufficientFunds(req.payer_account_id, total_required)) {
        return EngineError{
            ErrorCode::InsufficientFunds,
            "Insufficient funds in account " + req.payer_account_id,
            false
        };
    }

    // Check daily/monthly limits
    if (!IsWithinLimits(req.payer_account_id, total_required)) {
        return EngineError{
            ErrorCode::LimitExceeded,
            "Transaction exceeds daily/monthly limit for account " + req.payer_account_id,
            false
        };
    }

    return std::nullopt; // Valid
}

} // namespace payment_engine
