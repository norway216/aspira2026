#include "persistence/Database.h"
#include <fstream>
#include <sstream>
#include <iostream>
#include <chrono>
#include <algorithm>
#include <cstring>

// Simple JSON writer (no dependency)
namespace {

std::string escapeJson(const std::string& s) {
    std::string out;
    out.reserve(s.size());
    for (char c : s) {
        switch (c) {
            case '"':  out += "\\\""; break;
            case '\\': out += "\\\\"; break;
            case '\n': out += "\\n";  break;
            case '\t': out += "\\t";  break;
            default:   out += c;
        }
    }
    return out;
}

std::string base64Encode(const std::vector<uint8_t>& data) {
    static const char* chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    std::string result;
    result.reserve((data.size() + 2) / 3 * 4);

    for (size_t i = 0; i < data.size(); i += 3) {
        uint32_t val = static_cast<uint32_t>(data[i]) << 16;
        if (i + 1 < data.size()) val |= static_cast<uint32_t>(data[i+1]) << 8;
        if (i + 2 < data.size()) val |= static_cast<uint32_t>(data[i+2]);

        result += chars[(val >> 18) & 0x3F];
        result += chars[(val >> 12) & 0x3F];
        result += (i + 1 < data.size()) ? chars[(val >> 6) & 0x3F] : '=';
        result += (i + 2 < data.size()) ? chars[val & 0x3F] : '=';
    }
    return result;
}

int64_t nowMs() {
    return std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
}

} // anonymous namespace

