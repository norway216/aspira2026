/**
 * @file    yolo_seg_inference.h
 * @brief   YOLOv8 分割模型 ONNX Runtime C++ 推理类（优化版）
 * @author  yolo_cpp project
 * @date    2026-06-01
 *
 * ============================================================================
 * 优化记录 (v2.0)
 * ============================================================================
 * 精度修复：
 *   - Mask 系数添加 sigmoid 激活（修复分割 Mask 精度）
 *
 * 速度优化：
 *   - 缓存 ONNX Runtime name pointers 和 MemoryInfo，避免每帧重建
 *   - 启用 inter-op 并行 + 内存模式优化
 *   - 预分配 mask/postprocess 缓冲区，消除重复分配
 *   - Anchor 扫描循环优化（class score stride 预计算）
 *   - NMS 原地筛选，减少 vector 分配
 *   - 移除废弃的 decodeBbox 死代码
 * ============================================================================
 */

#pragma once

#include <opencv2/core.hpp>
#include <opencv2/imgproc.hpp>
#include <onnxruntime_cxx_api.h>

#include <string>
#include <vector>
#include <memory>
#include <chrono>

// ============================================================================
// 数据结构
// ============================================================================

struct SegmentationResult {
    cv::Rect bbox;
    int classId;
    std::string className;
    float confidence;
    cv::Mat mask;

    bool operator<(const SegmentationResult& other) const {
        return confidence > other.confidence;
    }
};

struct InferenceResult {
    std::vector<SegmentationResult> detections;
    double preprocessMs;
    double inferenceMs;
    double postprocessMs;
};

// ============================================================================
// 配置参数
// ============================================================================

struct InferenceConfig {
    int inputWidth   = 640;
    int inputHeight  = 640;
    int numClasses   = 80;
    int maskChannels = 32;
    int maskProtoH   = 160;
    int maskProtoW   = 160;

    float confThreshold = 0.25f;
    float iouThreshold  = 0.45f;
    float maskThreshold = 0.5f;

    int  numThreads    = 4;
    int  interOpThreads = 2;       ///< inter-op 并行线程数（新增）
    bool enableProfiling = false;
    bool enableMemoryPattern = true;  ///< 启用内存模式优化（新增）
};

// ============================================================================
// YoloSegInference 类（优化版）
// ============================================================================

class YoloSegInference {
public:
    YoloSegInference();
    explicit YoloSegInference(const InferenceConfig& config);
    ~YoloSegInference();

    YoloSegInference(const YoloSegInference&) = delete;
    YoloSegInference& operator=(const YoloSegInference&) = delete;

    bool initialize(const std::string& modelPath, int numThreads = 4);
    InferenceResult infer(const cv::Mat& image);

    cv::Mat visualize(
        const cv::Mat& image,
        const std::vector<SegmentationResult>& detections,
        bool drawMask = true
    ) const;

    bool isInitialized() const { return m_initialized; }
    const InferenceConfig& getConfig() const { return m_config; }

private:
    // ── 预处理/后处理 ──────────────────────────────────────────────────
    Ort::Value preprocess(const cv::Mat& image);
    std::vector<SegmentationResult> postprocess(
        const float* output0Tensor,
        const float* output1Tensor,
        const cv::Size& originalSize
    );

    // ── 工具函数 ───────────────────────────────────────────────────────
    static cv::Mat letterbox(const cv::Mat& image, int targetW, int targetH,
                             float& scale, int& dx, int& dy);
    static float computeIoU(const cv::Rect& a, const cv::Rect& b);
    static std::vector<int> nms(
        const std::vector<cv::Rect>& boxes,
        const std::vector<float>& scores,
        float iouThreshold
    );

    // ── 成员变量 ──────────────────────────────────────────────────────
    InferenceConfig m_config;
    bool m_initialized = false;

    // ONNX Runtime 核心
    std::unique_ptr<Ort::Env> m_env;
    std::unique_ptr<Ort::Session> m_session;
    Ort::AllocatorWithDefaultOptions m_allocator;

    // 模型元信息
    std::vector<std::string> m_inputNames;
    std::vector<std::string> m_outputNames;
    std::vector<int64_t> m_inputShape;
    std::vector<int64_t> m_output0Shape;
    std::vector<int64_t> m_output1Shape;

    // 缓存的 name pointers（避免每次推理时重建）
    std::vector<const char*> m_inputNamePtrs;
    std::vector<const char*> m_outputNamePtrs;

    // 缓存的 MemoryInfo（避免重复创建）
    Ort::MemoryInfo m_cachedMemoryInfo{nullptr};

    // 输入数据缓冲区（Ort::Value 不拷贝数据）
    std::vector<float> m_inputBuffer;

    // 预分配的 mask 缓冲区（避免每次后处理重复分配）
    std::vector<float> m_maskWorkBuffer;  // 160*160 的 mask 工作区

    // Letterbox 变换参数
    float m_letterboxScale = 1.0f;
    int m_letterboxDx = 0;
    int m_letterboxDy = 0;

    // COCO 类别名称
    static const std::vector<std::string> COCO_CLASSES;
};
