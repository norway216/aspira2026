#include "security/TemplateEncryptor.h"
#include "security/CryptoProvider.h"
#include <cstring>

namespace iris {

TemplateEncryptor::TemplateEncryptor(CryptoProvider& crypto)
    : m_crypto(crypto) {}

std::vector<uint8_t> TemplateEncryptor::encryptCode(
    const std::vector<uint8_t>& irisCode,
    const std::string& userId) {

    auto userKey = deriveUserKey(userId);
    return m_crypto.encrypt(irisCode, userKey);
}

std::vector<uint8_t> TemplateEncryptor::decryptCode(
    const std::vector<uint8_t>& encryptedBlob,
    const std::string& userId) {

    auto userKey = deriveUserKey(userId);
    return m_crypto.decrypt(encryptedBlob, userKey);
}

std::vector<uint8_t> TemplateEncryptor::encryptEmbedding(
    const std::vector<float>& embedding,
    const std::string& userId) {

    // Convert float vector to byte array
    std::vector<uint8_t> bytes;
    bytes.reserve(embedding.size() * sizeof(float));
    for (float v : embedding) {
        uint32_t bits;
        std::memcpy(&bits, &v, sizeof(float));
        bytes.push_back((bits >> 24) & 0xFF);
        bytes.push_back((bits >> 16) & 0xFF);
        bytes.push_back((bits >> 8)  & 0xFF);
        bytes.push_back(bits & 0xFF);
    }

    auto userKey = deriveUserKey(userId);
    return m_crypto.encrypt(bytes, userKey);
}

std::vector<float> TemplateEncryptor::decryptEmbedding(
    const std::vector<uint8_t>& encryptedBlob,
    const std::string& userId) {

    auto userKey = deriveUserKey(userId);
    auto bytes = m_crypto.decrypt(encryptedBlob, userKey);

    std::vector<float> embedding;
    embedding.reserve(bytes.size() / sizeof(float));

    for (size_t i = 0; i + 3 < bytes.size(); i += 4) {
        uint32_t bits = (static_cast<uint32_t>(bytes[i])   << 24)
                      | (static_cast<uint32_t>(bytes[i+1]) << 16)
                      | (static_cast<uint32_t>(bytes[i+2]) << 8)
                      | (static_cast<uint32_t>(bytes[i+3]));
        float v;
        std::memcpy(&v, &bits, sizeof(float));
        embedding.push_back(v);
    }

    return embedding;
}

std::vector<uint8_t> TemplateEncryptor::deriveUserKey(const std::string& userId) {
    return m_crypto.deriveKey(userId, 32);
}

} // namespace iris
