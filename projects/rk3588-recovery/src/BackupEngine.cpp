// RK3588 Recovery — Backup & Restore Engine Implementation
#include "recovery/BackupEngine.h"
#include "recovery/Logger.h"
#include <fstream>
#include <cstdio>
#include <cstring>
#include <chrono>
#include <unistd.h>
#include <sys/stat.h>

namespace rk3588 {
namespace recovery {

BackupEngine::BackupEngine(PartitionManager* pm, ChecksumVerifier* verifier)
    : pm_(pm), verifier_(verifier) {}

BackupEngine::~BackupEngine() { cancel(); }

void BackupEngine::cancel() {
    cancelled_ = true;
    // Wait up to 2 seconds for running operation to stop
    for (int i = 0; i < 20 && running_; i++) {
        std::this_thread::sleep_for(std::chrono::milliseconds(100));
    }
}

Progress BackupEngine::current_progress() const {
    std::lock_guard<std::mutex> lock(progress_mutex_);
    return progress_;
}

// ─── System Install ────────────────────────────────────
bool BackupEngine::install_system(const std::string& image_path,
                                   const std::string& target_partition,
                                   ProgressCallback callback) {
    running_ = true;
    cancelled_ = false;

    auto* partition = pm_->get_partition(target_partition);
    if (!partition) {
        running_ = false;
        return false;
    }

    // Get source size
    struct stat st;
    if (stat(image_path.c_str(), &st) != 0) {
        running_ = false;
        return false;
    }
    uint64_t src_size = static_cast<uint64_t>(st.st_size);

    // Check target has enough space
    if (!pm_->has_enough_space(target_partition, src_size)) {
        running_ = false;
        return false;
    }

    // Update progress
    {
        std::lock_guard<std::mutex> lock(progress_mutex_);
        progress_ = Progress{};
        progress_.status = Status::RUNNING;
        progress_.bytes_total = src_size;
        progress_.started_at = std::time(nullptr);
        snprintf(progress_.message, sizeof(progress_.message),
                 "Installing system from %s to %s",
                 image_path.c_str(), partition->device_path.c_str());
    }
    if (callback) callback(progress_);

    // Build command: decompress and write
    std::string target_dev = partition->device_path;
    std::string cmd;

    bool is_compressed = (image_path.find(".xz") != std::string::npos ||
                          image_path.find(".gz") != std::string::npos);

    if (is_compressed) {
        cmd = "xzcat '" + image_path + "' | dd of='" + target_dev +
              "' bs=4M status=none 2>&1";
    } else {
        cmd = "dd if='" + image_path + "' of='" + target_dev +
              "' bs=4M status=none 2>&1";
    }

    // Execute with progress tracking
    bool result = run_piped_command(cmd, src_size, callback);

    if (result && !cancelled_) {
        // Verify written image
        {
            std::lock_guard<std::mutex> lock(progress_mutex_);
            progress_.status = Status::VERIFYING;
            snprintf(progress_.message, sizeof(progress_.message),
                     "Verifying installed image...");
        }
        if (callback) callback(progress_);

        auto checksum = verifier_->compute(target_dev, ChecksumType::SHA256);
        result = checksum.verified;
    }

    {
        std::lock_guard<std::mutex> lock(progress_mutex_);
        progress_.status = (result && !cancelled_) ? Status::COMPLETED : Status::FAILED;
        progress_.percent = result ? 100 : progress_.percent;
        progress_.updated_at = std::time(nullptr);
    }
    if (callback) callback(progress_);

    sync(); // force write to disk
    running_ = false;
    return result && !cancelled_;
}

// ─── System Backup ─────────────────────────────────────
bool BackupEngine::backup_system(const std::string& target_partition,
                                  const std::string& output_path,
                                  ProgressCallback callback) {
    running_ = true;
    cancelled_ = false;

    auto* partition = pm_->get_partition(target_partition);
    if (!partition) { running_ = false; return false; }

    uint64_t part_size = partition->size_bytes;

    {
        std::lock_guard<std::mutex> lock(progress_mutex_);
        progress_ = Progress{};
        progress_.status = Status::RUNNING;
        progress_.bytes_total = part_size;
        progress_.started_at = std::time(nullptr);
        snprintf(progress_.message, sizeof(progress_.message),
                 "Backing up %s to %s", target_partition.c_str(), output_path.c_str());
    }
    if (callback) callback(progress_);

    // dd from partition, pipe through xz
    std::string raw_file = output_path + ".raw";
    std::string cmd = "dd if='" + partition->device_path + "' of='" + raw_file +
                      "' bs=4M status=none 2>&1";

    bool result = run_piped_command(cmd, part_size, callback);

    if (result && !cancelled_) {
        // Compress with xz
        result = compress_xz(raw_file, output_path, callback);
        // Remove raw file
        std::remove(raw_file.c_str());
    }

    {
        std::lock_guard<std::mutex> lock(progress_mutex_);
        progress_.status = (result && !cancelled_) ? Status::COMPLETED : Status::FAILED;
        progress_.updated_at = std::time(nullptr);
    }
    if (callback) callback(progress_);

    running_ = false;
    return result && !cancelled_;
}

// ─── System Restore ────────────────────────────────────
bool BackupEngine::restore_system(const std::string& backup_path,
                                   const std::string& target_partition,
                                   ProgressCallback callback) {
    // Same as install_system — decompress backup to partition
    return install_system(backup_path, target_partition, callback);
}

// ─── App Backup ────────────────────────────────────────
bool BackupEngine::backup_apps(const std::vector<std::string>& paths,
                                const std::string& output_path,
                                ProgressCallback callback) {
    running_ = true;
    cancelled_ = false;

    // Calculate total size
    uint64_t total_size = 0;
    for (const auto& p : paths) {
        struct stat st;
        if (stat(p.c_str(), &st) == 0) {
            total_size += st.st_size;
        }
    }

    {
        std::lock_guard<std::mutex> lock(progress_mutex_);
        progress_ = Progress{};
        progress_.status = Status::RUNNING;
        progress_.bytes_total = total_size;
        progress_.started_at = std::time(nullptr);
    }
    if (callback) callback(progress_);

    // tar.xz the paths
    std::string cmd = "tar -cJf '" + output_path + "'";
    for (const auto& p : paths) {
        cmd += " '" + p + "'";
    }
    cmd += " 2>&1";

    bool result = run_piped_command(cmd, total_size, callback);

    {
        std::lock_guard<std::mutex> lock(progress_mutex_);
        progress_.status = (result && !cancelled_) ? Status::COMPLETED : Status::FAILED;
        progress_.updated_at = std::time(nullptr);
    }
    if (callback) callback(progress_);

    running_ = false;
    return result && !cancelled_;
}

// ─── App Restore ───────────────────────────────────────
bool BackupEngine::restore_apps(const std::string& backup_path,
                                 const std::string& target_root,
                                 ProgressCallback callback) {
    running_ = true;
    cancelled_ = false;

    struct stat st;
    stat(backup_path.c_str(), &st);

    {
        std::lock_guard<std::mutex> lock(progress_mutex_);
        progress_ = Progress{};
        progress_.status = Status::RUNNING;
        progress_.bytes_total = static_cast<uint64_t>(st.st_size);
        progress_.started_at = std::time(nullptr);
    }
    if (callback) callback(progress_);

    std::string cmd = "tar -xJf '" + backup_path + "' -C '" + target_root + "' 2>&1";
    bool result = run_piped_command(cmd, static_cast<uint64_t>(st.st_size), callback);

    {
        std::lock_guard<std::mutex> lock(progress_mutex_);
        progress_.status = (result && !cancelled_) ? Status::COMPLETED : Status::FAILED;
        progress_.updated_at = std::time(nullptr);
    }
    if (callback) callback(progress_);

    running_ = false;
    return result && !cancelled_;
}

// ─── Run command with progress tracking ────────────────
bool BackupEngine::run_piped_command(const std::string& cmd,
                                      uint64_t estimated_size,
                                      ProgressCallback callback) {
    FILE* pipe = popen(cmd.c_str(), "r");
    if (!pipe) {
        running_ = false;
        return false;
    }

    auto start_time = std::chrono::steady_clock::now();
    uint64_t bytes_read = 0;
    char buf[1024];
    auto last_update = start_time;

    while (fgets(buf, sizeof(buf), pipe) != nullptr && !cancelled_) {
        bytes_read += strlen(buf);

        auto now = std::chrono::steady_clock::now();
        auto elapsed_ms = std::chrono::duration_cast<std::chrono::milliseconds>(
            now - last_update).count();

        if (elapsed_ms >= PROGRESS_UPDATE_MS) {
            auto total_elapsed = std::chrono::duration_cast<std::chrono::seconds>(
                now - start_time).count();

            std::lock_guard<std::mutex> lock(progress_mutex_);
            if (estimated_size > 0) {
                progress_.percent = static_cast<int>(
                    (bytes_read * 100) / estimated_size);
            }
            progress_.bytes_done = bytes_read;
            if (total_elapsed > 0) {
                progress_.speed_bps = bytes_read / static_cast<uint64_t>(total_elapsed);
            }
            progress_.updated_at = std::time(nullptr);
            last_update = now;

            if (callback) callback(progress_);
        }
    }

    int ret = pclose(pipe);
    return ret == 0 && !cancelled_;
}

bool BackupEngine::compress_xz(const std::string& input,
                                const std::string& output,
                                ProgressCallback callback) {
    std::string cmd = "xz -9 -c '" + input + "' > '" + output + "' 2>&1";
    struct stat st;
    stat(input.c_str(), &st);
    return run_piped_command(cmd, static_cast<uint64_t>(st.st_size), callback);
}

bool BackupEngine::decompress_xz(const std::string& input,
                                  const std::string& output,
                                  ProgressCallback callback) {
    std::string cmd = "xzcat '" + input + "' > '" + output + "' 2>&1";
    struct stat st;
    stat(input.c_str(), &st);
    return run_piped_command(cmd, static_cast<uint64_t>(st.st_size) * 3, callback);
}

} // namespace recovery
} // namespace rk3588
