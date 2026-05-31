#pragma once

#include <opencv2/core.hpp>
#include <opencv2/objdetect.hpp>
#include <string>
#include <vector>

namespace iris {

/// Detection result for a single eye
struct EyeBox {
    cv::Rect bbox;
    float confidence = 0.0f;
    int label        = -1;   // 0 = left eye, 1 = right eye
};

/// Eye detector using Haar cascades (classical CV fallback).
/// Can be swapped with YOLOv5n/SCRFD ONNX inference for production.
class EyeDetector {
public:
    EyeDetector();

    /// Initialize with optional Haar cascade path.
    /// If empty, uses OpenCV's built-in cascade.
    bool initialize(const std::string& cascadePath = "");

    /// Detect eyes in a frame. Returns detected eye boxes.
    std::vector<EyeBox> detect(const cv::Mat& frame);

    /// Extract an eye ROI from the frame given a detection box
    static cv::Mat extractEyeRoi(const cv::Mat& frame, const EyeBox& box,
                                  int roiSize = 256);

    /// Get the best (most confident) eye ROI, or a centered fallback
    cv::Mat getBestEyeRoi(const cv::Mat& frame);

    bool isInitialized() const { return m_initialized; }

private:
    cv::CascadeClassifier m_cascade;
    bool m_initialized = false;
};

} // namespace iris
