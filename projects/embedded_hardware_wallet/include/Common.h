#pragma once

#include <cstdint>
#include <string>
#include <vector>
#include <array>
#include <format>
#include <chrono>
#include <mutex>
#include <iostream>

namespace ehw {

// ============================================================================
// Constants
// ============================================================================
inline constexpr size_t AES_256_KEY_SIZE = 32;
inline constexpr size_t AES_GCM_IV_SIZE = 12;
inline constexpr size_t AES_GCM_TAG_SIZE = 16;
inline constexpr size_t SHA256_DIGEST_SIZE = 32;
inline constexpr size_t HMAC_SHA512_SIZE = 64;
inline constexpr size_t EC_PRIVKEY_SIZE = 32;
inline constexpr size_t EC_PUBKEY_COMPRESSED_SIZE = 33;
inline constexpr size_t EC_PUBKEY_UNCOMPRESSED_SIZE = 65;
inline constexpr size_t EC_SIGNATURE_MAX_SIZE = 72;
inline constexpr size_t BIP39_ENTROPY_128 = 16;   // 12 words
inline constexpr size_t BIP39_ENTROPY_256 = 32;   // 24 words
inline constexpr size_t BIP39_SEED_SIZE = 64;
inline constexpr size_t BIP39_WORD_COUNT = 2048;
inline constexpr size_t BIP39_PBKDF2_ITERATIONS = 2048;
inline constexpr size_t SHAMIR_MAX_SHARES = 255;
inline constexpr size_t TRANSACTION_MAX_INPUTS = 256;
inline constexpr size_t TRANSACTION_MAX_OUTPUTS = 256;

// ============================================================================
// Error codes
// ============================================================================
enum class WalletError : int32_t {
    Success = 0,
    // Crypto errors
    CryptoInitFailed = 100,
    KeyGenFailed = 101,
    SignFailed = 102,
    VerifyFailed = 103,
    EncryptFailed = 104,
    DecryptFailed = 105,
    RandomFailed = 106,
    // Wallet errors
    WalletNotInitialized = 200,
    InvalidMnemonic = 201,
    InvalidPassphrase = 202,
    InvalidKey = 203,
    InvalidAddress = 204,
    KeyStorageFailed = 205,
    // Transaction errors
    InvalidTransaction = 300,
    InsufficientFunds = 301,
    InvalidSignature = 302,
    SerializationFailed = 303,
    // Backup errors
    BackupFailed = 400,
    RecoveryFailed = 401,
    InvalidShare = 402,
    InsufficientShares = 403,
    ShareCountMismatch = 404,
    // Network errors
    NetworkError = 500,
    ConnectionFailed = 501,
    Timeout = 502,
    TLSHandshakeFailed = 503,
    // Security errors
    SecurityViolation = 600,
    MemoryLockFailed = 601,
    IntegrityCheckFailed = 602,
    // General
    UnknownError = 999
};

inline const char* walletErrorStr(WalletError e) {
    switch (e) {
        case WalletError::Success: return "Success";
        case WalletError::CryptoInitFailed: return "Crypto initialization failed";
        case WalletError::KeyGenFailed: return "Key generation failed";
        case WalletError::SignFailed: return "Signature failed";
        case WalletError::VerifyFailed: return "Verification failed";
        case WalletError::EncryptFailed: return "Encryption failed";
        case WalletError::DecryptFailed: return "Decryption failed";
        case WalletError::RandomFailed: return "Random number generation failed";
        case WalletError::WalletNotInitialized: return "Wallet not initialized";
        case WalletError::InvalidMnemonic: return "Invalid mnemonic phrase";
        case WalletError::InvalidPassphrase: return "Invalid passphrase";
        case WalletError::InvalidKey: return "Invalid key";
        case WalletError::InvalidAddress: return "Invalid address";
        case WalletError::KeyStorageFailed: return "Key storage failed";
        case WalletError::InvalidTransaction: return "Invalid transaction";
        case WalletError::InsufficientFunds: return "Insufficient funds";
        case WalletError::InvalidSignature: return "Invalid signature";
        case WalletError::SerializationFailed: return "Serialization failed";
        case WalletError::BackupFailed: return "Backup failed";
        case WalletError::RecoveryFailed: return "Recovery failed";
        case WalletError::InvalidShare: return "Invalid share";
        case WalletError::InsufficientShares: return "Insufficient shares";
        case WalletError::ShareCountMismatch: return "Share count mismatch";
        case WalletError::NetworkError: return "Network error";
        case WalletError::ConnectionFailed: return "Connection failed";
        case WalletError::Timeout: return "Operation timed out";
        case WalletError::TLSHandshakeFailed: return "TLS handshake failed";
        case WalletError::SecurityViolation: return "Security violation";
        case WalletError::MemoryLockFailed: return "Memory lock failed";
        case WalletError::IntegrityCheckFailed: return "Integrity check failed";
        default: return "Unknown error";
    }
}

// ============================================================================
// Result type
// ============================================================================
template <typename T>
struct Result {
    T value{};
    WalletError error = WalletError::Success;

