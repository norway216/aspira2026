#pragma once

#include "SecureBuffer.h"
#include "util/Result.h"

/**
 * Ed25519 digital signatures using libsodium.
 */
namespace SignatureHelper {

struct KeyPair {
    SecureBuffer publicKey;   // 32 bytes
    SecureBuffer secretKey;   // 64 bytes (includes copy of public key)
};

// Generate a new Ed25519 key pair
Result<KeyPair> generateKeyPair();

// Generate key pair from a seed
Result<KeyPair> generateFromSeed(const SecureBuffer& seed);

// Sign a message with a secret key
// Returns the 64-byte signature
Result<SecureBuffer> sign(const unsigned char* message, size_t msgLen,
                          const unsigned char* secretKey);

// Verify a signature
Result<bool> verify(const unsigned char* message, size_t msgLen,
                    const unsigned char* signature,
                    const unsigned char* publicKey);

// Convenience overloads
Result<SecureBuffer> sign(const SecureBuffer& message, const SecureBuffer& secretKey);
Result<bool> verify(const SecureBuffer& message, const SecureBuffer& signature,
                    const SecureBuffer& publicKey);

} // namespace SignatureHelper
