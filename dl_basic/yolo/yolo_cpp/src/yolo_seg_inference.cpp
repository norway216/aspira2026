/**
 * @file    yolo_seg_inference.cpp
 * @brief   YOLOv8 分割模型 ONNX Runtime C++ 推理实现（优化版 v2.0）
 * @details
 *   v2.0 优化内容:
 *   1. [精度] Mask 系数添加 sigmoid 激活
 *   2. [速度] 缓存 name pointers + MemoryInfo，避免每帧重建
 *   3. [速度] ONNX RT 启用 inter-op 并行 + 内存模式优化
 *   4. [速度] 预分配 mask 工作缓冲区
 *   5. [速度] Anchor 扫描 class score stride 预计算
 *   6. [速度] NMS 实现优化
 *   7. [清理] 移除废弃的 decodeBbox 死代码
 */

#include "yolo_seg_inference.h"

#include <algorithm>
#include <cmath>
#include <cstring>
#include <iostream>
#include <numeric>

#include <opencv2/dnn.hpp>
#include <opencv2/imgcodecs.hpp>
#include <opencv2/imgproc.hpp>

// ============================================================================
// COCO 80 类名称
// ============================================================================
const std::vector<std::string> YoloSegInference::COCO_CLASSES = {
    "person","bicycle","car","motorcycle","airplane","bus","train","truck",
    "boat","traffic light","fire hydrant","stop sign","parking meter","bench",
    "bird","cat","dog","horse","sheep","cow","elephant","bear","zebra",
    "giraffe","backpack","umbrella","handbag","tie","suitcase","frisbee",
    "skis","snowboard","sports ball","kite","baseball bat","baseball glove",
    "skateboard","surfboard","tennis racket","bottle","wine glass","cup",
    "fork","knife","spoon","bowl","banana","apple","sandwich","orange",
    "broccoli","carrot","hot dog","pizza","donut","cake","chair","couch",
    "potted plant","bed","dining table","toilet","tv","laptop","mouse",
    "remote","keyboard","cell phone","microwave","oven","toaster","sink",
    "refrigerator","book","clock","vase","scissors","teddy bear",
    "hair drier","toothbrush"
};

// ============================================================================
// 构造函数
// ============================================================================

YoloSegInference::YoloSegInference()
    : m_config(InferenceConfig{})
{
    m_env = std::make_unique<Ort::Env>(ORT_LOGGING_LEVEL_WARNING, "YoloSegInference");
}

YoloSegInference::YoloSegInference(const InferenceConfig& config)
    : m_config(config)
{
    m_env = std::make_unique<Ort::Env>(ORT_LOGGING_LEVEL_WARNING, "YoloSegInference");
}

YoloSegInference::~YoloSegInference()
{
    if (m_config.enableProfiling) {
        std::cout << "[YoloSegInference] 引擎已释放" << std::endl;
    }
}

// ============================================================================
// 初始化
// ============================================================================

