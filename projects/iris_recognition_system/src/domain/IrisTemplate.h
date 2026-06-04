#pragma once

#include <cstddef>
#include <cstdint>
#include <string>
#include <vector>

namespace iris {

struct IrisTemplate {
    std::string id;
    std::string user_id;
    std::string eye_side;       // "left" or "right"
    std::vector<uint8_t> iris_code;  // binary IrisCode
    std::vector<uint8_t> mask_code;  // valid bits mask
    std::vector<float> embedding;    // float embedding (for NN path)
    int embedding_dim    = 0;
    std::string model_version;
    float quality_score  = 0.0f;
    int64_t created_at   = 0;
};

} // namespace iris
