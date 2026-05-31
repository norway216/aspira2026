#pragma once

#include <QString>
#include <QByteArray>
#include <QDateTime>
#include "crypto/SecureBuffer.h"

namespace CryptoUtils {

// Generate a UUID-like identifier (hex encoded random bytes)
QString generateId();

// Get current timestamp as milliseconds since epoch
qint64 currentTimestampMs();

// Convert timestamp to QDateTime
QDateTime timestampToDateTime(qint64 ts);

// Convert QDateTime to timestamp
qint64 dateTimeToTimestamp(const QDateTime& dt);

// Compute SHA-256 hash of data (uses BLAKE2b-256 via libsodium)
SecureBuffer hashData(const QByteArray& data);
QString hashDataHex(const QByteArray& data);

// Constant-time secure comparison
bool secureCompare(const unsigned char* a, const unsigned char* b, size_t len);
bool secureCompare(const SecureBuffer& a, const SecureBuffer& b);

} // namespace CryptoUtils
