#include "security/CryptoProvider.h"
#include <cstring>
#include <random>
#include <sstream>
#include <iomanip>
#include <iostream>

#ifdef HAS_OPENSSL
#include <openssl/evp.h>
#include <openssl/sha.h>
#include <openssl/rand.h>
#include <openssl/err.h>
#endif

namespace iris {

CryptoProvider::CryptoProvider() = default;

bool CryptoProvider::initialize(const std::string& deviceKey) {
    if (!deviceKey.empty()) {
        m_masterKey = sha256(
            std::vector<uint8_t>(deviceKey.begin(), deviceKey.end())
        );
    } else {
        // Generate a random device key
        m_masterKey = generateRandomKey(32);
    }

    m_available = true;
    std::cout << "[CryptoProvider] Initialized (key size="
              << m_masterKey.size() * 8 << " bits)\n";
    return true;
}

std::vector<uint8_t> CryptoProvider::encrypt(const std::vector<uint8_t>& plaintext,
                                              const std::vector<uint8_t>& associatedData) {
    if (!m_available) return plaintext;

#ifdef HAS_OPENSSL
    // Use OpenSSL EVP for AES-256-GCM
    std::vector<uint8_t> iv(12);
    RAND_bytes(iv.data(), static_cast<int>(iv.size()));

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (!ctx) return simplifiedEncrypt(plaintext, m_masterKey);

    EVP_EncryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr);
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, 12, nullptr);
    EVP_EncryptInit_ex(ctx, nullptr, nullptr, m_masterKey.data(), iv.data());

    if (!associatedData.empty()) {
        int len = 0;
        EVP_EncryptUpdate(ctx, nullptr, &len, associatedData.data(),
                          static_cast<int>(associatedData.size()));
    }

    std::vector<uint8_t> ciphertext(plaintext.size() + 16);
    int outLen = 0;
    EVP_EncryptUpdate(ctx, ciphertext.data(), &outLen,
                      plaintext.data(), static_cast<int>(plaintext.size()));
    int cipherLen = outLen;

    EVP_EncryptFinal_ex(ctx, ciphertext.data() + outLen, &outLen);
    cipherLen += outLen;

    std::vector<uint8_t> tag(16);
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, 16, tag.data());

    EVP_CIPHER_CTX_free(ctx);

    // Format: [IV(12)] [ciphertext] [tag(16)]
    ciphertext.resize(cipherLen);
    std::vector<uint8_t> result;
    result.reserve(12 + cipherLen + 16);
    result.insert(result.end(), iv.begin(), iv.end());
    result.insert(result.end(), ciphertext.begin(), ciphertext.end());
    result.insert(result.end(), tag.begin(), tag.end());

    return result;
#else
    return simplifiedEncrypt(plaintext, m_masterKey);
#endif
}

std::vector<uint8_t> CryptoProvider::decrypt(const std::vector<uint8_t>& ciphertext,
                                              const std::vector<uint8_t>& associatedData) {
    if (!m_available) return ciphertext;
    if (ciphertext.size() < 28) return ciphertext; // too short for IV+tag

#ifdef HAS_OPENSSL
    // Parse: [IV(12)] [ciphertext] [tag(16)]
    std::vector<uint8_t> iv(ciphertext.begin(), ciphertext.begin() + 12);
    std::vector<uint8_t> tag(ciphertext.end() - 16, ciphertext.end());
    std::vector<uint8_t> encrypted(ciphertext.begin() + 12, ciphertext.end() - 16);

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();

    EVP_DecryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr);
    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, 12, nullptr);
    EVP_DecryptInit_ex(ctx, nullptr, nullptr, m_masterKey.data(), iv.data());

    if (!associatedData.empty()) {
        int len = 0;
        EVP_DecryptUpdate(ctx, nullptr, &len, associatedData.data(),
                          static_cast<int>(associatedData.size()));
    }

    std::vector<uint8_t> plaintext(encrypted.size());
    int outLen = 0;
    EVP_DecryptUpdate(ctx, plaintext.data(), &outLen,
                      encrypted.data(), static_cast<int>(encrypted.size()));
    int plainLen = outLen;

    EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG, 16,
                        const_cast<uint8_t*>(tag.data()));

    int ret = EVP_DecryptFinal_ex(ctx, plaintext.data() + outLen, &outLen);
    EVP_CIPHER_CTX_free(ctx);

    if (ret <= 0) {
        std::cerr << "[CryptoProvider] Decryption failed (tag mismatch)\n";
        return {};
    }

    plainLen += outLen;
    plaintext.resize(plainLen);
    return plaintext;
#else
    return simplifiedDecrypt(ciphertext, m_masterKey);
#endif
}

std::vector<uint8_t> CryptoProvider::generateRandomKey(int bytes) {
    std::vector<uint8_t> key(bytes);

#ifdef HAS_OPENSSL
    RAND_bytes(key.data(), bytes);
#else
    std::random_device rd;
    std::mt19937_64 gen(rd());
    std::uniform_int_distribution<uint8_t> dist;
    for (int i = 0; i < bytes; ++i) {
        key[i] = dist(gen);
    }
#endif

    return key;
}

