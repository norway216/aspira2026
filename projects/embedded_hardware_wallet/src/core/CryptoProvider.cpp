#include "CryptoProvider.h"
#include <span>
#include <cassert>
#include <cstring>

// Suppress OpenSSL 3.0 deprecation warnings for legacy EC API
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wdeprecated-declarations"

namespace ehw {

void CryptoProvider::handleOpenSSLError(const char* context) {
    char errBuf[256];
    unsigned long err = ERR_get_error();
    ERR_error_string_n(err, errBuf, sizeof(errBuf));
    log(LogLevel::Error, "OpenSSL error in {}: {}", context, errBuf);
}

CryptoProvider::CryptoProvider() {
    m_initialized = true;
    log(LogLevel::Info, "CryptoProvider initialized (OpenSSL {})",
        OpenSSL_version(OPENSSL_FULL_VERSION_STRING));
}

CryptoProvider::~CryptoProvider() {
    m_initialized = false;
}

CryptoProvider& crypto() {
    static CryptoProvider instance;
    return instance;
}

// Detect available EC curve: prefer secp256k1, fall back to P-256
int CryptoProvider::getCurveNid() const {
    // Try secp256k1 first
    EC_KEY* testKey = EC_KEY_new_by_curve_name(NID_secp256k1);
    if (testKey) {
        EC_KEY_free(testKey);
        return NID_secp256k1;
    }
    // Fall back to P-256 (prime256v1)
    log(LogLevel::Info, "secp256k1 not available, using P-256 (prime256v1)");
    return NID_X9_62_prime256v1;
}

// ---- Random Number Generation ----
Result<void> CryptoProvider::randomBytes(uint8_t* out, size_t len) {
    if (RAND_bytes(out, static_cast<int>(len)) != 1) {
        handleOpenSSLError("RAND_bytes");
        return {.error = WalletError::RandomFailed};
    }
    return {};
}

// ---- AES-256-GCM Encrypt ----
Result<ByteVector> CryptoProvider::aesGcmEncrypt(
    std::span<const uint8_t> key,
    std::span<const uint8_t> plaintext,
    std::span<const uint8_t> aad)
{
    if (key.size() != AES_256_KEY_SIZE) {
        return {.error = WalletError::EncryptFailed};
    }

    uint8_t iv[AES_GCM_IV_SIZE];
    if (auto r = randomBytes(iv, sizeof(iv)); !r.ok()) {
        return {.error = r.error};
    }

    EvpCipherCtxPtr ctx(EVP_CIPHER_CTX_new());
    if (!ctx) {
        handleOpenSSLError("EVP_CIPHER_CTX_new");
        return {.error = WalletError::EncryptFailed};
    }

    if (EVP_EncryptInit_ex(ctx.get(), EVP_aes_256_gcm(), nullptr, nullptr, nullptr) != 1) {
        handleOpenSSLError("EVP_EncryptInit_ex");
        return {.error = WalletError::EncryptFailed};
    }

    if (EVP_CIPHER_CTX_ctrl(ctx.get(), EVP_CTRL_GCM_SET_IVLEN,
                            AES_GCM_IV_SIZE, nullptr) != 1) {
        handleOpenSSLError("SET_IVLEN");
        return {.error = WalletError::EncryptFailed};
    }

    if (EVP_EncryptInit_ex(ctx.get(), nullptr, nullptr, key.data(), iv) != 1) {
        handleOpenSSLError("EVP_EncryptInit_ex key/IV");
        return {.error = WalletError::EncryptFailed};
    }

    // Set AAD if provided
    if (!aad.empty()) {
        int outLen = 0;
        if (EVP_EncryptUpdate(ctx.get(), nullptr, &outLen,
                              aad.data(), static_cast<int>(aad.size())) != 1) {
            handleOpenSSLError("AAD update");
            return {.error = WalletError::EncryptFailed};
        }
    }

    // Encrypt
    ByteVector ciphertext(plaintext.size() + AES_GCM_TAG_SIZE + 16);
    int outLen = 0;
    if (EVP_EncryptUpdate(ctx.get(), ciphertext.data(), &outLen,
                          plaintext.data(), static_cast<int>(plaintext.size())) != 1) {
        handleOpenSSLError("EncryptUpdate");
        return {.error = WalletError::EncryptFailed};
    }

    int totalLen = outLen;
    if (EVP_EncryptFinal_ex(ctx.get(), ciphertext.data() + totalLen, &outLen) != 1) {
        handleOpenSSLError("EncryptFinal_ex");
        return {.error = WalletError::EncryptFailed};
    }
    totalLen += outLen;

    // Get tag
    uint8_t tag[AES_GCM_TAG_SIZE];
    if (EVP_CIPHER_CTX_ctrl(ctx.get(), EVP_CTRL_GCM_GET_TAG,
                            AES_GCM_TAG_SIZE, tag) != 1) {
        handleOpenSSLError("GET_TAG");
        return {.error = WalletError::EncryptFailed};
    }

    ciphertext.resize(totalLen);

    // Output format: IV (12) || ciphertext || tag (16)
    ByteVector result;
    result.reserve(AES_GCM_IV_SIZE + ciphertext.size() + AES_GCM_TAG_SIZE);
    result.insert(result.end(), iv, iv + AES_GCM_IV_SIZE);
    result.insert(result.end(), ciphertext.begin(), ciphertext.end());
    result.insert(result.end(), tag, tag + AES_GCM_TAG_SIZE);

    return {.value = std::move(result)};
}

// ---- AES-256-GCM Decrypt ----
Result<ByteVector> CryptoProvider::aesGcmDecrypt(
    std::span<const uint8_t> key,
    std::span<const uint8_t> ciphertext)
{
    if (key.size() != AES_256_KEY_SIZE ||
        ciphertext.size() < AES_GCM_IV_SIZE + AES_GCM_TAG_SIZE) {
        return {.error = WalletError::DecryptFailed};
    }

    const uint8_t* iv = ciphertext.data();
    size_t ctLen = ciphertext.size() - AES_GCM_IV_SIZE - AES_GCM_TAG_SIZE;
    const uint8_t* ct = ciphertext.data() + AES_GCM_IV_SIZE;
    const uint8_t* tag = ct + ctLen;

    EvpCipherCtxPtr ctx(EVP_CIPHER_CTX_new());
    if (!ctx) {
        return {.error = WalletError::DecryptFailed};
    }

    if (EVP_DecryptInit_ex(ctx.get(), EVP_aes_256_gcm(), nullptr, nullptr, nullptr) != 1) {
        return {.error = WalletError::DecryptFailed};
    }

    if (EVP_CIPHER_CTX_ctrl(ctx.get(), EVP_CTRL_GCM_SET_IVLEN,
                            AES_GCM_IV_SIZE, nullptr) != 1) {
        return {.error = WalletError::DecryptFailed};
    }

    if (EVP_DecryptInit_ex(ctx.get(), nullptr, nullptr, key.data(), iv) != 1) {
        return {.error = WalletError::DecryptFailed};
    }

    ByteVector plaintext(ctLen);
    int outLen = 0;
    if (EVP_DecryptUpdate(ctx.get(), plaintext.data(), &outLen,
                          ct, static_cast<int>(ctLen)) != 1) {
        return {.error = WalletError::DecryptFailed};
    }

    // Set expected tag
    if (EVP_CIPHER_CTX_ctrl(ctx.get(), EVP_CTRL_GCM_SET_TAG,
                            AES_GCM_TAG_SIZE, const_cast<uint8_t*>(tag)) != 1) {
        return {.error = WalletError::DecryptFailed};
    }

    int finalLen = 0;
    if (EVP_DecryptFinal_ex(ctx.get(), plaintext.data() + outLen, &finalLen) != 1) {
        return {.error = WalletError::DecryptFailed};
    }

    plaintext.resize(outLen + finalLen);
    return {.value = std::move(plaintext)};
}

// ---- SHA-256 ----
Result<std::array<uint8_t, SHA256_DIGEST_SIZE>> CryptoProvider::sha256(
    std::span<const uint8_t> data)
{
    std::array<uint8_t, SHA256_DIGEST_SIZE> hash{};
    EvpMdCtxPtr ctx(EVP_MD_CTX_new());
    if (!ctx || EVP_DigestInit_ex(ctx.get(), EVP_sha256(), nullptr) != 1 ||
        EVP_DigestUpdate(ctx.get(), data.data(), data.size()) != 1 ||
        EVP_DigestFinal_ex(ctx.get(), hash.data(), nullptr) != 1) {
        handleOpenSSLError("SHA256");
        return {.error = WalletError::CryptoInitFailed};
    }
    return {.value = hash};
}

Result<ByteVector> CryptoProvider::sha256(const ByteVector& data) {
    auto r = sha256(std::span<const uint8_t>(data.data(), data.size()));
    if (!r.ok()) return {.error = r.error};
    return {.value = ByteVector(r.value.begin(), r.value.end())};
}

// ---- Double SHA-256 ----
Result<std::array<uint8_t, SHA256_DIGEST_SIZE>> CryptoProvider::doubleSha256(
    std::span<const uint8_t> data)
{
    auto first = sha256(data);
    if (!first.ok()) return {.error = first.error};
    return sha256(std::span<const uint8_t>(first.value.data(), first.value.size()));
}

// ---- HMAC-SHA512 ----
Result<std::array<uint8_t, HMAC_SHA512_SIZE>> CryptoProvider::hmacSha512(
    std::span<const uint8_t> key,
    std::span<const uint8_t> data)
{
    std::array<uint8_t, HMAC_SHA512_SIZE> result{};
    unsigned int len = HMAC_SHA512_SIZE;
    if (!HMAC(EVP_sha512(), key.data(), static_cast<int>(key.size()),
              data.data(), data.size(), result.data(), &len)) {
        handleOpenSSLError("HMAC-SHA512");
        return {.error = WalletError::CryptoInitFailed};
    }
    return {.value = result};
}

// ---- PBKDF2-HMAC-SHA512 ----
Result<ByteVector> CryptoProvider::pbkdf2HmacSha512(
    std::span<const uint8_t> password,
    std::span<const uint8_t> salt,
    uint32_t iterations,
    size_t keyLen)
{
    ByteVector key(keyLen);
    if (PKCS5_PBKDF2_HMAC(
            reinterpret_cast<const char*>(password.data()),
            static_cast<int>(password.size()),
            salt.data(), static_cast<int>(salt.size()),
            static_cast<int>(iterations),
            EVP_sha512(),
            static_cast<int>(keyLen), key.data()) != 1) {
        handleOpenSSLError("PBKDF2");
        return {.error = WalletError::KeyGenFailed};
    }
    return {.value = std::move(key)};
}

// ---- ECDSA Key Generation ----
Result<std::pair<SecureByteVector, ByteVector>> CryptoProvider::generateKeyPair() {
    int curveNid = getCurveNid();

    EvpPkeyCtxPtr pctx(EVP_PKEY_CTX_new_id(EVP_PKEY_EC, nullptr));
    if (!pctx) {
        handleOpenSSLError("PKEY_CTX_new");
        return {.error = WalletError::KeyGenFailed};
    }

    if (EVP_PKEY_keygen_init(pctx.get()) != 1 ||
        EVP_PKEY_CTX_set_ec_paramgen_curve_nid(pctx.get(), curveNid) != 1) {
        handleOpenSSLError("keygen_init/curve");
        return {.error = WalletError::KeyGenFailed};
    }

    EVP_PKEY* rawPkey = nullptr;
    if (EVP_PKEY_keygen(pctx.get(), &rawPkey) != 1) {
        handleOpenSSLError("keygen");
        return {.error = WalletError::KeyGenFailed};
    }
    EvpPkeyPtr pkey(rawPkey);

    // Extract private key
    SecureByteVector privKey(EC_PRIVKEY_SIZE);
    size_t privLen = EC_PRIVKEY_SIZE;
    if (EVP_PKEY_get_raw_private_key(pkey.get(), privKey.data(), &privLen) != 1) {
        handleOpenSSLError("get_raw_private_key");
        return {.error = WalletError::KeyGenFailed};
    }
    privKey.resize(privLen);

    // Extract compressed public key (33 bytes)
    ByteVector pubKey(EC_PUBKEY_COMPRESSED_SIZE);
    size_t pubLen = EC_PUBKEY_COMPRESSED_SIZE;
    if (EVP_PKEY_get_raw_public_key(pkey.get(), pubKey.data(), &pubLen) != 1) {
        handleOpenSSLError("get_raw_public_key");
        return {.error = WalletError::KeyGenFailed};
    }
    pubKey.resize(pubLen);

    return {.value = {std::move(privKey), std::move(pubKey)}};
}

// ---- ECDSA Sign ----
Result<ByteVector> CryptoProvider::ecdsaSign(
    std::span<const uint8_t> privateKey,
    std::span<const uint8_t> digest)
{
    if (privateKey.size() != EC_PRIVKEY_SIZE || digest.size() != SHA256_DIGEST_SIZE) {
        return {.error = WalletError::SignFailed};
    }

    // Create EVP_PKEY from raw private key
    EvpPkeyPtr pkey(EVP_PKEY_new_raw_private_key(
        EVP_PKEY_EC, nullptr, privateKey.data(), privateKey.size()));
    if (!pkey) {
        handleOpenSSLError("new_raw_private_key");
        return {.error = WalletError::SignFailed};
    }

    EvpMdCtxPtr mdctx(EVP_MD_CTX_new());
    if (!mdctx) {
        return {.error = WalletError::SignFailed};
    }

    if (EVP_DigestSignInit(mdctx.get(), nullptr, EVP_sha256(), nullptr, pkey.get()) != 1) {
        handleOpenSSLError("DigestSignInit");
        return {.error = WalletError::SignFailed};
    }

    size_t sigLen = 0;
    if (EVP_DigestSign(mdctx.get(), nullptr, &sigLen, digest.data(), digest.size()) != 1) {
        handleOpenSSLError("DigestSign size query");
        return {.error = WalletError::SignFailed};
    }

    ByteVector signature(sigLen);
    if (EVP_DigestSign(mdctx.get(), signature.data(), &sigLen,
                       digest.data(), digest.size()) != 1) {
        handleOpenSSLError("DigestSign");
        return {.error = WalletError::SignFailed};
    }
    signature.resize(sigLen);
    return {.value = std::move(signature)};
}

// ---- ECDSA Verify ----
Result<bool> CryptoProvider::ecdsaVerify(
    std::span<const uint8_t> publicKey,
    std::span<const uint8_t> digest,
    std::span<const uint8_t> signature)
{
    if (digest.size() != SHA256_DIGEST_SIZE) {
        return {.error = WalletError::VerifyFailed};
    }

    EvpPkeyPtr pkey(EVP_PKEY_new_raw_public_key(
        EVP_PKEY_EC, nullptr, publicKey.data(), publicKey.size()));
    if (!pkey) {
        handleOpenSSLError("new_raw_public_key");
        return {.error = WalletError::VerifyFailed};
    }

    EvpMdCtxPtr mdctx(EVP_MD_CTX_new());
    if (!mdctx) {
        return {.error = WalletError::VerifyFailed};
    }

    if (EVP_DigestVerifyInit(mdctx.get(), nullptr, EVP_sha256(), nullptr, pkey.get()) != 1) {
        handleOpenSSLError("DigestVerifyInit");
        return {.error = WalletError::VerifyFailed};
    }

    int result = EVP_DigestVerify(mdctx.get(), signature.data(), signature.size(),
                                   digest.data(), digest.size());
    if (result == 1) return {.value = true};
    if (result == 0) return {.value = false};

    handleOpenSSLError("DigestVerify");
    return {.error = WalletError::VerifyFailed};
}

// ---- Public Key Derivation ----
Result<ByteVector> CryptoProvider::privateKeyToPublicKey(
    std::span<const uint8_t> privateKey,
    bool compressed)
{
    if (privateKey.size() != EC_PRIVKEY_SIZE) {
        return {.error = WalletError::InvalidKey};
    }

    EvpPkeyPtr pkey(EVP_PKEY_new_raw_private_key(
        EVP_PKEY_EC, nullptr, privateKey.data(), privateKey.size()));
    if (!pkey) {
        handleOpenSSLError("new_raw_private_key");
        return {.error = WalletError::InvalidKey};
    }

    size_t expectedSize = compressed ? EC_PUBKEY_COMPRESSED_SIZE : EC_PUBKEY_UNCOMPRESSED_SIZE;
    ByteVector pubKey(expectedSize);
    size_t pubLen = expectedSize;

    if (!compressed) {
        // For uncompressed, get the EC_KEY and convert the point
        const EC_KEY* ecKey = EVP_PKEY_get0_EC_KEY(pkey.get());
        if (!ecKey) return {.error = WalletError::InvalidKey};

        const EC_POINT* point = EC_KEY_get0_public_key(ecKey);
        const EC_GROUP* group = EC_KEY_get0_group(ecKey);
        if (!point || !group) return {.error = WalletError::InvalidKey};

        pubLen = EC_POINT_point2oct(group, point, POINT_CONVERSION_UNCOMPRESSED,
                                     pubKey.data(), expectedSize, nullptr);
        if (pubLen == 0) {
            handleOpenSSLError("point2oct");
            return {.error = WalletError::InvalidKey};
        }
    } else {
        if (EVP_PKEY_get_raw_public_key(pkey.get(), pubKey.data(), &pubLen) != 1) {
            handleOpenSSLError("get_raw_public_key");
            return {.error = WalletError::InvalidKey};
        }
    }

    pubKey.resize(pubLen);
    return {.value = std::move(pubKey)};
}

// ---- Constant-Time Comparison ----
bool CryptoProvider::constantTimeCompare(
    std::span<const uint8_t> a,
    std::span<const uint8_t> b)
{
    if (a.size() != b.size()) return false;
    return CRYPTO_memcmp(a.data(), b.data(), a.size()) == 0;
}

#pragma GCC diagnostic pop

} // namespace ehw
