#pragma once

#include <cstddef>

class CryptoProvider {
public:
    static CryptoProvider& instance();

    bool initialize();
    bool isInitialized() const { return m_initialized; }
    bool randomBytes(unsigned char* buf, size_t len);
    unsigned int randomUInt();
    bool hasArmCryptoExtensions() const { return m_hasArmCrypto; }

private:
    CryptoProvider() = default;
    ~CryptoProvider() = default;
    CryptoProvider(const CryptoProvider&) = delete;
    CryptoProvider& operator=(const CryptoProvider&) = delete;

    void detectArmCryptoExtensions();
    bool m_initialized = false;
    bool m_hasArmCrypto = false;
};
