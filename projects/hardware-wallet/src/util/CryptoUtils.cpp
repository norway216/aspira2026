#include "CryptoUtils.h"
#include "crypto/CryptoProvider.h"
#include <openssl/evp.h>
#include <QDateTime>

namespace CryptoUtils {

QString generateId() {
    SecureBuffer random = SecureBuffer::random(16);
    return random.toHex();
}

qint64 currentTimestampMs() {
    return QDateTime::currentMSecsSinceEpoch();
}

QDateTime timestampToDateTime(qint64 ts) {
    return QDateTime::fromMSecsSinceEpoch(ts);
}

qint64 dateTimeToTimestamp(const QDateTime& dt) {
    return dt.toMSecsSinceEpoch();
}

SecureBuffer hashData(const QByteArray& data) {
    SecureBuffer hash(32);
    unsigned int hashLen = 0;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    EVP_DigestUpdate(ctx, data.constData(), data.size());
    EVP_DigestFinal_ex(ctx, hash.data(), &hashLen);
    EVP_MD_CTX_free(ctx);
    return hash;
}

QString hashDataHex(const QByteArray& data) {
    return hashData(data).toHex();
}

bool secureCompare(const unsigned char* a, const unsigned char* b, size_t len) {
    return CRYPTO_memcmp(a, b, len) == 0;
}

bool secureCompare(const SecureBuffer& a, const SecureBuffer& b) {
    if (a.size() != b.size()) return false;
    return CRYPTO_memcmp(a.data(), b.data(), a.size()) == 0;
}

} // namespace CryptoUtils
