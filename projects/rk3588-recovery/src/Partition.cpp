// RK3588 Recovery — Partition Manager Implementation
#include "recovery/Partition.h"
#include <fstream>
#include <cstring>
#include <sys/stat.h>
#include <unistd.h>

namespace rk3588 {
namespace recovery {

PartitionManager::PartitionManager() = default;
PartitionManager::~PartitionManager() { unmount_all(); }

bool PartitionManager::detect_partitions(const std::string& config_path) {
    // Read partition layout from config file
    std::ifstream cfg(config_path);
    if (!cfg.is_open()) return false;

    partitions_.clear();
    std::string line;
    while (std::getline(cfg, line)) {
        if (line.empty() || line[0] == '#') continue;
        // Format: name device_path mount_point fs_type size_bytes
        PartitionInfo p;
        char dev[256], mp[256], fs[64];
        uint64_t sz;
        if (sscanf(line.c_str(), "%255s %255s %63s %lu %255s",
                   dev, mp, fs, &sz, dev) >= 4) {
            // Reparse: name device mount fs size
            char name[128];
            if (sscanf(line.c_str(), "%127s %255s %255s %63s %lu",
                       name, dev, mp, fs, &sz) >= 5) {
                p.name = name;
                p.device_path = dev;
                p.mount_point = mp;
                p.fs_type = fs;
                p.size_bytes = sz;
                partitions_.push_back(p);
            }
        }
    }
    return !partitions_.empty();
}

bool PartitionManager::create_simulated_partition(
    const std::string& name, const std::string& file_path,
    uint64_t size_bytes, const std::string& fs_type) {

    // Create a sparse file as simulated block device
    std::ofstream file(file_path, std::ios::binary | std::ios::trunc);
    if (!file.is_open()) return false;

    // Write 4KB header to allocate
    char header[4096] = {0};
    snprintf(header, sizeof(header), "RK3588_SIM_PART:%s:%s", name.c_str(), fs_type.c_str());
    file.write(header, sizeof(header));
    file.close();

    // Truncate to full size (sparse on supporting filesystems)
    truncate(file_path.c_str(), static_cast<off_t>(size_bytes));

    PartitionInfo p;
    p.name = name;
    p.device_path = file_path;
    p.mount_point = "/mnt/" + name;
    p.fs_type = fs_type;
    p.size_bytes = size_bytes;
    partitions_.push_back(p);

    // Create mount point directory
    std::string cmd = "mkdir -p " + p.mount_point;
    system(cmd.c_str());

    return true;
}

bool PartitionManager::mount(const std::string& name) {
    auto* p = const_cast<PartitionInfo*>(get_partition(name));
    if (!p || p->mounted) return false;

    // In simulator: just mark as mounted
    // On real HW: run mount command
    p->mounted = true;
    return true;
}

bool PartitionManager::unmount(const std::string& name) {
    auto* p = const_cast<PartitionInfo*>(get_partition(name));
    if (!p || !p->mounted) return false;
    p->mounted = false;
    return true;
}

bool PartitionManager::mount_all() {
    for (auto& p : partitions_) {
        if (!p.mounted) p.mounted = true;
    }
    return true;
}

bool PartitionManager::unmount_all() {
    for (auto& p : partitions_) p.mounted = false;
    return true;
}

const PartitionInfo* PartitionManager::get_partition(const std::string& name) const {
    for (const auto& p : partitions_) {
        if (p.name == name) return &p;
    }
    return nullptr;
}

std::vector<PartitionInfo> PartitionManager::list_partitions() const {
    return partitions_;
}

uint64_t PartitionManager::free_space(const std::string& name) const {
    auto* p = get_partition(name);
    if (!p) return 0;
    struct stat st;
    if (stat(p->mount_point.c_str(), &st) == 0) {
        return p->size_bytes - st.st_blocks * 512;
    }
    return p->size_bytes; // return full size if can't determine
}

bool PartitionManager::has_enough_space(const std::string& name,
                                         uint64_t required_bytes) const {
    return free_space(name) >= required_bytes;
}

std::string PartitionManager::get_device_path(const std::string& name) const {
    auto* p = get_partition(name);
    return p ? p->device_path : "";
}

std::string PartitionManager::get_mount_point(const std::string& name) const {
    auto* p = get_partition(name);
    return p ? p->mount_point : "";
}

} // namespace recovery
} // namespace rk3588
