#pragma once

#include <QObject>

#include "util/Result.h"
#include "domain/AuditLogEntry.h"

#include <vector>

class Database;
class AuditLogRepository;

/**
 * AuditService - provides access to the audit log and integrity verification.
 */
class AuditService : public QObject {
    Q_OBJECT
public:
    explicit AuditService(Database* db, QObject* parent = nullptr);

    // Get paginated audit log entries
    void getEntries(int offset = 0, int limit = 50);

    // Verify the hash chain integrity
    void verifyIntegrity();

    // Get entry count
    void getEntryCount();

    // Log an event
    void logEvent(const QString& action, const QString& userId, const QString& details);

signals:
    void entriesLoaded(Result<std::vector<AuditLogEntry>> result);
    void integrityVerified(Result<bool> result);
    void entryCountLoaded(Result<int> result);

private:
    Database* m_db;
    AuditLogRepository* m_auditRepo;
};
