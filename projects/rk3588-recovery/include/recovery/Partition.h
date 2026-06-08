// RK3588 Recovery — Partition Manager
// Handles partition mounting, unmounting, space checking.
// In simulator mode, uses files as block device substitutes.
#pragma once

#include "Types.h"
#include <vector>
#include <string>
#include <cstdint>

namespace rk3588 {
namespace recovery {

class PartitionManager {
public:
    PartitionManager();
    ~PartitionManager();

    // Detect all partitions from config
    bool detect_partitions(const std::string& config_path);

    // Mount / unmount
    bool mount(const std::string& name);
    bool unmount(const std::string& name);
    bool mount_all();
    bool unmount_all();

    // Query
    const PartitionInfo* get_partition(const std::string& name) const;
    std::vector<PartitionInfo> list_partitions() const;
    uint64_t free_space(const std::string& name) const;
    bool has_enough_space(const std::string& name, uint64_t required_bytes) const;

    // Simulator: create test partitions as files
    bool create_simulated_partition(const std::string& name,
                                     const std::string& file_path,
                                     uint64_t size_bytes,
                                     const std::string& fs_type);

    // Direct access for dd operations
    std::string get_device_path(const std::string& name) const;
    std::string get_mount_point(const std::string& name) const;

private:
    std::vector<PartitionInfo> partitions_;
    std::string sim_base_path_; // base path for simulated devices
};

} // namespace recovery
} // namespace rk3588
