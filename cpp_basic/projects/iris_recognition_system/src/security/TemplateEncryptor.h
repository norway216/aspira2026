#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace iris {

class CryptoProvider;

/// Encrypts and decrypts iris template data
class TemplateEncryptor {
public:
    explicit TemplateEncryptor(CryptoProvider& crypto);

    /// Encrypt an IrisCode for secure storage
    std::vector<uint8_t> encryptCode(const std::vector<uint8_t>& irisCode,
                                      const std::string& userId);

    /// Decrypt an IrisCode from secure storage
    std::vector<uint8_t> decryptCode(const std::vector<uint8_t>& encryptedBlob,
                                      const std::string& userId);

    /// Encrypt a float embedding vector
    std::vector<uint8_t> encryptEmbedding(const std::vector<float>& embedding,
                                           const std::string& userId);

    /// Decrypt a float embedding vector
    std::vector<float> decryptEmbedding(const std::vector<uint8_t>& encryptedBlob,
                                         const std::string& userId);

private:
    std::vector<uint8_t> deriveUserKey(const std::string& userId);

    CryptoProvider& m_crypto;
};

} // namespace iris