bool YoloSegInference::initialize(const std::string& modelPath, int numThreads)
{
    if (m_initialized) {
        std::cerr << "[错误] 引擎已经初始化" << std::endl;
        return false;
    }

    try {
        Ort::SessionOptions sessionOptions;

        // 线程配置
        if (numThreads > 0) {
            sessionOptions.SetIntraOpNumThreads(numThreads);
        }
        sessionOptions.SetInterOpNumThreads(m_config.interOpThreads);

        // 图优化级别
        sessionOptions.SetGraphOptimizationLevel(
            GraphOptimizationLevel::ORT_ENABLE_ALL
        );

        // [优化] 启用内存模式优化 — 复用内存减少分配开销
        if (m_config.enableMemoryPattern) {
            sessionOptions.EnableMemPattern();
        }

        // [优化] 设置并行执行模式
        sessionOptions.SetExecutionMode(ExecutionMode::ORT_PARALLEL);

        // 加载模型
        m_session = std::make_unique<Ort::Session>(
            *m_env, modelPath.c_str(), sessionOptions
        );

        // ── 查询输入输出元信息并缓存 name pointers ──────────────────
        size_t numInputNodes = m_session->GetInputCount();
        size_t numOutputNodes = m_session->GetOutputCount();

        // 先预留空间，避免 push_back 时 vector 重分配导致 c_str() 失效
        m_inputNames.reserve(numInputNodes);
        m_outputNames.reserve(numOutputNodes);

        std::cout << "[模型信息] 输入节点数: " << numInputNodes << std::endl;

        for (size_t i = 0; i < numInputNodes; ++i) {
            auto namePtr = m_session->GetInputNameAllocated(i, m_allocator);
            m_inputNames.push_back(namePtr.get());

            auto typeInfo = m_session->GetInputTypeInfo(i);
            auto tensorInfo = typeInfo.GetTensorTypeAndShapeInfo();
            m_inputShape = tensorInfo.GetShape();

            std::cout << "  输入[" << i << "]: " << m_inputNames[i]
                      << " 形状=[";
            for (size_t j = 0; j < m_inputShape.size(); ++j) {
                std::cout << m_inputShape[j];
                if (j < m_inputShape.size() - 1) std::cout << ", ";
            }
            std::cout << "]" << std::endl;
        }

        std::cout << "[模型信息] 输出节点数: " << numOutputNodes << std::endl;

        for (size_t i = 0; i < numOutputNodes; ++i) {
            auto namePtr = m_session->GetOutputNameAllocated(i, m_allocator);
            m_outputNames.push_back(namePtr.get());

            auto typeInfo = m_session->GetOutputTypeInfo(i);
            auto tensorInfo = typeInfo.GetTensorTypeAndShapeInfo();
            auto shape = tensorInfo.GetShape();

            if (i == 0) m_output0Shape = shape;
            if (i == 1) m_output1Shape = shape;

            std::cout << "  输出[" << i << "]: " << m_outputNames[i]
                      << " 形状=[";
            for (size_t j = 0; j < shape.size(); ++j) {
                std::cout << shape[j];
                if (j < shape.size() - 1) std::cout << ", ";
            }
            std::cout << "]" << std::endl;
        }

        // 在名字全部收集完毕后，再构建 c_str() 指针缓存
        // 避免 vector 重分配时 c_str() 失效
        m_inputNamePtrs.clear();
        for (const auto& name : m_inputNames) {
            m_inputNamePtrs.push_back(name.c_str());
        }
        m_outputNamePtrs.clear();
        for (const auto& name : m_outputNames) {
            m_outputNamePtrs.push_back(name.c_str());
        }

        // [优化] 缓存 CPU MemoryInfo，避免每次预处理重复创建
        m_cachedMemoryInfo = Ort::MemoryInfo::CreateCpu(
            OrtArenaAllocator, OrtMemTypeDefault
        );

        // [优化] 预分配输入和 mask 缓冲区
        const size_t inputSize = 1 * 3 * m_config.inputHeight * m_config.inputWidth;
        m_inputBuffer.resize(inputSize);

        const size_t maskWorkSize = m_config.maskProtoH * m_config.maskProtoW;
        m_maskWorkBuffer.resize(maskWorkSize);

        m_initialized = true;
        std::cout << "[初始化] ONNX 模型加载成功: " << modelPath << std::endl;
        return true;

    } catch (const Ort::Exception& e) {
        std::cerr << "[错误] ONNX Runtime 初始化失败: " << e.what() << std::endl;
        return false;
    } catch (const std::exception& e) {
        std::cerr << "[错误] 初始化异常: " << e.what() << std::endl;
        return false;
    }
}

// ============================================================================
// 主推理接口
// ============================================================================

