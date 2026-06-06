#include "storage/AccountStore.h"

namespace payment_engine {

AccountStore::AccountStore() {
    SeedDefaultAccounts();
}

bool AccountStore::CreateAccount(const AccountData& acct) {
    std::unique_lock lock(mutex_);
    if (accounts_.find(acct.id) != accounts_.end()) {
        return false; // Already exists
    }
    auto ptr = std::make_unique<AccountData>(acct);
    accounts_[acct.id] = std::move(ptr);
    return true;
}

std::optional<std::reference_wrapper<AccountData>> AccountStore::GetAccount(const std::string& id) {
    std::shared_lock lock(mutex_);
    auto it = accounts_.find(id);
    if (it == accounts_.end()) {
        return std::nullopt;
    }
    return std::ref(*it->second);
}

std::optional<AccountData> AccountStore::GetAccountCopy(const std::string& id) const {
    std::shared_lock lock(mutex_);
    auto it = accounts_.find(id);
    if (it == accounts_.end()) {
        return std::nullopt;
    }
    return *it->second;
}

bool AccountStore::UpdateBalance(const std::string& id, int64_t new_balance, int64_t new_reserved) {
    std::shared_lock lock(mutex_);
    auto it = accounts_.find(id);
    if (it == accounts_.end()) {
        return false;
    }
    it->second->balance.store(new_balance, std::memory_order_release);
    it->second->reserved_balance.store(new_reserved, std::memory_order_release);
    return true;
}

void AccountStore::SeedDefaultAccounts() {
    // Seed 4 default accounts with 1,000,000.00 (100,000,000 cents) each
    auto make = [](const std::string& id, const std::string& merchant,
                   const std::string& currency) {
        AccountData acct;
        acct.id = id;
        acct.merchant_id = merchant;
        acct.currency = currency;
        acct.balance.store(100000000, std::memory_order_relaxed);
        acct.reserved_balance.store(0, std::memory_order_relaxed);
        acct.status = "active";
        acct.daily_limit = 10000000;
        acct.monthly_limit = 100000000;
        return acct;
    };

    CreateAccount(make("merchant1_USD", "merchant1", "USD"));
    CreateAccount(make("merchant1_CNY", "merchant1", "CNY"));
    CreateAccount(make("merchant2_USD", "merchant2", "USD"));
    CreateAccount(make("merchant2_CNY", "merchant2", "CNY"));

    std::cout << "[AccountStore] Seeded 4 default accounts with 100,000,000 cents each" << std::endl;
}

} // namespace payment_engine
