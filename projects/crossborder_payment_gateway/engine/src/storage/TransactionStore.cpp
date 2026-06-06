#include "storage/TransactionStore.h"

namespace payment_engine {

void TransactionStore::Append(const TxnRecord& txn) {
    std::unique_lock lock(mutex_);
    transactions_.push_back(txn);
    total_count_.fetch_add(1, std::memory_order_release);
    total_volume_.fetch_add(txn.source_amount, std::memory_order_release);
}

std::optional<TransactionStore::TxnRecord> TransactionStore::Get(const std::string& id) const {
    std::shared_lock lock(mutex_);
    for (const auto& txn : transactions_) {
        if (txn.id == id) {
            return txn;
        }
    }
    return std::nullopt;
}

std::vector<TransactionStore::TxnRecord> TransactionStore::GetRecent(int count) const {
    std::shared_lock lock(mutex_);
    std::vector<TxnRecord> result;
    if (transactions_.empty()) return result;

    size_t start = (transactions_.size() > static_cast<size_t>(count))
                       ? transactions_.size() - count
                       : 0;
    result.reserve(transactions_.size() - start);
    for (size_t i = start; i < transactions_.size(); ++i) {
        result.push_back(transactions_[i]);
    }
    return result;
}

} // namespace payment_engine
