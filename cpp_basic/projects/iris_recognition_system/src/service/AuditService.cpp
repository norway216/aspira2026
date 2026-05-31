#include "service/AuditService.h"
#include "persistence/Database.h"

namespace iris {

AuditService::AuditService(Database& database)
    : m_database(database) {}

void AuditService::log(const std::string& action, const std::string& userId,
                        const std::string& details) {
    m_database.logAction(action, userId, details);
}

std::vector<AuditLogEntry> AuditService::getLogs(int limit) {
    return m_database.getAuditLogs(limit);
}

} // namespace iris