InferenceResult YoloSegInference::infer(const cv::Mat& image)
{
    InferenceResult result;

    if (!m_initialized || image.empty()) {
        std::cerr << "[错误] 引擎未初始化或图片为空" << std::endl;
        return result;
    }

    try {
        // ── 预处理 ──────────────────────────────────────────────────────
        auto t0 = std::chrono::high_resolution_clock::now();
        Ort::Value inputTensor = preprocess(image);
        auto t1 = std::chrono::high_resolution_clock::now();
        result.preprocessMs = std::chrono::duration<double, std::milli>(t1 - t0).count();

        if (!inputTensor.IsTensor()) {
            std::cerr << "[错误] 预处理失败" << std::endl;
            return result;
        }

        // ── ONNX Runtime 推理 ───────────────────────────────────────────
        // [优化] 使用缓存的 name pointers，无需每次重建 vector<const char*>
        auto outputTensors = m_session->Run(
            Ort::RunOptions{nullptr},
            m_inputNamePtrs.data(),
            &inputTensor,
            1,
            m_outputNamePtrs.data(),
            m_outputNamePtrs.size()
        );

        auto t2 = std::chrono::high_resolution_clock::now();
        result.inferenceMs = std::chrono::duration<double, std::milli>(t2 - t1).count();

        // ── 后处理 ──────────────────────────────────────────────────────
        const float* output0Data = outputTensors[0].GetTensorData<float>();
        const float* output1Data = outputTensors[1].GetTensorData<float>();

        result.detections = postprocess(
            output0Data, output1Data,
            cv::Size(image.cols, image.rows)
        );

        auto t3 = std::chrono::high_resolution_clock::now();
        result.postprocessMs = std::chrono::duration<double, std::milli>(t3 - t2).count();

    } catch (const Ort::Exception& e) {
        std::cerr << "[错误] ONNX Runtime 推理失败: " << e.what() << std::endl;
    } catch (const std::exception& e) {
        std::cerr << "[错误] 推理异常: " << e.what() << std::endl;
    }

    return result;
}

// ============================================================================
// 预处理
// ============================================================================

Ort::Value YoloSegInference::preprocess(const cv::Mat& image)
{
    // Letterbox + BGR→RGB + 归一化 + HWC→CHW
    cv::Mat letterboxed = letterbox(image,
                                     m_config.inputWidth,
                                     m_config.inputHeight,
                                     m_letterboxScale,
                                     m_letterboxDx,
                                     m_letterboxDy);

    // 使用 OpenCV dnn::blobFromImage 一步完成 BGR→RGB + /255 + CHW
    cv::Mat blob = cv::dnn::blobFromImage(
        letterboxed,
        1.0 / 255.0,
        cv::Size(m_config.inputWidth, m_config.inputHeight),
        cv::Scalar(0, 0, 0),
        true,   // swapRB: BGR → RGB
        false   // crop
    );

    // [优化] 直接拷贝到预分配的缓冲区（避免重新分配 vector）
    float* blobData = reinterpret_cast<float*>(blob.data);
    const size_t totalElements = blob.total();
    if (m_inputBuffer.size() < totalElements) {
        m_inputBuffer.resize(totalElements);
    }
    std::memcpy(m_inputBuffer.data(), blobData, totalElements * sizeof(float));

    std::vector<int64_t> inputShape = {
        1, 3, m_config.inputHeight, m_config.inputWidth
    };

    // [优化] 使用缓存的 MemoryInfo
    return Ort::Value::CreateTensor<float>(
        m_cachedMemoryInfo,
        m_inputBuffer.data(),
        totalElements * sizeof(float),
        inputShape.data(),
        inputShape.size()
    );
}

// ============================================================================
// 后处理（优化版）
// ============================================================================

