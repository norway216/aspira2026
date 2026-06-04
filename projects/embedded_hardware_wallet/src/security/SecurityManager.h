#pragma once

#include "Common.h"
#include "core/CryptoProvider.h"
#include "core/SecureMemory.h"
#include <mutex>
#include <atomic>
#include <chrono>

namespace ehw {

/**
 * SecurityManager — Hardware security module for the embedded wallet.
 *
 * Features:
 * - Hardware RNG access (/dev/hwrng fallback to OpenSSL CSPRNG)
 * - Secure enclave simulation (isolated memory operations)
 * - Timing attack mitigation (constant-time comparison, randomized delays)
 * - Side-channel resistance (operation time randomization)
 * - Stack canary & ASLR status validation
 * - Secure wipe of temporary cryptographic buffers
 * - Intrusion detection simulation
 */
class SecurityManager {
public:
    SecurityManager();
    ~SecurityManager();

    SecurityManager(const SecurityManager&) = delete;
    SecurityManager& operator=(const SecurityManager&) = delete;

    // ---- Initialization ----
    Result<void> initialize();

    // ---- Hardware RNG ----
    // Get cryptographically secure random bytes (prefers HW RNG)
    Result<void> getHardwareRandom(uint8_t* out, size_t len);

    // Check if hardware RNG is available
    bool hasHardwareRNG() const { return m_hasHwRng; }

    // ---- Secure Enclave Simulation ----
    // Execute a function in an isolated secure context
    template <typename F>
    Result<std::invoke_result_t<F>> executeInEnclave(F&& func) {
        std::lock_guard lock(m_enclaveMutex);
        m_inEnclave.store(true, std::memory_order_release);

        try {
            auto result = func();
            m_inEnclave.store(false, std::memory_order_release);
            return {.value = std::move(result)};
        } catch (const std::exception& e) {
            m_inEnclave.store(false, std::memory_order_release);
            log(LogLevel::Error, "Enclave execution failed: {}", e.what());
            return {.error = WalletError::SecurityViolation};
        }
    }

    // ---- Timing Attack Mitigation ----
    // Constant-time comparison
    template <typename T>
    static bool constantTimeCompare(const T& a, const T& b) {
        return CryptoProvider::constantTimeCompare(
            std::span<const uint8_t>(reinterpret_cast<const uint8_t*>(&a), sizeof(T)),
            std::span<const uint8_t>(reinterpret_cast<const uint8_t*>(&b), sizeof(T))
        );
    }

    // Add random delay to mask operation timing
    static void randomTimingDelay(uint32_t maxMicroseconds = 1000);

    // ---- Security Validation ----
    // Check stack canaries are active
    static bool checkStackProtection();

    // Verify ASLR is enabled
    static bool checkASLR();

    // Check that we're not being debugged (basic anti-debug)
    static bool checkAntiDebug();

    // ---- Secure Wipe ----
    void secureWipe(void* ptr, size_t size);

    // ---- Integrity Check ----
    // Verify code integrity (simple checksum)
    Result<bool> verifyIntegrity(std::span<const uint8_t> codeSegment,
                                  std::span<const uint8_t> expectedHash);

    // ---- Security Events ----
    uint32_t securityEventCount() const { return m_securityEvents.load(); }
    void resetSecurityEventCount() { m_securityEvents.store(0); }

    enum class SecurityLevel : uint8_t {
        Standard = 0,
        Enhanced = 1,
        Maximum = 2
    };
    void setSecurityLevel(SecurityLevel level) { m_securityLevel = level; }
    SecurityLevel securityLevel() const { return m_securityLevel; }

private:
    bool probeHardwareRNG();
    void setupGuardPages();

    bool m_hasHwRng = false;
    int m_hwrngFd = -1;
    std::mutex m_enclaveMutex;
    std::atomic<bool> m_inEnclave{false};
    std::atomic<uint32_t> m_securityEvents{0};
    SecurityLevel m_securityLevel{SecurityLevel::Enhanced};
    void* m_guardPage = nullptr;
};

} // namespace ehw
