#pragma once

#include "domain/AuditLog.h"
#include <string>
#include <vector>

namespace iris {

class Database;

/// Service for audit logging
class AuditService {
public:
    explicit AuditService(Database& database);

    void log(const std::string& action, const std::string& userId = "",
             const std::string& details = "");

    std::vector<AuditLogEntry> getLogs(int limit = 100);

private:
    Database& m_database;
};

} // namespace iris
