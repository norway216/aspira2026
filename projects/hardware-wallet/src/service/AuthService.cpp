#include "AuthService.h"
#include "SecurityService.h"
#include "persistence/Database.h"
#include "persistence/UserRepository.h"
#include "persistence/WalletRepository.h"
#include "persistence/AuditLogRepository.h"
#include "domain/AuditLogEntry.h"
#include "worker/CryptoTask.h"
#include "worker/SessionMonitor.h"
#include "crypto/KDFHelper.h"
#include "crypto/KeyManager.h"
#include "crypto/AEADHelper.h"
#include "util/CryptoUtils.h"
#include "app/Constants.h"

#include <QJsonDocument>
#include <QJsonObject>
#include <QDebug>

AuthService::AuthService(Database* db, SecurityService* securityService,
                         QThreadPool* cryptoPool, QObject* parent)
    : QObject(parent)
    , m_db(db)
    , m_securityService(securityService)
    , m_cryptoPool(cryptoPool)
{
    m_userRepo = new UserRepository(db);
    m_walletRepo = new WalletRepository(db);
    m_auditRepo = new AuditLogRepository(db);
    m_sessionMonitor = new SessionMonitor(this);

    connect(m_sessionMonitor, &SessionMonitor::sessionExpired, this, [this]() {
        if (m_loggedIn) {
            m_auditRepo->append(AuditLogEntry::ACTION_SESSION_EXPIRED, m_currentUserId, "{}");
            logout();
            emit sessionExpired();
        }
    });

    connect(m_sessionMonitor, &SessionMonitor::lockWarning, this, [this](int seconds) {
        emit autoLockWarning(seconds);
    });
}

AuthService::~AuthService() {
    delete m_userRepo;
    delete m_walletRepo;
    delete m_auditRepo;
}

void AuthService::registerUser(const QString& username, const QString& password) {
    if (username.length() < 3) {
        emit registerCompleted(false, "Username must be at least 3 characters");
        emit userRegistered(User{});
        return;
    }

    if (password.length() < AppConstants::MIN_PASSWORD_LENGTH) {
        emit registerCompleted(false, QString("Password must be at least %1 characters").arg(AppConstants::MIN_PASSWORD_LENGTH));
        emit userRegistered(User{});
        return;
    }

    auto existing = m_userRepo->findByUsername(username);
    if (existing.isOk() && existing.value().has_value()) {
        emit registerCompleted(false, QString("User '%1' already exists").arg(username));
        emit userRegistered(User{});
        return;
    }

    // Generate salt (stored as QByteArray)
    auto saltResult = KeyManager::createSalt();
    if (saltResult.isFail()) {
        emit registerCompleted(false, saltResult.error().message);
        emit userRegistered(User{});
        return;
    }
    QByteArray salt = saltResult.value().toQByteArray();

    // Generate password hash
    auto hashResult = KDFHelper::hashPassword(password, saltResult.value());
    if (hashResult.isFail()) {
        emit registerCompleted(false, hashResult.error().message);
        emit userRegistered(User{});
        return;
    }
    QByteArray passwordHash = hashResult.value().toHex().toUtf8();

    // Generate DEK
    auto dekResult = KeyManager::createDEK();
    if (dekResult.isFail()) {
        emit registerCompleted(false, dekResult.error().message);
        emit userRegistered(User{});
        return;
    }
    SecureBuffer dek = std::move(dekResult.value());

    // Derive KEK and wrap DEK
    auto kekResult = KDFHelper::deriveKEK(password, saltResult.value());
    if (kekResult.isFail()) {
        emit registerCompleted(false, kekResult.error().message);
        emit userRegistered(User{});
        return;
    }

    auto wrappedDekResult = KeyManager::wrapDEK(dek, kekResult.value());
    if (wrappedDekResult.isFail()) {
        emit registerCompleted(false, wrappedDekResult.error().message);
        emit userRegistered(User{});
        return;
    }
    QByteArray encryptedDEK = wrappedDekResult.value().toQByteArray();

    // Create user
    User user;
    user.id = CryptoUtils::generateId();
    user.username = username;
    user.encryptedDEK = encryptedDEK;
    user.passwordVerifier = passwordHash;
    user.salt = salt;
    user.kdfOpsLimit = AppConstants::KDF_OPS_LIMIT;
    user.kdfMemLimit = AppConstants::KDF_MEM_LIMIT;
    user.createdAt = CryptoUtils::currentTimestampMs();

    auto insertResult = m_userRepo->insert(user);
    if (insertResult.isFail()) {
        emit registerCompleted(false, insertResult.error().message);
        emit userRegistered(User{});
        return;
    }

    m_auditRepo->append(AuditLogEntry::ACTION_REGISTER, user.id,
        QString("{\"username\":\"%1\"}").arg(username));
    qDebug() << "User registered:" << username << "id:" << user.id;
    emit userRegistered(user);
    emit registerCompleted(true, {});
}

