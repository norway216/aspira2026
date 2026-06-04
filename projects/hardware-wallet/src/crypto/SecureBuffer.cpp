#include "SecureBuffer.h"
#include "CryptoProvider.h"
#include <openssl/crypto.h>
#include <cstring>
#include <QDebug>

SecureBuffer::SecureBuffer() noexcept : m_data(nullptr), m_size(0) {}

SecureBuffer::SecureBuffer(size_t size) : m_data(nullptr), m_size(0) {
    alloc(size);
}

SecureBuffer::~SecureBuffer() noexcept {
    free();
}

SecureBuffer::SecureBuffer(SecureBuffer&& other) noexcept
    : m_data(other.m_data), m_size(other.m_size) {
    other.m_data = nullptr;
    other.m_size = 0;
}

SecureBuffer& SecureBuffer::operator=(SecureBuffer&& other) noexcept {
    if (this != &other) {
        free();
        m_data = other.m_data;
        m_size = other.m_size;
        other.m_data = nullptr;
        other.m_size = 0;
    }
    return *this;
}

void SecureBuffer::clear() noexcept {
    free();
}

void SecureBuffer::resize(size_t newSize) {
    if (newSize == m_size && m_data != nullptr) return;
    if (newSize == 0) { free(); return; }

    unsigned char* newData = static_cast<unsigned char*>(OPENSSL_secure_malloc(newSize));
    if (!newData) throw std::bad_alloc();

    if (m_data && m_size > 0) {
        size_t copySize = std::min(m_size, newSize);
        std::memcpy(newData, m_data, copySize);
    }
    free();
    m_data = newData;
    m_size = newSize;
}

void SecureBuffer::alloc(size_t size) {
    if (size == 0) return;
    m_data = static_cast<unsigned char*>(OPENSSL_secure_malloc(size));
    if (!m_data) throw std::bad_alloc();
    m_size = size;
}

void SecureBuffer::free() noexcept {
    if (m_data) {
        OPENSSL_cleanse(m_data, m_size);
        OPENSSL_secure_free(m_data);
        m_data = nullptr;
    }
    m_size = 0;
}

QByteArray SecureBuffer::toQByteArray() const {
    if (!m_data || m_size == 0) return {};
    return QByteArray(reinterpret_cast<const char*>(m_data), static_cast<int>(m_size));
}

QString SecureBuffer::toHex() const {
    if (!m_data || m_size == 0) return {};
    return QByteArray(reinterpret_cast<const char*>(m_data), static_cast<int>(m_size)).toHex();
}

QString SecureBuffer::toBase64() const {
    if (!m_data || m_size == 0) return {};
    return toQByteArray().toBase64();
}

SecureBuffer SecureBuffer::fromHex(const QString& hex) {
    QByteArray bin = QByteArray::fromHex(hex.toLatin1());
    SecureBuffer buf(bin.size());
    std::memcpy(buf.data(), bin.constData(), bin.size());
    return buf;
}

SecureBuffer SecureBuffer::fromBase64(const QString& b64) {
    QByteArray bin = QByteArray::fromBase64(b64.toLatin1());
    SecureBuffer buf(bin.size());
    std::memcpy(buf.data(), bin.constData(), bin.size());
    return buf;
}

SecureBuffer SecureBuffer::fromQByteArray(const QByteArray& ba) {
    if (ba.isEmpty()) return SecureBuffer();
    SecureBuffer buf(ba.size());
    std::memcpy(buf.data(), ba.constData(), ba.size());
    return buf;
}

SecureBuffer SecureBuffer::random(size_t size) {
    SecureBuffer buf(size);
    CryptoProvider::instance().randomBytes(buf.data(), size);
    return buf;
}
