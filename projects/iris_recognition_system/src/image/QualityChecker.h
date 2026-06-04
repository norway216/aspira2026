#pragma once

#include "domain/RecognitionResult.h"
#include <opencv2/core.hpp>

namespace iris {

/// Evaluates image quality for iris recognition
class QualityChecker {
public:
    explicit QualityChecker(float threshold = 0.35f);

    /// Check the quality of an eye ROI image
    QualityResult check(const cv::Mat& eyeRoi);

    /// Set quality threshold
    void setThreshold(float t) { m_threshold = t; }
    float getThreshold() const { return m_threshold; }

private:
    /// Compute sharpness using Laplacian variance
    static float computeSharpness(const cv::Mat& gray);

    /// Compute mean brightness (normalized)
    static float computeBrightness(const cv::Mat& gray);

    /// Estimate occlusion ratio from pixel distribution
    static float estimateOcclusion(const cv::Mat& gray);

    /// Compute overall quality score
    static float computeOverall(float sharpness, float brightness, float occlusion);

    float m_threshold;
};

} // namespace iris
