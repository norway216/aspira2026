#pragma once

#include "domain/RecognitionResult.h"
#include <opencv2/core.hpp>
#include <vector>

namespace iris {

class IrisLivenessDetector;

/// Service wrapper for liveness detection with frame history management
class LivenessService {
public:
    explicit LivenessService(IrisLivenessDetector& detector);

    /// Check liveness with frame history
    LivenessResult check(const cv::Mat& eyeRoi);

    /// Clear frame history
    void reset();

private:
    IrisLivenessDetector& m_detector;
    std::vector<cv::Mat> m_recentFrames;
    static constexpr size_t MAX_HISTORY = 16;
};

} // namespace iris
