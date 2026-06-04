#pragma once

#include "Common.h"
#include "core/CryptoProvider.h"
#include "core/SecureMemory.h"
#include <vector>
#include <string>
#include <span>
#include <shared_mutex>

namespace ehw {

/**
 * BackupManager — Distributed backup using Shamir's Secret Sharing (SSS) over GF(256).
 *
 * Features:
 * - Split a secret into N shares, requiring K to reconstruct
 * - Each share is encrypted with AES-256-GCM
 * - Share integrity via HMAC-SHA256
 * - Share serialization in Base64 for easy storage/transfer
 * - Incremental backup support with version tracking
 */
class BackupManager {
public:
    BackupManager();
    ~BackupManager();

    // ---- Shamir's Secret Sharing ----
    // Split secret into N shares, requiring K shares to recover
    // N: total shares (2-255), K: threshold (2-N)
    Result<std::vector<BackupShare>> splitSecret(
        std::span<const uint8_t> secret,
        uint8_t totalShares,
        uint8_t threshold,
        std::span<const uint8_t> encryptionKey
    );

    // Recover secret from K shares
    Result<SecureByteVector> recoverSecret(
        const std::vector<BackupShare>& shares,
        std::span<const uint8_t> encryptionKey
    );

    // ---- Share Serialization ----
    // Encode a share to a portable string (Base64)
    static std::string shareToString(const BackupShare& share);

    // Decode a share from a string
    static Result<BackupShare> shareFromString(std::string_view encoded);

    // ---- Backup Operations ----
    // Create a full backup of wallet seed
    Result<std::vector<std::string>> createBackup(
        std::span<const uint8_t> seed,
        std::string_view passphrase,
        uint8_t totalShares,
        uint8_t threshold
    );

    // Recover wallet seed from backup shares
    Result<SecureByteVector> recoverFromBackup(
        const std::vector<std::string>& encodedShares,
        std::string_view passphrase,
        uint8_t threshold
    );

    // ---- Incremental Backup ----
    struct BackupVersion {
        uint32_t version = 0;
        std::string timestamp;
        std::vector<std::string> shares;
        uint8_t totalShares = 0;
        uint8_t threshold = 0;
    };

    void addBackupVersion(const BackupVersion& version);
    const std::vector<BackupVersion>& getBackupHistory() const;

private:
    // GF(256) arithmetic
    static uint8_t gf256Mul(uint8_t a, uint8_t b);
    static uint8_t gf256Add(uint8_t a, uint8_t b);
    static uint8_t gf256Div(uint8_t a, uint8_t b);

    // Lagrange interpolation for SSS recovery
    static uint8_t lagrangeInterpolate(
        const std::vector<std::pair<uint8_t, uint8_t>>& points,
        uint8_t x
    );

    // Derive encryption key from passphrase for shares
    Result<ByteVector> deriveShareKey(std::string_view passphrase);

    mutable std::shared_mutex m_versionMutex;
    std::vector<BackupVersion> m_backupVersions;
};

} // namespace ehw
