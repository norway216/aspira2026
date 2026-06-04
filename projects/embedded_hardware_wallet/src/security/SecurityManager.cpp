#include "SecurityManager.h"
#include <unistd.h>
#include <fcntl.h>
#include <sys/mman.h>
#include <sys/ptrace.h>
#include <thread>
#include <random>

namespace ehw {

SecurityManager::SecurityManager() {
    log(LogLevel::Info, "SecurityManager created");
    probeHardwareRNG();
}

SecurityManager::~SecurityManager() {
    if (m_hwrngFd >= 0) {
        close(m_hwrngFd);
        m_hwrngFd = -1;
    }
    if (m_guardPage) {
        munmap(m_guardPage, 4096);
        m_guardPage = nullptr;
    }
}

Result<void> SecurityManager::initialize() {
    // Setup guard pages for key operations
    setupGuardPages();

    // Validate security features
    if (!checkStackProtection()) {
        log(LogLevel::Warning, "Stack protection may not be active");
    }
    if (!checkASLR()) {
        log(LogLevel::Warning, "ASLR may not be active");
    }

    log(LogLevel::Info, "SecurityManager initialized (HW RNG: {}, Level: {})",
        m_hasHwRng ? "yes" : "no",
        static_cast<int>(m_securityLevel));

    return {};
}

// ---- Hardware RNG ----
bool SecurityManager::probeHardwareRNG() {
    // Try /dev/hwrng first
    m_hwrngFd = open("/dev/hwrng", O_RDONLY | O_NONBLOCK);
    if (m_hwrngFd >= 0) {
        m_hasHwRng = true;
        return true;
    }
    // Try /dev/random
    m_hwrngFd = open("/dev/random", O_RDONLY | O_NONBLOCK);
    if (m_hwrngFd >= 0) {
        m_hasHwRng = true;
        return true;
    }
    m_hasHwRng = false;
    return false;
}

Result<void> SecurityManager::getHardwareRandom(uint8_t* out, size_t len) {
    if (m_hasHwRng && m_hwrngFd >= 0) {
        ssize_t readBytes = read(m_hwrngFd, out, len);
        if (static_cast<size_t>(readBytes) == len) {
            return {};
        }
    }
    // Fallback to OpenSSL CSPRNG
    return crypto().randomBytes(out, len);
}

// ---- Random Timing Delay ----
void SecurityManager::randomTimingDelay(uint32_t maxMicroseconds) {
    if (maxMicroseconds == 0) return;

    uint8_t rnd[2];
    if (RAND_bytes(rnd, sizeof(rnd)) != 1) return;

    uint16_t delay = (static_cast<uint16_t>(rnd[0]) << 8 | rnd[1]) % maxMicroseconds;
    std::this_thread::sleep_for(std::chrono::microseconds(delay));
}

// ---- Stack Protection Check ----
bool SecurityManager::checkStackProtection() {
#if defined(__STACK_CHK_GUARD) || defined(__SSP__) || defined(__SSP_STRONG__)
    return true;
#else
    // Check if compiler defines __stack_chk_guard
    return false;
#endif
}

// ---- ASLR Check ----
bool SecurityManager::checkASLR() {
    // Read /proc/self/maps and check for randomized addresses
    int fd = open("/proc/self/maps", O_RDONLY);
    if (fd < 0) return false;

    char buf[256];
    ssize_t n = read(fd, buf, sizeof(buf) - 1);
    close(fd);

    if (n <= 0) return false;
    buf[n] = '\0';

    // Simple check: if the first mapped address is not a fixed one, ASLR is likely active
    // Non-PIE binaries typically load at 0x400000
    std::string_view maps(buf, n);
    return !maps.starts_with("00400000");
}

// ---- Anti-Debug ----
bool SecurityManager::checkAntiDebug() {
    // ptrace check: a process can only be traced by one tracer
    if (ptrace(PTRACE_TRACEME, 0, nullptr, nullptr) == -1) {
        return false; // Already being traced
    }
    // Detach immediately
    ptrace(PTRACE_DETACH, 0, nullptr, nullptr);
    return true;
}

// ---- Secure Wipe ----
void SecurityManager::secureWipe(void* ptr, size_t size) {
    if (ptr && size > 0) {
        OPENSSL_cleanse(ptr, size);
    }
}

// ---- Guard Pages ----
void SecurityManager::setupGuardPages() {
    // Allocate a page with no access — accesses will segfault
    // This guards against buffer overflows into adjacent key material
    void* page = mmap(nullptr, 4096, PROT_NONE,
                      MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
    if (page != MAP_FAILED) {
        m_guardPage = page;
    }
}

// ---- Integrity Verification ----
Result<bool> SecurityManager::verifyIntegrity(
    std::span<const uint8_t> codeSegment,
    std::span<const uint8_t> expectedHash)
{
    auto actualHash = crypto().sha256(codeSegment);
    if (!actualHash.ok()) return {.error = actualHash.error};

    bool match = constantTimeEquals(actualHash.value.data(),
                                     expectedHash.data(),
                                     SHA256_DIGEST_SIZE);
    if (!match) {
        m_securityEvents.fetch_add(1, std::memory_order_relaxed);
        log(LogLevel::Error, "Integrity check failed!");
    }
    return {.value = match};
}

} // namespace ehw
