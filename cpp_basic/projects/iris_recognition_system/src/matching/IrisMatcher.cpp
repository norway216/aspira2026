#include "matching/IrisMatcher.h"
#include <cmath>
#include <algorithm>
#include <limits>
#include <stdexcept>

namespace iris {

IrisMatcher::IrisMatcher(float threshold)
    : m_threshold(threshold) {}

MatchResult IrisMatcher::matchOne(const IrisEmbedding& query,
                                   const IrisTemplate& tmpl) {
    MatchResult result;
    result.user_id = tmpl.user_id;

    // Prefer IrisCode matching if available
    if (!query.iris_code.empty() && !tmpl.iris_code.empty()) {
        float bestDist = 1.0f;

        // Try multiple rotation shifts
        for (int shift = -MAX_ROTATION_SHIFT; shift <= MAX_ROTATION_SHIFT; ++shift) {
            float dist = hammingDistance(
                query.iris_code, query.mask_code,
                tmpl.iris_code, tmpl.mask_code,
                shift);
            if (dist < bestDist) bestDist = dist;
        }

        result.hamming_dist = bestDist;
        result.similarity    = 1.0f - bestDist;
        result.matched       = (bestDist <= m_threshold);
    }
    // Fallback to cosine similarity on float embeddings
    else if (!query.vector.empty() && !tmpl.embedding.empty()) {
        float sim = cosineSimilarity(query.vector, tmpl.embedding);
        result.similarity = sim;
        result.hamming_dist = 1.0f - sim;
        result.matched = (sim >= (1.0f - m_threshold));
    }
    else {
        result.similarity = 0.0f;
        result.hamming_dist = 1.0f;
        result.matched = false;
    }

    return result;
}

MatchResult IrisMatcher::search(const IrisEmbedding& query,
                                 const std::vector<IrisTemplate>& templates) {
    MatchResult best;
    best.similarity = -1.0f;
    best.hamming_dist = 1.0f;
    best.matched = false;

    for (const auto& tmpl : templates) {
        MatchResult r = matchOne(query, tmpl);
        if (r.similarity > best.similarity) {
            best = r;
        }
    }

    return best;
}

float IrisMatcher::hammingDistance(const std::vector<uint8_t>& codeA,
                                    const std::vector<uint8_t>& maskA,
                                    const std::vector<uint8_t>& codeB,
                                    const std::vector<uint8_t>& maskB,
                                    int shiftBits) {
    if (codeA.empty() || codeB.empty()) return 1.0f;

    size_t numBytes = std::min({codeA.size(), codeB.size(),
                                 maskA.empty() ? codeA.size() : maskA.size(),
                                 maskB.empty() ? codeB.size() : maskB.size()});

    if (numBytes == 0) return 1.0f;

    size_t totalBits   = numBytes * 8;
    size_t validBits   = 0;
    size_t differBits  = 0;

    // Fast path: no rotation shift, use byte-level XOR + popcount (SIMD-friendly)
    if (shiftBits == 0) {
        bool hasMasks = !maskA.empty() || !maskB.empty();
        for (size_t i = 0; i < numBytes; ++i) {
            uint8_t commonMask = 0xFF;
            if (hasMasks) {
                uint8_t ma = (i < maskA.size()) ? maskA[i] : 0xFF;
                uint8_t mb = (i < maskB.size()) ? maskB[i] : 0xFF;
                commonMask = ma & mb;
            }
            if (commonMask == 0) continue;
            uint8_t diff = (codeA[i] ^ codeB[i]) & commonMask;
            differBits += static_cast<size_t>(
                __builtin_popcount(static_cast<unsigned int>(diff)));
            validBits  += static_cast<size_t>(
                __builtin_popcount(static_cast<unsigned int>(commonMask)));
        }
    } else {
        // Rotation shift path: bit-by-bit with shift (rarely used; only during
        // rotation-compensated matching across the 33-shift loop)
        for (size_t byteIdx = 0; byteIdx < numBytes; ++byteIdx) {
            for (int bit = 0; bit < 8; ++bit) {
                size_t bitPos = byteIdx * 8 + bit;
                int shiftedPos = static_cast<int>(bitPos) + shiftBits;
                if (shiftedPos < 0 || shiftedPos >= static_cast<int>(totalBits)) continue;
                size_t shiftedByte = static_cast<size_t>(shiftedPos) / 8;
                int shiftedBit    = shiftedPos % 8;

                bool validA = maskA.empty() ||
                    (byteIdx < maskA.size() && (maskA[byteIdx] & (1 << bit)));
                bool validB = maskB.empty() ||
                    (shiftedByte < maskB.size() && (maskB[shiftedByte] & (1 << shiftedBit)));

                if (!validA || !validB) continue;
                ++validBits;

                uint8_t bitA = (codeA[byteIdx] >> bit) & 1;
                uint8_t bitB = (codeB[shiftedByte] >> shiftedBit) & 1;
                if (bitA != bitB) ++differBits;
            }
        }
    }

    if (validBits == 0) return 1.0f;
    return static_cast<float>(differBits) / static_cast<float>(validBits);
}

float IrisMatcher::cosineSimilarity(const std::vector<float>& a,
                                     const std::vector<float>& b) {
    if (a.empty() || b.empty()) return 0.0f;

    size_t n = std::min(a.size(), b.size());
    double dot = 0.0, normA = 0.0, normB = 0.0;

    for (size_t i = 0; i < n; ++i) {
        dot   += static_cast<double>(a[i]) * static_cast<double>(b[i]);
        normA += static_cast<double>(a[i]) * static_cast<double>(a[i]);
        normB += static_cast<double>(b[i]) * static_cast<double>(b[i]);
    }

    double denom = std::sqrt(normA) * std::sqrt(normB);
    if (denom < 1e-10) return 0.0f;

    return std::clamp(static_cast<float>(dot / denom), -1.0f, 1.0f);
}

int IrisMatcher::popcount(uint8_t byte) {
    // Built-in popcount
    return __builtin_popcount(static_cast<unsigned int>(byte));
}

int IrisMatcher::popcountArray(const std::vector<uint8_t>& data) {
    int count = 0;
    for (uint8_t byte : data) {
        count += popcount(byte);
    }
    return count;
}

} // namespace iris
