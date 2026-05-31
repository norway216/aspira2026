#pragma once

#include "SecureBuffer.h"
#include "util/Result.h"

#include <QByteArray>

/**
 * AEAD encryption using XChaCha20-Poly1305 (libsodium).
 *
 * Encrypted format: [NONCE(24 bytes)][CIPHERTEXT + TAG]
 * Nonce is randomly generated for each encryption.
 */
namespace AEADHelper {

// Encrypt plaintext with key. Returns encrypted blob with nonce prepended.
Result<SecureBuffer> encrypt(const SecureBuffer& plaintext, const SecureBuffer& key);
Result<SecureBuffer> encrypt(const unsigned char* plaintext, size_t plainLen, const unsigned char* key);

// Decrypt ciphertext blob (with prepended nonce). Returns plaintext.
Result<SecureBuffer> decrypt(const SecureBuffer& ciphertextWithNonce, const SecureBuffer& key);
Result<SecureBuffer> decrypt(const unsigned char* ciphertext, size_t cipherLen, const unsigned char* key);

} // namespace AEADHelper
