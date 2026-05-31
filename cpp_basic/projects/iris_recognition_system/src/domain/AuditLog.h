#pragma once

#include <cstdint>
#include <string>

namespace iris {

struct AuditLogEntry {
    int64_t id         = 0;
    std::string action;
    std::string user_id;
    std::string details;
    int64_t created_at = 0;
};

} // namespace iris
