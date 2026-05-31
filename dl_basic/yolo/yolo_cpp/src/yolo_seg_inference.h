/**
 * @file    yolo_seg_inference.h
 * @brief   YOLOv8 分割模型 ONNX Runtime C++ 推理类
 * @author  yolo_cpp project
 * @date    2026-05-31
 *
 * ============================================================================
 * 功能概述
 * ============================================================================
 * 本文件定义了 YoloSegInference 类，用于在 C++ 环境下使用 ONNX Runtime
 * 加载 YOLOv8 分割模型并执行推理。
 *
 * 完整管线：
 *   1. 加载 ONNX 模型 → 2. 图片预处理 → 3. ONNX Runtime 推理
 *   → 4. 后处理（解析检测框 + 生成实例分割Mask）
 *   → 5. 可视化（绘制Mask、边界框、标签）
 *
 * 模型输入： images [1, 3, 640, 640] float32, BCHW, RGB, 归一化到[0,1]
 * 模型输出：
 *   - output0 [1, 116, 8400] 检测预测
 *       116 = 4(bbox) + 80(COCO类别分数) + 32(Mask系数)
 *   - output1 [1, 32, 160, 160] 原型Mask
 *
 * 目标平台： Linux / 嵌入式设备 (RK3588 / Jetson 等)
 * ============================================================================
 */

#pragma once

// ── 第三方库头文件 ──────────────────────────────────────────────────────
#include <opencv2/core.hpp>       // cv::Mat, cv::Rect, cv::Scalar
#include <opencv2/imgproc.hpp>    // cv::resize, cv::warpAffine 等
#include <onnxruntime_cxx_api.h>  // Ort::Env, Ort::Session, Ort::Value

// ── 标准库头文件 ──────────────────────────────────────────────────────
#include <string>                 // std::string
#include <vector>                 // std::vector
#include <memory>                 // std::unique_ptr
#include <chrono>                 // 计时

// ============================================================================
// 数据结构定义
// ============================================================================

/// @brief 单次检测结果（一个实例）
struct SegmentationResult {
    cv::Rect bbox;              ///< 边界框 (x, y, width, height) — 原图坐标
    int classId;                ///< 类别 ID (0-79, COCO 数据集)
    std::string className;      ///< 类别名称（如 "car", "person"）
    float confidence;           ///< 置信度 [0.0, 1.0]
    cv::Mat mask;               ///< 实例分割 Mask（单通道 CV_8UC1, 0或255, 原图尺寸）

    /// @brief 按置信度降序排序（用于 NMS 后的结果排序）
    bool operator<(const SegmentationResult& other) const {
        return confidence > other.confidence;  // 降序
    }
};

/// @brief 一次推理的完整结果
struct InferenceResult {
    std::vector<SegmentationResult> detections;  ///< 所有检测到的实例
    double preprocessMs;     ///< 预处理耗时（毫秒）
    double inferenceMs;      ///< 推理耗时（毫秒）
    double postprocessMs;    ///< 后处理耗时（毫秒）
};

// ============================================================================
// 配置参数结构
// ============================================================================

/// @brief 推理参数配置（可根据嵌入式设备性能调整）
struct InferenceConfig {
    // ── 模型参数 ──────────────────────────────────────────────────────
    int inputWidth   = 640;    ///< 模型输入宽度
    int inputHeight  = 640;    ///< 模型输入高度
    int numClasses   = 80;     ///< COCO 类别数
    int maskChannels = 32;     ///< Mask 系数维度（YOLOv8-seg 为 32）
    int maskProtoH   = 160;    ///< 原型 Mask 高度
    int maskProtoW   = 160;    ///< 原型 Mask 宽度

    // ── 后处理参数 ────────────────────────────────────────────────────
    float confThreshold = 0.25f;     ///< 置信度阈值（低于此值的检测被丢弃）
    float iouThreshold  = 0.45f;     ///< NMS IoU 阈值
    float maskThreshold = 0.5f;      ///< Mask 二值化阈值

    // ── 性能参数 ──────────────────────────────────────────────────────
    int   numThreads    = 4;         ///< ONNX Runtime 线程数
    bool  enableProfiling = false;   ///< 是否启用详细性能日志
};

// ============================================================================
// YoloSegInference 类
// ============================================================================

/**
 * @class YoloSegInference
 * @brief YOLOv8 分割模型 ONNX Runtime C++ 推理引擎
 *
 * 使用示例：
 * @code
 *   #include "yolo_seg_inference.h"
 *
 *   using namespace yolo_cpp;
 *
 *   YoloSegInference engine;
 *   engine.initialize("models/yolov8n-seg.onnx");
 *
 *   cv::Mat image = cv::imread("images/road.jpg");
 *   auto result = engine.infer(image);
 *
 *   cv::Mat vis = engine.visualize(image, result.detections);
 *   cv::imwrite("images/road_result.jpg", vis);
 * @endcode
 */
class YoloSegInference {
public:
    // ========================================================================
    // 构造与析构
    // ========================================================================

    /// @brief 默认构造函数，使用默认配置
    YoloSegInference();

    /// @brief 使用自定义配置构造
    /// @param config 推理配置参数
    explicit YoloSegInference(const InferenceConfig& config);

    /// @brief 析构函数（自动释放 ONNX Runtime 资源）
    ~YoloSegInference();

    // ── 禁止拷贝（ONNX Runtime 对象不可拷贝） ──────────────────────
    YoloSegInference(const YoloSegInference&) = delete;
    YoloSegInference& operator=(const YoloSegInference&) = delete;

    // ========================================================================
    // 公共 API
    // ========================================================================

