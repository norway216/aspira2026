#pragma once

#include "util/Result.h"
#include "domain/AuditLogEntry.h"

#include <vector>

class Database;

class AuditLogRepository {
public:
    explicit AuditLogRepository(Database* db);

    // Insert a new audit log entry (computes hash from previous entry)
    Result<AuditLogEntry> append(const QString& action, const QString& userId,
                                  const QString& details);

    // Get paginated entries
    Result<std::vector<AuditLogEntry>> findAll(int offset = 0, int limit = 50);

    // Get the last (most recent) entry for hash chain
    Result<std::optional<AuditLogEntry>> lastEntry();

    // Verify the entire hash chain integrity
    Result<bool> verifyChain();

    // Count entries
    Result<int> count();

    // Get all entries (for backup)
    Result<std::vector<AuditLogEntry>> getAllEntries();

private:
    Database* m_db;
    AuditLogEntry entryFromQuery(class QSqlQuery& query) const;
};