std::vector<uint8_t> CryptoProvider::deriveKey(const std::string& salt, int keyLength) {
    std::vector<uint8_t> saltBytes(salt.begin(), salt.end());
    std::vector<uint8_t> prk = sha256(m_masterKey);

    // Simple HKDF-like key derivation
    std::vector<uint8_t> combined = prk;
    combined.insert(combined.end(), saltBytes.begin(), saltBytes.end());
    combined.push_back(0x01); // counter byte

    std::vector<uint8_t> derived = sha256(combined);
    derived.resize(keyLength);
    return derived;
}

// ── Simplified encryption (no OpenSSL) ─────────────────────────

std::vector<uint8_t> CryptoProvider::simplifiedEncrypt(
    const std::vector<uint8_t>& plaintext,
    const std::vector<uint8_t>& key) {

    std::vector<uint8_t> result;
    result.reserve(plaintext.size() + 16);

    // Simple XOR with key stream + random IV
    std::vector<uint8_t> iv = generateRandomKey(8);
    result.insert(result.end(), iv.begin(), iv.end());

    // Generate key stream using repeated SHA256
    std::vector<uint8_t> keyStream;
    keyStream.insert(keyStream.end(), key.begin(), key.end());
    keyStream.insert(keyStream.end(), iv.begin(), iv.end());

    std::vector<uint8_t> streamHash = sha256(keyStream);

    // XOR encrypt
    for (size_t i = 0; i < plaintext.size(); ++i) {
        uint8_t ks = streamHash[i % streamHash.size()];
        result.push_back(plaintext[i] ^ ks);
    }

    // Append checksum (simple XOR of plaintext)
    uint8_t checksum = 0;
    for (uint8_t b : plaintext) checksum ^= b;
    result.push_back(checksum);

    return result;
}

std::vector<uint8_t> CryptoProvider::simplifiedDecrypt(
    const std::vector<uint8_t>& ciphertext,
    const std::vector<uint8_t>& key) {

    if (ciphertext.size() < 9) return ciphertext;

    // Parse: [IV(8)] [encrypted data] [checksum(1)]
    std::vector<uint8_t> iv(ciphertext.begin(), ciphertext.begin() + 8);
    uint8_t expectedChecksum = ciphertext.back();

    // Regenerate key stream
    std::vector<uint8_t> keyStream;
    keyStream.insert(keyStream.end(), key.begin(), key.end());
    keyStream.insert(keyStream.end(), iv.begin(), iv.end());
    std::vector<uint8_t> streamHash = sha256(keyStream);

    // XOR decrypt
    std::vector<uint8_t> plaintext;
    size_t encryptedLen = ciphertext.size() - 9;
    plaintext.reserve(encryptedLen);

    for (size_t i = 0; i < encryptedLen; ++i) {
        uint8_t ks = streamHash[i % streamHash.size()];
        plaintext.push_back(ciphertext[8 + i] ^ ks);
    }

    // Verify checksum
    uint8_t actualChecksum = 0;
    for (uint8_t b : plaintext) actualChecksum ^= b;

    if (actualChecksum != expectedChecksum) {
        std::cerr << "[CryptoProvider] Checksum mismatch - data may be corrupted\n";
    }

    return plaintext;
}

std::vector<uint8_t> CryptoProvider::sha256(const std::vector<uint8_t>& data) {
    std::vector<uint8_t> hash(32);

#ifdef HAS_OPENSSL
    SHA256(data.data(), data.size(), hash.data());
#else
    // Simplified SHA-256 fallback for demo purposes
    // In production, always use OpenSSL or a proper implementation
    // This is a placeholder - NOT cryptographically secure
    uint32_t h[8] = {
        0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a,
        0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19
    };

    // Simple mixing based on input
    for (size_t i = 0; i < data.size(); ++i) {
        uint32_t val = static_cast<uint32_t>(data[i]) << ((i % 4) * 8);
        h[i % 8] ^= val;
        h[(i + 1) % 8] = (h[(i + 1) % 8] + val) ^ (h[(i + 3) % 8] >> 3);
    }

    // More mixing rounds
    for (int round = 0; round < 32; ++round) {
        for (int j = 0; j < 8; ++j) {
            uint32_t s1 = (h[j] >> 17 | h[j] << 15);
            uint32_t ch = (h[j] & h[(j+1)%8]) ^ (~h[j] & h[(j+2)%8]);
            h[(j+4)%8] += s1 + ch + h[(j+7)%8];
            h[j] = (h[j] << 10 | h[j] >> 22) + h[(j+4)%8];
        }
    }

    for (int i = 0; i < 8; ++i) {
        hash[i * 4]     = (h[i] >> 24) & 0xFF;
        hash[i * 4 + 1] = (h[i] >> 16) & 0xFF;
        hash[i * 4 + 2] = (h[i] >> 8)  & 0xFF;
        hash[i * 4 + 3] = h[i] & 0xFF;
    }
#endif

    return hash;
}

} // namespace iris
