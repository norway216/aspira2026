#pragma once

#include <opencv2/core.hpp>
#include <opencv2/objdetect.hpp>
#include <memory>
#include <string>
#include <vector>

namespace iris {

/// Detection result for a single eye
struct EyeBox {
    cv::Rect bbox;
    float confidence = 0.0f;
    int label        = -1;   // 0 = left eye, 1 = right eye
};

// Forward declaration
class YoloDetector;

/// Eye detector using Haar cascades (classical CV fallback)
/// or YOLO ONNX inference (production).
class EyeDetector {
public:
    EyeDetector();
    ~EyeDetector();

    /// Initialize with optional Haar cascade path.
    /// If empty, uses OpenCV's built-in cascade.
    bool initialize(const std::string& cascadePath = "");

    /// Enable YOLO-based face detection.
    /// @param modelPath  Path to YOLO ONNX model
    /// @param useGPU     Not yet implemented (CPU only)
    /// @return true if model loaded successfully
    bool enableYolo(const std::string& modelPath, bool useGPU = false);

    /// Check if YOLO is enabled
    bool isYoloEnabled() const { return m_yoloEnabled; }

    /// Detect eyes in a frame. Returns detected eye boxes.
    /// Uses YOLO if enabled, otherwise falls back to Haar cascade.
    std::vector<EyeBox> detect(const cv::Mat& frame);

    /// Extract an eye ROI from the frame given a detection box
    static cv::Mat extractEyeRoi(const cv::Mat& frame, const EyeBox& box,
                                  int roiSize = 256);

    /// Get the best (most confident) eye ROI, or a centered fallback
    cv::Mat getBestEyeRoi(const cv::Mat& frame);

    bool isInitialized() const { return m_initialized || m_yoloEnabled; }

private:
    cv::CascadeClassifier m_cascade;
    bool m_initialized = false;

    // YOLO path
    std::unique_ptr<YoloDetector> m_yoloDetector;
    bool m_yoloEnabled = false;
};

} // namespace iris
