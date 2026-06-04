#pragma once

#include "domain/User.h"
#include "domain/IrisTemplate.h"
#include "domain/AuditLog.h"
#include <string>
#include <vector>
#include <memory>

#ifdef HAS_SQLITE3
#include <sqlite3.h>
#endif

namespace iris {

/// Database abstraction for users, templates, and audit logs.
/// Uses SQLite3 if available, otherwise falls back to JSON file storage.
class Database {
public:
    Database();
    ~Database();

    /// Open/create the database
    bool open(const std::string& path);

    /// Close the database
    void close();

    /// Check if database is open
    bool isOpen() const;

    // ── User operations ────────────────────────────────────────

    bool insertUser(const User& user);
    bool deleteUser(const std::string& userId);
    std::vector<User> getAllUsers();
    User getUser(const std::string& userId);

    // ── Template operations ────────────────────────────────────

    bool insertTemplate(const IrisTemplate& tmpl);
    bool deleteTemplate(const std::string& templateId);
    std::vector<IrisTemplate> getTemplatesForUser(const std::string& userId);
    std::vector<IrisTemplate> getAllTemplates();
    IrisTemplate getTemplate(const std::string& templateId);

    // ── Audit log operations ───────────────────────────────────

    void logAction(const std::string& action, const std::string& userId = "",
                   const std::string& details = "");
    std::vector<AuditLogEntry> getAuditLogs(int limit = 100);

private:
#ifdef HAS_SQLITE3
    sqlite3* m_db = nullptr;
    bool createTables();
    bool executeSql(const std::string& sql);
#endif

    std::string m_path;
    bool m_opened = false;

    // JSON-based fallback storage
    bool loadFromFile();
    bool saveToFile();

    std::vector<User> m_users;
    std::vector<IrisTemplate> m_templates;
    std::vector<AuditLogEntry> m_auditLogs;
};

} // namespace iris
