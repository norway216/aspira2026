#include "KDFHelper.h"
#include "CryptoConfig.h"
#include <openssl/evp.h>
#include <openssl/kdf.h>
#include <cstring>

namespace KDFHelper {

Result<SecureBuffer> hashPassword(const QString& password,
                                  const SecureBuffer& salt,
                                  unsigned long long opsLimit,
                                  size_t memLimit) {
    (void)opsLimit; (void)memLimit;
    if (password.isEmpty()) {
        return Result<SecureBuffer>::fail(ErrorCode::KDFFailed, "Password is empty");
    }

    QByteArray pwBytes = password.toUtf8();
    SecureBuffer hash(CryptoConfig::HASH_SIZE);

    int ret = PKCS5_PBKDF2_HMAC(
        pwBytes.constData(), pwBytes.size(),
        salt.data(), salt.size(),
        CryptoConfig::PBKDF2_ITERATIONS,
        EVP_sha256(),
        CryptoConfig::KEY_SIZE,
        hash.data()
    );

    if (ret != 1) {
        return Result<SecureBuffer>::fail(ErrorCode::KDFFailed, "PBKDF2 failed");
    }

    return Result<SecureBuffer>::ok(std::move(hash));
}

Result<bool> verifyPassword(const QString& password, const SecureBuffer& storedHash) {
    (void)password; (void)storedHash;
    return Result<bool>::fail(ErrorCode::KDFFailed, "Use hashPassword with stored salt and compare results");
}

Result<SecureBuffer> deriveKEK(const QString& password,
                               const SecureBuffer& salt,
                               unsigned long long opsLimit,
                               size_t memLimit) {
    return hashPassword(password, salt, opsLimit, memLimit);
}

Result<SecureBuffer> deriveBackupKey(const QString& backupPassword,
                                     const SecureBuffer& salt,
                                     unsigned long long opsLimit,
                                     size_t memLimit) {
    return deriveKEK(backupPassword, salt, opsLimit, memLimit);
}

} // namespace KDFHelper
