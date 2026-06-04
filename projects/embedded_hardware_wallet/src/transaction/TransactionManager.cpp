#include "TransactionManager.h"
#include <chrono>

namespace ehw {

TransactionManager::TransactionManager() = default;
TransactionManager::~TransactionManager() = default;

// ---- Transaction Creation ----
Result<Transaction> TransactionManager::createTransaction(
    const std::vector<TransactionInput>& inputs,
    const std::vector<TransactionOutput>& outputs,
    uint32_t lockTime)
{
    if (inputs.empty() || inputs.size() > TRANSACTION_MAX_INPUTS) {
        return {.error = WalletError::InvalidTransaction};
    }
    if (outputs.empty() || outputs.size() > TRANSACTION_MAX_OUTPUTS) {
        return {.error = WalletError::InvalidTransaction};
    }

    Transaction tx;
    tx.version = 1;
    tx.inputs = inputs;
    tx.outputs = outputs;
    tx.lockTime = lockTime;

    // Serialize and compute hash
    auto serialized = serialize(tx);
    if (!serialized.ok()) return {.error = serialized.error};

    auto txHash = crypto().doubleSha256(std::span<const uint8_t>(serialized.value));
    if (!txHash.ok()) return {.error = txHash.error};

    tx.txHash = ByteVector(txHash.value.begin(), txHash.value.end());

    return {.value = std::move(tx)};
}

// ---- SIG Hash Computation ----
Result<std::array<uint8_t, SHA256_DIGEST_SIZE>> TransactionManager::computeSigHash(
    const Transaction& tx,
    size_t inputIndex,
    std::span<const uint8_t> scriptPubKey)
{
    if (inputIndex >= tx.inputs.size()) {
        return {.error = WalletError::InvalidTransaction};
    }

    // Simplified SIGHASH_ALL: serialize tx with empty scripts except for the
    // input being signed (which gets the scriptPubKey).
    // Real implementations would follow BIP143 for segwit.

    Transaction copy = tx;
    for (auto& input : copy.inputs) {
        input.scriptSig.clear();
    }
    copy.inputs[inputIndex].scriptSig = ByteVector(scriptPubKey.begin(), scriptPubKey.end());

    auto serialized = serialize(copy, false);
    if (!serialized.ok()) return {.error = serialized.error};

    // Append SIGHASH_ALL (0x01) as 4-byte little-endian
    serialized.value.push_back(0x01);
    serialized.value.push_back(0x00);
    serialized.value.push_back(0x00);
    serialized.value.push_back(0x00);

    return crypto().doubleSha256(std::span<const uint8_t>(serialized.value));
}

// ---- Sign Transaction ----
Result<Transaction> TransactionManager::signTransaction(
    const Transaction& tx,
    std::span<const uint8_t> privateKey,
    std::span<const uint8_t> publicKey)
{
    Transaction signedTx = tx;

    for (size_t i = 0; i < tx.inputs.size(); ++i) {
        // Compute signature hash for this input
        auto sigHash = computeSigHash(tx, i, tx.outputs[0].scriptPubKey);
        if (!sigHash.ok()) return {.error = sigHash.error};

        // Sign the hash
        auto sig = crypto().ecdsaSign(
            privateKey,
            std::span<const uint8_t>(sigHash.value.data(), sigHash.value.size())
        );
        if (!sig.ok()) return {.error = sig.error};

        // Append SIGHASH_ALL (0x01)
        sig.value.push_back(0x01);

        // Build scriptSig: <sig> <pubkey>
        ByteVector scriptSig;
        scriptSig.push_back(static_cast<uint8_t>(sig.value.size())); // PUSH sig
        scriptSig.insert(scriptSig.end(), sig.value.begin(), sig.value.end());
        scriptSig.push_back(static_cast<uint8_t>(publicKey.size())); // PUSH pubkey
        scriptSig.insert(scriptSig.end(), publicKey.begin(), publicKey.end());

        signedTx.inputs[i].scriptSig = std::move(scriptSig);
    }

    signedTx.isSigned = true;

    // Re-serialize to get final tx
    auto finalSerialized = serialize(signedTx);
    if (!finalSerialized.ok()) return {.error = finalSerialized.error};
    signedTx.signedTx = std::move(finalSerialized.value);

    // Compute final tx hash
    auto txHash = computeTxHash(signedTx);
    if (!txHash.ok()) return {.error = txHash.error};
    signedTx.txHash = ByteVector(txHash.value.begin(), txHash.value.end());

    return {.value = std::move(signedTx)};
}

// ---- Verify Input Signature ----
Result<bool> TransactionManager::verifyInput(
    const Transaction& tx,
    size_t inputIndex,
    std::span<const uint8_t> publicKey,
    std::span<const uint8_t> signature)
{
    if (inputIndex >= tx.inputs.size()) {
        return {.error = WalletError::InvalidTransaction};
    }

    auto sigHash = computeSigHash(tx, inputIndex, tx.inputs[inputIndex].scriptSig);
    if (!sigHash.ok()) return {.error = sigHash.error};

    // Remove SIGHASH byte from signature before verifying
    ByteVector sig(signature.begin(), signature.end() - 1);

    return crypto().ecdsaVerify(
        publicKey,
        std::span<const uint8_t>(sigHash.value.data(), sigHash.value.size()),
        sig
    );
}

// ---- Serialization (simplified binary format) ----
Result<ByteVector> TransactionManager::serialize(const Transaction& tx, bool includeWitness) {
    (void)includeWitness;
    ByteVector data;

    // Version (4 bytes, little-endian)
    data.push_back(tx.version & 0xFF);
    data.push_back((tx.version >> 8) & 0xFF);
    data.push_back((tx.version >> 16) & 0xFF);
    data.push_back((tx.version >> 24) & 0xFF);

    // Input count (varint — simplified)
    auto writeVarInt = [&data](uint64_t val) {
        if (val < 0xFD) {
            data.push_back(static_cast<uint8_t>(val));
        } else if (val <= 0xFFFF) {
            data.push_back(0xFD);
            data.push_back(val & 0xFF);
            data.push_back((val >> 8) & 0xFF);
        } else {
            data.push_back(0xFF);
            for (int i = 0; i < 8; ++i) {
                data.push_back((val >> (i * 8)) & 0xFF);
            }
        }
    };

    writeVarInt(tx.inputs.size());
    for (const auto& input : tx.inputs) {
        // Previous tx hash (32 bytes)
        size_t hashSize = std::min(input.txHash.size(), size_t(32));
        data.insert(data.end(), input.txHash.begin(), input.txHash.begin() + hashSize);
        while (data.size() % 32 != 0) data.push_back(0); // Pad to 32

        // Output index (4 bytes LE)
        data.push_back(input.outputIndex & 0xFF);
        data.push_back((input.outputIndex >> 8) & 0xFF);
        data.push_back((input.outputIndex >> 16) & 0xFF);
        data.push_back((input.outputIndex >> 24) & 0xFF);

        // ScriptSig (varint length + data)
        writeVarInt(input.scriptSig.size());
        data.insert(data.end(), input.scriptSig.begin(), input.scriptSig.end());

        // Sequence (4 bytes LE)
        data.push_back(input.sequence & 0xFF);
        data.push_back((input.sequence >> 8) & 0xFF);
        data.push_back((input.sequence >> 16) & 0xFF);
        data.push_back((input.sequence >> 24) & 0xFF);
    }

    // Output count
    writeVarInt(tx.outputs.size());
    for (const auto& output : tx.outputs) {
        // Amount (8 bytes LE, satoshis)
        for (int i = 0; i < 8; ++i) {
            data.push_back((output.amount >> (i * 8)) & 0xFF);
        }

        // ScriptPubKey
        writeVarInt(output.scriptPubKey.size());
        data.insert(data.end(), output.scriptPubKey.begin(), output.scriptPubKey.end());
    }

    // Lock time (4 bytes LE)
    data.push_back(tx.lockTime & 0xFF);
    data.push_back((tx.lockTime >> 8) & 0xFF);
    data.push_back((tx.lockTime >> 16) & 0xFF);
    data.push_back((tx.lockTime >> 24) & 0xFF);

    return {.value = std::move(data)};
}

// ---- Deserialize ----
Result<Transaction> TransactionManager::deserialize(std::span<const uint8_t> data) {
    if (data.size() < 10) {
        return {.error = WalletError::SerializationFailed};
    }

    size_t pos = 0;
    auto readU32 = [&](size_t& p) -> uint32_t {
        if (p + 4 > data.size()) return 0;
        uint32_t v = data[p] | (data[p+1] << 8) | (data[p+2] << 16) | (data[p+3] << 24);
        p += 4;
        return v;
    };
    auto readU64 = [&](size_t& p) -> uint64_t {
        if (p + 8 > data.size()) return 0;
        uint64_t v = 0;
        for (int i = 0; i < 8; ++i) v |= static_cast<uint64_t>(data[p+i]) << (i*8);
        p += 8;
        return v;
    };
    auto readVarInt = [&](size_t& p) -> uint64_t {
        if (p >= data.size()) return 0;
        uint8_t first = data[p++];
        if (first < 0xFD) return first;
        if (first == 0xFD) return readU32(p) & 0xFFFF;
        return readU64(p);
    };
    auto readBytes = [&](size_t& p, size_t len) -> ByteVector {
        len = std::min(len, data.size() - p);
        ByteVector v(data.begin() + p, data.begin() + p + len);
        p += len;
        return v;
    };

    Transaction tx;
    tx.version = readU32(pos);

    // Inputs
    size_t inputCount = readVarInt(pos);
    tx.inputs.reserve(inputCount);
    for (size_t i = 0; i < inputCount; ++i) {
        TransactionInput input;
        input.txHash = readBytes(pos, 32);
        input.outputIndex = readU32(pos);
        size_t scriptLen = readVarInt(pos);
        input.scriptSig = readBytes(pos, scriptLen);
        input.sequence = readU32(pos);
        tx.inputs.push_back(std::move(input));
    }

    // Outputs
    size_t outputCount = readVarInt(pos);
    tx.outputs.reserve(outputCount);
    for (size_t i = 0; i < outputCount; ++i) {
        TransactionOutput output;
        output.amount = readU64(pos);
        size_t scriptLen = readVarInt(pos);
        output.scriptPubKey = readBytes(pos, scriptLen);
        tx.outputs.push_back(std::move(output));
    }

    tx.lockTime = readU32(pos);

    return {.value = std::move(tx)};
}

// ---- Compute TxHash ----
Result<std::array<uint8_t, SHA256_DIGEST_SIZE>> TransactionManager::computeTxHash(
    const Transaction& tx)
{
    auto serialized = serialize(tx);
    if (!serialized.ok()) return {.error = serialized.error};
    return crypto().doubleSha256(std::span<const uint8_t>(serialized.value));
}

// ---- Fee Calculation ----
uint64_t TransactionManager::calculateFee(
    const std::vector<TransactionInput>& inputs,
    const std::vector<TransactionOutput>& outputs,
    uint64_t feeRate)
{
    // Rough estimate: ~148 bytes per input, ~34 bytes per output, +10 bytes overhead
    size_t estimatedSize = inputs.size() * 148 + outputs.size() * 34 + 10;
    return estimatedSize * feeRate;
}

// ---- Transaction History ----
void TransactionManager::addToHistory(const Transaction& tx) {
    std::unique_lock lock(m_historyMutex);
    m_transactionHistory.push_back(tx);
}

const std::vector<Transaction>& TransactionManager::getHistory() const {
    std::shared_lock lock(m_historyMutex);
    return m_transactionHistory;
}

size_t TransactionManager::getHistoryCount() const {
    std::shared_lock lock(m_historyMutex);
    return m_transactionHistory.size();
}

} // namespace ehw
