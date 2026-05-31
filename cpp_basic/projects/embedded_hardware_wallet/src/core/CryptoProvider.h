#pragma once

#include "Common.h"
#include "SecureMemory.h"
#include <openssl/evp.h>
#include <openssl/ec.h>
#include <openssl/bn.h>
#include <openssl/rand.h>
#include <openssl/hmac.h>
#include <openssl/err.h>
#include <openssl/obj_mac.h>
#include <openssl/core_names.h>
#include <openssl/param_build.h>
#include <memory>
#include <cstring>
#include <span>

namespace ehw {

// ---- RAII Wrappers for OpenSSL types ----
struct EvpCipherCtxDeleter {
    void operator()(EVP_CIPHER_CTX* ctx) const { if (ctx) EVP_CIPHER_CTX_free(ctx); }
};
using EvpCipherCtxPtr = std::unique_ptr<EVP_CIPHER_CTX, EvpCipherCtxDeleter>;

struct EvpMdCtxDeleter {
    void operator()(EVP_MD_CTX* ctx) const { if (ctx) EVP_MD_CTX_free(ctx); }
};
using EvpMdCtxPtr = std::unique_ptr<EVP_MD_CTX, EvpMdCtxDeleter>;

struct EvpPkeyCtxDeleter {
    void operator()(EVP_PKEY_CTX* ctx) const { if (ctx) EVP_PKEY_CTX_free(ctx); }
};
using EvpPkeyCtxPtr = std::unique_ptr<EVP_PKEY_CTX, EvpPkeyCtxDeleter>;

struct EvpPkeyDeleter {
    void operator()(EVP_PKEY* pkey) const { if (pkey) EVP_PKEY_free(pkey); }
};
using EvpPkeyPtr = std::unique_ptr<EVP_PKEY, EvpPkeyDeleter>;

/**
 * RAII wrapper for OpenSSL EVP contexts.
 * Provides hardware-accelerated cryptographic operations.
 */
class CryptoProvider {
public:
    CryptoProvider();
    ~CryptoProvider();

    CryptoProvider(const CryptoProvider&) = delete;
    CryptoProvider& operator=(const CryptoProvider&) = delete;

    // ---- Random Number Generation ----
    Result<void> randomBytes(uint8_t* out, size_t len);

    // ---- AES-256-GCM ----
    Result<ByteVector> aesGcmEncrypt(
        std::span<const uint8_t> key,
        std::span<const uint8_t> plaintext,
        std::span<const uint8_t> aad = {}
    );
    Result<ByteVector> aesGcmDecrypt(
        std::span<const uint8_t> key,
        std::span<const uint8_t> ciphertext
    );

    // ---- SHA-256 ----
    Result<std::array<uint8_t, SHA256_DIGEST_SIZE>> sha256(std::span<const uint8_t> data);
    Result<ByteVector> sha256(const ByteVector& data);

    // Double SHA-256
    Result<std::array<uint8_t, SHA256_DIGEST_SIZE>> doubleSha256(std::span<const uint8_t> data);

    // ---- HMAC-SHA512 ----
    Result<std::array<uint8_t, HMAC_SHA512_SIZE>> hmacSha512(
        std::span<const uint8_t> key,
        std::span<const uint8_t> data
    );

    // ---- PBKDF2-HMAC-SHA512 ----
    Result<ByteVector> pbkdf2HmacSha512(
        std::span<const uint8_t> password,
        std::span<const uint8_t> salt,
        uint32_t iterations,
        size_t keyLen
    );

    // ---- ECDSA (secp256k1 or P-256) ----
    Result<std::pair<SecureByteVector, ByteVector>> generateKeyPair();
    Result<ByteVector> ecdsaSign(
        std::span<const uint8_t> privateKey,
        std::span<const uint8_t> digest
    );
    Result<bool> ecdsaVerify(
        std::span<const uint8_t> publicKey,
        std::span<const uint8_t> digest,
        std::span<const uint8_t> signature
    );
    Result<ByteVector> privateKeyToPublicKey(
        std::span<const uint8_t> privateKey,
        bool compressed = true
    );

    static bool constantTimeCompare(std::span<const uint8_t> a, std::span<const uint8_t> b);

private:
    void handleOpenSSLError(const char* context);
    int getCurveNid() const;
    bool m_initialized = false;
};

CryptoProvider& crypto();

} // namespace ehw
