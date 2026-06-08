// RK3588 Recovery System — Common Types
// High-performance, high-stability recovery/backup/restore system
#pragma once

#include <cstdint>
#include <string>
#include <ctime>

namespace rk3588 {
namespace recovery {

// ─── Operation Types ────────────────────────────────────
enum class Operation : uint8_t {
    INSTALL_SYSTEM   = 0,
    BACKUP_SYSTEM    = 1,
    RESTORE_SYSTEM   = 2,
    BACKUP_APPS      = 3,
    RESTORE_APPS     = 4,
    VERIFY_IMAGE     = 5,
    NONE             = 0xFF
};

inline const char* operation_name(Operation op) {
    switch (op) {
        case Operation::INSTALL_SYSTEM:  return "Install System";
        case Operation::BACKUP_SYSTEM:   return "Backup System";
        case Operation::RESTORE_SYSTEM:  return "Restore System";
        case Operation::BACKUP_APPS:     return "Backup Apps";
        case Operation::RESTORE_APPS:    return "Restore Apps";
        case Operation::VERIFY_IMAGE:    return "Verify Image";
        default:                         return "Unknown";
    }
}

// ─── Operation Status ────────────────────────────────────
enum class Status : uint8_t {
    PENDING     = 0,
    RUNNING     = 1,
    COMPLETED   = 2,
    FAILED      = 3,
    CANCELLED   = 4,
    VERIFYING   = 5
};

// ─── Progress Information ────────────────────────────────
struct Progress {
    Status      status     = Status::PENDING;
    int         percent    = 0;
    uint64_t    bytes_done = 0;
    uint64_t    bytes_total = 0;
    uint64_t    speed_bps  = 0;    // bytes per second
    int         eta_seconds = 0;
    char        message[256] = {0};
    std::time_t started_at = 0;
    std::time_t updated_at = 0;
};

// ─── Partition Info ──────────────────────────────────────
struct PartitionInfo {
    std::string name;         // e.g., "system", "recovery", "userdata"
    std::string device_path;  // e.g., "/dev/mmcblk0p2"
    std::string mount_point;  // e.g., "/mnt/system"
    std::string fs_type;      // "ext4", "fat32"
    uint64_t    size_bytes = 0;
    uint64_t    used_bytes = 0;
    bool        mounted = false;
};

// ─── Checksum Type ───────────────────────────────────────
enum class ChecksumType : uint8_t {
    MD5    = 0,
    SHA256 = 1,
    CRC32  = 2
};

// ─── Checksum Result ─────────────────────────────────────
struct ChecksumResult {
    ChecksumType type;
    char         hash[128] = {0};
    bool         verified = false;
};

// ─── Log Entry ───────────────────────────────────────────
struct LogEntry {
    std::time_t timestamp;
    Operation   operation;
    Status      status;
    char        message[512] = {0};
};

// ─── Compression Type ────────────────────────────────────
enum class Compression : uint8_t {
    NONE = 0,
    XZ   = 1,
    GZIP = 2,
    ZSTD = 3
};

// ─── Boot Mode ───────────────────────────────────────────
enum class BootMode : uint8_t {
    NORMAL   = 0,
    RECOVERY = 1,
    FASTBOOT = 2
};

// ─── I/O Constants for High Performance ──────────────────
constexpr uint64_t BLOCK_SIZE       = 4 * 1024 * 1024;  // 4MB dd blocks
constexpr uint64_t BUFFER_SIZE      = 64 * 1024;         // 64KB read buffer
constexpr int      MAX_LOG_ENTRIES  = 10000;
constexpr int      PROGRESS_UPDATE_MS = 100;

} // namespace recovery
} // namespace rk3588
