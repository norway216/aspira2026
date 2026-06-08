// RK3588 Recovery — Operation Logger
// Append-only log for all recovery operations.
#pragma once

#include "Types.h"
#include <vector>
#include <string>
#include <fstream>
#include <mutex>

namespace rk3588 {
namespace recovery {

class Logger {
public:
    explicit Logger(const std::string& log_path);
    ~Logger();

    void log(Operation op, Status status, const char* fmt, ...);
    std::vector<LogEntry> get_entries(int limit = 100) const;
    std::vector<LogEntry> get_entries_by_operation(Operation op, int limit = 50) const;
    void clear();
    size_t entry_count() const;

private:
    std::string log_path_;
    mutable std::mutex mutex_;
    std::vector<LogEntry> entries_;
    std::ofstream file_;

    void write_to_file(const LogEntry& entry);
    void load_from_file();
};

} // namespace recovery
} // namespace rk3588
