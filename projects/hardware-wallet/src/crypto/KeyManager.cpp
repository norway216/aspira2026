#include "KeyManager.h"
#include "AEADHelper.h"
#include "CryptoConfig.h"
#include "CryptoProvider.h"

Result<SecureBuffer> KeyManager::createDEK() {
    return Result<SecureBuffer>::ok(SecureBuffer::random(CryptoConfig::KEY_SIZE));
}

Result<SecureBuffer> KeyManager::createSalt() {
    return Result<SecureBuffer>::ok(SecureBuffer::random(CryptoConfig::SALT_SIZE));
}

Result<SecureBuffer> KeyManager::createWalletSeed() {
    return Result<SecureBuffer>::ok(SecureBuffer::random(CryptoConfig::SEED_SIZE));
}

Result<SecureBuffer> KeyManager::createNonce() {
    return Result<SecureBuffer>::ok(SecureBuffer::random(16));
}

Result<SecureBuffer> KeyManager::wrapDEK(const SecureBuffer& dek, const SecureBuffer& kek) {
    if (dek.size() != CryptoConfig::KEY_SIZE || kek.size() != CryptoConfig::KEY_SIZE) {
        return Result<SecureBuffer>::fail(ErrorCode::EncryptionFailed, "Invalid key size");
    }
    return AEADHelper::encrypt(dek, kek);
}

Result<SecureBuffer> KeyManager::unwrapDEK(const SecureBuffer& encryptedDek, const SecureBuffer& kek) {
    if (kek.size() != CryptoConfig::KEY_SIZE) {
        return Result<SecureBuffer>::fail(ErrorCode::DecryptionFailed, "Invalid KEK size");
    }
    auto result = AEADHelper::decrypt(encryptedDek, kek);
    if (result.isOk() && result.value().size() != CryptoConfig::KEY_SIZE) {
        return Result<SecureBuffer>::fail(ErrorCode::DecryptionFailed, "Decrypted DEK has wrong size");
    }
    return result;
}
