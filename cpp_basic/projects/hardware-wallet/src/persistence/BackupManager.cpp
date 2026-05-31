#include "BackupManager.h"
#include "Database.h"
#include "UserRepository.h"
#include "WalletRepository.h"
#include "TransactionRepository.h"
#include "AuditLogRepository.h"
#include "crypto/KDFHelper.h"
#include "crypto/AEADHelper.h"
#include "crypto/KeyManager.h"
#include "crypto/CryptoConfig.h"
#include "util/CryptoUtils.h"

#include <QFile>
#include <QSqlQuery>
#include <QJsonDocument>
#include <QJsonArray>
#include <QJsonObject>
#include <QDebug>
#include <cstring>

BackupManager::BackupManager(Database* db) : m_db(db) {}

Result<void> BackupManager::exportBackup(const QString& filePath, const QString& backupPassword) {
    auto payloadResult = buildPayload();
    if (payloadResult.isFail()) {
        return Result<void>::fail(payloadResult.error().code, payloadResult.error().message);
    }

    QJsonDocument doc(payloadResult.value());
    QByteArray payloadBytes = doc.toJson(QJsonDocument::Compact);

    auto saltResult = KeyManager::createSalt();
    if (saltResult.isFail()) return Result<void>::fail(saltResult.error().code, saltResult.error().message);

    unsigned long long opsLimit = 4ULL;
    size_t memLimit = 268435456ULL;

    auto kekResult = KDFHelper::deriveBackupKey(backupPassword, saltResult.value(), opsLimit, memLimit);
    if (kekResult.isFail()) {
        return Result<void>::fail(kekResult.error().code, "Failed to derive backup key");
    }

    auto encryptResult = AEADHelper::encrypt(
        reinterpret_cast<const unsigned char*>(payloadBytes.constData()),
        payloadBytes.size(), kekResult.value().data());
    if (encryptResult.isFail()) {
        return Result<void>::fail(encryptResult.error().code, "Failed to encrypt backup");
    }

    const auto& salt = saltResult.value();
    const auto& encrypted = encryptResult.value();

    QByteArray backupData;
    backupData.append(reinterpret_cast<const char*>(salt.data()), static_cast<int>(salt.size()));
    backupData.append(reinterpret_cast<const char*>(&opsLimit), sizeof(opsLimit));
    backupData.append(reinterpret_cast<const char*>(&memLimit), sizeof(memLimit));
    backupData.append(reinterpret_cast<const char*>(encrypted.data()), static_cast<int>(encrypted.size()));

    QString tempPath = filePath + ".tmp";
    QFile tempFile(tempPath);
    if (!tempFile.open(QIODevice::WriteOnly)) {
        return Result<void>::fail(ErrorCode::FileIOError, "Failed to create temporary backup file");
    }

    qint64 written = tempFile.write(backupData);
    if (written != backupData.size()) {
        tempFile.remove();
        return Result<void>::fail(ErrorCode::FileIOError, "Failed to write backup data");
    }

    if (!tempFile.flush()) {
        tempFile.remove();
        return Result<void>::fail(ErrorCode::FileIOError, "Failed to flush backup file");
    }
    tempFile.close();

    if (QFile::exists(filePath)) QFile::remove(filePath);
    if (!QFile::rename(tempPath, filePath)) {
        QFile::remove(tempPath);
        return Result<void>::fail(ErrorCode::FileIOError, "Failed to finalize backup file");
    }

    qDebug() << "Backup exported to:" << filePath << "size:" << backupData.size() << "bytes";
    return Result<void>::ok();
}

Result<int> BackupManager::importBackup(const QString& filePath, const QString& backupPassword) {
    QFile file(filePath);
    if (!file.open(QIODevice::ReadOnly)) {
        return Result<int>::fail(ErrorCode::BackupFileNotFound, "Backup file not found");
    }

    QByteArray backupData = file.readAll();
    file.close();

    if (backupData.size() < static_cast<int>(CryptoConfig::SALT_SIZE + sizeof(unsigned long long) + sizeof(size_t) + CryptoConfig::ENCRYPTED_OVERHEAD + 1)) {
        return Result<int>::fail(ErrorCode::BackupFileCorrupted, "Backup file is too small");
    }

    const unsigned char* ptr = reinterpret_cast<const unsigned char*>(backupData.constData());

    SecureBuffer salt(CryptoConfig::SALT_SIZE);
    std::memcpy(salt.data(), ptr, CryptoConfig::SALT_SIZE);
    ptr += CryptoConfig::SALT_SIZE;

    unsigned long long opsLimit;
    std::memcpy(&opsLimit, ptr, sizeof(opsLimit));
    ptr += sizeof(opsLimit);

    size_t memLimit;
    std::memcpy(&memLimit, ptr, sizeof(memLimit));
    ptr += sizeof(memLimit);

    size_t encryptedSize = backupData.size() - (ptr - reinterpret_cast<const unsigned char*>(backupData.constData()));

    auto kekResult = KDFHelper::deriveBackupKey(backupPassword, salt, opsLimit, memLimit);
    if (kekResult.isFail()) {
        return Result<int>::fail(kekResult.error().code, "Failed to derive backup key");
    }

    auto decryptResult = AEADHelper::decrypt(ptr, encryptedSize, kekResult.value().data());
    if (decryptResult.isFail()) {
        return Result<int>::fail(ErrorCode::BackupDecryptionFailed, "Wrong backup password or corrupted file");
    }

    QByteArray jsonBytes(reinterpret_cast<const char*>(decryptResult.value().data()),
                         static_cast<int>(decryptResult.value().size()));

    QJsonParseError parseError;
    QJsonDocument doc = QJsonDocument::fromJson(jsonBytes, &parseError);
    if (parseError.error != QJsonParseError::NoError) {
        return Result<int>::fail(ErrorCode::BackupFileCorrupted,
            QString("Invalid JSON in backup: %1").arg(parseError.errorString()));
    }

    if (!doc.isObject()) {
        return Result<int>::fail(ErrorCode::BackupFileCorrupted, "Backup JSON is not an object");
    }

    QJsonObject payload = doc.object();
    if (payload.value("format").toString() != "rk3568-wallet-backup") {
        return Result<int>::fail(ErrorCode::BackupVersionMismatch, "Unknown backup format");
    }

    return restoreFromPayload(payload);
}

