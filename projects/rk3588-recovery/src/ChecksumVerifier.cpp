// RK3588 Recovery — Checksum Verifier Implementation
#include "recovery/ChecksumVerifier.h"
#include <fstream>
#include <cstdio>
#include <cstring>
#include <openssl/evp.h>

namespace rk3588 {
namespace recovery {

ChecksumResult ChecksumVerifier::compute(const std::string& file_path,
                                          ChecksumType type) {
    ChecksumResult result;
    result.type = type;
    result.verified = false;

    std::ifstream file(file_path, std::ios::binary);
    if (!file.is_open()) return result;

    // Use EVP API for both MD5 and SHA256
    const EVP_MD* md = nullptr;
    switch (type) {
        case ChecksumType::MD5:    md = EVP_md5(); break;
        case ChecksumType::SHA256: md = EVP_sha256(); break;
        default: return result;
    }

    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (!ctx) return result;

    EVP_DigestInit_ex(ctx, md, nullptr);

    uint8_t buffer[HASH_BUFFER_SIZE];
    while (file.read(reinterpret_cast<char*>(buffer), sizeof(buffer)) ||
           file.gcount() > 0) {
        EVP_DigestUpdate(ctx, buffer, static_cast<size_t>(file.gcount()));
    }

    unsigned char hash[EVP_MAX_MD_SIZE];
    unsigned int hash_len = 0;
    EVP_DigestFinal_ex(ctx, hash, &hash_len);
    EVP_MD_CTX_free(ctx);

    // Convert to hex string
    for (unsigned int i = 0; i < hash_len; i++) {
        snprintf(result.hash + i * 2, 3, "%02x", hash[i]);
    }
    result.verified = true;
    return result;
}

bool ChecksumVerifier::verify(const std::string& file_path,
                               const std::string& expected_hash,
                               ChecksumType type) {
    auto result = compute(file_path, type);
    if (!result.verified) return false;
    return strcasecmp(result.hash, expected_hash.c_str()) == 0;
}

std::string ChecksumVerifier::compute_md5_hex(const uint8_t* data, size_t len) {
    unsigned char hash[MD5_DIGEST_LENGTH];
    MD5(data, len, hash);
    char hex[MD5_DIGEST_LENGTH * 2 + 1];
    for (int i = 0; i < MD5_DIGEST_LENGTH; i++) {
        snprintf(hex + i * 2, 3, "%02x", hash[i]);
    }
    return std::string(hex);
}

std::string ChecksumVerifier::compute_sha256_hex(const uint8_t* data, size_t len) {
    unsigned char hash[SHA256_DIGEST_LENGTH];
    SHA256(data, len, hash);
    char hex[SHA256_DIGEST_LENGTH * 2 + 1];
    for (int i = 0; i < SHA256_DIGEST_LENGTH; i++) {
        snprintf(hex + i * 2, 3, "%02x", hash[i]);
    }
    return std::string(hex);
}

bool ChecksumVerifier::files_identical(const std::string& path_a,
                                        const std::string& path_b) {
    std::ifstream fa(path_a, std::ios::binary);
    std::ifstream fb(path_b, std::ios::binary);
    if (!fa.is_open() || !fb.is_open()) return false;

    char ba[HASH_BUFFER_SIZE], bb[HASH_BUFFER_SIZE];
    while (fa && fb) {
        fa.read(ba, sizeof(ba));
        fb.read(bb, sizeof(bb));
        if (fa.gcount() != fb.gcount()) return false;
        if (memcmp(ba, bb, static_cast<size_t>(fa.gcount())) != 0) return false;
    }
    return true;
}

} // namespace recovery
} // namespace rk3588
