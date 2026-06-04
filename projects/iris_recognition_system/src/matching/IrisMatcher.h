#pragma once

#include "domain/IrisTemplate.h"
#include "domain/RecognitionResult.h"  // MatchResult
#include "model/IrisFeatureExtractor.h" // IrisEmbedding
#include <cstdint>
#include <string>
#include <vector>

namespace iris {

/// Iris matcher using Hamming distance between IrisCodes
/// with rotation compensation (Daugman's approach).
class IrisMatcher {
public:
    explicit IrisMatcher(float threshold = 0.35f);

    /// Match a query embedding against a registered template
    MatchResult matchOne(const IrisEmbedding& query,
                          const IrisTemplate& registeredTemplate);

    /// Search for the best match among multiple templates
    MatchResult search(const IrisEmbedding& query,
                        const std::vector<IrisTemplate>& templates);

    /// Compute Hamming distance between two IrisCodes
    /// Accounts for mask bits and rotation compensation
    float hammingDistance(const std::vector<uint8_t>& codeA,
                           const std::vector<uint8_t>& maskA,
                           const std::vector<uint8_t>& codeB,
                           const std::vector<uint8_t>& maskB,
                           int shiftBits = 0);

    /// Cosine similarity for float embeddings
    static float cosineSimilarity(const std::vector<float>& a,
                                   const std::vector<float>& b);

    void setThreshold(float t) { m_threshold = t; }
    float getThreshold() const { return m_threshold; }

private:
    /// Count set bits in a byte
    static int popcount(uint8_t byte);

    /// Count set bits in a byte array
    static int popcountArray(const std::vector<uint8_t>& data);

    float m_threshold;
    static constexpr int MAX_ROTATION_SHIFT = 16; // bits shift for rotation
};

} // namespace iris