    /**
     * @brief 初始化推理引擎：加载 ONNX 模型并创建推理会话
     * @param modelPath ONNX 模型文件路径
     * @param numThreads ONNX Runtime 线程数（0=自动检测）
     * @return true 成功, false 失败
     */
    bool initialize(const std::string& modelPath, int numThreads = 4);

    /**
     * @brief 执行一次完整推理（预处理 + 推理 + 后处理）
     * @param image 输入图片 (BGR 格式，CV_8UC3)
     * @return 推理结果（包含检测列表和各阶段耗时）
     */
    InferenceResult infer(const cv::Mat& image);

    /**
     * @brief 将检测结果可视化绘制到原图上
     * @param image 原始图片 (BGR)
     * @param detections 检测结果列表
     * @param drawMask 是否绘制半透明 Mask（默认 true）
     * @return 绘制后的图片（BGR 格式）
     */
    cv::Mat visualize(
        const cv::Mat& image,
        const std::vector<SegmentationResult>& detections,
        bool drawMask = true
    ) const;

    /**
     * @brief 检查引擎是否已初始化
     * @return true 已初始化
     */
    bool isInitialized() const { return m_initialized; }

    /**
     * @brief 获取配置的只读引用
     */
    const InferenceConfig& getConfig() const { return m_config; }

private:
    // ========================================================================
    // 内部实现（PIMPL 模式 — 隐藏 ONNX Runtime 具体类型）
    // ========================================================================

    /// @brief 预处理：将 OpenCV Mat 转换为 ONNX Runtime 输入 Tensor
    /// @param image 输入图片 (BGR, CV_8UC3)
    /// @return ONNX Runtime Value（即 Tensor），失败返回 nullptr
    Ort::Value preprocess(const cv::Mat& image);

    /// @brief 后处理：解析模型输出，生成检测和分割结果
    /// @param output0Tensor output0 的 float* 数据指针
    /// @param output1Tensor output1 的 float* 数据指针
    /// @param originalSize 原始图片尺寸 (width, height)
    /// @return 检测结果列表
    std::vector<SegmentationResult> postprocess(
        const float* output0Tensor,
        const float* output1Tensor,
        const cv::Size& originalSize
    );

    // ========================================================================
    // 辅助函数（静态方法）
    // ========================================================================

    /// @brief Letterbox 缩放：保持宽高比缩放到目标尺寸，不足部分填充灰色
    /// @return 缩放后的图片 + 变换参数 (scale, dx, dy)
    static cv::Mat letterbox(const cv::Mat& image, int targetW, int targetH,
                             float& scale, int& dx, int& dy);

    /// @brief 计算两个边界框的 IoU（交并比）
    static float computeIoU(const cv::Rect& a, const cv::Rect& b);

    /// @brief 非极大值抑制 (NMS)：移除重叠度过高的检测框
    static std::vector<int> nms(
        const std::vector<cv::Rect>& boxes,
        const std::vector<float>& scores,
        float iouThreshold
    );

    /// @brief YOLO 边界框解码：从 [cx, cy, w, h] 转换为 [x1, y1, x2, y2]
    /// @param cx 中心点 x（相对于网格单元，范围 [0, 1]）
    /// @param cy 中心点 y（相对于网格单元，范围 [0, 1]）
    /// @param w  宽度（相对于输入尺寸，范围 [0, 1]）
    /// @param h  高度（相对于输入尺寸，范围 [0, 1]）
    /// @param gridX 网格列索引
    /// @param gridY 网格行索引
    /// @param stride 当前特征图的步长（如 8, 16, 32）
    /// @param imgW 输入图片宽度
    /// @param imgH 输入图片高度
    /// @return 解码后的边界框 (x, y, width, height) — 输入图片坐标系
    static cv::Rect decodeBbox(
        float cx, float cy, float w, float h,
        int gridX, int gridY, int stride,
        int imgW, int imgH
    );

    // ========================================================================
    // 成员变量
    // ========================================================================

    InferenceConfig m_config;                          ///< 推理配置
    bool m_initialized = false;                        ///< 是否已初始化

    // ── ONNX Runtime 核心对象 ────────────────────────────────────────
    // 使用 unique_ptr + 自定义删除器管理 Ort 对象生命周期
    // 注意：Ort::Env 必须在 Ort::Session 之前创建，在之后销毁
    std::unique_ptr<Ort::Env> m_env;                   ///< ONNX Runtime 环境
    std::unique_ptr<Ort::Session> m_session;            ///< ONNX Runtime 推理会话
    Ort::AllocatorWithDefaultOptions m_allocator;       ///< 内存分配器（用于获取字符串）

    // ── 模型输入输出元信息（初始化时缓存，避免每次推理时查询） ────
    std::vector<std::string> m_inputNames;               ///< 输入节点名称列表
    std::vector<std::string> m_outputNames;              ///< 输出节点名称列表
    std::vector<int64_t> m_inputShape;                  ///< 输入张量形状 [1, 3, 640, 640]
    std::vector<int64_t> m_output0Shape;                ///< output0 形状 [1, 116, 8400]
    std::vector<int64_t> m_output1Shape;                ///< output1 形状 [1, 32, 160, 160]

    // ── Letterbox 变换参数（预处理时记录，后处理时用于坐标逆变换） ──
    float m_letterboxScale = 1.0f;                      ///< 缩放比例
    int m_letterboxDx = 0;                              ///< X 方向填充偏移
    int m_letterboxDy = 0;                              ///< Y 方向填充偏移

    // ── 输入数据缓冲区（Ort::Value 不复制数据，需要保持其生命周期） ──
    std::vector<float> m_inputBuffer;                   ///< 输入 tensor 数据缓冲区

    // ── COCO 类别名称 ────────────────────────────────────────────────
    static const std::vector<std::string> COCO_CLASSES;
};
