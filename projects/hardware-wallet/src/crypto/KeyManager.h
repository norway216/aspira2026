#pragma once

#include "SecureBuffer.h"
#include "util/Result.h"

#include <QString>

/**
 * Manages the Data Encryption Key (DEK) lifecycle:
 * - DEK creation (random generation)
 * - DEK wrapping (encrypt with KEK for storage)
 * - DEK unwrapping (decrypt with KEK for use)
 */
class KeyManager {
public:
    // Generate a random 32-byte DEK
    static Result<SecureBuffer> createDEK();

    // Generate a random salt for password hashing
    static Result<SecureBuffer> createSalt();

    // Generate a random seed for wallet key generation
    static Result<SecureBuffer> createWalletSeed();

    // Generate a random nonce for transactions
    static Result<SecureBuffer> createNonce();

    // Wrap (encrypt) DEK with KEK for database storage
    static Result<SecureBuffer> wrapDEK(const SecureBuffer& dek, const SecureBuffer& kek);

    // Unwrap (decrypt) DEK with KEK from database
    static Result<SecureBuffer> unwrapDEK(const SecureBuffer& encryptedDek, const SecureBuffer& kek);
};