std::vector<SegmentationResult> YoloSegInference::postprocess(
    const float* output0Tensor,
    const float* output1Tensor,
    const cv::Size& originalSize)
{
    std::vector<SegmentationResult> results;

    // ── 维度常量 ──────────────────────────────────────────────────────────
    const int numChannels  = static_cast<int>(m_output0Shape[1]);  // 116
    const int numAnchors   = static_cast<int>(m_output0Shape[2]);  // 8400
    const int bboxChannels = 4;
    const int classChannels = m_config.numClasses;   // 80
    const int maskChannels  = m_config.maskChannels; // 32
    const int protoH = static_cast<int>(m_output1Shape[2]);  // 160
    const int protoW = static_cast<int>(m_output1Shape[3]);  // 160
    const int protoArea = protoH * protoW;                   // 25600
    const int stride = numAnchors;                           // 8400

    // 验证维度
    if (numChannels != bboxChannels + classChannels + maskChannels) {
        std::cerr << "[错误] output0 通道数不匹配" << std::endl;
        return results;
    }

    // [优化] 预计算各段在 stride 上的起始偏移
    const int bboxOffset  = 0;                          // channels [0, 4)
    const int classOffset = bboxChannels;               // channels [4, 84)
    const int maskOffset  = bboxChannels + classChannels; // channels [84, 116)

    // ── 第1遍：收集候选检测 ─────────────────────────────────────────────
    struct RawDetection {
        cv::Rect bbox;
        float confidence;
        int classId;
        float maskCoeffs[32];  // [优化] 使用栈数组替代 std::vector
    };

    std::vector<RawDetection> candidates;
    candidates.reserve(64);  // [优化] 预分配典型候选数（通常 < 50）

    // [优化] 使用局部变量缓存常量，便于编译器优化
    const float confThresh = m_config.confThreshold;
    const float inputW = static_cast<float>(m_config.inputWidth);
    const float inputH = static_cast<float>(m_config.inputHeight);

    for (int a = 0; a < numAnchors; ++a) {
        // ── 边界框（已解码为绝对坐标） ──────────────────────────────────
        float cx = output0Tensor[a + bboxOffset * stride];
        float cy = output0Tensor[a + 1 * stride];
        float bw = output0Tensor[a + 2 * stride];
        float bh = output0Tensor[a + 3 * stride];

        // ── 类别分数（已 sigmoid），找最高分 ───────────────────────────
        float maxScore = 0.0f;
        int bestClassId = 0;

        // [优化] 内联循环 — 编译器可自动向量化 (SIMD)
        const float* scorePtr = output0Tensor + a + classOffset * stride;
        for (int c = 0; c < classChannels; ++c) {
            float s = scorePtr[c * stride];  // 注意：c * stride 是跨通道步长
            if (s > maxScore) {
                maxScore = s;
                bestClassId = c;
            }
        }

        if (maxScore < confThresh) continue;

        // ── Mask 系数（读取32个 float） ────────────────────────────────
        RawDetection det;
        det.confidence = maxScore;
        det.classId = bestClassId;

        // [优化] 用 memcpy 批量复制 32 个 float
        const float* coeffPtr = output0Tensor + a + maskOffset * stride;
        for (int k = 0; k < maskChannels; ++k) {
            det.maskCoeffs[k] = coeffPtr[k * stride];
        }

        // ── 构造边界框 [cx,cy,w,h] → [x1,y1,w,h] ──────────────────────
        float x1 = cx - bw * 0.5f;
        float y1 = cy - bh * 0.5f;
        x1 = std::max(0.0f, std::min(x1, inputW - 1.0f));
        y1 = std::max(0.0f, std::min(y1, inputH - 1.0f));
        bw = std::max(1.0f, std::min(bw, inputW - x1));
        bh = std::max(1.0f, std::min(bh, inputH - y1));

        det.bbox = cv::Rect(
            static_cast<int>(x1), static_cast<int>(y1),
            static_cast<int>(bw), static_cast<int>(bh)
        );

        candidates.push_back(std::move(det));
    }

    if (m_config.enableProfiling) {
        std::cout << "[后处理] 候选数: " << candidates.size()
                  << " / " << numAnchors << std::endl;
    }

    if (candidates.empty()) return results;

    // ── 第2步：NMS ──────────────────────────────────────────────────────
    std::vector<cv::Rect> boxes;
    std::vector<float> scores;
    boxes.reserve(candidates.size());
    scores.reserve(candidates.size());
    for (const auto& c : candidates) {
        boxes.push_back(c.bbox);
        scores.push_back(c.confidence);
    }

    std::vector<int> keepIndices = nms(boxes, scores, m_config.iouThreshold);

    if (keepIndices.empty()) return results;

    // ── 第3步：为每个保留的检测生成实例分割 Mask ─────────────────────
    // [优化] 预分配 mask 工作缓冲区
    if (m_maskWorkBuffer.size() < static_cast<size_t>(protoArea)) {
        m_maskWorkBuffer.resize(protoArea);
    }
    float* maskWork = m_maskWorkBuffer.data();

    for (int idx : keepIndices) {
        const auto& cand = candidates[idx];

        // ── 3a. 计算实例 Mask: mask = sigmoid(coeffs @ proto) ───────────
        // [优化] 清零预分配缓冲区（避免每次分配新 vector）
        std::memset(maskWork, 0, protoArea * sizeof(float));

        for (int k = 0; k < maskChannels; ++k) {
            float coeff = cand.maskCoeffs[k];

            // [精度修复] Mask 系数需要 sigmoid 激活
            // ultralytics ONNX 输出中 class score 已 sigmoid，但 mask 系数未激活
            coeff = 1.0f / (1.0f + std::exp(-coeff));

            const float* protoChannel = output1Tensor + k * protoArea;

            // [优化] 内联累加，编译器可向量化
            for (int p = 0; p < protoArea; ++p) {
                maskWork[p] += coeff * protoChannel[p];
            }
        }

        // ── 3b. Sigmoid + 转换为 CV_8U Mask ───────────────────────────
        cv::Mat mask160(protoH, protoW, CV_32FC1, maskWork);
        cv::Mat mask640;

        // 组合 sigmoid + resize 为一次操作
        // 先对 160x160 做 sigmoid，再 resize 到 640x640
        for (int p = 0; p < protoArea; ++p) {
            maskWork[p] = 1.0f / (1.0f + std::exp(-maskWork[p]));
        }

        cv::resize(mask160, mask640,
                    cv::Size(m_config.inputWidth, m_config.inputHeight),
                    0, 0, cv::INTER_LINEAR);

        // ── 3c. 二值化 + 裁剪到检测框 ──────────────────────────────────
        cv::Mat maskBinary;
        cv::threshold(mask640, maskBinary, m_config.maskThreshold, 255.0,
                      cv::THRESH_BINARY);
        maskBinary.convertTo(maskBinary, CV_8UC1);

        cv::Mat maskCropped = cv::Mat::zeros(m_config.inputHeight,
                                              m_config.inputWidth, CV_8UC1);
        cv::Rect clampedBbox = cand.bbox & cv::Rect(0, 0,
                                                     m_config.inputWidth,
                                                     m_config.inputHeight);
        if (clampedBbox.width > 0 && clampedBbox.height > 0) {
            maskBinary(clampedBbox).copyTo(maskCropped(clampedBbox));
        }

        // ── 3d. 逆 letterbox 变换 → 原图坐标 ───────────────────────────
        cv::Mat maskNoPad;
        cv::Rect imageRoi(m_letterboxDx, m_letterboxDy,
                           m_config.inputWidth - 2 * m_letterboxDx,
                           m_config.inputHeight - 2 * m_letterboxDy);
        if (imageRoi.width > 0 && imageRoi.height > 0) {
            maskCropped(imageRoi).copyTo(maskNoPad);
        } else {
            maskCropped.copyTo(maskNoPad);
        }

        cv::Mat maskOriginal;
        cv::resize(maskNoPad, maskOriginal, originalSize,
                    0, 0, cv::INTER_LINEAR);
        cv::threshold(maskOriginal, maskOriginal, 127, 255, cv::THRESH_BINARY);
        maskOriginal.convertTo(maskOriginal, CV_8UC1);

        // ── 3e. Bbox 坐标逆变换 ────────────────────────────────────────
        float x1Orig = (cand.bbox.x - m_letterboxDx) / m_letterboxScale;
        float y1Orig = (cand.bbox.y - m_letterboxDy) / m_letterboxScale;
        float wOrig  = cand.bbox.width / m_letterboxScale;
        float hOrig  = cand.bbox.height / m_letterboxScale;

        cv::Rect bboxOriginal(
            static_cast<int>(x1Orig), static_cast<int>(y1Orig),
            static_cast<int>(wOrig), static_cast<int>(hOrig)
        );
        bboxOriginal &= cv::Rect(0, 0, originalSize.width, originalSize.height);

        // ── 3f. 构建结果 ────────────────────────────────────────────────
        results.push_back({
            bboxOriginal,
            cand.classId,
            (cand.classId >= 0 && cand.classId < static_cast<int>(COCO_CLASSES.size()))
                ? COCO_CLASSES[cand.classId] : "unknown",
            cand.confidence,
            maskOriginal
        });
    }

    std::sort(results.begin(), results.end());

    if (m_config.enableProfiling) {
        std::cout << "[后处理] 候选数: " << candidates.size()
                  << " → NMS后: " << results.size() << std::endl;
    }

    return results;
}

