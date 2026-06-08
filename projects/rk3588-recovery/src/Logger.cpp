// RK3588 Recovery — Logger Implementation
#include "recovery/Logger.h"
#include <cstdarg>
#include <cstdio>
#include <cstring>
#include <algorithm>

namespace rk3588 {
namespace recovery {

Logger::Logger(const std::string& log_path) : log_path_(log_path) {
    file_.open(log_path_, std::ios::app);
    load_from_file();
}

Logger::~Logger() {
    if (file_.is_open()) file_.close();
}

void Logger::log(Operation op, Status status, const char* fmt, ...) {
    std::lock_guard<std::mutex> lock(mutex_);

    LogEntry entry;
    entry.timestamp = std::time(nullptr);
    entry.operation = op;
    entry.status = status;

    va_list args;
    va_start(args, fmt);
    vsnprintf(entry.message, sizeof(entry.message), fmt, args);
    va_end(args);

    entries_.push_back(entry);
    if (entries_.size() > static_cast<size_t>(MAX_LOG_ENTRIES)) {
        entries_.erase(entries_.begin());
    }

    write_to_file(entry);
}

std::vector<LogEntry> Logger::get_entries(int limit) const {
    std::lock_guard<std::mutex> lock(mutex_);
    if (limit <= 0 || limit >= static_cast<int>(entries_.size())) {
        return entries_;
    }
    return std::vector<LogEntry>(
        entries_.end() - limit, entries_.end());
}

std::vector<LogEntry> Logger::get_entries_by_operation(Operation op, int limit) const {
    std::lock_guard<std::mutex> lock(mutex_);
    std::vector<LogEntry> result;
    for (auto it = entries_.rbegin(); it != entries_.rend(); ++it) {
        if (it->operation == op) {
            result.push_back(*it);
            if (--limit <= 0) break;
        }
    }
    std::reverse(result.begin(), result.end());
    return result;
}

void Logger::clear() {
    std::lock_guard<std::mutex> lock(mutex_);
    entries_.clear();
    file_.close();
    file_.open(log_path_, std::ios::trunc);
    file_.close();
    file_.open(log_path_, std::ios::app);
}

size_t Logger::entry_count() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return entries_.size();
}

void Logger::write_to_file(const LogEntry& entry) {
    if (!file_.is_open()) return;
    char time_buf[32];
    struct tm tm_info;
    localtime_r(&entry.timestamp, &tm_info);
    strftime(time_buf, sizeof(time_buf), "%Y-%m-%d %H:%M:%S", &tm_info);

    file_ << "[" << time_buf << "] "
          << "[" << operation_name(entry.operation) << "] "
          << "[" << static_cast<int>(entry.status) << "] "
          << entry.message << std::endl;
    file_.flush();
}

void Logger::load_from_file() {
    // Load existing log entries on startup
    std::ifstream in(log_path_);
    if (!in.is_open()) return;
    std::string line;
    while (std::getline(in, line) && entries_.size() < static_cast<size_t>(MAX_LOG_ENTRIES)) {
        if (line.empty()) continue;
        LogEntry entry;
        entry.timestamp = std::time(nullptr);
        entry.operation = Operation::NONE;
        entry.status = Status::COMPLETED;
        strncpy(entry.message, line.c_str(), sizeof(entry.message) - 1);
        entries_.push_back(entry);
    }
}

} // namespace recovery
} // namespace rk3588
