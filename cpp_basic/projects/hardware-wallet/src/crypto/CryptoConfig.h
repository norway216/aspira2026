#pragma once

#include <cstddef>

namespace CryptoConfig {
    // Key sizes (bytes)
    constexpr size_t KEY_SIZE = 32;          // AES-256 key size
    constexpr size_t SALT_SIZE = 16;         // Salt for PBKDF2
    constexpr size_t NONCE_SIZE = 12;        // AES-256-GCM IV/nonce size
    constexpr size_t TAG_SIZE = 16;          // GCM authentication tag
    constexpr size_t SIGNATURE_SIZE = 64;    // Ed25519 signature size
    constexpr size_t PUBLIC_KEY_SIZE = 32;   // Ed25519 public key
    constexpr size_t SECRET_KEY_SIZE = 32;   // Ed25519 private key (seed)
    constexpr size_t SEED_SIZE = 32;         // Wallet seed
    constexpr size_t HASH_SIZE = 32;         // SHA-256 hash size

    // PBKDF2 parameters
    constexpr int PBKDF2_ITERATIONS = 100000;  // High iteration count
    constexpr int PBKDF2_ITERATIONS_MODERATE = 10000;  // For testing

    // Encrypted blob format: [NONCE(12)][CIPHERTEXT + TAG]
    constexpr size_t ENCRYPTED_OVERHEAD = NONCE_SIZE + TAG_SIZE;
}
