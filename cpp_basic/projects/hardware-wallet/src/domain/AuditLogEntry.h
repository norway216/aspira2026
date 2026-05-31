#pragma once

#include <QString>
#include <QDateTime>

struct AuditLogEntry {
    qint64 id = 0;                 // Auto-increment
    QString action;                // "LOGIN", "LOGOUT", "REGISTER", "SEND_TX", etc.
    QString userId;
    QString details;               // JSON string with event details
    QString previousHash;          // Hex-encoded BLAKE2b hash of previous entry
    QString hash;                  // Hex-encoded BLAKE2b hash of this entry
    qint64 createdAt = 0;

    // Well-known action types
    static const char* ACTION_LOGIN;
    static const char* ACTION_LOGOUT;
    static const char* ACTION_REGISTER;
    static const char* ACTION_SEND_TX;
    static const char* ACTION_RECEIVE_TX;
    static const char* ACTION_BACKUP_EXPORT;
    static const char* ACTION_BACKUP_IMPORT;
    static const char* ACTION_PASSWORD_CHANGE;
    static const char* ACTION_WALLET_CREATE;
    static const char* ACTION_SESSION_EXPIRED;
    static const char* ACTION_ACCOUNT_LOCKED;
    static const char* ACTION_LOGIN_FAILED;
    static const char* ACTION_SECURITY_SELF_DESTRUCT;
    static const char* ACTION_WALLET_DELETE;
    static const char* ACTION_WALLET_WIPED;
};

inline const char* AuditLogEntry::ACTION_LOGIN = "LOGIN";
inline const char* AuditLogEntry::ACTION_LOGOUT = "LOGOUT";
inline const char* AuditLogEntry::ACTION_REGISTER = "REGISTER";
inline const char* AuditLogEntry::ACTION_SEND_TX = "SEND_TX";
inline const char* AuditLogEntry::ACTION_RECEIVE_TX = "RECEIVE_TX";
inline const char* AuditLogEntry::ACTION_BACKUP_EXPORT = "BACKUP_EXPORT";
inline const char* AuditLogEntry::ACTION_BACKUP_IMPORT = "BACKUP_IMPORT";
inline const char* AuditLogEntry::ACTION_PASSWORD_CHANGE = "PASSWORD_CHANGE";
inline const char* AuditLogEntry::ACTION_WALLET_CREATE = "WALLET_CREATE";
inline const char* AuditLogEntry::ACTION_WALLET_DELETE = "WALLET_DELETE";
inline const char* AuditLogEntry::ACTION_SESSION_EXPIRED = "SESSION_EXPIRED";
inline const char* AuditLogEntry::ACTION_ACCOUNT_LOCKED = "ACCOUNT_LOCKED";
inline const char* AuditLogEntry::ACTION_LOGIN_FAILED = "LOGIN_FAILED";
inline const char* AuditLogEntry::ACTION_SECURITY_SELF_DESTRUCT = "SECURITY_SELF_DESTRUCT";
inline const char* AuditLogEntry::ACTION_WALLET_WIPED = "WALLET_WIPED";
