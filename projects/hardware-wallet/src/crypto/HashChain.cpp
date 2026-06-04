#include "HashChain.h"
#include "CryptoConfig.h"
#include <openssl/evp.h>
#include <cstring>

namespace HashChain {

const char* GENESIS_HASH_HEX = "0000000000000000000000000000000000000000000000000000000000000000";

SecureBuffer genesisHash() {
    return SecureBuffer::fromHex(QString::fromLatin1(GENESIS_HASH_HEX));
}

Result<SecureBuffer> computeHash(const SecureBuffer& previousHash,
                                 const QString& eventType,
                                 const QString& userId,
                                 const QString& message,
                                 qint64 timestamp) {
    if (previousHash.size() != CryptoConfig::HASH_SIZE) {
        return Result<SecureBuffer>::fail(ErrorCode::KDFFailed, "Previous hash must be 32 bytes");
    }

    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    if (!ctx) {
        return Result<SecureBuffer>::fail(ErrorCode::KDFFailed, "Failed to create digest context");
    }

    if (EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr) != 1) {
        EVP_MD_CTX_free(ctx);
        return Result<SecureBuffer>::fail(ErrorCode::KDFFailed, "DigestInit failed");
    }

    EVP_DigestUpdate(ctx, previousHash.data(), previousHash.size());

    QByteArray eventBytes = eventType.toUtf8();
    EVP_DigestUpdate(ctx, eventBytes.constData(), eventBytes.size());

    QByteArray userIdBytes = userId.toUtf8();
    EVP_DigestUpdate(ctx, userIdBytes.constData(), userIdBytes.size());

    QByteArray msgBytes = message.toUtf8();
    EVP_DigestUpdate(ctx, msgBytes.constData(), msgBytes.size());

    EVP_DigestUpdate(ctx, &timestamp, sizeof(timestamp));

    SecureBuffer hash(CryptoConfig::HASH_SIZE);
    unsigned int hashLen = 0;
    EVP_DigestFinal_ex(ctx, hash.data(), &hashLen);
    EVP_MD_CTX_free(ctx);

    return Result<SecureBuffer>::ok(std::move(hash));
}

Result<bool> verifyLink(const SecureBuffer& previousHash,
                        const QString& eventType,
                        const QString& userId,
                        const QString& message,
                        qint64 timestamp,
                        const SecureBuffer& expectedHash) {
    auto computedResult = computeHash(previousHash, eventType, userId, message, timestamp);
    if (computedResult.isFail()) {
        return Result<bool>::fail(computedResult.error().code, computedResult.error().message);
    }

    const auto& computed = computedResult.value();
    bool match = (computed.size() == expectedHash.size()) &&
                 (CRYPTO_memcmp(computed.data(), expectedHash.data(), computed.size()) == 0);

    return Result<bool>::ok(match);
}

Result<SecureBuffer> blake2b(const unsigned char* data, size_t len) {
    if (!data || len == 0) {
        return Result<SecureBuffer>::fail(ErrorCode::KDFFailed, "Empty input for hash");
    }

    SecureBuffer hash(CryptoConfig::HASH_SIZE);
    unsigned int hashLen = 0;

    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    EVP_DigestUpdate(ctx, data, len);
    EVP_DigestFinal_ex(ctx, hash.data(), &hashLen);
    EVP_MD_CTX_free(ctx);

    return Result<SecureBuffer>::ok(std::move(hash));
}

Result<SecureBuffer> blake2b(const QString& data) {
    QByteArray bytes = data.toUtf8();
    return blake2b(reinterpret_cast<const unsigned char*>(bytes.constData()), bytes.size());
}

} // namespace HashChain
