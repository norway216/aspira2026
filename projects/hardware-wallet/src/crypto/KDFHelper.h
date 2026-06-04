#pragma once

#include "SecureBuffer.h"
#include "CryptoConfig.h"
#include "util/Result.h"

#include <QString>

namespace KDFHelper {

// Hash a password using Argon2id for storage verification
// Returns the hash string (includes salt and params encoded)
Result<SecureBuffer> hashPassword(const QString& password,
                                  const SecureBuffer& salt,
                                  unsigned long long opsLimit = 4ULL,
                                  size_t memLimit = 268435456ULL);

// Verify a password against a stored hash
// hashAndSalt: the stored hash from hashPassword
Result<bool> verifyPassword(const QString& password, const SecureBuffer& hashAndSalt);

// Derive a Key Encryption Key (KEK) from password and salt
// Used for encrypting/decrypting DEK
Result<SecureBuffer> deriveKEK(const QString& password,
                               const SecureBuffer& salt,
                               unsigned long long opsLimit = 4ULL,
                               size_t memLimit = 268435456ULL);

// Derive a key for backup encryption
Result<SecureBuffer> deriveBackupKey(const QString& backupPassword,
                                     const SecureBuffer& salt,
                                     unsigned long long opsLimit = 4ULL,
                                     size_t memLimit = 268435456ULL);

} // namespace KDFHelper
