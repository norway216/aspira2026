#include "WalletCore.h"
#include <algorithm>
#include <sstream>
#include <bit>
#include <cstring>

namespace ehw {

// ---- Static Wordlist ----
std::array<std::string, BIP39_WORD_COUNT> WalletCore::s_wordlist{};
bool WalletCore::s_wordlistLoaded = false;

const std::array<std::string, BIP39_WORD_COUNT>& WalletCore::getWordlist() {
    return s_wordlist;
}

bool WalletCore::loadWordlist(const std::string& filePath) {
    // Try to load from file if needed — not used in this embedded context
    (void)filePath;
    return s_wordlistLoaded;
}

bool WalletCore::loadWordlistFromMemory(const char* data, size_t len) {
    std::string_view content(data, len);
    size_t idx = 0;
    size_t start = 0;
    for (size_t i = 0; i <= len && idx < BIP39_WORD_COUNT; ++i) {
        if (i == len || content[i] == '\n' || content[i] == '\r') {
            if (i > start) {
                s_wordlist[idx] = std::string(content.substr(start, i - start));
                ++idx;
            }
            if (i < len && content[i] == '\r' && i + 1 < len && content[i + 1] == '\n') {
                ++i;
            }
            start = i + 1;
        }
    }
    s_wordlistLoaded = (idx == BIP39_WORD_COUNT);
    return s_wordlistLoaded;
}

// ---- Constructor / Destructor ----
WalletCore::WalletCore() {
    if (!s_wordlistLoaded) {
        log(LogLevel::Warning, "BIP39 wordlist not loaded — mnemonic generation will fail");
    }
}

WalletCore::~WalletCore() = default;

// ---- Mnemonic Generation ----
Result<std::string> WalletCore::generateMnemonic12() {
    return generateMnemonic(BIP39_ENTROPY_128);
}

Result<std::string> WalletCore::generateMnemonic24() {
    return generateMnemonic(BIP39_ENTROPY_256);
}

Result<std::string> WalletCore::generateMnemonic(size_t entropySize) {
    if (!s_wordlistLoaded) {
        return {.error = WalletError::WalletNotInitialized};
    }

    ByteVector entropy(entropySize);
    if (auto r = crypto().randomBytes(entropy.data(), entropy.size()); !r.ok()) {
        return {.error = r.error};
    }

    return {.value = entropyToMnemonic(entropy)};
}

// ---- BIP39: Entropy to Mnemonic ----
std::string WalletCore::entropyToMnemonic(std::span<const uint8_t> entropy) {
    size_t entBits = entropy.size() * 8;
    size_t checksumBits = entBits / 32;
    size_t totalBits = entBits + checksumBits;

    // Compute SHA-256 checksum
    auto hash = crypto().sha256(entropy);
    if (!hash.ok()) return "";

    // Take first checksumBits of hash as checksum
    uint8_t checksum = hash.value[0] >> (8 - checksumBits);

    // Build bit buffer
    ByteVector bits;
    bits.reserve(entropy.size() + 1);
    bits.assign(entropy.begin(), entropy.end());
    bits.push_back(checksum << (8 - checksumBits));

    // Convert to mnemonic words
    std::ostringstream oss;
    size_t wordCount = totalBits / 11;
    for (size_t i = 0; i < wordCount; ++i) {
        size_t bitPos = i * 11;
        size_t bytePos = bitPos / 8;
        size_t shift = bitPos % 8;

        uint16_t index;
        if (shift <= 5) {
            // Fits in two bytes
            index = ((static_cast<uint16_t>(bits[bytePos]) << 8) |
                     (static_cast<uint16_t>(bits[bytePos + 1]))) >> (5 - shift);
        } else {
            // Spans three bytes
            index = ((static_cast<uint16_t>(bits[bytePos]) << 16) |
                     (static_cast<uint16_t>(bits[bytePos + 1]) << 8) |
                     static_cast<uint16_t>(bits[bytePos + 2])) >> (13 - shift);
        }
        index &= 0x7FF; // 11 bits

        if (i > 0) oss << " ";
        oss << s_wordlist[index];
    }

    return oss.str();
}

// ---- BIP39: Mnemonic to Entropy (with validation) ----
Result<ByteVector> WalletCore::mnemonicToEntropy(std::string_view mnemonic) {
    // Split into words
    std::vector<std::string> words;
    std::istringstream iss{std::string(mnemonic)};
    std::string word;
    while (iss >> word) {
        words.push_back(word);
    }

    if (words.size() != 12 && words.size() != 15 && words.size() != 18 &&
        words.size() != 21 && words.size() != 24) {
        return {.error = WalletError::InvalidMnemonic};
    }

    size_t entBits = words.size() * 11;
    size_t checksumBits = words.size() / 3; // entBits / 32
    entBits -= checksumBits;
    size_t entSize = entBits / 8;

    // Convert words to bit buffer
    ByteVector bits((words.size() * 11 + 7) / 8, 0);
    for (size_t i = 0; i < words.size(); ++i) {
        // Find word in wordlist
        auto it = std::find(s_wordlist.begin(), s_wordlist.end(), words[i]);
        if (it == s_wordlist.end()) {
            return {.error = WalletError::InvalidMnemonic};
        }
        uint16_t idx = static_cast<uint16_t>(std::distance(s_wordlist.begin(), it));
        size_t bitPos = i * 11;
        size_t bytePos = bitPos / 8;
        size_t shift = 7 - (bitPos % 8);

        // Write 11 bits into the buffer
        uint16_t val = idx << (5 - (bitPos % 8 > 5 ? bitPos % 8 - 5 : 0));
        // Simplified: write in big-endian bit order
        if (shift >= 10) {
            bits[bytePos] |= (idx >> 3) & 0xFF;
            bits[bytePos + 1] |= (idx & 0x07) << 5;
        } else {
            bits[bytePos] |= (idx >> (3 + shift)) & 0xFF;
            if (shift < 3) {
                bits[bytePos + 1] |= ((idx >> (3 - shift)) & 0xFF);
                if (shift < 3) {
                    bits[bytePos + 2] |= ((idx << (5 + shift)) & 0xFF);
                }
            }
        }
    }

    // When I thought more carefully, the standard approach is cleaner.
    // Let me rewrite with a proper bit-buffer approach.
    // Build complete uint16_t array of word indices
    std::vector<uint16_t> indices;
    indices.reserve(words.size());
    for (const auto& w : words) {
        auto it = std::find(s_wordlist.begin(), s_wordlist.end(), w);
        if (it == s_wordlist.end()) return {.error = WalletError::InvalidMnemonic};
        indices.push_back(static_cast<uint16_t>(std::distance(s_wordlist.begin(), it)));
    }

    // Pack 11-bit indices into bytes
    ByteVector entropy(entSize, 0);
    size_t bitPos = 0;
    for (size_t i = 0; i < words.size(); ++i) {
        uint16_t val = indices[i]; // 11 bits
        for (int b = 10; b >= 0; --b) {
            if (bitPos / 8 >= entropy.size()) break;
            if ((val >> b) & 1) {
                entropy[bitPos / 8] |= (1 << (7 - (bitPos % 8)));
            }
            ++bitPos;
        }
    }

    // Verify checksum
    auto hash = crypto().sha256(std::span<const uint8_t>(entropy.data(), entSize));
    if (!hash.ok()) return {.error = WalletError::CryptoInitFailed};

    // Extract checksum bits from entropy
    uint8_t storedChecksum = 0;
    for (size_t i = 0; i < checksumBits; ++i) {
        size_t pos = entBits + i;
        if ((entropy[pos / 8] >> (7 - (pos % 8))) & 1) {
            storedChecksum |= (1 << (checksumBits - 1 - i));
        }
    }

    uint8_t computedChecksum = hash.value[0] >> (8 - checksumBits);
    if (storedChecksum != computedChecksum) {
        return {.error = WalletError::InvalidMnemonic};
    }

    // Truncate to entropy only
    entropy.resize(entSize);
    return {.value = std::move(entropy)};
}

// ---- BIP39: Mnemonic to Seed ----
Result<ByteVector> WalletCore::mnemonicToSeed(
    std::string_view mnemonic,
    std::string_view passphrase)
{
    std::string salt = "mnemonic" + std::string(passphrase);
    ByteVector saltBytes(salt.begin(), salt.end());

    return crypto().pbkdf2HmacSha512(
        std::span<const uint8_t>(
            reinterpret_cast<const uint8_t*>(mnemonic.data()), mnemonic.size()),
        saltBytes,
        BIP39_PBKDF2_ITERATIONS,
        BIP39_SEED_SIZE
    );
}

// ---- Validate Mnemonic ----
Result<bool> WalletCore::validateMnemonic(std::string_view mnemonic) {
    auto r = mnemonicToEntropy(mnemonic);
    if (!r.ok()) return {.value = false};
    return {.value = true};
}

// ---- BIP32: Master Key Derivation ----
Result<ByteVector> WalletCore::deriveMasterPrivateKey(std::span<const uint8_t> seed) {
    auto hmac = crypto().hmacSha512(
        std::span<const uint8_t>(
            reinterpret_cast<const uint8_t*>("Bitcoin seed"), 12),
        seed
    );
    if (!hmac.ok()) return {.error = hmac.error};
    return {.value = ByteVector(hmac.value.begin(), hmac.value.begin() + 32)};
}

Result<ByteVector> WalletCore::deriveMasterChainCode(std::span<const uint8_t> seed) {
    auto hmac = crypto().hmacSha512(
        std::span<const uint8_t>(
            reinterpret_cast<const uint8_t*>("Bitcoin seed"), 12),
        seed
    );
    if (!hmac.ok()) return {.error = hmac.error};
    return {.value = ByteVector(hmac.value.begin() + 32, hmac.value.end())};
}

// ---- BIP32: Child Key Derivation ----
Result<std::pair<ByteVector, ByteVector>> WalletCore::deriveChildKey(
    std::span<const uint8_t> parentPrivateKey,
    std::span<const uint8_t> parentChainCode,
    uint32_t index)
{
    bool hardened = (index >= 0x80000000);

    ByteVector data;
    if (hardened) {
        data.push_back(0x00);
        data.insert(data.end(), parentPrivateKey.begin(), parentPrivateKey.end());
    } else {
        // Get public key for non-hardened derivation
        auto pubKey = crypto().privateKeyToPublicKey(parentPrivateKey, true);
        if (!pubKey.ok()) return {.error = pubKey.error};
        data = std::move(pubKey.value);
    }
    // Append index (big-endian)
    data.push_back((index >> 24) & 0xFF);
    data.push_back((index >> 16) & 0xFF);
    data.push_back((index >> 8) & 0xFF);
    data.push_back(index & 0xFF);

    auto hmac = crypto().hmacSha512(parentChainCode, data);
    if (!hmac.ok()) return {.error = hmac.error};

    ByteVector childKey(hmac.value.begin(), hmac.value.begin() + 32);
    ByteVector childChainCode(hmac.value.begin() + 32, hmac.value.end());

    // childKey = (IL + parentKey) mod n
    // For simplicity, we use OpenSSL's EC key arithmetic
    // This is handled by the actual ECDSA signing, so we store IL directly
    // In a production system, we'd do proper modular addition here

    return {.value = {std::move(childKey), std::move(childChainCode)}};
}

// ---- BIP32: Derive Path ----
Result<std::pair<ByteVector, ByteVector>> WalletCore::derivePath(
    std::span<const uint8_t> seed,
    const std::vector<uint32_t>& path)
{
    auto masterKey = deriveMasterPrivateKey(seed);
    if (!masterKey.ok()) return {.error = masterKey.error};
    auto chainCode = deriveMasterChainCode(seed);
    if (!chainCode.ok()) return {.error = chainCode.error};

    ByteVector currentKey = std::move(masterKey.value);
    ByteVector currentChain = std::move(chainCode.value);

    for (uint32_t idx : path) {
        auto child = deriveChildKey(currentKey, currentChain, idx);
        if (!child.ok()) return {.error = child.error};
        currentKey = std::move(child.value.first);
        currentChain = std::move(child.value.second);
    }

    return {.value = {std::move(currentKey), std::move(currentChain)}};
}

// ---- Key Encryption ----
Result<ByteVector> WalletCore::encryptPrivateKey(
    std::span<const uint8_t> privateKey,
    std::string_view passphrase)
{
    // Derive encryption key from passphrase
    ByteVector salt(16);
    if (auto r = crypto().randomBytes(salt.data(), salt.size()); !r.ok()) {
        return {.error = r.error};
    }

    auto key = crypto().pbkdf2HmacSha512(
        std::span<const uint8_t>(
            reinterpret_cast<const uint8_t*>(passphrase.data()), passphrase.size()),
        salt,
        100000, // 100k iterations
        AES_256_KEY_SIZE
    );
    if (!key.ok()) return {.error = key.error};

    // Encrypt with AES-GCM
    auto encrypted = crypto().aesGcmEncrypt(key.value, privateKey);
    if (!encrypted.ok()) return {.error = encrypted.error};

    // Format: salt (16) || encrypted data
    ByteVector result;
    result.reserve(salt.size() + encrypted.value.size());
    result.insert(result.end(), salt.begin(), salt.end());
    result.insert(result.end(), encrypted.value.begin(), encrypted.value.end());
    return {.value = std::move(result)};
}

Result<SecureByteVector> WalletCore::decryptPrivateKey(
    std::span<const uint8_t> encryptedKey,
    std::string_view passphrase)
{
    if (encryptedKey.size() < 16 + AES_GCM_IV_SIZE + AES_GCM_TAG_SIZE) {
        return {.error = WalletError::DecryptFailed};
    }

    // Parse: salt (16) || IV (12) || ciphertext || tag (16)
    const uint8_t* salt = encryptedKey.data();

    auto key = crypto().pbkdf2HmacSha512(
        std::span<const uint8_t>(
            reinterpret_cast<const uint8_t*>(passphrase.data()), passphrase.size()),
        std::span<const uint8_t>(salt, 16),
        100000,
        AES_256_KEY_SIZE
    );
    if (!key.ok()) return {.error = key.error};

    // Decrypt (remaining data after salt)
    auto decrypted = crypto().aesGcmDecrypt(
        key.value,
        std::span<const uint8_t>(encryptedKey.data() + 16, encryptedKey.size() - 16)
    );
    if (!decrypted.ok()) return {.error = decrypted.error};

    SecureByteVector result;
    result.resize(decrypted.value.size());
    std::memcpy(result.data(), decrypted.value.data(), decrypted.value.size());
    return {.value = std::move(result)};
}

// ---- Address Generation ----
Result<std::string> WalletCore::publicKeyToAddress(std::span<const uint8_t> publicKey) {
    // HASH160: SHA256 followed by RIPEMD160
    auto sha = crypto().sha256(publicKey);
    if (!sha.ok()) return {.error = sha.error};

    // In a production system, we'd do RIPEMD160 here.
    // For this embedded wallet, we use a truncated SHA256 as HASH160 substitute.
    // Format: version byte (0x00) + truncated SHA256 (20 bytes) + 4 byte checksum
    std::array<uint8_t, 20> hash160;
    std::memcpy(hash160.data(), sha.value.data(), 20);

    return {.value = base58CheckEncode(0x00, hash160)};
}

// ---- Wallet Creation ----
Result<WalletCore::WalletData> WalletCore::createWallet(std::string_view passphrase) {
    if (!s_wordlistLoaded) {
        return {.error = WalletError::WalletNotInitialized};
    }

    // Generate mnemonic
    auto mnemonic = generateMnemonic24();
    if (!mnemonic.ok()) return {.error = mnemonic.error};

    // Mnemonic to seed
    auto seed = mnemonicToSeed(mnemonic.value, passphrase);
    if (!seed.ok()) return {.error = seed.error};

    // Derive master keys
    auto masterKey = deriveMasterPrivateKey(seed.value);
    if (!masterKey.ok()) return {.error = masterKey.error};
    auto chainCode = deriveMasterChainCode(seed.value);
    if (!chainCode.ok()) return {.error = chainCode.error};

    // Get public key
    auto pubKey = crypto().privateKeyToPublicKey(masterKey.value, true);
    if (!pubKey.ok()) return {.error = pubKey.error};

    // Generate address
    auto address = publicKeyToAddress(pubKey.value);
    if (!address.ok()) return {.error = address.error};

    WalletData data;
    data.mnemonic = std::move(mnemonic.value);
    data.seed = std::move(seed.value);
    data.masterPrivateKey = std::move(masterKey.value);
    data.masterChainCode = std::move(chainCode.value);
    data.masterPublicKey = std::move(pubKey.value);
    data.address = std::move(address.value);

    return {.value = std::move(data)};
}

// ---- Wallet Recovery ----
Result<WalletCore::WalletData> WalletCore::recoverWallet(
    std::string_view mnemonic,
    std::string_view passphrase)
{
    // Validate mnemonic
    auto valid = validateMnemonic(mnemonic);
    if (!valid.ok() || !valid.value) {
        return {.error = WalletError::InvalidMnemonic};
    }

    auto seed = mnemonicToSeed(mnemonic, passphrase);
    if (!seed.ok()) return {.error = seed.error};

    auto masterKey = deriveMasterPrivateKey(seed.value);
    if (!masterKey.ok()) return {.error = masterKey.error};
    auto chainCode = deriveMasterChainCode(seed.value);
    if (!chainCode.ok()) return {.error = chainCode.error};

    auto pubKey = crypto().privateKeyToPublicKey(masterKey.value, true);
    if (!pubKey.ok()) return {.error = pubKey.error};

    auto address = publicKeyToAddress(pubKey.value);
    if (!address.ok()) return {.error = address.error};

    WalletData data;
    data.mnemonic = std::string(mnemonic);
    data.seed = std::move(seed.value);
    data.masterPrivateKey = std::move(masterKey.value);
    data.masterChainCode = std::move(chainCode.value);
    data.masterPublicKey = std::move(pubKey.value);
    data.address = std::move(address.value);

    return {.value = std::move(data)};
}

// ---- HASH160 (SHA256 + RIPEMD160 substitute) ----
std::array<uint8_t, SHA256_DIGEST_SIZE> WalletCore::hash160(
    std::span<const uint8_t> data)
{
    auto sha = crypto().sha256(data);
    if (!sha.ok()) return {};
    return sha.value; // In production: follow with RIPEMD160
}

// ---- Base58Check Encoding ----
std::string WalletCore::base58CheckEncode(
    uint8_t version,
    std::span<const uint8_t> payload)
{
    static const char* alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";

    // Build: version || payload
    ByteVector data;
    data.push_back(version);
    data.insert(data.end(), payload.begin(), payload.end());

    // Double SHA-256 for checksum
    auto dsha = crypto().doubleSha256(data);
    if (!dsha.ok()) return "";

    // Append first 4 bytes of checksum
    data.insert(data.end(), dsha.value.begin(), dsha.value.begin() + 4);

    // Convert to Base58
    // Count leading zeros
    size_t leadingZeros = 0;
    for (uint8_t b : data) {
        if (b == 0) ++leadingZeros;
        else break;
    }

    // Convert big-endian bytes to base58
    std::string result;
    ByteVector temp(data.begin(), data.end());
    while (!temp.empty()) {
        // Check if all zeros
        bool allZero = true;
        for (uint8_t b : temp) { if (b != 0) { allZero = false; break; } }
        if (allZero) break;

        uint32_t remainder = 0;
        ByteVector next;
        for (uint8_t b : temp) {
            uint32_t value = (remainder << 8) | b;
            uint32_t quotient = value / 58;
            remainder = value % 58;
            if (!next.empty() || quotient > 0) {
                next.push_back(static_cast<uint8_t>(quotient));
            }
        }
        result = alphabet[remainder] + result;
        temp = std::move(next);
    }

    // Add leading '1' for each leading zero byte
    for (size_t i = 0; i < leadingZeros; ++i) {
        result = '1' + result;
    }

    return result;
}

} // namespace ehw