// ============================================================================
// Letterbox 缩放
// ============================================================================

cv::Mat YoloSegInference::letterbox(
    const cv::Mat& image, int targetW, int targetH,
    float& scale, int& dx, int& dy)
{
    int origW = image.cols;
    int origH = image.rows;

    // 计算缩放比例（保持宽高比）
    float r = std::min(
        static_cast<float>(targetW) / origW,
        static_cast<float>(targetH) / origH
    );
    // [优化] 使用 round 避免浮点精度问题导致少1像素
    int newW = static_cast<int>(std::round(origW * r));
    int newH = static_cast<int>(std::round(origH * r));

    // 确保 newW/newH 不超过 target
    newW = std::min(newW, targetW);
    newH = std::min(newH, targetH);

    scale = r;
    dx = (targetW - newW) / 2;
    dy = (targetH - newH) / 2;

    // [优化] 缩小场景用 INTER_AREA（比 INTER_LINEAR 更快且抗锯齿更好）
    cv::Mat resized;
    cv::resize(image, resized, cv::Size(newW, newH), 0, 0,
               (newW < origW) ? cv::INTER_AREA : cv::INTER_LINEAR);

    cv::Mat canvas(targetH, targetW, CV_8UC3, cv::Scalar(114, 114, 114));
    cv::Rect roi(dx, dy, newW, newH);
    if (roi.width > 0 && roi.height > 0) {
        resized.copyTo(canvas(roi));
    }

    return canvas;
}

