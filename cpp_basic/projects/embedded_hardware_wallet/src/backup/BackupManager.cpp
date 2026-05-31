#include "BackupManager.h"
#include <algorithm>
#include <sstream>
#include <chrono>
#include <iomanip>
#include <cstring>

namespace ehw {

BackupManager::BackupManager() = default;
BackupManager::~BackupManager() = default;

// ---- GF(256) Arithmetic ----
// Using AES Rijndael's finite field: x^8 + x^4 + x^3 + x + 1

uint8_t BackupManager::gf256Add(uint8_t a, uint8_t b) {
    return a ^ b; // Addition is XOR in GF(256)
}

uint8_t BackupManager::gf256Mul(uint8_t a, uint8_t b) {
    uint8_t p = 0;
    for (int i = 0; i < 8; ++i) {
        if (b & 1) p ^= a;
        bool hiBit = (a & 0x80) != 0;
        a <<= 1;
        if (hiBit) a ^= 0x1B; // Reduction polynomial
        b >>= 1;
    }
    return p;
}

uint8_t BackupManager::gf256Div(uint8_t a, uint8_t b) {
    if (b == 0) return 0;
    // Division in GF(256): multiply by inverse
    // Compute inverse via extended Euclidean algorithm (brute-force for small field)
    for (uint16_t i = 1; i < 256; ++i) {
        if (gf256Mul(b, static_cast<uint8_t>(i)) == a) {
            return static_cast<uint8_t>(i);
        }
    }
    return 0;
}

// ---- Lagrange Interpolation at x=0 ----
uint8_t BackupManager::lagrangeInterpolate(
    const std::vector<std::pair<uint8_t, uint8_t>>& points,
    uint8_t x)
{
    uint8_t result = 0;
    for (size_t i = 0; i < points.size(); ++i) {
        uint8_t term = points[i].second; // y_i
        for (size_t j = 0; j < points.size(); ++j) {
            if (i == j) continue;
            // term *= (x - x_j) / (x_i - x_j)
            uint8_t numerator = gf256Add(x, points[j].first);   // x - x_j
            uint8_t denominator = gf256Add(points[i].first, points[j].first); // x_i - x_j
            term = gf256Mul(term, gf256Div(numerator, denominator));
        }
        result = gf256Add(result, term);
    }
    return result;
}

// ---- Shamir's Secret Sharing: Split ----
Result<std::vector<BackupShare>> BackupManager::splitSecret(
    std::span<const uint8_t> secret,
    uint8_t totalShares,
    uint8_t threshold,
    std::span<const uint8_t> encryptionKey)
{
    if (totalShares < 2 || totalShares > SHAMIR_MAX_SHARES ||
        threshold < 2 || threshold > totalShares) {
        return {.error = WalletError::BackupFailed};
    }

    // For each byte of the secret, create a polynomial of degree (threshold-1)
    // f(x) = secret + a_1*x + a_2*x^2 + ... + a_{k-1}*x^{k-1}
    // Each share gets f(share_index) for each byte

    std::vector<ByteVector> shareData(totalShares);
    for (auto& sd : shareData) {
        sd.resize(secret.size());
    }

    // Generate random coefficients for each byte position
    ByteVector coefficients(threshold - 1);

    for (size_t byteIdx = 0; byteIdx < secret.size(); ++byteIdx) {
        // Generate random coefficients
        for (uint8_t c = 0; c < threshold - 1; ++c) {
            if (auto r = crypto().randomBytes(&coefficients[c], 1); !r.ok()) {
                return {.error = r.error};
            }
            if (coefficients[c] == 0) coefficients[c] = 1; // Avoid degenerate polynomials
        }

        // Evaluate polynomial at each share index (1, 2, ..., totalShares)
        for (uint8_t shareIdx = 1; shareIdx <= totalShares; ++shareIdx) {
            uint8_t result = secret[byteIdx];
            uint8_t xPower = shareIdx;

            for (uint8_t c = 0; c < threshold - 1; ++c) {
                // result += coeff[c] * x^(c+1)
                result = gf256Add(result, gf256Mul(coefficients[c], xPower));
                xPower = gf256Mul(xPower, shareIdx); // x^{c+1}
            }

            shareData[shareIdx - 1][byteIdx] = result;
        }
    }

    // Encrypt each share and add HMAC
    std::vector<BackupShare> shares;
    shares.reserve(totalShares);

    for (uint8_t i = 0; i < totalShares; ++i) {
        // Encrypt share data
        auto encrypted = crypto().aesGcmEncrypt(encryptionKey, shareData[i]);
        if (!encrypted.ok()) return {.error = WalletError::EncryptFailed};

        // Compute HMAC over encrypted data
        auto hmacKey = crypto().sha256(
            std::span<const uint8_t>(encryptionKey.data(), encryptionKey.size()));
        if (!hmacKey.ok()) return {.error = hmacKey.error};

        // HMAC-SHA256 (truncated to first 16 bytes for compactness)
        auto hmac = crypto().hmacSha512(
            std::span<const uint8_t>(hmacKey.value.data(), SHA256_DIGEST_SIZE),
            encrypted.value
        );
        if (!hmac.ok()) return {.error = hmac.error};

        BackupShare share;
        share.index = i + 1;
        share.data = std::move(encrypted.value);
        share.hmac = ByteVector(hmac.value.begin(), hmac.value.begin() + 16);
        shares.push_back(std::move(share));
    }

    return {.value = std::move(shares)};
}

// ---- Shamir's Secret Sharing: Recover ----
Result<SecureByteVector> BackupManager::recoverSecret(
    const std::vector<BackupShare>& shares,
    std::span<const uint8_t> encryptionKey)
{
    if (shares.empty()) {
        return {.error = WalletError::InsufficientShares};
    }

    // Decrypt each share
    std::vector<std::pair<uint8_t, ByteVector>> decryptedShares;

    for (const auto& share : shares) {
        // Verify HMAC
        auto hmacKey = crypto().sha256(
            std::span<const uint8_t>(encryptionKey.data(), encryptionKey.size()));
        if (!hmacKey.ok()) return {.error = hmacKey.error};

        auto computedHmac = crypto().hmacSha512(
            std::span<const uint8_t>(hmacKey.value.data(), SHA256_DIGEST_SIZE),
            share.data
        );
        if (!computedHmac.ok()) return {.error = computedHmac.error};

        // Compare first 16 bytes
        if (!constantTimeEquals(share.hmac.data(),
                                computedHmac.value.data(),
                                std::min(share.hmac.size(), size_t(16)))) {
            return {.error = WalletError::InvalidShare};
        }

        // Decrypt
        auto decrypted = crypto().aesGcmDecrypt(encryptionKey, share.data);
        if (!decrypted.ok()) return {.error = WalletError::DecryptFailed};

        decryptedShares.emplace_back(share.index, std::move(decrypted.value));
    }

    if (decryptedShares.empty()) {
        return {.error = WalletError::InsufficientShares};
    }

    // Determine secret length from first decrypted share
    size_t secretLen = decryptedShares[0].second.size();

    // Recover each byte using Lagrange interpolation at x=0
    SecureByteVector recovered(secretLen);
    for (size_t byteIdx = 0; byteIdx < secretLen; ++byteIdx) {
        std::vector<std::pair<uint8_t, uint8_t>> points;
        for (const auto& [idx, data] : decryptedShares) {
            points.emplace_back(idx, data[byteIdx]);
        }
        recovered[byteIdx] = lagrangeInterpolate(points, 0);
    }

    return {.value = std::move(recovered)};
}

// ---- Share Serialization ----
std::string BackupManager::shareToString(const BackupShare& share) {
    // Format: index (1 byte) || data_len (2 bytes BE) || data || hmac
    ByteVector bin;
    bin.push_back(share.index);
    bin.push_back((share.data.size() >> 8) & 0xFF);
    bin.push_back(share.data.size() & 0xFF);
    bin.insert(bin.end(), share.data.begin(), share.data.end());
    bin.insert(bin.end(), share.hmac.begin(), share.hmac.end());

    // Base64 encode
    static const char* b64chars =
        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    std::string result;
    result.reserve(((bin.size() + 2) / 3) * 4);

    for (size_t i = 0; i < bin.size(); i += 3) {
        uint32_t val = static_cast<uint32_t>(bin[i]) << 16;
        if (i + 1 < bin.size()) val |= static_cast<uint32_t>(bin[i + 1]) << 8;
        if (i + 2 < bin.size()) val |= static_cast<uint32_t>(bin[i + 2]);

        result.push_back(b64chars[(val >> 18) & 0x3F]);
        result.push_back(b64chars[(val >> 12) & 0x3F]);
        result.push_back((i + 1 < bin.size()) ? b64chars[(val >> 6) & 0x3F] : '=');
        result.push_back((i + 2 < bin.size()) ? b64chars[val & 0x3F] : '=');
    }
    return result;
}

Result<BackupShare> BackupManager::shareFromString(std::string_view encoded) {
    // Base64 decode lookup table
    static const std::array<int, 256> b64table = []() {
        std::array<int, 256> t{};
        t.fill(-1);
        const char* chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
        for (int i = 0; i < 64; ++i) t[static_cast<uint8_t>(chars[i])] = i;
        return t;
    }();

    ByteVector bin;
    bin.reserve((encoded.size() * 3) / 4);

    uint32_t val = 0;
    int bits = 0;
    for (char c : encoded) {
        if (c == '=' || c == '\n' || c == '\r') break;
        int v = b64table[static_cast<uint8_t>(c)];
        if (v < 0) continue;
        val = (val << 6) | static_cast<uint32_t>(v);
        bits += 6;
        if (bits >= 8) {
            bits -= 8;
            bin.push_back((val >> bits) & 0xFF);
        }
    }

    if (bin.size() < 3) {
        return {.error = WalletError::InvalidShare};
    }

    // Parse: index || data_len || data || hmac
    BackupShare share;
    share.index = bin[0];
    size_t dataLen = (static_cast<size_t>(bin[1]) << 8) | bin[2];

    if (bin.size() < 3 + dataLen) {
        return {.error = WalletError::InvalidShare};
    }

    share.data = ByteVector(bin.begin() + 3, bin.begin() + 3 + dataLen);
    size_t hmacStart = 3 + dataLen;
    share.hmac = ByteVector(bin.begin() + hmacStart, bin.end());

    return {.value = std::move(share)};
}

// ---- Derive Share Encryption Key ----
Result<ByteVector> BackupManager::deriveShareKey(std::string_view passphrase) {
    ByteVector salt(16);
    if (auto r = crypto().randomBytes(salt.data(), salt.size()); !r.ok()) {
        return {.error = r.error};
    }
    return crypto().pbkdf2HmacSha512(
        std::span<const uint8_t>(
            reinterpret_cast<const uint8_t*>(passphrase.data()), passphrase.size()),
        std::span<const uint8_t>(salt.data(), salt.size()),
        100000,
        AES_256_KEY_SIZE
    );
}

// ---- Create Backup ----
Result<std::vector<std::string>> BackupManager::createBackup(
    std::span<const uint8_t> seed,
    std::string_view passphrase,
    uint8_t totalShares,
    uint8_t threshold)
{
    // Derive encryption key from passphrase
    auto encKey = deriveShareKey(passphrase);
    if (!encKey.ok()) return {.error = encKey.error};

    // Split seed using SSS
    auto shares = splitSecret(seed, totalShares, threshold, encKey.value);
    if (!shares.ok()) return {.error = shares.error};

    // Encode shares to strings
    std::vector<std::string> encoded;
    encoded.reserve(shares.value.size());
    for (const auto& share : shares.value) {
        encoded.push_back(shareToString(share));
    }

    // Record backup version
    auto now = std::chrono::system_clock::now();
    auto time = std::chrono::system_clock::to_time_t(now);
    std::ostringstream timeStr;
    timeStr << std::put_time(std::gmtime(&time), "%Y-%m-%dT%H:%M:%SZ");

    BackupVersion version{
        .version = static_cast<uint32_t>(m_backupVersions.size() + 1),
        .timestamp = timeStr.str(),
        .shares = encoded,
        .totalShares = totalShares,
        .threshold = threshold
    };
    addBackupVersion(version);

    return {.value = std::move(encoded)};
}

// ---- Recover from Backup ----
Result<SecureByteVector> BackupManager::recoverFromBackup(
    const std::vector<std::string>& encodedShares,
    std::string_view passphrase,
    uint8_t threshold)
{
    if (encodedShares.size() < threshold) {
        return {.error = WalletError::InsufficientShares};
    }

    // Decode shares
    std::vector<BackupShare> shares;
    shares.reserve(encodedShares.size());
    for (const auto& encoded : encodedShares) {
        auto share = shareFromString(encoded);
        if (!share.ok()) return {.error = share.error};
        shares.push_back(std::move(share.value));
    }

    // Derive encryption key
    auto encKey = deriveShareKey(passphrase);
    if (!encKey.ok()) return {.error = encKey.error};

    // Recover secret
    return recoverSecret(shares, encKey.value);
}

// ---- Backup Version History ----
void BackupManager::addBackupVersion(const BackupVersion& version) {
    std::unique_lock lock(m_versionMutex);
    m_backupVersions.push_back(version);
}

const std::vector<BackupManager::BackupVersion>& BackupManager::getBackupHistory() const {
    std::shared_lock lock(m_versionMutex);
    return m_backupVersions;
}

} // namespace ehw
