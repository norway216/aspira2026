#pragma once

#include "image/IrisNormalizer.h"  // IrisBoundaries
#include <opencv2/core.hpp>
#include <string>

namespace iris {

/// Result of iris segmentation
struct IrisSegmentationResult {
    cv::Mat iris_mask;          // binary mask: iris region
    cv::Mat pupil_mask;         // binary mask: pupil region
    IrisBoundaries boundaries;  // detected circle boundaries
    float quality_score = 0.0f; // segmentation quality estimate
};

/// Iris segmenter using Hough circle detection (classical CV).
/// In production, replace with U-Net / Mobile-UNet ONNX inference.
class IrisSegmenter {
public:
    IrisSegmenter();

    /// Initialize (no model needed for classical CV approach)
    bool initialize(const std::string& /*modelPath*/ = "") { return true; }

    /// Segment iris from an eye ROI image
    IrisSegmentationResult segment(const cv::Mat& eyeRoi);

    /// Set parameters for Hough circle detection
    void setIrisRadiusRange(int minR, int maxR) {
        m_minIrisRadius = minR;
        m_maxIrisRadius = maxR;
    }

    void setPupilRadiusRange(int minR, int maxR) {
        m_minPupilRadius = minR;
        m_maxPupilRadius = maxR;
    }

private:
    /// Detect iris outer boundary using Hough circles
    bool detectIrisBoundary(const cv::Mat& gray, cv::Point2f& center, float& radius);

    /// Detect pupil boundary using Hough circles (after iris is found)
    bool detectPupilBoundary(const cv::Mat& gray,
                              const cv::Point2f& irisCenter, float irisRadius,
                              cv::Point2f& pupilCenter, float& pupilRadius);

    /// Generate segmentation masks from boundaries
    void generateMasks(const cv::Size& size,
                       const IrisBoundaries& boundaries,
                       cv::Mat& irisMask, cv::Mat& pupilMask);

    /// Refine boundaries using edge information
    void refineBoundaries(const cv::Mat& gray, IrisBoundaries& boundaries);

    int m_minIrisRadius  = 30;
    int m_maxIrisRadius  = 110;
    int m_minPupilRadius = 10;
    int m_maxPupilRadius = 50;
};

} // namespace iris