    bool ok() const { return error == WalletError::Success; }
    explicit operator bool() const { return ok(); }
};

template <>
struct Result<void> {
    WalletError error = WalletError::Success;
    bool ok() const { return error == WalletError::Success; }
    explicit operator bool() const { return ok(); }
};

// ============================================================================
// Core data types
// ============================================================================
using ByteVector = std::vector<uint8_t>;
using SecureByteVector = ByteVector;  // Tagged alias for secure memory

struct KeyMetadata {
    std::string label;
    std::string address;
    std::chrono::system_clock::time_point created;
    bool isEncrypted = true;
    uint32_t derivationPath[5] = {0, 0, 0, 0, 0};
    uint8_t depth = 0;
};

struct TransactionInput {
    ByteVector txHash;       // 32 bytes - previous tx hash
    uint32_t outputIndex = 0;
    ByteVector scriptSig;    // Unlocking script
    uint64_t sequence = 0xFFFFFFFF;
    ByteVector pubKey;       // For verification
};

struct TransactionOutput {
    uint64_t amount = 0;     // In satoshis
    ByteVector scriptPubKey; // Locking script
    std::string address;
};

struct Transaction {
    uint32_t version = 1;
    std::vector<TransactionInput> inputs;
    std::vector<TransactionOutput> outputs;
    uint32_t lockTime = 0;
    uint64_t fee = 0;

    // Computed
    ByteVector txHash;       // 32 bytes
    ByteVector signedTx;
    bool isSigned = false;
};

struct BackupShare {
    uint8_t index = 0;
    ByteVector data;         // Encrypted share data
    ByteVector hmac;         // Integrity check (HMAC-SHA256)
};

struct WalletState {
    bool initialized = false;
    bool locked = true;
    std::string address;
    uint64_t balance = 0;
    size_t transactionCount = 0;
    int backupShareCount = 0;
    int backupThreshold = 0;
    bool networkConnected = false;
};

// ============================================================================
// Logger utility
// ============================================================================
enum class LogLevel : uint8_t {
    Debug = 0,
    Info = 1,
    Warning = 2,
    Error = 3,
    None = 4
};

inline LogLevel g_minLogLevel = LogLevel::Info;
inline std::mutex g_logMutex;

template <typename... Args>
void log(LogLevel level, std::string_view fmt, Args&&... args) {
    if (level < g_minLogLevel) return;
    std::lock_guard lock(g_logMutex);
    const char* prefix = "";
    switch (level) {
        case LogLevel::Debug: prefix = "[DBG] "; break;
        case LogLevel::Info:  prefix = "[INF] "; break;
        case LogLevel::Warning: prefix = "[WRN] "; break;
        case LogLevel::Error: prefix = "[ERR] "; break;
        default: break;
    }
    try {
        std::cout << prefix << std::vformat(fmt, std::make_format_args(args...)) << std::endl;
    } catch (const std::format_error&) {
        std::cout << prefix << fmt << std::endl;
    }
}

// ============================================================================
// Utility functions
// ============================================================================
inline ByteVector hexToBytes(std::string_view hex) {
    ByteVector bytes;
    bytes.reserve(hex.size() / 2);
    for (size_t i = 0; i + 1 < hex.size(); i += 2) {
        auto nibble = [](char c) -> uint8_t {
            if (c >= '0' && c <= '9') return c - '0';
            if (c >= 'a' && c <= 'f') return c - 'a' + 10;
            if (c >= 'A' && c <= 'F') return c - 'A' + 10;
            return 0;
        };
        bytes.push_back((nibble(hex[i]) << 4) | nibble(hex[i + 1]));
    }
    return bytes;
}

inline std::string bytesToHex(const uint8_t* data, size_t len) {
    static const char hexChars[] = "0123456789abcdef";
    std::string result;
    result.reserve(len * 2);
    for (size_t i = 0; i < len; ++i) {
        result.push_back(hexChars[data[i] >> 4]);
        result.push_back(hexChars[data[i] & 0x0F]);
    }
    return result;
}

inline std::string bytesToHex(const ByteVector& data) {
    return bytesToHex(data.data(), data.size());
}

// Constant-time comparison (timing attack mitigation)
inline bool constantTimeEquals(const uint8_t* a, const uint8_t* b, size_t len) {
    uint8_t diff = 0;
    for (size_t i = 0; i < len; ++i) {
        diff |= a[i] ^ b[i];
    }
    return diff == 0;
}

} // namespace ehw
