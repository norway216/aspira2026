#include "matching/DecisionEngine.h"
#include <iomanip>
#include <sstream>

namespace iris {

DecisionEngine::DecisionEngine() = default;

FinalDecision DecisionEngine::decide(const MatchResult& match,
                                      const LivenessResult& liveness,
                                      const QualityResult& quality) {
    FinalDecision decision;

    decision.match_score    = match.similarity;
    decision.liveness_score = liveness.live_score;
    decision.quality_score  = quality.overall;
    decision.user_id        = match.user_id;

    // Weighted composite score
    float composite = m_matchWeight    * match.similarity
                    + m_livenessWeight * liveness.live_score
                    + m_qualityWeight  * quality.overall;

    // Decision logic
    std::ostringstream reason;

    if (!quality.passed) {
        decision.accepted = false;
        reason << "Quality check failed: " << quality.reason << "; ";
    }

    if (!liveness.passed) {
        decision.accepted = false;
        reason << "Liveness check failed (score="
               << std::fixed << std::setprecision(2) << liveness.live_score
               << ", type=" << liveness.attack_type << "); ";
    }

    if (!match.matched) {
        decision.accepted = false;
        reason << "No matching template found (similarity="
               << std::fixed << std::setprecision(3) << match.similarity << "); ";
    }

    // If all checks passed
    if (quality.passed && liveness.passed && match.matched) {
        decision.accepted = true;
        reason << "Accepted: composite="
               << std::fixed << std::setprecision(3) << composite;
        decision.username = match.username;
    }

    decision.reason = reason.str();
    return decision;
}

} // namespace iris