void AuthService::login(const QString& username, const QString& password) {
    if (m_loggedIn) {
        emit loginCompleted(false, "Already logged in");
        emit userLoggedIn(User{});
        return;
    }

    auto existing = m_userRepo->findByUsername(username);
    if (existing.isFail()) {
        emit loginCompleted(false, existing.error().message);
        emit userLoggedIn(User{});
        return;
    }

    auto optUser = existing.value();
    if (!optUser.has_value()) {
        emit loginCompleted(false, QString("User '%1' not found").arg(username));
        emit userLoggedIn(User{});
        return;
    }

    User user = optUser.value();

    // Check if wallet was wiped (per architecture §8.2)
    if (user.walletWiped) {
        emit loginCompleted(false,
            "Wallet data has been wiped. Please restore from encrypted backup.");
        emit userLoggedIn(User{});
        return;
    }

    // Check if account is temporarily locked
    if (user.isLocked()) {
        qint64 remainingMs = user.lockedUntil - CryptoUtils::currentTimestampMs();
        int remainingSec = static_cast<int>(remainingMs / 1000);
        emit loginCompleted(false,
            QString("Account locked. Try again in %1 seconds").arg(remainingSec));
        emit userLoggedIn(User{});
        return;
    }

    // Convert stored data to SecureBuffer for crypto ops
    SecureBuffer storedSalt = SecureBuffer::fromQByteArray(user.salt);
    SecureBuffer storedHash = SecureBuffer::fromHex(QString::fromUtf8(user.passwordVerifier));

    // Hash the provided password
    auto hashResult = KDFHelper::hashPassword(password, storedSalt,
                                               user.kdfOpsLimit, user.kdfMemLimit);
    if (hashResult.isFail()) {
        m_userRepo->incrementFailedAttempts(user.id);
        emit loginCompleted(false, hashResult.error().message);
        emit userLoggedIn(User{});
        return;
    }

    bool passwordMatch = CryptoUtils::secureCompare(hashResult.value(), storedHash);
    if (!passwordMatch) {
        handleLoginFailure(user);
        emit userLoggedIn(User{});
        return;
    }

    // Unwrap DEK
    auto kekResult = KDFHelper::deriveKEK(password, storedSalt,
                                           user.kdfOpsLimit, user.kdfMemLimit);
    if (kekResult.isFail()) {
        emit loginCompleted(false, kekResult.error().message);
        emit userLoggedIn(User{});
        return;
    }

    SecureBuffer encryptedDek = SecureBuffer::fromQByteArray(user.encryptedDEK);
    auto dekResult = KeyManager::unwrapDEK(encryptedDek, kekResult.value());
    if (dekResult.isFail()) {
        emit loginCompleted(false, "Failed to unwrap DEK");
        emit userLoggedIn(User{});
        return;
    }

    // Reset failed attempts on successful login
    m_userRepo->resetFailedAttempts(user.id);
    m_userRepo->updateLoginTime(user.id, CryptoUtils::currentTimestampMs());

    // Set session state
    m_loggedIn = true;
    m_currentUserId = user.id;
    m_currentUsername = user.username;
    m_currentDEK = std::move(dekResult.value());

    m_sessionMonitor->reset();
    m_sessionMonitor->start();

    m_auditRepo->append(AuditLogEntry::ACTION_LOGIN, user.id,
        QString("{\"username\":\"%1\"}").arg(username));
    qDebug() << "User logged in:" << username;
    emit userLoggedIn(user);
    emit loginCompleted(true, {});
}

void AuthService::logout() {
    if (!m_loggedIn) {
        emit logoutCompleted();
        return;
    }

    m_auditRepo->append(AuditLogEntry::ACTION_LOGOUT, m_currentUserId, "{}");
    clearSessionSecrets();
    qDebug() << "User logged out";
    emit logoutCompleted();
}

