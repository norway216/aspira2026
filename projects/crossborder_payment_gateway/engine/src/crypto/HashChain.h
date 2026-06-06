#pragma once

#include <string>
#include <mutex>
#include <vector>
#include <utility>
#include "engine/Common.h"

namespace payment_engine {

/**
 * SHA-256 Hash Chain for audit trail integrity.
 *
 * Each transaction is chained to the previous one via:
 *   hash = SHA256(prev_hash + "|" + txn_id + "|" + src_amount + "|"
 *                  + target_currency + "|" + target_amount + "|" + timestamp)
 *
 * Checkpoints are saved every N transactions for faster verification.
 */
class HashChain {
public:
    explicit HashChain(int checkpoint_interval = 1000);

    /**
     * Append a transaction to the hash chain.
     * Returns the new hash.
     */
    std::string Append(const std::string& transaction_id, int64_t source_amount,
                      const std::string& target_currency, int64_t target_amount,
                      int64_t timestamp_ns);

    /**
     * Verify a hash against its inputs.
     */
    bool Verify(const std::string& hash, const std::string& prev_hash,
                const std::string& txn_id, int64_t src_amt,
                const std::string& tgt_cur, int64_t tgt_amt, int64_t ts);

    std::string GetCurrentHash() const;
    std::vector<std::pair<std::string, int64_t>> GetCheckpoints() const;

    /** Force a checkpoint save. */
    void SaveCheckpoint();

private:
    std::string current_hash_;
    mutable std::mutex mutex_;
    int64_t txn_since_checkpoint_ = 0;
    int checkpoint_interval_ = 1000;
    std::vector<std::pair<std::string, int64_t>> checkpoints_;

    static std::string SHA256Hex(const std::string& input);
};

} // namespace payment_engine
