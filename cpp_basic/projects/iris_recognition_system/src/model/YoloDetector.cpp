#include "model/YoloDetector.h"
#include "nn/OnnxRuntimeEngine.h"
#include <opencv2/imgproc.hpp>
#include <iostream>
#include <algorithm>
#include <cmath>

namespace iris {

// ── YOLOv5 anchor definitions (for 640x640 input) ────────────────
// Standard YOLOv5n anchors for COCO (3 anchors per detection head)
static const std::vector<std::vector<float>> YOLOV5_ANCHORS = {
    {10, 13,   16, 30,   33, 23},     // P3/8  (small)
    {30, 61,   62, 45,   59, 119},    // P4/16 (medium)
    {116, 90,  156, 198, 373, 326}    // P5/32 (large)
};

// ── Number of anchors per detection head ─────────────────────────
static constexpr int YOLOV5_NA = 3;

// ── Constructor / Destructor ─────────────────────────────────────

YoloDetector::YoloDetector()
    : m_engine(std::make_unique<OnnxRuntimeEngine>()) {}

YoloDetector::~YoloDetector() = default;

// ── Model loading ────────────────────────────────────────────────

bool YoloDetector::loadModel(const std::string& modelPath, bool /*useGPU*/, int /*numThreads*/) {
    if (!m_engine->loadModel(modelPath)) {
        return false;
    }

    // Auto-detect YOLO version from number of outputs
    size_t numOutputs = m_engine->numOutputs();
    m_isYolov8 = (numOutputs == 1);

    // Read input dimensions from model
    auto inputShape = m_engine->getInputShape(0);
    if (inputShape.size() >= 4) {
        // ONNX: [batch, channels, height, width]
        m_inputHeight = static_cast<int>(inputShape[2]);
        m_inputWidth  = static_cast<int>(inputShape[3]);
        if (m_inputHeight <= 0) m_inputHeight = 640;
        if (m_inputWidth  <= 0) m_inputWidth  = 640;
    }

    // Detect number of classes from output shape
    if (m_isYolov8) {
        // YOLOv8 output: [1, 4+nc, num_proposals]
        auto outShape = m_engine->getOutputShape(0);
        if (outShape.size() >= 2) {
            m_numClasses = static_cast<int>(outShape[1]) - 4;
            if (m_numClasses <= 0) m_numClasses = 1;
        }
    } else {
        // YOLOv5: detect from first output channel count
        auto outShape = m_engine->getOutputShape(0);
        if (outShape.size() >= 2) {
            int channels = static_cast<int>(outShape[1]);
            m_numClasses = channels / YOLOV5_NA - 5;
            if (m_numClasses <= 0) m_numClasses = 1;
        }
    }

    std::cout << "[YoloDetector] Model loaded: " << (m_isYolov8 ? "YOLOv8/11" : "YOLOv5")
              << " format, " << m_numClasses << " class(es), "
              << m_inputWidth << "x" << m_inputHeight << " input\n";

    return true;
}

bool YoloDetector::isLoaded() const {
    return m_engine->isLoaded();
}

// ── Preprocessing ────────────────────────────────────────────────

YoloDetector::PreprocessResult YoloDetector::preprocess(const cv::Mat& frame) {
    PreprocessResult result;

    // Letterbox: resize keeping aspect ratio, pad to square
    cv::Size targetSize(m_inputWidth, m_inputHeight);
    float scale = std::min(
        static_cast<float>(m_inputWidth)  / frame.cols,
        static_cast<float>(m_inputHeight) / frame.rows);
    int scaledW = static_cast<int>(frame.cols * scale);
    int scaledH = static_cast<int>(frame.rows * scale);

    cv::Mat resized;
    cv::resize(frame, resized, cv::Size(scaledW, scaledH), 0, 0, cv::INTER_LINEAR);

    // Create letterbox canvas (gray 114 padding like YOLO training)
    cv::Mat letterbox(targetSize, CV_8UC3, cv::Scalar(114, 114, 114));
    int padX = (m_inputWidth  - scaledW) / 2;
    int padY = (m_inputHeight - scaledH) / 2;
    resized.copyTo(letterbox(cv::Rect(padX, padY, scaledW, scaledH)));

    result.scaleX = scale;
    result.scaleY = scale;
    result.padX   = static_cast<float>(padX);
    result.padY   = static_cast<float>(padY);

    // Convert BGR to RGB, normalize to [0,1], layout NCHW
    cv::Mat rgb;
    cv::cvtColor(letterbox, rgb, cv::COLOR_BGR2RGB);
    rgb.convertTo(rgb, CV_32F, 1.0 / 255.0);

    // NCHW blob: [1, 3, H, W]
    size_t blobSize = 1 * 3 * static_cast<size_t>(m_inputHeight) * static_cast<size_t>(m_inputWidth);
    result.blob.resize(blobSize);

    // Extract channels (OpenCV stores as HWC)
    for (int h = 0; h < m_inputHeight; ++h) {
        const float* row = rgb.ptr<float>(h);
        for (int w = 0; w < m_inputWidth; ++w) {
            size_t idx = static_cast<size_t>(h) * static_cast<size_t>(m_inputWidth) + static_cast<size_t>(w);
            // Channel 0 (R): row[w*3 + 0] -> blob[0 * H * W + h * W + w]
            result.blob[0 * m_inputHeight * m_inputWidth + idx] = row[w * 3 + 0];
            // Channel 1 (G)
            result.blob[1 * m_inputHeight * m_inputWidth + idx] = row[w * 3 + 1];
            // Channel 2 (B)
            result.blob[2 * m_inputHeight * m_inputWidth + idx] = row[w * 3 + 2];
        }
    }

    return result;
}

// ── Detection ────────────────────────────────────────────────────

std::vector<YoloDetection> YoloDetector::detect(const cv::Mat& frame,
                                                  float confThreshold,
                                                  float nmsThreshold) {
    m_lastDetections.clear();

    if (!m_engine->isLoaded() || frame.empty()) {
        return {};
    }

    // Preprocess
    auto prep = preprocess(frame);

    // Run inference
    std::vector<int64_t> inputShape = {1, 3, m_inputHeight, m_inputWidth};
    std::vector<float> output = m_engine->infer(prep.blob, inputShape);

    if (output.empty()) {
        std::cerr << "[YoloDetector] Inference returned empty output\n";
        return {};
    }

    // Decode based on YOLO version
    if (m_isYolov8) {
        m_lastDetections = decodeYolov8Output(
            output, frame.cols, frame.rows,
            prep.scaleX, prep.scaleY, prep.padX, prep.padY,
            confThreshold);
    } else {
        // For YOLOv5, we need to split the flat output into per-head tensors.
        // Since infer() concatenates all outputs, we need to know the sizes.
        std::vector<std::vector<float>> splitOutputs;
        size_t offset = 0;
        for (size_t i = 0; i < m_engine->numOutputs(); ++i) {
            auto outShape = m_engine->getOutputShape(i);
            size_t numElements = 1;
            for (auto d : outShape) {
                if (d > 0) numElements *= static_cast<size_t>(d);
            }
            if (offset + numElements <= output.size()) {
                splitOutputs.emplace_back(
                    output.begin() + static_cast<long>(offset),
                    output.begin() + static_cast<long>(offset + numElements));
                offset += numElements;
            }
        }

        if (!splitOutputs.empty()) {
            m_lastDetections = decodeYolov5Outputs(
                splitOutputs, frame.cols, frame.rows,
                prep.scaleX, prep.scaleY, prep.padX, prep.padY,
                confThreshold);
        }
    }

    // Apply NMS
    m_lastDetections = nms(m_lastDetections, nmsThreshold);

    return m_lastDetections;
}

// ── YOLOv8 Decoder ───────────────────────────────────────────────

std::vector<YoloDetection> YoloDetector::decodeYolov8Output(
    const std::vector<float>& output,
    int imgWidth, int imgHeight,
    float scaleX, float scaleY, float padX, float padY,
    float confThreshold) {

    // YOLOv8/11 output: [1, 4 + num_classes, num_proposals]
    // num_proposals = (H/8)*(W/8) + (H/16)*(W/16) + (H/32)*(W/32)
    // For 640x640: 80*80 + 40*40 + 20*20 = 8400

    int fieldsPerProposal = 4 + m_numClasses;  // cx, cy, w, h, cls0, cls1, ...
    if (fieldsPerProposal <= 0) return {};

    // Extract dimension info from output shape
    auto outShape = m_engine->getOutputShape(0);
    int numProposals = 0;
    if (outShape.size() >= 3) {
        numProposals = static_cast<int>(outShape[2]);
    }
    if (numProposals <= 0) {
        // Infer from output size
        if (output.size() % fieldsPerProposal == 0) {
            numProposals = static_cast<int>(output.size()) / fieldsPerProposal;
        }
    }
    if (numProposals <= 0) return {};

    // Output is in channel-major format: [fields, proposals]
    // Transpose mentally for easier parsing: each proposal = contiguous slice
    std::vector<YoloDetection> detections;

    for (int i = 0; i < numProposals; ++i) {
        // Read fields for this proposal from strided layout
        float cx = output[0 * numProposals + i];
        float cy = output[1 * numProposals + i];
        float w  = output[2 * numProposals + i];
        float h  = output[3 * numProposals + i];

        // Find max class confidence
        float maxConf = 0.0f;
        int bestClass = 0;
        for (int c = 0; c < m_numClasses; ++c) {
            float conf = output[(4 + c) * numProposals + i];
            if (conf > maxConf) {
                maxConf = conf;
                bestClass = c;
            }
        }

        if (maxConf < confThreshold) continue;

        // Convert from model space to original image space
        // 1. Scale back from letterbox
        float bx = (cx - padX) / scaleX;
        float by = (cy - padY) / scaleY;
        float bw = w / scaleX;
        float bh = h / scaleY;

        // 2. Convert from center to top-left
        int x0 = std::max(0, static_cast<int>(bx - bw / 2.0f));
        int y0 = std::max(0, static_cast<int>(by - bh / 2.0f));
        int x1 = std::min(imgWidth,  static_cast<int>(bx + bw / 2.0f));
        int y1 = std::min(imgHeight, static_cast<int>(by + bh / 2.0f));

        if (x1 <= x0 || y1 <= y0) continue;

        YoloDetection det;
        det.bbox       = cv::Rect(x0, y0, x1 - x0, y1 - y0);
        det.confidence = maxConf;
        det.class_id   = bestClass;
        detections.push_back(det);
    }

    return detections;
}

// ── YOLOv5 Decoder ───────────────────────────────────────────────

std::vector<YoloDetection> YoloDetector::decodeYolov5Outputs(
    const std::vector<std::vector<float>>& outputs,
    int imgWidth, int imgHeight,
    float scaleX, float scaleY, float padX, float padY,
    float confThreshold) {

    // YOLOv5 has 3 detection heads (P3/8, P4/16, P5/32)
    // Each output shape: [1, na * (5 + nc), grid_h, grid_w]
    // where na = 3 (anchors per grid cell)
    // Layout within channels: for each anchor a:
    //   [tx, ty, tw, th, obj, cls0, cls1, ...] (repeated na times)

    int nClasses = m_numClasses;
    int nChannelsPerAnchor = 5 + nClasses;  // cx, cy, w, h, obj + class scores
    int channelsPerGrid = YOLOV5_NA * nChannelsPerAnchor;

    std::vector<YoloDetection> detections;

    for (size_t headIdx = 0; headIdx < outputs.size() && headIdx < 3; ++headIdx) {
        const auto& output = outputs[headIdx];

        auto outShape = m_engine->getOutputShape(headIdx);
        int gridH = 0, gridW = 0;
        if (outShape.size() >= 4) {
            gridH = static_cast<int>(outShape[2]);
            gridW = static_cast<int>(outShape[3]);
        }

        // Infer grid size
        if (gridH <= 0 || gridW <= 0) {
            int totalCells = static_cast<int>(output.size()) / channelsPerGrid;
            gridH = static_cast<int>(std::sqrt(static_cast<double>(totalCells) *
                                        m_inputHeight / m_inputWidth));
            gridW = totalCells / gridH;
        }
        if (gridH <= 0 || gridW <= 0) continue;

        float stride = static_cast<float>(m_inputHeight) / gridH;

        const auto& anchors = YOLOV5_ANCHORS[headIdx];

        for (int gy = 0; gy < gridH; ++gy) {
            for (int gx = 0; gx < gridW; ++gx) {
                for (int a = 0; a < YOLOV5_NA; ++a) {
                    // Read values from channel-major layout
                    int chIdx = a * nChannelsPerAnchor;
                    float tx = output[chIdx * gridH * gridW + gy * gridW + gx];
                    float ty = output[(chIdx + 1) * gridH * gridW + gy * gridW + gx];
                    float tw = output[(chIdx + 2) * gridH * gridW + gy * gridW + gx];
                    float th = output[(chIdx + 3) * gridH * gridW + gy * gridW + gx];
                    float objScore = 1.0f / (1.0f + std::exp(-output[(chIdx + 4) * gridH * gridW + gy * gridW + gx]));

                    // Find best class
                    float maxClsConf = 0.0f;
                    int bestClass = 0;
                    for (int c = 0; c < nClasses; ++c) {
                        float clsVal = output[(chIdx + 5 + c) * gridH * gridW + gy * gridW + gx];
                        float clsProb = 1.0f / (1.0f + std::exp(-clsVal));
                        if (clsProb > maxClsConf) {
                            maxClsConf = clsProb;
                            bestClass = c;
                        }
                    }

                    float totalConf = objScore * maxClsConf;
                    if (totalConf < confThreshold) continue;

                    // Decode bounding box (YOLOv5 anchor-based)
                    float aw = anchors[a * 2];
                    float ah = anchors[a * 2 + 1];

                    float cx = (static_cast<float>(gx) + (1.0f / (1.0f + std::exp(-tx)) * 2.0f - 0.5f))
                               * stride;
                    float cy = (static_cast<float>(gy) + (1.0f / (1.0f + std::exp(-ty)) * 2.0f - 0.5f))
                               * stride;
                    float w  = aw * std::exp(tw) * 2.0f;
                    float h  = ah * std::exp(th) * 2.0f;

                    // Scale from letterbox to original
                    float bx = (cx - padX) / scaleX;
                    float by = (cy - padY) / scaleY;
                    float bw = w / scaleX;
                    float bh = h / scaleY;

                    int x0 = std::max(0, static_cast<int>(bx - bw / 2.0f));
                    int y0 = std::max(0, static_cast<int>(by - bh / 2.0f));
                    int x1 = std::min(imgWidth,  static_cast<int>(bx + bw / 2.0f));
                    int y1 = std::min(imgHeight, static_cast<int>(by + bh / 2.0f));

                    if (x1 <= x0 || y1 <= y0) continue;

                    YoloDetection det;
                    det.bbox       = cv::Rect(x0, y0, x1 - x0, y1 - y0);
                    det.confidence = totalConf;
                    det.class_id   = bestClass;
                    detections.push_back(det);
                }
            }
        }
    }

    return detections;
}

// ── NMS ──────────────────────────────────────────────────────────

std::vector<YoloDetection> YoloDetector::nms(
    std::vector<YoloDetection>& detections, float nmsThreshold) {

    if (detections.empty()) return {};

    // Sort by confidence descending
    std::sort(detections.begin(), detections.end(),
              [](const YoloDetection& a, const YoloDetection& b) {
                  return a.confidence > b.confidence;
              });

    std::vector<YoloDetection> kept;
    std::vector<bool> suppressed(detections.size(), false);

    for (size_t i = 0; i < detections.size(); ++i) {
        if (suppressed[i]) continue;
        kept.push_back(detections[i]);

        for (size_t j = i + 1; j < detections.size(); ++j) {
            if (suppressed[j]) continue;
            if (computeIoU(detections[i].bbox, detections[j].bbox) >= nmsThreshold) {
                suppressed[j] = true;
            }
        }
    }

    return kept;
}

float YoloDetector::computeIoU(const cv::Rect& a, const cv::Rect& b) {
    int xL = std::max(a.x, b.x);
    int yT = std::max(a.y, b.y);
    int xR = std::min(a.x + a.width,  b.x + b.width);
    int yB = std::min(a.y + a.height, b.y + b.height);

    if (xR <= xL || yB <= yT) return 0.0f;

    float interArea = static_cast<float>((xR - xL) * (yB - yT));
    float unionArea = static_cast<float>(a.area() + b.area()) - interArea;

    return (unionArea > 0.0f) ? interArea / unionArea : 0.0f;
}

// ── Eye ROI Extraction ───────────────────────────────────────────

cv::Rect YoloDetector::getEyeRoiFromFace(const cv::Rect& faceBbox,
                                           const cv::Size& /*frameSize*/) {
    // Eyes are typically in the upper-central region of the face
    // Proportion: x + 10% width, y + 15% height, 80% width, 35% height
    int ex = faceBbox.x + static_cast<int>(faceBbox.width  * 0.10f);
    int ey = faceBbox.y + static_cast<int>(faceBbox.height * 0.15f);
    int ew = static_cast<int>(faceBbox.width  * 0.80f);
    int eh = static_cast<int>(faceBbox.height * 0.35f);

    return cv::Rect(ex, ey, ew, eh);
}

cv::Mat YoloDetector::extractEyeRoi(const cv::Mat& frame) {
    if (frame.empty() || !m_engine->isLoaded()) {
        return cv::Mat(256, 256, frame.type(), cv::Scalar(0));
    }

    auto detections = detect(frame);
    if (detections.empty()) {
        // Fallback: assume face is centered
        int halfSize = 128;
        int cx = frame.cols / 2;
        int cy = frame.rows / 3;
        cv::Rect fallback(cx - halfSize, cy - halfSize, halfSize * 2, halfSize * 2);
        cv::Mat eyeRoi;
        cv::resize(frame(fallback & cv::Rect(0, 0, frame.cols, frame.rows)),
                   eyeRoi, cv::Size(256, 256), 0, 0, cv::INTER_LINEAR);
        return eyeRoi;
    }

    // Use highest-confidence face detection
    auto eyeRect = getEyeRoiFromFace(detections[0].bbox, frame.size());
    cv::Rect clamped = eyeRect & cv::Rect(0, 0, frame.cols, frame.rows);
    if (clamped.width <= 0 || clamped.height <= 0) {
        return cv::Mat(256, 256, frame.type(), cv::Scalar(0));
    }

    cv::Mat eyeRoi;
    cv::resize(frame(clamped), eyeRoi, cv::Size(256, 256), 0, 0, cv::INTER_LINEAR);
    return eyeRoi;
}

} // namespace iris