// ============================================================================
// IoU 计算
// ============================================================================

float YoloSegInference::computeIoU(const cv::Rect& a, const cv::Rect& b)
{
    int interLeft   = std::max(a.x, b.x);
    int interTop    = std::max(a.y, b.y);
    int interRight  = std::min(a.x + a.width, b.x + b.width);
    int interBottom = std::min(a.y + a.height, b.y + b.height);

    if (interLeft >= interRight || interTop >= interBottom) return 0.0f;

    float interArea = static_cast<float>(
        (interRight - interLeft) * (interBottom - interTop)
    );
    float areaA = static_cast<float>(a.width * a.height);
    float areaB = static_cast<float>(b.width * b.height);

    return interArea / (areaA + areaB - interArea);
}

// ============================================================================
// NMS（优化版 — 原地筛选）
// ============================================================================

std::vector<int> YoloSegInference::nms(
    const std::vector<cv::Rect>& boxes,
    const std::vector<float>& scores,
    float iouThreshold)
{
    const size_t n = boxes.size();

    // 创建索引并按分数降序排列
    std::vector<int> indices(n);
    std::iota(indices.begin(), indices.end(), 0);
    std::sort(indices.begin(), indices.end(),
              [&scores](int a, int b) { return scores[a] > scores[b]; });

    // [优化] 使用 bool 数组标记被抑制的框，避免重复创建 vector
    std::vector<bool> suppressed(n, false);
    std::vector<int> keep;
    keep.reserve(std::min(n, size_t(20)));  // 典型结果 < 20个

    for (size_t i = 0; i < n; ++i) {
        int idxI = indices[i];
        if (suppressed[idxI]) continue;

        keep.push_back(idxI);

        // 抑制所有与当前框 IoU 超过阈值的后续框
        for (size_t j = i + 1; j < n; ++j) {
            int idxJ = indices[j];
            if (suppressed[idxJ]) continue;

            if (computeIoU(boxes[idxI], boxes[idxJ]) > iouThreshold) {
                suppressed[idxJ] = true;
            }
        }
    }

    return keep;
}