Result<QJsonObject> BackupManager::verifyBackup(const QString& filePath, const QString& backupPassword) {
    auto importResult = importBackup(filePath, backupPassword);
    if (importResult.isFail()) {
        QJsonObject errorObj;
        errorObj["valid"] = false;
        errorObj["error"] = importResult.error().message;
        return Result<QJsonObject>::ok(errorObj);
    }
    QJsonObject result;
    result["valid"] = true;
    result["users_restored"] = importResult.value();
    return Result<QJsonObject>::ok(result);
}

Result<QJsonObject> BackupManager::buildPayload() {
    UserRepository userRepo(m_db);
    WalletRepository walletRepo(m_db);
    TransactionRepository txRepo(m_db);
    AuditLogRepository auditRepo(m_db);

    QJsonObject payload;
    payload["format"] = "rk3568-wallet-backup";
    payload["version"] = 1;
    payload["created_at"] = CryptoUtils::currentTimestampMs();

    // Users
    auto usersResult = userRepo.findAll();
    if (usersResult.isFail()) return Result<QJsonObject>::fail(usersResult.error().code, usersResult.error().message);

    QJsonArray usersArray;
    for (const auto& user : usersResult.value()) {
        QJsonObject obj;
        obj["id"] = user.id;
        obj["username"] = user.username;
        obj["encrypted_dek"] = QString::fromLatin1(user.encryptedDEK.toBase64());
        obj["password_verifier"] = QString::fromLatin1(user.passwordVerifier);
        obj["salt"] = QString::fromLatin1(user.salt.toBase64());
        obj["kdf_ops_limit"] = static_cast<qint64>(user.kdfOpsLimit);
        obj["kdf_mem_limit"] = static_cast<qint64>(user.kdfMemLimit);
        obj["created_at"] = user.createdAt;
        obj["failed_attempts"] = user.failedAttempts;
        obj["locked_until"] = user.lockedUntil;
        usersArray.append(obj);
    }
    payload["users"] = usersArray;

    // Wallets
    auto walletsResult = walletRepo.findAll();
    if (walletsResult.isFail()) return Result<QJsonObject>::fail(walletsResult.error().code, walletsResult.error().message);

    QJsonArray walletsArray;
    for (const auto& wallet : walletsResult.value()) {
        QJsonObject obj;
        obj["id"] = wallet.id;
        obj["user_id"] = wallet.userId;
        obj["encrypted_seed"] = QString::fromLatin1(wallet.encryptedSeed.toBase64());
        obj["public_key"] = QString::fromLatin1(wallet.publicKey.toBase64());
        obj["balance"] = wallet.balance;
        obj["created_at"] = wallet.createdAt;
        walletsArray.append(obj);
    }
    payload["wallets"] = walletsArray;

    // Transactions
    auto txsResult = txRepo.findAll();
    if (txsResult.isFail()) return Result<QJsonObject>::fail(txsResult.error().code, txsResult.error().message);

    QJsonArray txsArray;
    for (const auto& tx : txsResult.value()) {
        QJsonObject obj;
        obj["id"] = tx.id;
        obj["wallet_id"] = tx.walletId;
        obj["to_public_key"] = tx.toPublicKey;
        obj["amount"] = tx.amount;
        obj["fee"] = tx.fee;
        obj["signature"] = QString::fromLatin1(tx.signature.toBase64());
        obj["nonce"] = tx.nonce;
        obj["status"] = tx.status;
        obj["memo"] = tx.memo;
        obj["created_at"] = tx.createdAt;
        txsArray.append(obj);
    }
    payload["transactions"] = txsArray;

    // Audit logs
    auto auditResult = auditRepo.getAllEntries();
    if (auditResult.isFail()) return Result<QJsonObject>::fail(auditResult.error().code, auditResult.error().message);

    QJsonArray auditArray;
    for (const auto& entry : auditResult.value()) {
        QJsonObject obj;
        obj["action"] = entry.action;
        obj["user_id"] = entry.userId;
        obj["details"] = entry.details;
        obj["previous_hash"] = entry.previousHash;
        obj["hash"] = entry.hash;
        obj["created_at"] = entry.createdAt;
        auditArray.append(obj);
    }
    payload["audit_logs"] = auditArray;

    QByteArray payloadBytes = QJsonDocument(payload).toJson(QJsonDocument::Compact);
    payload["integrity_hash"] = CryptoUtils::hashDataHex(payloadBytes);

    return Result<QJsonObject>::ok(payload);
}

