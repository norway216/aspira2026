#include "crypto/AESCipher.h"
#include <openssl/evp.h>
#include <cstring>
#include <random>
#include <algorithm>

namespace payment_engine {

AESCipher::AESCipher(const std::string& key_hex) {
    key_ = HexToBytes(key_hex);
}

std::vector<uint8_t> AESCipher::HexToBytes(const std::string& hex) {
    std::vector<uint8_t> bytes;
    bytes.reserve(hex.size() / 2);
    for (size_t i = 0; i + 1 < hex.size(); i += 2) {
        auto h = std::stoul(hex.substr(i, 2), nullptr, 16);
        bytes.push_back(static_cast<uint8_t>(h));
    }
    return bytes;
}

std::string AESCipher::Base64Encode(const std::vector<uint8_t>& data) {
    if (data.empty()) return {};
    int encoded_len = 4 * ((data.size() + 2) / 3);
    std::string encoded(encoded_len, '\0');

    EVP_ENCODE_CTX* ctx = EVP_ENCODE_CTX_new();
    EVP_EncodeInit(ctx);
    int out_len = 0;
    EVP_EncodeUpdate(ctx, reinterpret_cast<unsigned char*>(&encoded[0]), &out_len,
                     data.data(), static_cast<int>(data.size()));
    int final_len = 0;
    EVP_EncodeFinal(ctx, reinterpret_cast<unsigned char*>(&encoded[0]) + out_len, &final_len);
    EVP_ENCODE_CTX_free(ctx);

    // Remove newlines that OpenSSL base64 adds
    encoded.erase(std::remove(encoded.begin(), encoded.end(), '\n'), encoded.end());
    return encoded;
}

std::vector<uint8_t> AESCipher::Base64Decode(const std::string& b64) {
    if (b64.empty()) return {};
    std::string input = b64;
    // Add padding if needed
    while (input.size() % 4 != 0) input += '=';

    std::vector<uint8_t> decoded(input.size());
    EVP_ENCODE_CTX* ctx = EVP_ENCODE_CTX_new();
    EVP_DecodeInit(ctx);
    int out_len = 0;
    EVP_DecodeUpdate(ctx, decoded.data(), &out_len,
                     reinterpret_cast<const unsigned char*>(input.data()),
                     static_cast<int>(input.size()));
    int final_len = 0;
    EVP_DecodeFinal(ctx, decoded.data() + out_len, &final_len);
    EVP_ENCODE_CTX_free(ctx);

    decoded.resize(out_len + final_len);
    return decoded;
}

std::optional<std::string> AESCipher::Encrypt(const std::string& plaintext) {
    if (key_.size() != 32) return std::nullopt;

    // Generate random 12-byte IV
    std::vector<uint8_t> iv(12);
    std::random_device rd;
    for (size_t i = 0; i < iv.size(); ++i) {
        iv[i] = static_cast<uint8_t>(rd());
    }

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (!ctx) return std::nullopt;

    // Initialize encryption
    if (EVP_EncryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }

    // Set IV length (default is 12 for GCM)
    if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, static_cast<int>(iv.size()), nullptr) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }

    // Set key and IV
    if (EVP_EncryptInit_ex(ctx, nullptr, nullptr, key_.data(), iv.data()) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }

    // Encrypt
    std::vector<uint8_t> ciphertext(plaintext.size() + 16);
    int out_len = 0;
    if (EVP_EncryptUpdate(ctx, ciphertext.data(), &out_len,
                          reinterpret_cast<const unsigned char*>(plaintext.data()),
                          static_cast<int>(plaintext.size())) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }
    int total_len = out_len;

    // Finalize
    if (EVP_EncryptFinal_ex(ctx, ciphertext.data() + total_len, &out_len) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }
    total_len += out_len;
    ciphertext.resize(total_len);

    // Get GCM tag (16 bytes)
    std::vector<uint8_t> tag(16);
    if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, static_cast<int>(tag.size()), tag.data()) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }
    EVP_CIPHER_CTX_free(ctx);

    // Concatenate: IV || ciphertext || tag
    std::vector<uint8_t> result;
    result.reserve(iv.size() + ciphertext.size() + tag.size());
    result.insert(result.end(), iv.begin(), iv.end());
    result.insert(result.end(), ciphertext.begin(), ciphertext.end());
    result.insert(result.end(), tag.begin(), tag.end());

    return Base64Encode(result);
}

std::optional<std::string> AESCipher::Decrypt(const std::string& ciphertext_b64) {
    if (key_.size() != 32) return std::nullopt;

    auto decoded = Base64Decode(ciphertext_b64);
    if (decoded.size() < 12 + 16) return std::nullopt; // Need at least IV + tag

    // Extract IV (12 bytes), ciphertext, and tag (16 bytes)
    std::vector<uint8_t> iv(decoded.begin(), decoded.begin() + 12);
    std::vector<uint8_t> tag(decoded.end() - 16, decoded.end());
    std::vector<uint8_t> ciphertext(decoded.begin() + 12, decoded.end() - 16);

    EVP_CIPHER_CTX* ctx = EVP_CIPHER_CTX_new();
    if (!ctx) return std::nullopt;

    if (EVP_DecryptInit_ex(ctx, EVP_aes_256_gcm(), nullptr, nullptr, nullptr) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }

    if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, static_cast<int>(iv.size()), nullptr) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }

    if (EVP_DecryptInit_ex(ctx, nullptr, nullptr, key_.data(), iv.data()) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }

    std::vector<uint8_t> plaintext(ciphertext.size() + 16);
    int out_len = 0;
    if (EVP_DecryptUpdate(ctx, plaintext.data(), &out_len,
                          ciphertext.data(), static_cast<int>(ciphertext.size())) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }
    int total_len = out_len;

    // Set expected tag for verification
    if (EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG, static_cast<int>(tag.size()), tag.data()) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt;
    }

    // Finalize (verifies tag)
    if (EVP_DecryptFinal_ex(ctx, plaintext.data() + total_len, &out_len) != 1) {
        EVP_CIPHER_CTX_free(ctx);
        return std::nullopt; // Tag verification failed
    }
    total_len += out_len;
    plaintext.resize(total_len);
    EVP_CIPHER_CTX_free(ctx);

    return std::string(reinterpret_cast<const char*>(plaintext.data()), plaintext.size());
}

} // namespace payment_engine