void AuthService::changePassword(const QString& oldPassword, const QString& newPassword) {
    if (!m_loggedIn) {
        emit passwordChangeCompleted(false, "Not logged in");
        return;
    }

    if (newPassword.length() < AppConstants::MIN_PASSWORD_LENGTH) {
        emit passwordChangeCompleted(false,
            QString("Password must be at least %1 characters").arg(AppConstants::MIN_PASSWORD_LENGTH));
        return;
    }

    auto userResult = m_userRepo->findById(m_currentUserId);
    if (userResult.isFail() || !userResult.value().has_value()) {
        emit passwordChangeCompleted(false, "Current user not found");
        return;
    }

    User user = userResult.value().value();
    SecureBuffer storedSalt = SecureBuffer::fromQByteArray(user.salt);
    SecureBuffer storedHash = SecureBuffer::fromHex(QString::fromUtf8(user.passwordVerifier));

    auto oldHashResult = KDFHelper::hashPassword(oldPassword, storedSalt,
                                                  user.kdfOpsLimit, user.kdfMemLimit);
    if (oldHashResult.isFail() || !CryptoUtils::secureCompare(oldHashResult.value(), storedHash)) {
        emit passwordChangeCompleted(false, "Current password is incorrect");
        return;
    }

    auto newSaltResult = KeyManager::createSalt();
    QByteArray newSalt = newSaltResult.value().toQByteArray();
    auto newHashResult = KDFHelper::hashPassword(newPassword, newSaltResult.value());
    QByteArray newHash = newHashResult.value().toHex().toUtf8();

    auto newKekResult = KDFHelper::deriveKEK(newPassword, newSaltResult.value());
    auto newWrappedDekResult = KeyManager::wrapDEK(m_currentDEK, newKekResult.value());
    QByteArray newEncryptedDEK = newWrappedDekResult.value().toQByteArray();

    m_userRepo->updatePasswordVerifier(m_currentUserId, newHash, newEncryptedDEK);
    m_auditRepo->append(AuditLogEntry::ACTION_PASSWORD_CHANGE, m_currentUserId, "{}");
    emit passwordChangeCompleted(true, {});
}

void AuthService::handleLoginFailure(const User& user)
{
    // 1. Increment failed attempts
    auto incRes = m_userRepo->incrementFailedAttempts(user.id);
    if (incRes.isFail()) {
        emit loginCompleted(false, incRes.error().message);
        return;
    }

    // 2. Refresh user to get updated attempt count
    auto refreshedRes = m_userRepo->findById(user.id);
    if (refreshedRes.isFail() || !refreshedRes.value().has_value()) {
        emit loginCompleted(false, "Failed to reload user state");
        return;
    }

    User refreshed = refreshedRes.value().value();

    // 3. Write LOGIN_FAILED audit entry
    QJsonObject details;
    details["failed_attempts"] = refreshed.failedAttempts;
    details["max_attempts"] = AppConstants::MAX_FAILED_ATTEMPTS;

    m_auditRepo->append(
        AuditLogEntry::ACTION_LOGIN_FAILED,
        user.id,
        QString::fromUtf8(QJsonDocument(details).toJson(QJsonDocument::Compact))
    );

    // 4. Check if threshold reached -> trigger self-destruct
    if (refreshed.failedAttempts >= AppConstants::MAX_FAILED_ATTEMPTS) {
        qWarning() << "Max failed attempts reached for user:" << user.username
                    << "— wiping wallet data";

        if constexpr (AppConstants::WIPE_WALLET_ON_MAX_FAILED_ATTEMPTS) {
            auto wipeRes = m_securityService->wipeUserWalletData(
                user.id,
                QString("Password failed %1 times").arg(AppConstants::MAX_FAILED_ATTEMPTS)
            );

            // Clear in-memory secrets regardless
            clearSessionSecrets();

            if (wipeRes.isFail()) {
                emit loginCompleted(false,
                    QString("Password failed %1 times, but wallet wipe failed: %2")
                        .arg(AppConstants::MAX_FAILED_ATTEMPTS)
                        .arg(wipeRes.error().message));
                return;
            }

            emit loginCompleted(false,
                QString("Password failed %1 times. Wallet data has been securely wiped.")
                    .arg(AppConstants::MAX_FAILED_ATTEMPTS));
            return;
        } else {
            // Fallback: lock account instead
            qint64 lockUntil = CryptoUtils::currentTimestampMs()
                               + AppConstants::LOCKOUT_DURATION_SECONDS * 1000;
            m_userRepo->lockUser(user.id, lockUntil);
            emit loginCompleted(false,
                QString("Account locked for %1 seconds").arg(AppConstants::LOCKOUT_DURATION_SECONDS));
            return;
        }
    }

    // 5. Return remaining attempts warning
    int remaining = AppConstants::MAX_FAILED_ATTEMPTS - refreshed.failedAttempts;

    if (remaining == 1) {
        emit loginCompleted(false,
            QString("Invalid password. %1 attempt remaining. "
                    "WARNING: Another incorrect attempt will permanently delete your wallet data.")
                .arg(remaining));
    } else {
        emit loginCompleted(false,
            QString("Invalid password. %1 attempts remaining before wallet data is wiped.")
                .arg(remaining));
    }
}

void AuthService::clearSessionSecrets()
{
    if (m_currentDEK.size() > 0) {
        m_currentDEK.clear();
    }

    m_currentUserId.clear();
    m_currentUsername.clear();
    m_loggedIn = false;

    if (m_sessionMonitor) {
        m_sessionMonitor->stop();
    }
}

void AuthService::resetAutoLock() {
    if (m_sessionMonitor) m_sessionMonitor->reset();
}