Result<int> BackupManager::restoreFromPayload(const QJsonObject& payload) {
    UserRepository userRepo(m_db);
    WalletRepository walletRepo(m_db);
    TransactionRepository txRepo(m_db);
    AuditLogRepository auditRepo(m_db);

    auto txResult = m_db->beginTransaction();
    if (txResult.isFail()) return Result<int>::fail(txResult.error().code, txResult.error().message);

    int usersRestored = 0;

    QJsonArray usersArray = payload.value("users").toArray();
    for (const auto& userVal : usersArray) {
        QJsonObject obj = userVal.toObject();
        User user;
        user.id = obj.value("id").toString();
        user.username = obj.value("username").toString();
        user.encryptedDEK = QByteArray::fromBase64(obj.value("encrypted_dek").toString().toLatin1());
        user.passwordVerifier = obj.value("password_verifier").toString().toLatin1();
        user.salt = QByteArray::fromBase64(obj.value("salt").toString().toLatin1());
        user.kdfOpsLimit = static_cast<unsigned long long>(obj.value("kdf_ops_limit").toDouble());
        user.kdfMemLimit = static_cast<size_t>(obj.value("kdf_mem_limit").toDouble());
        user.createdAt = obj.value("created_at").toVariant().toLongLong();
        user.failedAttempts = obj.value("failed_attempts").toInt(0);
        user.lockedUntil = obj.value("locked_until").toVariant().toLongLong();

        auto existing = userRepo.findByUsername(user.username);
        if (existing.isOk() && existing.value().has_value()) continue;

        if (userRepo.insert(user).isOk()) usersRestored++;
    }

    QJsonArray walletsArray = payload.value("wallets").toArray();
    for (const auto& wv : walletsArray) {
        QJsonObject obj = wv.toObject();
        Wallet wallet;
        wallet.id = obj.value("id").toString();
        wallet.userId = obj.value("user_id").toString();
        wallet.encryptedSeed = QByteArray::fromBase64(obj.value("encrypted_seed").toString().toLatin1());
        wallet.publicKey = QByteArray::fromBase64(obj.value("public_key").toString().toLatin1());
        wallet.balance = obj.value("balance").toDouble();
        wallet.createdAt = obj.value("created_at").toVariant().toLongLong();

        auto existing = walletRepo.findById(wallet.id);
        if (existing.isOk() && existing.value().has_value()) continue;
        walletRepo.insert(wallet);
    }

    QJsonArray txsArray = payload.value("transactions").toArray();
    for (const auto& tv : txsArray) {
        QJsonObject obj = tv.toObject();
        Transaction tx;
        tx.id = obj.value("id").toString();
        tx.walletId = obj.value("wallet_id").toString();
        tx.toPublicKey = obj.value("to_public_key").toString();
        tx.amount = obj.value("amount").toDouble();
        tx.fee = obj.value("fee").toDouble();
        tx.signature = QByteArray::fromBase64(obj.value("signature").toString().toLatin1());
        tx.nonce = obj.value("nonce").toString();
        tx.status = obj.value("status").toInt();
        tx.memo = obj.value("memo").toString();
        tx.createdAt = obj.value("created_at").toVariant().toLongLong();

        auto existing = txRepo.findById(tx.id);
        if (existing.isOk() && existing.value().has_value()) continue;
        txRepo.insert(tx);
    }

    QJsonArray auditArray = payload.value("audit_logs").toArray();
    for (const auto& av : auditArray) {
        QJsonObject obj = av.toObject();
        QSqlQuery query(m_db->sqlDatabase());
        query.prepare(QStringLiteral(
            "INSERT OR IGNORE INTO audit_logs (action, user_id, details, previous_hash, hash, created_at) "
            "VALUES (?, ?, ?, ?, ?, ?)"));
        query.addBindValue(obj.value("action").toString());
        query.addBindValue(obj.value("user_id").toString());
        query.addBindValue(obj.value("details").toString());
        query.addBindValue(obj.value("previous_hash").toString());
        query.addBindValue(obj.value("hash").toString());
        query.addBindValue(obj.value("created_at").toVariant().toLongLong());
        query.exec();
    }

    return m_db->commit().isOk() ? Result<int>::ok(usersRestored)
           : Result<int>::fail(ErrorCode::DatabaseTransactionFailed, "Commit failed");
}
