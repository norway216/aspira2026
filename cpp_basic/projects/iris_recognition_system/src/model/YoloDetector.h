#pragma once

#include <opencv2/core.hpp>
#include <memory>
#include <string>
#include <vector>

namespace iris {

/// Single detection result from YOLO
struct YoloDetection {
    cv::Rect bbox;
    float confidence = 0.0f;
    int class_id     = 0;
};

/// YOLO-based face detector using ONNX Runtime.
/// Auto-detects YOLOv5 (3 outputs, anchor-based) vs YOLOv8 (1 output, anchor-free).
/// Designed to work with ONNX models at opset=11 for OpenCV 4.6 compatibility.
class YoloDetector {
public:
    YoloDetector();
    ~YoloDetector();

    YoloDetector(const YoloDetector&)            = delete;
    YoloDetector& operator=(const YoloDetector&) = delete;

    /// Load ONNX model from file.
    /// @param modelPath  Path to .onnx model file
    /// @param useGPU     Not yet implemented (CPU only)
    /// @param numThreads Number of intra-op threads for ONNX Runtime
    bool loadModel(const std::string& modelPath, bool useGPU = false, int numThreads = 4);

    /// Check if model is loaded and ready
    bool isLoaded() const;

    /// Run detection on a frame
    /// @param frame          BGR input image (any size, will be letterbox-resized)
    /// @param confThreshold  Minimum confidence [0,1]
    /// @param nmsThreshold   IoU threshold for NMS [0,1]
    std::vector<YoloDetection> detect(const cv::Mat& frame,
                                       float confThreshold = 0.5f,
                                       float nmsThreshold = 0.45f);

    /// Extract eye ROI from a face bounding box using proportional cropping.
    /// Returns the upper-central region of the face where eyes are expected.
    /// face.x + 10% width, face.y + 15% height, 80% width, 35% height
    static cv::Rect getEyeRoiFromFace(const cv::Rect& faceBbox, const cv::Size& frameSize);

    /// Convenience: detect face, then extract eye ROI
    cv::Mat extractEyeRoi(const cv::Mat& frame);

    /// Get last detections (for display / debugging)
    const std::vector<YoloDetection>& getLastDetections() const { return m_lastDetections; }

    /// Model input dimensions
    int inputWidth() const  { return m_inputWidth; }
    int inputHeight() const { return m_inputHeight; }

private:
    // ── Preprocessing ─────────────────────────────────────────────

    struct PreprocessResult {
        std::vector<float> blob;   // NCHW float data, normalized to [0,1]
        float scaleX;              // original_w to input_w mapping
        float scaleY;
        float padX;                // letterbox padding offset
        float padY;
    };

    PreprocessResult preprocess(const cv::Mat& frame);

    // ── Post-processing ───────────────────────────────────────────

    /// YOLOv8 format: single output [1, 4+nc, num_proposals]
    std::vector<YoloDetection> decodeYolov8Output(
        const std::vector<float>& output,
        int imgWidth, int imgHeight,
        float scaleX, float scaleY, float padX, float padY,
        float confThreshold);

    /// YOLOv5 format: 3 outputs [1, na*(5+nc), grid_h, grid_w]
    std::vector<YoloDetection> decodeYolov5Outputs(
        const std::vector<std::vector<float>>& outputs,
        int imgWidth, int imgHeight,
        float scaleX, float scaleY, float padX, float padY,
        float confThreshold);

    /// Non-maximum suppression (greedy IoU)
    static std::vector<YoloDetection> nms(
        std::vector<YoloDetection>& detections, float nmsThreshold);

    /// Compute intersection-over-union of two boxes
    static float computeIoU(const cv::Rect& a, const cv::Rect& b);

    // ── State ─────────────────────────────────────────────────────

    std::unique_ptr<class OnnxRuntimeEngine> m_engine;
    int m_inputWidth   = 640;
    int m_inputHeight  = 640;

    bool m_isYolov8    = true;  // auto-detected: 1 output = v8, 3 outputs = v5
    int  m_numClasses  = 1;     // default: face-only model (will be detected)

    std::vector<YoloDetection> m_lastDetections;
};

} // namespace iris
