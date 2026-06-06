#pragma once

#include <string>
#include <unordered_map>
#include <memory>
#include <shared_mutex>
#include <optional>
#include <iostream>
#include "engine/Common.h"

namespace payment_engine {

struct AccountData {
    std::string id;
    std::string merchant_id;
    std::string currency;
    std::atomic<int64_t> balance{0};
    std::atomic<int64_t> reserved_balance{0};
    std::string status; // "active", "frozen", "closed"
    int64_t daily_limit = 10000000;
    int64_t daily_used = 0;
    int64_t monthly_limit = 100000000;
    int64_t monthly_used = 0;

    AccountData() = default;
    AccountData(const AccountData& other)
        : id(other.id)
        , merchant_id(other.merchant_id)
        , currency(other.currency)
        , status(other.status)
        , daily_limit(other.daily_limit)
        , daily_used(other.daily_used)
        , monthly_limit(other.monthly_limit)
        , monthly_used(other.monthly_used)
    {
        balance.store(other.balance.load(std::memory_order_relaxed), std::memory_order_relaxed);
        reserved_balance.store(other.reserved_balance.load(std::memory_order_relaxed), std::memory_order_relaxed);
    }
    AccountData& operator=(const AccountData& other) {
        if (this != &other) {
            id = other.id;
            merchant_id = other.merchant_id;
            currency = other.currency;
            status = other.status;
            daily_limit = other.daily_limit;
            daily_used = other.daily_used;
            monthly_limit = other.monthly_limit;
            monthly_used = other.monthly_used;
            balance.store(other.balance.load(std::memory_order_relaxed), std::memory_order_relaxed);
            reserved_balance.store(other.reserved_balance.load(std::memory_order_relaxed), std::memory_order_relaxed);
        }
        return *this;
    }
};

class AccountStore {
public:
    AccountStore();

    bool CreateAccount(const AccountData& acct);
    std::optional<std::reference_wrapper<AccountData>> GetAccount(const std::string& id);
    bool UpdateBalance(const std::string& id, int64_t new_balance, int64_t new_reserved);
    void SeedDefaultAccounts();

    // Returns a copy for thread-safe external access
    std::optional<AccountData> GetAccountCopy(const std::string& id) const;

private:
    std::unordered_map<std::string, std::unique_ptr<AccountData>> accounts_;
    mutable std::shared_mutex mutex_;
};

} // namespace payment_engine
