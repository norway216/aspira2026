#pragma once

#include "domain/RecognitionResult.h"
#include <string>

namespace iris {

/// Combines match, liveness, and quality scores into a final decision
class DecisionEngine {
public:
    DecisionEngine();

    /// Make a final accept/reject decision
    FinalDecision decide(const MatchResult& match,
                          const LivenessResult& liveness,
                          const QualityResult& quality);

    /// Set score weights
    void setWeights(float matchW, float livenessW, float qualityW) {
        m_matchWeight    = matchW;
        m_livenessWeight = livenessW;
        m_qualityWeight  = qualityW;
    }

private:
    float m_matchWeight    = 0.45f;
    float m_livenessWeight = 0.35f;
    float m_qualityWeight  = 0.20f;
};

} // namespace iris