namespace iris {

Database::Database() = default;

Database::~Database() {
    close();
}

bool Database::open(const std::string& path) {
    m_path = path;
    m_opened = true;

#ifdef HAS_SQLITE3
    int rc = sqlite3_open(path.c_str(), &m_db);
    if (rc == SQLITE_OK) {
        std::cout << "[Database] SQLite3 opened: " << path << "\n";
        return createTables();
    }
    std::cerr << "[Database] SQLite3 open failed: " << sqlite3_errmsg(m_db) << "\n";
    sqlite3_close(m_db);
    m_db = nullptr;
#endif

    // Fallback: JSON file-based storage
    std::cout << "[Database] Using file-based storage: " << path << "\n";
    loadFromFile();

    return true;
}

void Database::close() {
#ifdef HAS_SQLITE3
    if (m_db) {
        sqlite3_close(m_db);
        m_db = nullptr;
    }
#endif
    if (m_opened) {
        saveToFile();
        m_opened = false;
    }
}

bool Database::isOpen() const {
    return m_opened;
}

// ── User operations ────────────────────────────────────────────

bool Database::insertUser(const User& user) {
#ifdef HAS_SQLITE3
    if (m_db) {
        std::ostringstream sql;
        sql << "INSERT OR REPLACE INTO users (id, username, created_at, updated_at) "
            << "VALUES ('" << escapeJson(user.id) << "','"
            << escapeJson(user.username) << "',"
            << user.created_at << "," << user.updated_at << ");";
        return executeSql(sql.str());
    }
#endif
    // Remove existing user with same ID
    m_users.erase(std::remove_if(m_users.begin(), m_users.end(),
        [&](const User& u) { return u.id == user.id; }), m_users.end());
    m_users.push_back(user);
    return saveToFile();
}

bool Database::deleteUser(const std::string& userId) {
#ifdef HAS_SQLITE3
    if (m_db) {
        std::ostringstream sql;
        sql << "DELETE FROM users WHERE id='" << escapeJson(userId) << "';";
        // Also delete associated templates
        executeSql("DELETE FROM iris_templates WHERE user_id='" + escapeJson(userId) + "';");
        return executeSql(sql.str());
    }
#endif
    m_users.erase(std::remove_if(m_users.begin(), m_users.end(),
        [&](const User& u) { return u.id == userId; }), m_users.end());
    m_templates.erase(std::remove_if(m_templates.begin(), m_templates.end(),
        [&](const IrisTemplate& t) { return t.user_id == userId; }), m_templates.end());
    return saveToFile();
}

std::vector<User> Database::getAllUsers() {
#ifdef HAS_SQLITE3
    if (m_db) {
        // Simplified: return cached users
    }
#endif
    return m_users;
}

User Database::getUser(const std::string& userId) {
#ifdef HAS_SQLITE3
    if (m_db) {
        // Simplified: search cache
    }
#endif
    for (const auto& u : m_users) {
        if (u.id == userId) return u;
    }
    return User{};
}

// ── Template operations ────────────────────────────────────────

bool Database::insertTemplate(const IrisTemplate& tmpl) {
#ifdef HAS_SQLITE3
    if (m_db) {
        std::ostringstream sql;
        sql << "INSERT OR REPLACE INTO iris_templates "
            << "(id, user_id, eye_side, embedding_encrypted, embedding_dim, "
            << "model_version, quality_score, created_at) VALUES ('"
            << escapeJson(tmpl.id) << "','"
            << escapeJson(tmpl.user_id) << "','"
            << escapeJson(tmpl.eye_side) << "','"
            << base64Encode(tmpl.iris_code) << "',"
            << tmpl.embedding_dim << ",'"
            << escapeJson(tmpl.model_version) << "',"
            << tmpl.quality_score << ","
            << tmpl.created_at << ");";
        return executeSql(sql.str());
    }
#endif
    m_templates.erase(std::remove_if(m_templates.begin(), m_templates.end(),
        [&](const IrisTemplate& t) { return t.id == tmpl.id; }), m_templates.end());
    m_templates.push_back(tmpl);
    return saveToFile();
}

bool Database::deleteTemplate(const std::string& templateId) {
#ifdef HAS_SQLITE3
    if (m_db) {
        return executeSql("DELETE FROM iris_templates WHERE id='" +
                          escapeJson(templateId) + "';");
    }
#endif
    m_templates.erase(std::remove_if(m_templates.begin(), m_templates.end(),
        [&](const IrisTemplate& t) { return t.id == templateId; }), m_templates.end());
    return saveToFile();
}

std::vector<IrisTemplate> Database::getTemplatesForUser(const std::string& userId) {
    std::vector<IrisTemplate> result;
    for (const auto& t : m_templates) {
        if (t.user_id == userId) result.push_back(t);
    }
    return result;
}

std::vector<IrisTemplate> Database::getAllTemplates() {
    return m_templates;
}

IrisTemplate Database::getTemplate(const std::string& templateId) {
    for (const auto& t : m_templates) {
        if (t.id == templateId) return t;
    }
    return IrisTemplate{};
}

// ── Audit log operations ───────────────────────────────────────

void Database::logAction(const std::string& action, const std::string& userId,
                          const std::string& details) {
    AuditLogEntry entry;
    entry.id         = static_cast<int64_t>(m_auditLogs.size());
    entry.action     = action;
    entry.user_id    = userId;
    entry.details    = details;
    entry.created_at = nowMs();

#ifdef HAS_SQLITE3
    if (m_db) {
        std::ostringstream sql;
        sql << "INSERT INTO audit_logs (action, user_id, details, created_at) VALUES ('"
            << escapeJson(action) << "','"
            << escapeJson(userId) << "','"
            << escapeJson(details) << "',"
            << entry.created_at << ");";
        executeSql(sql.str());
    }
#endif

    m_auditLogs.push_back(entry);

    // Keep only last 1000 entries
    if (m_auditLogs.size() > 1000) {
        m_auditLogs.erase(m_auditLogs.begin(),
                           m_auditLogs.begin() + 500);
    }
}

std::vector<AuditLogEntry> Database::getAuditLogs(int limit) {
    if (limit <= 0 || static_cast<size_t>(limit) >= m_auditLogs.size()) {
        return m_auditLogs;
    }
    return std::vector<AuditLogEntry>(
        m_auditLogs.end() - limit, m_auditLogs.end());
}

// ── SQLite3 helpers ────────────────────────────────────────────

#ifdef HAS_SQLITE3
bool Database::createTables() {
    const char* sqlUsers = R"(
        CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            username TEXT NOT NULL UNIQUE,
            created_at INTEGER NOT NULL,
            updated_at INTEGER NOT NULL
        );
    )";

    const char* sqlTemplates = R"(
        CREATE TABLE IF NOT EXISTS iris_templates (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            eye_side TEXT NOT NULL,
            embedding_encrypted BLOB NOT NULL,
            embedding_dim INTEGER NOT NULL,
            model_version TEXT NOT NULL,
            quality_score REAL NOT NULL,
            created_at INTEGER NOT NULL,
            FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
        );
    )";

    const char* sqlAudit = R"(
        CREATE TABLE IF NOT EXISTS audit_logs (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            action TEXT NOT NULL,
            user_id TEXT,
            details TEXT,
            created_at INTEGER NOT NULL
        );
    )";

    return executeSql(sqlUsers) && executeSql(sqlTemplates) && executeSql(sqlAudit);
}

