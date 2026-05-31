#pragma once

#include <QObject>
#include <QThreadPool>
#include <memory>

#include "util/Result.h"
#include "domain/User.h"
#include "crypto/SecureBuffer.h"

class Database;
class UserRepository;
class WalletRepository;
class AuditLogRepository;
class SessionMonitor;
class SecurityService;

/**
 * AuthService - handles user registration, login, logout, session management.
 * All KDF operations are dispatched to the crypto thread pool.
 *
 * Per architecture §8: after MAX_FAILED_ATTEMPTS (=3) consecutive failed logins,
 * triggers SecurityService::wipeUserWalletData() instead of just locking the account.
 */
class AuthService : public QObject {
    Q_OBJECT
public:
    explicit AuthService(Database* db, SecurityService* securityService,
                        QThreadPool* cryptoPool, QObject* parent = nullptr);
    ~AuthService();

    // Register a new user
    void registerUser(const QString& username, const QString& password);

    // Login
    void login(const QString& username, const QString& password);

    // Logout current user
    void logout();

    // Change password (requires current password)
    void changePassword(const QString& oldPassword, const QString& newPassword);

    // Session state
    bool isLoggedIn() const { return m_loggedIn; }
    QString currentUserId() const { return m_currentUserId; }
    QString currentUsername() const { return m_currentUsername; }
    SecureBuffer& currentDEK() { return m_currentDEK; }
    const SecureBuffer& currentDEK() const { return m_currentDEK; }

    // Session monitor
    SessionMonitor* sessionMonitor() { return m_sessionMonitor; }
    void resetAutoLock();

signals:
    void registerCompleted(bool success, QString errorMessage);
    void loginCompleted(bool success, QString errorMessage);
    void logoutCompleted();
    void passwordChangeCompleted(bool success, QString errorMessage);
    void sessionExpired();
    void autoLockWarning(int secondsLeft);

    // Internal signals for data passing (not used by QML directly)
    void userRegistered(User user);
    void userLoggedIn(User user);

private:
    Database* m_db;
    UserRepository* m_userRepo;
    WalletRepository* m_walletRepo;
    AuditLogRepository* m_auditRepo;
    SecurityService* m_securityService;
    QThreadPool* m_cryptoPool;
    SessionMonitor* m_sessionMonitor;

    // Session state
    bool m_loggedIn = false;
    QString m_currentUserId;
    QString m_currentUsername;
    SecureBuffer m_currentDEK;

    // Clear in-memory secrets (per architecture §9.1)
    void clearSessionSecrets();

    // Handle login failure: increment attempts, optionally trigger wipe
    void handleLoginFailure(const User& user);

    // Internal helpers
    void handleRegisterCryptoDone(const QString& username, const QString& password,
                                   Result<SecureBuffer> salt,
                                   Result<SecureBuffer> passwordHash);
    void handleLoginCryptoDone(const QString& username, const QString& password,
                                Result<SecureBuffer> computedHash);
    void handleChangePasswordCryptoDone(const QString& oldPassword, const QString& newPassword);
};
