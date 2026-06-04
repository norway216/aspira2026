#pragma once

#include "Common.h"
#include "CryptoProvider.h"
#include "SecureMemory.h"
#include <string>
#include <array>
#include <span>

namespace ehw {

/**
 * WalletCore — BIP39 mnemonic generation/recovery + BIP32 HD wallet.
 *
 * BIP39: Mnemonic generation from entropy, mnemonic-to-seed via PBKDF2.
 * BIP32: Master key derivation from seed, child key derivation (hardened/non-hardened).
 * Key encryption: AES-256-GCM with user passphrase-derived key.
 */
class WalletCore {
public:
    WalletCore();
    ~WalletCore();

    WalletCore(const WalletCore&) = delete;
    WalletCore& operator=(const WalletCore&) = delete;

    // ---- BIP39 Mnemonic Generation ----
    // Generate a 12-word mnemonic (128 bits entropy)
    Result<std::string> generateMnemonic12();

    // Generate a 24-word mnemonic (256 bits entropy)
    Result<std::string> generateMnemonic24();

    // Validate a mnemonic phrase (checksum verification)
    Result<bool> validateMnemonic(std::string_view mnemonic);

    // Convert mnemonic to seed (BIP39: PBKDF2-HMAC-SHA512, 2048 iterations)
    Result<ByteVector> mnemonicToSeed(
        std::string_view mnemonic,
        std::string_view passphrase = ""
    );

    // ---- BIP32 HD Wallet ----
    // Derive master key from seed
    Result<ByteVector> deriveMasterPrivateKey(std::span<const uint8_t> seed);
    Result<ByteVector> deriveMasterChainCode(std::span<const uint8_t> seed);

    // Derive child key (hardened if index >= 0x80000000)
    Result<std::pair<ByteVector, ByteVector>> deriveChildKey(
        std::span<const uint8_t> parentPrivateKey,
        std::span<const uint8_t> parentChainCode,
        uint32_t index
    );

    // Derive BIP44 path: m/44'/coin'/account'/change/address_index
    Result<std::pair<ByteVector, ByteVector>> derivePath(
        std::span<const uint8_t> seed,
        const std::vector<uint32_t>& path
    );

    // ---- Key Encryption ----
    // Encrypt private key with user passphrase
    Result<ByteVector> encryptPrivateKey(
        std::span<const uint8_t> privateKey,
        std::string_view passphrase
    );

    // Decrypt private key
    Result<SecureByteVector> decryptPrivateKey(
        std::span<const uint8_t> encryptedKey,
        std::string_view passphrase
    );

    // ---- Address Generation ----
    // Generate a simple address from public key (SHA256 + RIPEMD160 + Base58Check)
    Result<std::string> publicKeyToAddress(std::span<const uint8_t> publicKey);

    // ---- Wallet Creation ----
    // Create a full wallet: returns mnemonic, seed, master key, and address
    struct WalletData {
        std::string mnemonic;
        ByteVector seed;
        ByteVector masterPrivateKey;
        ByteVector masterChainCode;
        ByteVector masterPublicKey;
        std::string address;
    };
    Result<WalletData> createWallet(std::string_view passphrase = "");

    // Recover wallet from mnemonic
    Result<WalletData> recoverWallet(
        std::string_view mnemonic,
        std::string_view passphrase = ""
    );

    // ---- BIP39 Wordlist ----
    static const std::array<std::string, BIP39_WORD_COUNT>& getWordlist();
    static bool loadWordlist(const std::string& filePath);
    static bool loadWordlistFromMemory(const char* data, size_t len);

private:
    // BIP39 internal helpers
    Result<std::string> generateMnemonic(size_t entropySize);
    static std::string entropyToMnemonic(std::span<const uint8_t> entropy);
    static Result<ByteVector> mnemonicToEntropy(std::string_view mnemonic);

    // BIP32 internal helpers
    static ByteVector serializeExtendedKey(
        std::span<const uint8_t> key,
        std::span<const uint8_t> chainCode,
        uint8_t depth,
        uint32_t parentFingerprint,
        uint32_t childIndex,
        bool isPrivate
    );

    static std::array<uint8_t, SHA256_DIGEST_SIZE> hash160(
        std::span<const uint8_t> data
    );

    static std::string base58CheckEncode(
        uint8_t version,
        std::span<const uint8_t> payload
    );

    // Wordlist
    static std::array<std::string, BIP39_WORD_COUNT> s_wordlist;
    static bool s_wordlistLoaded;
};

} // namespace ehw