// ============================================================================
// 可视化
// ============================================================================

cv::Mat YoloSegInference::visualize(
    const cv::Mat& image,
    const std::vector<SegmentationResult>& detections,
    bool drawMask) const
{
    static const std::vector<cv::Scalar> COLORS = {
        cv::Scalar(56,56,255), cv::Scalar(151,157,255), cv::Scalar(31,112,255),
        cv::Scalar(29,178,255), cv::Scalar(49,210,207), cv::Scalar(10,249,72),
        cv::Scalar(23,204,146), cv::Scalar(134,219,61), cv::Scalar(52,147,26),
        cv::Scalar(187,212,0), cv::Scalar(168,115,22), cv::Scalar(179,115,198),
    };

    cv::Mat output = image.clone();

    for (size_t i = 0; i < detections.size(); ++i) {
        const auto& det = detections[i];
        cv::Scalar color = COLORS[det.classId % COLORS.size()];

        // ── 半透明 Mask ─────────────────────────────────────────────────
        if (drawMask && !det.mask.empty()) {
            cv::Mat coloredMask(image.rows, image.cols, CV_8UC3, cv::Scalar(0,0,0));
            coloredMask.setTo(color, det.mask);
            cv::addWeighted(output, 1.0, coloredMask, 0.35, 0.0, output);

            // Mask 轮廓
            std::vector<std::vector<cv::Point>> contours;
            cv::findContours(det.mask.clone(), contours,
                             cv::RETR_EXTERNAL, cv::CHAIN_APPROX_SIMPLE);
            cv::drawContours(output, contours, -1,
                             cv::Scalar(255,255,255), 2);
        }

        // ── 边界框 ─────────────────────────────────────────────────────
        cv::rectangle(output, det.bbox, color, 2);

        // ── 标签 ───────────────────────────────────────────────────────
        std::string label = det.className + " "
                          + std::to_string(static_cast<int>(det.confidence * 100)) + "%";

        int baseline = 0;
        cv::Size textSize = cv::getTextSize(label, cv::FONT_HERSHEY_SIMPLEX,
                                             0.5, 1, &baseline);

        cv::Point labelTopLeft(
            det.bbox.x,
            std::max(det.bbox.y - textSize.height - 8, 0)
        );

        cv::rectangle(output, labelTopLeft,
                      cv::Point(labelTopLeft.x + textSize.width + 4,
                                labelTopLeft.y + textSize.height + 4),
                      color, cv::FILLED);

        cv::putText(output, label,
                    cv::Point(labelTopLeft.x + 2,
                              labelTopLeft.y + textSize.height + 1),
                    cv::FONT_HERSHEY_SIMPLEX, 0.5,
                    cv::Scalar(255,255,255), 1, cv::LINE_AA);
    }

    return output;
}
