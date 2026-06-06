#include "crypto/HashChain.h"
#include <openssl/evp.h>
#include <openssl/sha.h>
#include <sstream>
#include <iomanip>

namespace payment_engine {

HashChain::HashChain(int checkpoint_interval)
    : checkpoint_interval_(checkpoint_interval)
{
    // Initialize with a genesis hash
    current_hash_ = SHA256Hex("genesis|aspira-payment-engine-v1.0|0|||0");
    checkpoints_.emplace_back(current_hash_, 0);
}

std::string HashChain::SHA256Hex(const std::string& input) {
    unsigned char hash[SHA256_DIGEST_LENGTH];
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (!ctx) return std::string(64, '0');

    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    EVP_DigestUpdate(ctx, input.data(), input.size());
    unsigned int len = 0;
    EVP_DigestFinal_ex(ctx, hash, &len);
    EVP_MD_CTX_free(ctx);

    std::ostringstream oss;
    for (unsigned int i = 0; i < len; ++i) {
        oss << std::hex << std::setw(2) << std::setfill('0')
            << static_cast<int>(hash[i]);
    }
    return oss.str();
}

std::string HashChain::Append(const std::string& transaction_id, int64_t source_amount,
                              const std::string& target_currency, int64_t target_amount,
                              int64_t timestamp_ns)
{
    std::lock_guard lock(mutex_);

    std::ostringstream input;
    input << current_hash_ << "|"
          << transaction_id << "|"
          << source_amount << "|"
          << target_currency << "|"
          << target_amount << "|"
          << timestamp_ns;

    std::string prev_hash = current_hash_;
    current_hash_ = SHA256Hex(input.str());

    // Update checkpoint tracking
    txn_since_checkpoint_++;

    return current_hash_;
}

void HashChain::SaveCheckpoint() {
    std::lock_guard lock(mutex_);
    checkpoints_.emplace_back(current_hash_, txn_since_checkpoint_);
    txn_since_checkpoint_ = 0;
}

bool HashChain::Verify(const std::string& hash, const std::string& prev_hash,
                       const std::string& txn_id, int64_t src_amt,
                       const std::string& tgt_cur, int64_t tgt_amt, int64_t ts)
{
    std::ostringstream input;
    input << prev_hash << "|"
          << txn_id << "|"
          << src_amt << "|"
          << tgt_cur << "|"
          << tgt_amt << "|"
          << ts;

    std::string expected = SHA256Hex(input.str());
    return hash == expected;
}

std::string HashChain::GetCurrentHash() const {
    std::lock_guard lock(mutex_);
    return current_hash_;
}

std::vector<std::pair<std::string, int64_t>> HashChain::GetCheckpoints() const {
    std::lock_guard lock(mutex_);
    return checkpoints_;
}

} // namespace payment_engine
