#include "AEADHelper.h"
#include "CryptoConfig.h"
#include <openssl/evp.h>
#include <openssl/rand.h>
#include <cstring>

namespace AEADHelper {

Result<SecureBuffer> encrypt(const unsigned char* plaintext, size_t plainLen, const unsigned char* key) {
    if (!plaintext || plainLen == 0 || !key) {
        return Result<SecureBuffer>::fail(ErrorCode::EncryptionFailed, "Invalid input");
    }

    // Generate random nonce (IV)
    unsigned char nonce[CryptoConfig::NONCE_SIZE];
    if (RAND_bytes(nonce, CryptoConfig::NONCE_SIZE) != 1) {
        return Result<SecureBuffer>::fail(ErrorCode::EncryptionFailed, "Failed to generate nonce");
    }

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (!ctx) {
        return Result<SecureBuffer>::fail(ErrorCode::EncryptionFailed, "Failed to create cipher context");
    }

    if (EVP_EncryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::EncryptionFailed, "EncryptInit failed");
    }

    if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, CryptoConfig::NONCE_SIZE, nullptr) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::EncryptionFailed, "Set IV length failed");
    }

    if (EVP_EncryptInit_ex(ctx, nullptr, nullptr, key, nonce) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::EncryptionFailed, "Set key/IV failed");
    }

    // Allocate output: nonce + ciphertext + tag
    size_t cipherLen = CryptoConfig::NONCE_SIZE + plainLen + CryptoConfig::TAG_SIZE;
    SecureBuffer output(cipherLen);

    // Copy nonce to output
    std::memcpy(output.data(), nonce, CryptoConfig::NONCE_SIZE);

    unsigned char* ct = output.data() + CryptoConfig::NONCE_SIZE;
    int outLen = 0;

    if (EVP_EncryptUpdate(ctx, ct, &outLen, plaintext, static_cast<int>(plainLen)) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::EncryptionFailed, "EncryptUpdate failed");
    }

    int finalLen = 0;
    if (EVP_EncryptFinal_ex(ctx, ct + outLen, &finalLen) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::EncryptionFailed, "EncryptFinal failed");
    }

    // Get authentication tag
    if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, CryptoConfig::TAG_SIZE,
                             output.data() + CryptoConfig::NONCE_SIZE + outLen + finalLen) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::EncryptionFailed, "Get tag failed");
    }

    EVP_CIPHER_CTX_free(ctx);
    return Result<SecureBuffer>::ok(std::move(output));
}

Result<SecureBuffer> encrypt(const SecureBuffer& plaintext, const SecureBuffer& key) {
    return encrypt(plaintext.data(), plaintext.size(), key.data());
}

Result<SecureBuffer> decrypt(const unsigned char* ciphertextWithNonce, size_t cipherLen, const unsigned char* key) {
    if (!ciphertextWithNonce || cipherLen <= CryptoConfig::ENCRYPTED_OVERHEAD || !key) {
        return Result<SecureBuffer>::fail(ErrorCode::DecryptionFailed, "Invalid ciphertext");
    }

    const unsigned char* nonce = ciphertextWithNonce;
    const unsigned char* ct = ciphertextWithNonce + CryptoConfig::NONCE_SIZE;
    size_t ctLen = cipherLen - CryptoConfig::NONCE_SIZE - CryptoConfig::TAG_SIZE;

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (!ctx) {
        return Result<SecureBuffer>::fail(ErrorCode::DecryptionFailed, "Failed to create cipher context");
    }

    if (EVP_DecryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::DecryptionFailed, "DecryptInit failed");
    }

    if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, CryptoConfig::NONCE_SIZE, nullptr) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::DecryptionFailed, "Set IV length failed");
    }

    if (EVP_DecryptInit_ex(ctx, nullptr, nullptr, key, nonce) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::DecryptionFailed, "Set key/IV failed");
    }

    // Set expected tag
    if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG, CryptoConfig::TAG_SIZE,
                             const_cast<unsigned char*>(ct + ctLen)) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::DecryptionFailed, "Set tag failed");
    }

    SecureBuffer plaintext(ctLen);
    int outLen = 0;

    if (EVP_DecryptUpdate(ctx, plaintext.data(), &outLen, ct, static_cast<int>(ctLen)) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::DecryptionFailed, "DecryptUpdate failed");
    }

    int finalLen = 0;
    int ret = EVP_DecryptFinal_ex(ctx, plaintext.data() + outLen, &finalLen);
    EVP_CIPHER_CTX_free(ctx);

    if (ret != 1) {
        return Result<SecureBuffer>::fail(ErrorCode::DecryptionFailed, "AEAD decryption failed - wrong key or tampered data");
    }

    return Result<SecureBuffer>::ok(std::move(plaintext));
}

Result<SecureBuffer> decrypt(const SecureBuffer& ciphertextWithNonce, const SecureBuffer& key) {
    return decrypt(ciphertextWithNonce.data(), ciphertextWithNonce.size(), key.data());
}

} // namespace AEADHelper
