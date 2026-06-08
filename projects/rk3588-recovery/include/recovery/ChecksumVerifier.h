// RK3588 Recovery — Checksum Verification
// MD5 and SHA256 hash computation and verification.
#pragma once

#include "Types.h"
#include <string>
#include <cstdint>
#include <openssl/md5.h>
#include <openssl/sha.h>

namespace rk3588 {
namespace recovery {

class ChecksumVerifier {
public:
    ChecksumVerifier() = default;

    // Compute checksum of a file
    ChecksumResult compute(const std::string& file_path, ChecksumType type);

    // Verify file against expected checksum
    bool verify(const std::string& file_path, const std::string& expected_hash,
                ChecksumType type);

    // Stream-based compute (for pipes)
    std::string compute_md5_hex(const uint8_t* data, size_t len);
    std::string compute_sha256_hex(const uint8_t* data, size_t len);

    // Fast file comparison
    static bool files_identical(const std::string& path_a, const std::string& path_b);

private:
    static constexpr size_t HASH_BUFFER_SIZE = 64 * 1024; // 64KB read buffer
};

} // namespace recovery
} // namespace rk3588
