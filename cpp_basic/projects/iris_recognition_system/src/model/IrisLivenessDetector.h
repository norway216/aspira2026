#pragma once

#include "domain/RecognitionResult.h"  // LivenessResult
#include <opencv2/core.hpp>
#include <string>
#include <vector>

namespace iris {

/// Liveness detector using image analysis heuristics (classical CV fallback).
/// In production, replace with MiniFASNet/MobileNetV3-Liveness ONNX inference.
class IrisLivenessDetector {
public:
    IrisLivenessDetector();

    /// Initialize with optional model path
    bool initialize(const std::string& modelPath = "");

    /// Check liveness of an eye ROI. Optionally provide recent frames
    /// for motion-based analysis.
    LivenessResult check(const cv::Mat& eyeRoi,
                          const std::vector<cv::Mat>& recentFrames = {});

    void setThreshold(float t) { m_threshold = t; }

private:
    /// Analyze image texture statistics for print/screen artifacts
    static float analyzeTexture(const cv::Mat& gray);

    /// Detect Moire patterns (screen replay attack indicator)
    static float detectMoirePatterns(const cv::Mat& gray);

    /// Analyze local binary pattern distribution
    static float analyzeLBP(const cv::Mat& gray);

    /// Check for natural reflections (corneal reflection)
    static float detectSpecularReflection(const cv::Mat& gray);

    /// Frame-to-frame variation analysis for motion detection
    static float analyzeMotion(const std::vector<cv::Mat>& frames);

    float m_threshold = 0.55f;
    bool m_initialized = true;
};

} // namespace iris
