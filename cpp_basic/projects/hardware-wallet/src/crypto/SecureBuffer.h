#pragma once

#include "CryptoConfig.h"

#include <cstddef>
#include <QString>
#include <QByteArray>

/**
 * SecureBuffer - RAII wrapper around sodium_malloc/sodium_free.
 * Memory is allocated in locked pages (no swap) and zeroed on free.
 * Move-only, non-copyable.
 */
class SecureBuffer {
public:
    SecureBuffer() noexcept;
    explicit SecureBuffer(size_t size);
    ~SecureBuffer() noexcept;

    // Non-copyable
    SecureBuffer(const SecureBuffer&) = delete;
    SecureBuffer& operator=(const SecureBuffer&) = delete;

    // Movable
    SecureBuffer(SecureBuffer&& other) noexcept;
    SecureBuffer& operator=(SecureBuffer&& other) noexcept;

    // Accessors
    unsigned char* data() noexcept { return m_data; }
    const unsigned char* data() const noexcept { return m_data; }
    size_t size() const noexcept { return m_size; }
    bool empty() const noexcept { return m_size == 0 || m_data == nullptr; }

    // Clear (zero and free)
    void clear() noexcept;

    // Resize (allocate new, copy, zero old)
    void resize(size_t newSize);

    // Conversion helpers
    QByteArray toQByteArray() const;
    QString toHex() const;
    QString toBase64() const;

    static SecureBuffer fromHex(const QString& hex);
    static SecureBuffer fromBase64(const QString& b64);
    static SecureBuffer fromQByteArray(const QByteArray& ba);
    static SecureBuffer random(size_t size);

private:
    unsigned char* m_data = nullptr;
    size_t m_size = 0;

    void alloc(size_t size);
    void free() noexcept;
};
