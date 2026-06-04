#pragma once

#include <string>

namespace iris {

struct MatchResult {
    std::string user_id;
    std::string username;
    float similarity  = 0.0f;  // normalized [0,1], higher = better
    float hamming_dist = 1.0f; // Hamming distance, lower = better
    bool matched       = false;
};

struct LivenessResult {
    float live_score   = 0.0f;
    float spoof_score  = 0.0f;
    std::string attack_type;
    bool passed        = false;
};

struct QualityResult {
    bool passed        = false;
    float sharpness    = 0.0f;
    float brightness   = 0.0f;
    float occlusion    = 0.0f;
    float overall      = 0.0f;
    std::string reason;
};

struct FinalDecision {
    bool accepted      = false;
    std::string user_id;
    std::string username;
    float match_score  = 0.0f;
    float liveness_score = 0.0f;
    float quality_score  = 0.0f;
    std::string reason;
};

} // namespace iris