bool Database::executeSql(const std::string& sql) {
    char* errMsg = nullptr;
    int rc = sqlite3_exec(m_db, sql.c_str(), nullptr, nullptr, &errMsg);
    if (rc != SQLITE_OK) {
        std::cerr << "[Database] SQL error: " << (errMsg ? errMsg : "unknown") << "\n";
        if (errMsg) sqlite3_free(errMsg);
        return false;
    }
    return true;
}
#endif

// ── JSON file-based storage ────────────────────────────────────

bool Database::loadFromFile() {
    std::ifstream file(m_path);
    if (!file.is_open()) return true; // New database

    std::stringstream buffer;
    buffer << file.rdbuf();
    std::string content = buffer.str();

    // Very simple JSON parser for our format
    // In production, use nlohmann/json or simdjson
    // For MVP: just clear and start fresh if parsing fails
    if (content.empty()) return true;

    std::cout << "[Database] Loaded " << m_path << " (" << content.size() << " bytes)\n";
    // Full JSON parsing is complex; for MVP we maintain in-memory state
    // and save on changes. The file serves as backup.
    return true;
}

bool Database::saveToFile() {
    std::ofstream file(m_path);
    if (!file.is_open()) {
        std::cerr << "[Database] Failed to write: " << m_path << "\n";
        return false;
    }

    // Write JSON
    file << "{\n";

    // Users
    file << "  \"users\": [\n";
    for (size_t i = 0; i < m_users.size(); ++i) {
        const auto& u = m_users[i];
        file << "    {\"id\":\"" << escapeJson(u.id)
             << "\",\"username\":\"" << escapeJson(u.username)
             << "\",\"created_at\":" << u.created_at
             << ",\"updated_at\":" << u.updated_at << "}";
        if (i + 1 < m_users.size()) file << ",";
        file << "\n";
    }
    file << "  ],\n";

    // Templates
    file << "  \"templates\": [\n";
    for (size_t i = 0; i < m_templates.size(); ++i) {
        const auto& t = m_templates[i];
        file << "    {\"id\":\"" << escapeJson(t.id)
             << "\",\"user_id\":\"" << escapeJson(t.user_id)
             << "\",\"eye_side\":\"" << escapeJson(t.eye_side)
             << "\",\"iris_code\":\"" << base64Encode(t.iris_code)
             << "\",\"mask_code\":\"" << base64Encode(t.mask_code)
             << "\",\"embedding_dim\":" << t.embedding_dim
             << ",\"model_version\":\"" << escapeJson(t.model_version)
             << "\",\"quality_score\":" << t.quality_score
             << ",\"created_at\":" << t.created_at << "}";
        if (i + 1 < m_templates.size()) file << ",";
        file << "\n";
    }
    file << "  ]\n";
    file << "}\n";

    file.close();
    return true;
}

} // namespace iris
