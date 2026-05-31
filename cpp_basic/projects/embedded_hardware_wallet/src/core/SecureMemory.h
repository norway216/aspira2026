#pragma once

#include "Common.h"
#include <sys/mman.h>
#include <cstring>
#include <memory>
#include <utility>
#include <stdexcept>
#include <openssl/crypto.h>

namespace ehw {

/**
 * Secure memory allocator that locks pages to prevent swapping to disk.
 * On destruction, memory is zeroed before unlock.
 */
template <typename T>
class SecureAllocator {
public:
    using value_type = T;
    using size_type = std::size_t;
    using difference_type = std::ptrdiff_t;
    using propagate_on_container_move_assignment = std::true_type;

    constexpr SecureAllocator() noexcept = default;
    template <typename U>
    constexpr SecureAllocator(const SecureAllocator<U>&) noexcept {}

    T* allocate(std::size_t n) {
        if (n == 0) return nullptr;
        if (n > std::size_t(-1) / sizeof(T)) {
            throw std::bad_array_new_length();
        }
        std::size_t size = n * sizeof(T);
        // Round up to page size
        std::size_t pageSize = static_cast<std::size_t>(sysconf(_SC_PAGESIZE));
        std::size_t allocSize = ((size + pageSize - 1) / pageSize) * pageSize;

        void* ptr = mmap(nullptr, allocSize, PROT_READ | PROT_WRITE,
                         MAP_PRIVATE | MAP_ANONYMOUS | MAP_LOCKED, -1, 0);
        if (ptr == MAP_FAILED) {
            // Retry without MAP_LOCKED
            ptr = mmap(nullptr, allocSize, PROT_READ | PROT_WRITE,
                       MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
            if (ptr == MAP_FAILED) {
                throw std::bad_alloc();
            }
            // Try to lock
            mlock(ptr, allocSize);
        }
        return static_cast<T*>(ptr);
    }

    void deallocate(T* ptr, std::size_t n) noexcept {
        if (!ptr) return;
        std::size_t size = n * sizeof(T);
        std::size_t pageSize = static_cast<std::size_t>(sysconf(_SC_PAGESIZE));
        std::size_t allocSize = ((size + pageSize - 1) / pageSize) * pageSize;

        // Securely zero memory
        OPENSSL_cleanse(ptr, size);
        munlock(ptr, allocSize);
        munmap(ptr, allocSize);
    }
};

template <typename T, typename U>
bool operator==(const SecureAllocator<T>&, const SecureAllocator<U>&) noexcept { return true; }
template <typename T, typename U>
bool operator!=(const SecureAllocator<T>&, const SecureAllocator<U>&) noexcept { return false; }

/**
 * Vector with secure memory allocation — memory is locked and zeroed on destruction.
 */
template <typename T>
class SecureVector {
public:
    using value_type = T;
    using allocator_type = SecureAllocator<T>;

    SecureVector() = default;

    explicit SecureVector(size_t count) : m_data(nullptr), m_size(count) {
        if (count > 0) {
            m_data = m_alloc.allocate(count);
            std::memset(m_data, 0, count * sizeof(T));
        }
    }

    SecureVector(const T* data, size_t len) : m_data(nullptr), m_size(len) {
        if (len > 0) {
            m_data = m_alloc.allocate(len);
            std::memcpy(m_data, data, len * sizeof(T));
        }
    }

    ~SecureVector() {
        clear();
    }

    // Non-copyable
    SecureVector(const SecureVector&) = delete;
    SecureVector& operator=(const SecureVector&) = delete;

    // Movable
    SecureVector(SecureVector&& other) noexcept
        : m_data(std::exchange(other.m_data, nullptr))
        , m_size(std::exchange(other.m_size, 0)) {}

    SecureVector& operator=(SecureVector&& other) noexcept {
        if (this != &other) {
            clear();
            m_data = std::exchange(other.m_data, nullptr);
            m_size = std::exchange(other.m_size, 0);
        }
        return *this;
    }

    T* data() noexcept { return m_data; }
    const T* data() const noexcept { return m_data; }
    size_t size() const noexcept { return m_size; }
    bool empty() const noexcept { return m_size == 0; }
    T* begin() noexcept { return m_data; }
    T* end() noexcept { return m_data + m_size; }
    const T* begin() const noexcept { return m_data; }
    const T* end() const noexcept { return m_data + m_size; }
    T& operator[](size_t i) noexcept { return m_data[i]; }
    const T& operator[](size_t i) const noexcept { return m_data[i]; }

    void clear() {
        if (m_data && m_size > 0) {
            m_alloc.deallocate(m_data, m_size);
            m_data = nullptr;
            m_size = 0;
        }
    }

    void resize(size_t newSize) {
        T* newData = nullptr;
        if (newSize > 0) {
            newData = m_alloc.allocate(newSize);
            std::memset(newData, 0, newSize * sizeof(T));
            if (m_data && m_size > 0) {
                size_t copySize = std::min(m_size, newSize);
                std::memcpy(newData, m_data, copySize * sizeof(T));
            }
        }
        clear();
        m_data = newData;
        m_size = newSize;
    }

    std::vector<uint8_t> toVector() const {
        return std::vector<uint8_t>(m_data, m_data + m_size);
    }

private:
    T* m_data = nullptr;
    size_t m_size = 0;
    SecureAllocator<T> m_alloc;
};

/**
 * Secure memory guard — locks a region of memory for the lifetime of the guard.
 */
class SecureMemoryGuard {
public:
    explicit SecureMemoryGuard(void* ptr, size_t size)
        : m_ptr(ptr), m_size(size) {
        if (ptr && size > 0) {
            mlock(ptr, size);
        }
    }

    ~SecureMemoryGuard() {
        if (m_ptr && m_size > 0) {
            OPENSSL_cleanse(m_ptr, m_size);
            munlock(m_ptr, m_size);
        }
    }

    SecureMemoryGuard(const SecureMemoryGuard&) = delete;
    SecureMemoryGuard& operator=(const SecureMemoryGuard&) = delete;
    SecureMemoryGuard(SecureMemoryGuard&& other) noexcept
        : m_ptr(std::exchange(other.m_ptr, nullptr))
        , m_size(std::exchange(other.m_size, 0)) {}
    SecureMemoryGuard& operator=(SecureMemoryGuard&&) = delete;

private:
    void* m_ptr;
    size_t m_size;
};

} // namespace ehw
