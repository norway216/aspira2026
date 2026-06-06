#pragma once

#include <string>
#include <vector>
#include <optional>
#include <cstdint>

namespace payment_engine {

/**
 * AES-256-GCM encryption/decryption for sensitive data.
 *
 * Uses OpenSSL EVP interface.
 * Output format: base64(IV || ciphertext || GCM tag)
 *   IV: 12 bytes (random, generated per encryption)
 *   GCM tag: 16 bytes (appended by OpenSSL)
 */
class AESCipher {
public:
    /**
     * @param key_hex 64-character hex string (32 bytes for AES-256)
     */
    explicit AESCipher(const std::string& key_hex);

    /**
     * Encrypt plaintext.
     * Returns base64-encoded string: base64(IV || ciphertext || tag)
     * Returns std::nullopt on failure.
     */
    std::optional<std::string> Encrypt(const std::string& plaintext);

    /**
     * Decrypt ciphertext_b64 (base64(IV || ciphertext || tag)).
     * Returns plaintext on success, std::nullopt on failure.
     */
    std::optional<std::string> Decrypt(const std::string& ciphertext_b64);

private:
    std::vector<uint8_t> key_; // 32 bytes for AES-256

    static std::vector<uint8_t> HexToBytes(const std::string& hex);
    static std::string Base64Encode(const std::vector<uint8_t>& data);
    static std::vector<uint8_t> Base64Decode(const std::string& b64);
};

} // namespace payment_engine
