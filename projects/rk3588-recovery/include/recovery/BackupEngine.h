// RK3588 Recovery — Backup & Restore Engine
// Multi-threaded high-performance backup/restore with progress reporting.
// Uses 4MB block I/O (dd-style) for system images, tar.xz for app data.
#pragma once

#include "Types.h"
#include "Partition.h"
#include "ChecksumVerifier.h"
#include <functional>
#include <thread>
#include <atomic>
#include <mutex>

namespace rk3588 {
namespace recovery {

// Callback for progress updates
using ProgressCallback = std::function<void(const Progress&)>;

class BackupEngine {
public:
    BackupEngine(PartitionManager* pm, ChecksumVerifier* verifier);
    ~BackupEngine();

    // ─── System Operations ──────────────────────────────

    // Install system image from compressed file to system partition
    bool install_system(const std::string& image_path,
                        const std::string& target_partition,
                        ProgressCallback callback = nullptr);

    // Backup system partition to compressed image
    bool backup_system(const std::string& target_partition,
                       const std::string& output_path,
                       ProgressCallback callback = nullptr);

    // Restore system from backup image
    bool restore_system(const std::string& backup_path,
                        const std::string& target_partition,
                        ProgressCallback callback = nullptr);

    // ─── App Operations ─────────────────────────────────

    // Backup app data directories
    bool backup_apps(const std::vector<std::string>& paths,
                     const std::string& output_path,
                     ProgressCallback callback = nullptr);

    // Restore app data
    bool restore_apps(const std::string& backup_path,
                      const std::string& target_root,
                      ProgressCallback callback = nullptr);

    // ─── Control ────────────────────────────────────────

    void cancel();
    bool is_running() const { return running_.load(); }
    Progress current_progress() const;

private:
    // Low-level dd copy with progress
    bool dd_copy(const std::string& src, const std::string& dst,
                 uint64_t size, bool is_read, ProgressCallback callback);

    // Run external command with progress pipe
    bool run_piped_command(const std::string& cmd,
                           uint64_t estimated_size,
                           ProgressCallback callback);

    // Compress/decompress helpers
    bool compress_xz(const std::string& input, const std::string& output,
                     ProgressCallback callback);
    bool decompress_xz(const std::string& input, const std::string& output,
                       ProgressCallback callback);

    PartitionManager* pm_;
    ChecksumVerifier*  verifier_;
    std::atomic<bool>  running_{false};
    std::atomic<bool>  cancelled_{false};
    Progress           progress_;
    mutable std::mutex progress_mutex_;
};

} // namespace recovery
} // namespace rk3588
