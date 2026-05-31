#include "SignatureHelper.h"
#include "CryptoConfig.h"
#include <openssl/evp.h>
#include <openssl/pem.h>
#include <cstring>

namespace SignatureHelper {

Result<KeyPair> generateKeyPair() {
    EVP_PKEY_CTX* pctx = EVP_PKEY_CTX_new_id(EVP_PKEY_ED25519, nullptr);
    if (!pctx) {
        return Result<KeyPair>::fail(ErrorCode::KeyGenerationFailed, "Failed to create key context");
    }

    EVP_PKEY* pkey = nullptr;
    if (EVP_PKEY_keygen_init(pctx) != 1 ||
        EVP_PKEY_keygen(pctx, &pkey) != 1) {
        EVP_PKEY_CTX_free(pctx);
        return Result<KeyPair>::fail(ErrorCode::KeyGenerationFailed, "Key generation failed");
    }
    EVP_PKEY_CTX_free(pctx);

    // Extract raw keys
    size_t pubLen = CryptoConfig::PUBLIC_KEY_SIZE;
    size_t privLen = CryptoConfig::SECRET_KEY_SIZE;
    KeyPair kp;
    kp.publicKey = SecureBuffer(pubLen);
    kp.secretKey = SecureBuffer(privLen);

    if (EVP_PKEY_get_raw_public_key(pkey, kp.publicKey.data(), &pubLen) != 1 ||
        EVP_PKEY_get_raw_private_key(pkey, kp.secretKey.data(), &privLen) != 1) {
        EVP_PKEY_free(pkey);
        return Result<KeyPair>::fail(ErrorCode::KeyGenerationFailed, "Failed to extract raw keys");
    }

    EVP_PKEY_free(pkey);
    return Result<KeyPair>::ok(std::move(kp));
}

Result<KeyPair> generateFromSeed(const SecureBuffer& seed) {
    if (seed.size() != CryptoConfig::SEED_SIZE) {
        return Result<KeyPair>::fail(ErrorCode::KeyGenerationFailed, "Seed must be 32 bytes");
    }

    EVP_PKEY* pkey = EVP_PKEY_new_raw_private_key(EVP_PKEY_ED25519, nullptr,
                                                    seed.data(), seed.size());
    if (!pkey) {
        return Result<KeyPair>::fail(ErrorCode::KeyGenerationFailed, "Failed to create key from seed");
    }

    size_t pubLen = CryptoConfig::PUBLIC_KEY_SIZE;
    size_t privLen = CryptoConfig::SECRET_KEY_SIZE;
    KeyPair kp;
    kp.publicKey = SecureBuffer(pubLen);
    kp.secretKey = SecureBuffer(privLen);

    if (EVP_PKEY_get_raw_public_key(pkey, kp.publicKey.data(), &pubLen) != 1 ||
        EVP_PKEY_get_raw_private_key(pkey, kp.secretKey.data(), &privLen) != 1) {
        EVP_PKEY_free(pkey);
        return Result<KeyPair>::fail(ErrorCode::KeyGenerationFailed, "Failed to extract raw keys");
    }

    EVP_PKEY_free(pkey);
    return Result<KeyPair>::ok(std::move(kp));
}

Result<SecureBuffer> sign(const unsigned char* message, size_t msgLen,
                          const unsigned char* secretKey) {
    if (!message || msgLen == 0 || !secretKey) {
        return Result<SecureBuffer>::fail(ErrorCode::SignatureFailed, "Invalid input");
    }

    EVP_PKEY* pkey = EVP_PKEY_new_raw_private_key(EVP_PKEY_ED25519, nullptr,
                                                    secretKey, CryptoConfig::SECRET_KEY_SIZE);
    if (!pkey) {
        return Result<SecureBuffer>::fail(ErrorCode::SignatureFailed, "Failed to create key");
    }

    EVP_MD_CTX* mdctx = EVP_MD_CTX_new();
    if (!mdctx) {
        EVP_PKEY_free(pkey);
        return Result<SecureBuffer>::fail(ErrorCode::SignatureFailed, "Failed to create digest context");
    }

    if (EVP_DigestSignInit(mdctx, nullptr, nullptr, nullptr, pkey) != 1) {
        EVP_MD_CTX_free(mdctx);
        EVP_PKEY_free(pkey);
        return Result<SecureBuffer>::fail(ErrorCode::SignatureFailed, "SignInit failed");
    }

    size_t sigLen = CryptoConfig::SIGNATURE_SIZE;
    SecureBuffer sig(sigLen);

    if (EVP_DigestSign(mdctx, sig.data(), &sigLen, message, msgLen) != 1) {
        EVP_MD_CTX_free(mdctx);
        EVP_PKEY_free(pkey);
        return Result<SecureBuffer>::fail(ErrorCode::SignatureFailed, "Sign failed");
    }

    EVP_MD_CTX_free(mdctx);
    EVP_PKEY_free(pkey);
    return Result<SecureBuffer>::ok(std::move(sig));
}

Result<bool> verify(const unsigned char* message, size_t msgLen,
                    const unsigned char* signature,
                    const unsigned char* publicKey) {
    if (!message || msgLen == 0 || !signature || !publicKey) {
        return Result<bool>::fail(ErrorCode::VerificationFailed, "Invalid input");
    }

    EVP_PKEY* pkey = EVP_PKEY_new_raw_public_key(EVP_PKEY_ED25519, nullptr,
                                                   publicKey, CryptoConfig::PUBLIC_KEY_SIZE);
    if (!pkey) {
        return Result<bool>::fail(ErrorCode::VerificationFailed, "Failed to create key");
    }

    EVP_MD_CTX* mdctx = EVP_MD_CTX_new();
    if (!mdctx) {
        EVP_PKEY_free(pkey);
        return Result<bool>::fail(ErrorCode::VerificationFailed, "Failed to create digest context");
    }

    if (EVP_DigestVerifyInit(mdctx, nullptr, nullptr, nullptr, pkey) != 1) {
        EVP_MD_CTX_free(mdctx);
        EVP_PKEY_free(pkey);
        return Result<bool>::fail(ErrorCode::VerificationFailed, "VerifyInit failed");
    }

    int ret = EVP_DigestVerify(mdctx, signature, CryptoConfig::SIGNATURE_SIZE,
                                message, msgLen);
    EVP_MD_CTX_free(mdctx);
    EVP_PKEY_free(pkey);

    return Result<bool>::ok(ret == 1);
}

Result<SecureBuffer> sign(const SecureBuffer& message, const SecureBuffer& secretKey) {
    return sign(message.data(), message.size(), secretKey.data());
}

Result<bool> verify(const SecureBuffer& message, const SecureBuffer& signature,
                    const SecureBuffer& publicKey) {
    return verify(message.data(), message.size(), signature.data(), publicKey.data());
}

} // namespace SignatureHelper
