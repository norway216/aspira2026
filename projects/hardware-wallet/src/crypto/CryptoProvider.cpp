#include "CryptoProvider.h"
#include <openssl/crypto.h>
#include <openssl/rand.h>

#ifdef __aarch64__
#include <sys/auxv.h>
#include <asm/hwcap.h>
#endif

CryptoProvider& CryptoProvider::instance() {
    static CryptoProvider provider;
    return provider;
}

bool CryptoProvider::initialize() {
    if (m_initialized) return true;
    OPENSSL_init_crypto(OPENSSL_INIT_LOAD_CONFIG, nullptr);
    detectArmCryptoExtensions();
    m_initialized = true;
    return true;
}

bool CryptoProvider::randomBytes(unsigned char* buf, size_t len) {
    if (!m_initialized || !buf || len == 0) return false;
    return RAND_bytes(buf, static_cast<int>(len)) == 1;
}

unsigned int CryptoProvider::randomUInt() {
    unsigned int val = 0;
    RAND_bytes(reinterpret_cast<unsigned char*>(&val), sizeof(val));
    return val;
}

void CryptoProvider::detectArmCryptoExtensions() {
#ifdef __aarch64__
    long hwcaps = getauxval(AT_HWCAP);
    m_hasArmCrypto = (hwcaps & HWCAP_AES) && (hwcaps & HWCAP_SHA2);
#else
    m_hasArmCrypto = false;
#endif
}
