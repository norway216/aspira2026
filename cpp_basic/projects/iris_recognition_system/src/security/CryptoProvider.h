#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace iris {

/// Cryptographic operations for template protection.
/// Uses OpenSSL if available, otherwise falls back to simplified AES.
class CryptoProvider {
public:
    CryptoProvider();

    /// Initialize with a device-specific key
    bool initialize(const std::string& deviceKey = "");

    /// AES-256-GCM encrypt
    std::vector<uint8_t> encrypt(const std::vector<uint8_t>& plaintext,
                                  const std::vector<uint8_t>& associatedData = {});

    /// AES-256-GCM decrypt
    std::vector<uint8_t> decrypt(const std::vector<uint8_t>& ciphertext,
                                  const std::vector<uint8_t>& associatedData = {});

    /// Generate a random key
    static std::vector<uint8_t> generateRandomKey(int bytes = 32);

    /// HKDF-SHA256 key derivation
    std::vector<uint8_t> deriveKey(const std::string& salt, int keyLength = 32);

    bool isAvailable() const { return m_available; }

private:
    // Simplified AES-256-GCM using built-in operations
    // (production should use OpenSSL)
    static std::vector<uint8_t> simplifiedEncrypt(const std::vector<uint8_t>& plaintext,
                                                    const std::vector<uint8_t>& key);

    static std::vector<uint8_t> simplifiedDecrypt(const std::vector<uint8_t>& ciphertext,
                                                    const std::vector<uint8_t>& key);

    static std::vector<uint8_t> sha256(const std::vector<uint8_t>& data);

    std::vector<uint8_t> m_masterKey;
    bool m_available = false;
};

} // namespace iris
