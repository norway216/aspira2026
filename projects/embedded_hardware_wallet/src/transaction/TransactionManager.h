#pragma once

#include "Common.h"
#include "core/CryptoProvider.h"
#include <vector>
#include <string>
#include <span>
#include <shared_mutex>

namespace ehw {

/**
 * TransactionManager — Transaction construction, signing, serialization, and verification.
 *
 * Supports:
 * - Multi-input UTXO transactions
 * - ECDSA signing over transaction digest
 * - Binary serialization
 * - Parallel input verification via ThreadPool
 */
class TransactionManager {
public:
    TransactionManager();
    ~TransactionManager();

    // ---- Transaction Construction ----
    Result<Transaction> createTransaction(
        const std::vector<TransactionInput>& inputs,
        const std::vector<TransactionOutput>& outputs,
        uint32_t lockTime = 0
    );

    // ---- Signing ----
    // Sign a transaction with a private key (signs all inputs owned by this key)
    Result<Transaction> signTransaction(
        const Transaction& tx,
        std::span<const uint8_t> privateKey,
        std::span<const uint8_t> publicKey
    );

    // Verify a single input's signature
    Result<bool> verifyInput(
        const Transaction& tx,
        size_t inputIndex,
        std::span<const uint8_t> publicKey,
        std::span<const uint8_t> signature
    );

    // ---- Serialization ----
    // Serialize transaction to binary format
    Result<ByteVector> serialize(const Transaction& tx, bool includeWitness = false);

    // Deserialize transaction from binary
    Result<Transaction> deserialize(std::span<const uint8_t> data);

    // Compute transaction hash (double-SHA256 of serialized tx)
    Result<std::array<uint8_t, SHA256_DIGEST_SIZE>> computeTxHash(const Transaction& tx);

    // ---- Fee Calculation ----
    static uint64_t calculateFee(
        const std::vector<TransactionInput>& inputs,
        const std::vector<TransactionOutput>& outputs,
        uint64_t feeRate = 10  // satoshis per byte
    );

    // ---- Transaction History ----
    void addToHistory(const Transaction& tx);
    const std::vector<Transaction>& getHistory() const;
    size_t getHistoryCount() const;

private:
    // Compute the sighash for a single input (simplified SIGHASH_ALL)
    Result<std::array<uint8_t, SHA256_DIGEST_SIZE>> computeSigHash(
        const Transaction& tx,
        size_t inputIndex,
        std::span<const uint8_t> scriptPubKey
    );

    mutable std::shared_mutex m_historyMutex;
    std::vector<Transaction> m_transactionHistory;
};

} // namespace ehw
