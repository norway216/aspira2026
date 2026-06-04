#pragma once

#include "SecureBuffer.h"
#include "util/Result.h"

#include <QString>
#include <QDateTime>

/**
 * Hash chain for tamper-evident audit logging.
 * Each log entry's hash includes the previous entry's hash,
 * forming a chain where any tampering is detectable.
 *
 * Uses BLAKE2b-256 for hashing.
 */
namespace HashChain {

// Genesis hash - starting point of the chain
extern const char* GENESIS_HASH_HEX;

// Get the genesis hash as a SecureBuffer
SecureBuffer genesisHash();

// Compute hash for a new log entry
// currentHash = BLAKE2b(previousHash || eventType || userId || message || timestamp)
Result<SecureBuffer> computeHash(const SecureBuffer& previousHash,
                                 const QString& eventType,
                                 const QString& userId,
                                 const QString& message,
                                 qint64 timestamp);

// Verify a single chain link
Result<bool> verifyLink(const SecureBuffer& previousHash,
                        const QString& eventType,
                        const QString& userId,
                        const QString& message,
                        qint64 timestamp,
                        const SecureBuffer& expectedHash);

// General purpose BLAKE2b hashing
Result<SecureBuffer> blake2b(const unsigned char* data, size_t len);
Result<SecureBuffer> blake2b(const QString& data);

} // namespace HashChain
