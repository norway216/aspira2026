/**
 * @file    yolo_seg_inference.cpp
 * @brief   YOLOv8 分割模型 ONNX Runtime C++ 推理实现
 * @details 详细实现说明见各函数的中文注释。
 *          本文件实现了完整的 YOLO 分割推理管线，包括：
 *          预处理 → ONNX Runtime 推理 → 后处理 → 可视化
 *
 * 部署到嵌入式设备的注意事项：
 *  - 使用 CPU Execution Provider（ARM Linux 上编译 ONNX Runtime 时选择 CPU）
 *  - 可通过 InferenceConfig 调整线程数、置信度阈值等参数
 *  - 预处理中的 letterbox 和归一化可用 OpenCV 的 cv::dnn::blobFromImage 替代
 *  - 批量推理（Batch > 1）需要修改输入形状和预处理逻辑
 */

#include "yolo_seg_inference.h"

// ── 标准库 ──────────────────────────────────────────────────────────────
#include <algorithm>   // std::sort, std::max, std::min
#include <cmath>       // std::exp (sigmoid), std::sqrt
#include <cstring>     // std::memcpy, std::memset
#include <iostream>    // std::cout, std::cerr
#include <numeric>     // std::iota

// ── OpenCV 补充头文件 ────────────────────────────────────────────────
#include <opencv2/dnn.hpp>        // cv::dnn::blobFromImage
#include <opencv2/imgcodecs.hpp>  // cv::imread, cv::imwrite
#include <opencv2/imgproc.hpp>    // cv::findContours, cv::drawContours

// ============================================================================
// COCO 数据集 80 类名称定义
// ============================================================================
// YOLOv8-seg 模型在 COCO 数据集上训练，输出 80 个类别
// 注意：这些名称必须与训练时的类别顺序一致
const std::vector<std::string> YoloSegInference::COCO_CLASSES = {
    "person",        // 0
    "bicycle",       // 1
    "car",           // 2
    "motorcycle",    // 3
    "airplane",      // 4
    "bus",           // 5
    "train",         // 6
    "truck",         // 7
    "boat",          // 8
    "traffic light", // 9
    "fire hydrant",  // 10
    "stop sign",     // 11
    "parking meter", // 12
    "bench",         // 13
    "bird",          // 14
    "cat",           // 15
    "dog",           // 16
    "horse",         // 17
    "sheep",         // 18
    "cow",           // 19
    "elephant",      // 20
    "bear",          // 21
    "zebra",         // 22
    "giraffe",       // 23
    "backpack",      // 24
    "umbrella",      // 25
    "handbag",       // 26
    "tie",           // 27
    "suitcase",      // 28
    "frisbee",       // 29
    "skis",          // 30
    "snowboard",     // 31
    "sports ball",   // 32
    "kite",          // 33
    "baseball bat",  // 34
    "baseball glove",// 35
    "skateboard",    // 36
    "surfboard",     // 37
    "tennis racket", // 38
    "bottle",        // 39
    "wine glass",    // 40
    "cup",           // 41
    "fork",          // 42
    "knife",         // 43
    "spoon",         // 44
    "bowl",          // 45
    "banana",        // 46
    "apple",         // 47
    "sandwich",      // 48
    "orange",        // 49
    "broccoli",      // 50
    "carrot",        // 51
    "hot dog",       // 52
    "pizza",         // 53
    "donut",         // 54
    "cake",          // 55
    "chair",         // 56
    "couch",         // 57
    "potted plant",  // 58
    "bed",           // 59
    "dining table",  // 60
    "toilet",        // 61
    "tv",            // 62
    "laptop",        // 63
    "mouse",         // 64
    "remote",        // 65
    "keyboard",      // 66
    "cell phone",    // 67
    "microwave",     // 68
    "oven",          // 69
    "toaster",       // 70
    "sink",          // 71
    "refrigerator",  // 72
    "book",          // 73
    "clock",         // 74
    "vase",          // 75
    "scissors",      // 76
    "teddy bear",    // 77
    "hair drier",    // 78
    "toothbrush"     // 79
};

// ============================================================================
// 构造函数
// ============================================================================

YoloSegInference::YoloSegInference()
    : m_config(InferenceConfig{})
{
    // 默认使用 CPU 推理，创建 ONNX Runtime 环境
    // 嵌入式设备上通常使用 CPU；Jetson 上可替换为 CUDA
    // ORT_LOGGING_LEVEL_WARNING: 只输出警告和错误，减少日志噪音
    m_env = std::make_unique<Ort::Env>(
        ORT_LOGGING_LEVEL_WARNING,
        "YoloSegInference"
    );
}

YoloSegInference::YoloSegInference(const InferenceConfig& config)
    : m_config(config)
{
    m_env = std::make_unique<Ort::Env>(
        ORT_LOGGING_LEVEL_WARNING,
        "YoloSegInference"
    );
}

YoloSegInference::~YoloSegInference()
{
    // Ort::Session 和 Ort::Env 的 unique_ptr 自动调用析构
    // 注意析构顺序：先析构 m_session，再析构 m_env
    // unique_ptr 的成员按声明逆序析构，所以这里是对的（m_env 声明在 m_session 之前）
    std::cout << "[YoloSegInference] 引擎已释放" << std::endl;
}

// ============================================================================
// 初始化：加载 ONNX 模型
// ============================================================================

bool YoloSegInference::initialize(const std::string& modelPath, int numThreads)
{
    if (m_initialized) {
        std::cerr << "[错误] 引擎已经初始化，请勿重复调用 initialize()" << std::endl;
        return false;
    }

    try {
        // ── 第1步：配置 ONNX Runtime Session 选项 ─────────────────────────
        // SessionOptions 控制推理行为：线程数、图优化级别、执行提供者等
        Ort::SessionOptions sessionOptions;

        // 设置线程数（0 表示自动检测 CPU 核心数）
        if (numThreads > 0) {
            sessionOptions.SetIntraOpNumThreads(numThreads);
        }

        // 启用图优化（ORT_ENABLE_ALL = 启用所有优化pass）
        // 嵌入式设备上可设为 ORT_ENABLE_BASIC 以减少初始化时间
        sessionOptions.SetGraphOptimizationLevel(
            GraphOptimizationLevel::ORT_ENABLE_ALL
        );

        // ── 第2步：加载 ONNX 模型并创建会话 ───────────────────────────────
        // modelPath 支持相对路径和绝对路径
        // 第二个参数为模型数据的内存指针（从文件加载时传 nullptr）
        m_session = std::make_unique<Ort::Session>(
            *m_env,
            modelPath.c_str(),
            sessionOptions
        );

        // ── 第3步：查询模型的输入输出元信息 ───────────────────────────────
        // 缓存输入/输出节点的名称、形状，避免每次推理时重复查询
        Ort::MemoryInfo memoryInfo = Ort::MemoryInfo::CreateCpu(
            OrtArenaAllocator, OrtMemTypeDefault
        );

        // 获取输入信息
        size_t numInputNodes = m_session->GetInputCount();
        std::cout << "[模型信息] 输入节点数: " << numInputNodes << std::endl;

        for (size_t i = 0; i < numInputNodes; ++i) {
            // GetInputNameAllocated 返回 Ort::AllocatedStringPtr (unique_ptr<char>)
            // 通过 .get() 获取原始 const char* 指针
            auto namePtr = m_session->GetInputNameAllocated(i, m_allocator);
            m_inputNames.push_back(namePtr.get());
            // AllocatedStringPtr 在离开作用域时自动释放内存

            // 获取输入形状
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

        // 获取输出信息
        size_t numOutputNodes = m_session->GetOutputCount();
        std::cout << "[模型信息] 输出节点数: " << numOutputNodes << std::endl;

        for (size_t i = 0; i < numOutputNodes; ++i) {
            auto namePtr = m_session->GetOutputNameAllocated(i, m_allocator);
            m_outputNames.push_back(namePtr.get());

            auto typeInfo = m_session->GetOutputTypeInfo(i);
            auto tensorInfo = typeInfo.GetTensorTypeAndShapeInfo();
            auto shape = tensorInfo.GetShape();

            // 缓存 output0 和 output1 的形状
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

        m_initialized = true;
        std::cout << "[初始化] ONNX 模型加载成功: " << modelPath << std::endl;
        return true;

    } catch (const Ort::Exception& e) {
        // ONNX Runtime 异常：常见原因包括模型文件不存在、模型损坏、opset 不兼容等
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

    if (!m_initialized) {
        std::cerr << "[错误] 引擎未初始化，请先调用 initialize()" << std::endl;
        return result;
    }

    if (image.empty()) {
        std::cerr << "[错误] 输入图片为空" << std::endl;
        return result;
    }

    try {
        // ── 步骤1：预处理 ──────────────────────────────────────────────────
        auto t0 = std::chrono::high_resolution_clock::now();

        Ort::Value inputTensor = preprocess(image);

        auto t1 = std::chrono::high_resolution_clock::now();
        result.preprocessMs = std::chrono::duration<double, std::milli>(t1 - t0).count();

        if (!inputTensor.IsTensor()) {
            std::cerr << "[错误] 预处理失败" << std::endl;
            return result;
        }

        // ── 步骤2：ONNX Runtime 推理 ───────────────────────────────────────
        // Run() 的参数：
        //   - inputNames: 输入节点名称列表
        //   - inputValues: 输入 Tensor 列表
        //   - outputNames: 输出节点名称列表
        // 返回值是一个 vector<Ort::Value>，包含模型的全部输出

        // 将 std::string 转换为 const char* 数组（ONNX Runtime API 需要）
        std::vector<const char*> inputNamePtrs;
        std::vector<const char*> outputNamePtrs;
        for (const auto& name : m_inputNames) {
            inputNamePtrs.push_back(name.c_str());
        }
        for (const auto& name : m_outputNames) {
            outputNamePtrs.push_back(name.c_str());
        }

        auto outputTensors = m_session->Run(
            Ort::RunOptions{nullptr},    // 默认运行选项
            inputNamePtrs.data(),         // 输入名称数组
            &inputTensor,                 // 输入 Tensor 指针（单个输入）
            1,                            // 输入数量
            outputNamePtrs.data(),        // 输出名称数组
            outputNamePtrs.size()         // 输出数量
        );

        auto t2 = std::chrono::high_resolution_clock::now();
        result.inferenceMs = std::chrono::duration<double, std::milli>(t2 - t1).count();

        // ── 步骤3：后处理 ──────────────────────────────────────────────────
        // 获取原始数据指针
        // GetTensorMutableData<float>() 返回可写的 float* 指针
        const float* output0Data = outputTensors[0].GetTensorData<float>();
        const float* output1Data = outputTensors[1].GetTensorData<float>();

        result.detections = postprocess(
            output0Data,
            output1Data,
            cv::Size(image.cols, image.rows)  // 原图尺寸
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
// 预处理：图片 → ONNX Runtime Tensor
// ============================================================================

Ort::Value YoloSegInference::preprocess(const cv::Mat& image)
{
    // ── 步骤1：Letterbox 缩放（保持宽高比，填充到 640x640） ──────────────
    cv::Mat letterboxed = letterbox(image,
                                     m_config.inputWidth,
                                     m_config.inputHeight,
                                     m_letterboxScale,
                                     m_letterboxDx,
                                     m_letterboxDy);

    // ── 步骤2：使用 OpenCV dnn::blobFromImage 一步完成 ──────────────────
    // BGR→RGB + 归一化到[0,1] + HWC→CHW 转换
    // 参数说明：
    //   scalefactor=1.0/255.0: 将 uint8 [0,255] → float [0,1]
    //   size=(640,640): 已经是目标尺寸（letterbox 后）
    //   mean=(0,0,0): 不减去均值（YOLOv8 不需要）
    //   swapRB=true: BGR→RGB（OpenCV 读取为 BGR，模型期望 RGB）
    //   crop=false: 不裁剪
    cv::Mat blob = cv::dnn::blobFromImage(
        letterboxed,
        1.0 / 255.0,
        cv::Size(m_config.inputWidth, m_config.inputHeight),
        cv::Scalar(0, 0, 0),
        true,   // swapRB: BGR → RGB
        false   // crop
    );

    // blob 的形状是 [1, 3, 640, 640] (NCHW)，数据类型 CV_32F
    // 数据已经在连续内存中

    // ── 步骤3：创建 ONNX Runtime Tensor ──────────────────────────────────
    // blobFromImage 输出格式: NCHW [1, 3, 640, 640], CV_32F
    // 关键：Ort::Value::CreateTensor 不复制数据，只引用指针
    // 因此必须将数据复制到成员变量 m_inputBuffer 中保持其生命周期
    float* blobData = reinterpret_cast<float*>(blob.data);
    const size_t totalElements = blob.total();  // 1 * 3 * 640 * 640

    // 复制到成员变量，确保 Ort::Value 在推理时数据仍然有效
    m_inputBuffer.assign(blobData, blobData + totalElements);

    Ort::MemoryInfo memoryInfo = Ort::MemoryInfo::CreateCpu(
        OrtArenaAllocator, OrtMemTypeDefault
    );

    std::vector<int64_t> inputShape = {
        1, 3, m_config.inputHeight, m_config.inputWidth  // NCHW
    };

    return Ort::Value::CreateTensor<float>(
        memoryInfo,
        m_inputBuffer.data(),          // 指向成员变量，生命周期与对象一致
        m_inputBuffer.size() * sizeof(float),
        inputShape.data(),
        inputShape.size()
    );
}

// ============================================================================
// 后处理：模型输出 → 检测结果
// ============================================================================

std::vector<SegmentationResult> YoloSegInference::postprocess(
    const float* output0Tensor,
    const float* output1Tensor,
    const cv::Size& originalSize)
{
    std::vector<SegmentationResult> results;

    // ── 解析 output0 的维度 ──────────────────────────────────────────────
    // batchSize 固定为1; numChannels=116=4(bbox)+80(cls)+32(mask); numAnchors=8400
    [[maybe_unused]] const int batchSize   = static_cast<int>(m_output0Shape[0]);  // 1
    const int numChannels = static_cast<int>(m_output0Shape[1]);  // 116
    const int numAnchors  = static_cast<int>(m_output0Shape[2]);  // 8400

    const int bboxChannels  = 4;                          // cx, cy, w, h
    const int classChannels = m_config.numClasses;        // 80
    const int maskChannels  = m_config.maskChannels;      // 32

    // 验证输出维度正确性
    if (numChannels != bboxChannels + classChannels + maskChannels) {
        std::cerr << "[错误] output0 通道数不匹配: 期望 "
                  << (bboxChannels + classChannels + maskChannels)
                  << ", 实际 " << numChannels << std::endl;
        return results;
    }

    // ── 解析 output1（原型 Mask）的维度 ──────────────────────────────────
    [[maybe_unused]] const int protoB = static_cast<int>(m_output1Shape[0]);  // 1
    [[maybe_unused]] const int protoC = static_cast<int>(m_output1Shape[1]);  // 32
    const int protoH = static_cast<int>(m_output1Shape[2]);  // 160
    const int protoW = static_cast<int>(m_output1Shape[3]);  // 160

    // ── 第1遍遍历：收集所有超过置信度阈值的候选检测 ──────────────────────
    // 每个 anchor 点包含:
    //   [0:4]   = [cx, cy, w, h] 绝对坐标（640x640 空间，已解码）
    //   [4:84]  = 80 个类别分数（已 sigmoid）
    //   [84:116]= 32 个 mask 系数

    struct RawDetection {
        cv::Rect bbox;           ///< 边界框（640x640 输入空间，[x,y,w,h]）
        float confidence;        ///< 最高类别置信度
        int classId;             ///< 类别 ID
        std::vector<float> maskCoeffs;  ///< 32个mask系数
    };

    // 关键：output0 形状为 [1, 116, 8400]，C 行优先布局
    // data[channel * 8400 + anchor]
    const int stride = numAnchors;  // 8400 — 同一通道内相邻 anchor 的步长

    std::vector<RawDetection> candidates;

    for (int a = 0; a < numAnchors; ++a) {
        // ── 提取边界框参数（[cx, cy, w, h]，已解码为绝对坐标） ────────
        float cx = output0Tensor[a + 0 * stride];
        float cy = output0Tensor[a + 1 * stride];
        float bw = output0Tensor[a + 2 * stride];
        float bh = output0Tensor[a + 3 * stride];

        // ── 提取类别分数（已 sigmoid），找最大值 ────────────────────────
        float maxScore = 0.0f;
        int bestClassId = 0;

        for (int c = 0; c < classChannels; ++c) {
            float score = output0Tensor[a + (bboxChannels + c) * stride];
            if (score > maxScore) {
                maxScore = score;
                bestClassId = c;
            }
        }

        // ── 置信度过滤 ──────────────────────────────────────────────────
        if (maxScore < m_config.confThreshold) {
            continue;
        }

        // ── 提取 Mask 系数 ──────────────────────────────────────────────
        std::vector<float> maskCoeffs(maskChannels);
        for (int k = 0; k < maskChannels; ++k) {
            maskCoeffs[k] = output0Tensor[a + (bboxChannels + classChannels + k) * stride];
        }

        // ── 构造边界框：[cx, cy, w, h] → [x1, y1, w, h] ───────────────
        float x1 = cx - bw / 2.0f;
        float y1 = cy - bh / 2.0f;

        // 裁剪到 640×640 范围内
        x1 = std::max(0.0f, std::min(x1, static_cast<float>(m_config.inputWidth - 1)));
        y1 = std::max(0.0f, std::min(y1, static_cast<float>(m_config.inputHeight - 1)));
        bw = std::max(1.0f, std::min(bw, static_cast<float>(m_config.inputWidth - x1)));
        bh = std::max(1.0f, std::min(bh, static_cast<float>(m_config.inputHeight - y1)));

        cv::Rect bbox(
            static_cast<int>(x1),
            static_cast<int>(y1),
            static_cast<int>(bw),
            static_cast<int>(bh)
        );

        candidates.push_back({bbox, maxScore, bestClassId, std::move(maskCoeffs)});
    }

    // ── 性能日志 ──────────────────────────────────────────────────
    if (m_config.enableProfiling) {
        std::cout << "[后处理] 候选数: " << candidates.size()
                  << " / " << numAnchors << std::endl;
    }

    if (candidates.empty()) {
        return results;
    }

    // ── 第2步：NMS 非极大值抑制 ──────────────────────────────────────────
    // 提取所有候选的边界框和置信度用于 NMS
    std::vector<cv::Rect> boxes;
    std::vector<float> scores;

    for (const auto& cand : candidates) {
        boxes.push_back(cand.bbox);
        scores.push_back(cand.confidence);
    }

    // 运行 NMS，返回保留的索引列表
    std::vector<int> keepIndices = nms(boxes, scores, m_config.iouThreshold);

    // ── 第3步：为每个保留的检测生成实例分割 Mask ──────────────────────
    // 预先将原型 Mask 展平为 [32, 160*160] 矩阵
    const int protoArea = protoH * protoW;  // 160*160 = 25600
    // 注意：output1Tensor 的布局是 [1, 32, 160, 160]，即 CHW 格式
    // 每个 channel 是连续的 160×160 数据

    for (int idx : keepIndices) {
        const auto& cand = candidates[idx];

        // ── 3a. 计算实例 Mask ───────────────────────────────────────────
        // 公式: mask = sigmoid(mask_coefficients @ proto_masks_flat)
        // 其中 mask_coefficients: [32]
        //       proto_masks_flat: [32, 25600]
        //       mask: [25600] → reshape to [160, 160]
        //
        // 具体实现：
        //   mask[j] = sum_{k=0}^{31} (coeff[k] * proto[k, j])
        //   然后 sigmoid(mask[j])
        //
        // 由于每个位置的运算是独立的，直接逐像素计算

        std::vector<float> mask160(protoArea, 0.0f);

        for (int k = 0; k < maskChannels; ++k) {
            float coeff = cand.maskCoeffs[k];
            // 第 k 个原型 Mask 的数据在 output1 中的偏移
            const float* protoChannel = output1Tensor + k * protoArea;

            // 累加: mask += coeff * proto_channel
            for (int p = 0; p < protoArea; ++p) {
                mask160[p] += coeff * protoChannel[p];
            }
        }

        // ── 3b. Sigmoid 激活 ────────────────────────────────────────────
        // sigmoid(x) = 1 / (1 + exp(-x))
        for (int p = 0; p < protoArea; ++p) {
            mask160[p] = 1.0f / (1.0f + std::exp(-mask160[p]));
        }

        // ── 3c. 转换为 OpenCV Mat 并缩放到输入图片尺寸 (640×640) ───────
        cv::Mat maskProto(protoH, protoW, CV_32FC1, mask160.data());
        cv::Mat mask640;
        cv::resize(maskProto, mask640,
                    cv::Size(m_config.inputWidth, m_config.inputHeight),
                    0, 0, cv::INTER_LINEAR);

        // ── 3d. 二值化 ──────────────────────────────────────────────────
        // 大于阈值为前景（255），否则为背景（0）
        cv::Mat maskBinary;
        cv::threshold(mask640, maskBinary,
                      m_config.maskThreshold, 255.0,
                      cv::THRESH_BINARY);
        maskBinary.convertTo(maskBinary, CV_8UC1);

        // ── 3e. 裁剪到检测框范围内（排除其他目标的 Mask 区域） ─────────
        // 这一步是可选的，但能提高分割精度：将 mask 限制在检测框内
        cv::Mat maskCropped = cv::Mat::zeros(m_config.inputHeight,
                                              m_config.inputWidth, CV_8UC1);
        cv::Rect clampedBbox = cand.bbox & cv::Rect(0, 0,
                                                     m_config.inputWidth,
                                                     m_config.inputHeight);
        if (clampedBbox.width > 0 && clampedBbox.height > 0) {
            maskBinary(clampedBbox).copyTo(maskCropped(clampedBbox));
        }

        // ── 3f. 缩放 Mask 回原图尺寸 ────────────────────────────────────
        // 需要应用 letterbox 的逆变换：
        //   orig_coord = (coord_640 - padding_offset) / scale
        // 注意：mask 从 640×640 空间通过 cv::resize 缩放到原图尺寸
        // 但更准确的做法是先去除 letterbox padding，再缩放到原图
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

        // 再次二值化（resize 可能引入非整数像素值）
        cv::threshold(maskOriginal, maskOriginal, 127, 255, cv::THRESH_BINARY);
        maskOriginal.convertTo(maskOriginal, CV_8UC1);

        // ── 3g. 将边界框从 640×640 空间映射回原图坐标 ─────────────────
        // 逆 letterbox 变换: orig = (coord_640 - pad) / scale
        float x1Orig = (cand.bbox.x - m_letterboxDx) / m_letterboxScale;
        float y1Orig = (cand.bbox.y - m_letterboxDy) / m_letterboxScale;
        float wOrig = cand.bbox.width / m_letterboxScale;
        float hOrig = cand.bbox.height / m_letterboxScale;

        cv::Rect bboxOriginal(
            static_cast<int>(x1Orig),
            static_cast<int>(y1Orig),
            static_cast<int>(wOrig),
            static_cast<int>(hOrig)
        );

        // 裁剪到原图范围内
        bboxOriginal &= cv::Rect(0, 0, originalSize.width, originalSize.height);

        // ── 3h. 构建检测结果 ────────────────────────────────────────────
        SegmentationResult segResult;
        segResult.bbox = bboxOriginal;
        segResult.classId = cand.classId;
        segResult.className = (cand.classId >= 0 &&
                               cand.classId < static_cast<int>(COCO_CLASSES.size()))
                                  ? COCO_CLASSES[cand.classId]
                                  : "unknown";
        segResult.confidence = cand.confidence;
        segResult.mask = maskOriginal;

        results.push_back(std::move(segResult));
    }

    // ── 按置信度降序排序 ────────────────────────────────────────
    std::sort(results.begin(), results.end());

    // ── 性能日志 ─────────────────────────────────────────────────
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
    // ── 计算缩放比例（保持宽高比） ────────────────────────────────────
    int origW = image.cols;
    int origH = image.rows;

    // 选择较小的缩放比例，确保整张图能放入 640x640 范围内
    scale = std::min(
        static_cast<float>(targetW) / origW,
        static_cast<float>(targetH) / origH
    );

    // 缩放后的尺寸
    int newW = static_cast<int>(origW * scale);
    int newH = static_cast<int>(origH * scale);

    // 计算填充偏移量（居中放置）
    dx = (targetW - newW) / 2;
    dy = (targetH - newH) / 2;

    // ── 步骤1：缩放图片 ──────────────────────────────────────────────────
    cv::Mat resized;
    cv::resize(image, resized, cv::Size(newW, newH), 0, 0, cv::INTER_LINEAR);

    // ── 步骤2：创建灰色画布并居中放置缩放后的图片 ──────────────────────
    // 灰色填充颜色 (114, 114, 114) — 这是 YOLOv8 训练时使用的填充色
    cv::Mat canvas(targetH, targetW, CV_8UC3,
                   cv::Scalar(114, 114, 114));

    // 定义缩放后图片在画布中的 ROI（感兴趣区域）
    cv::Rect roi(dx, dy, newW, newH);
    if (roi.width > 0 && roi.height > 0 &&
        roi.x >= 0 && roi.y >= 0 &&
        roi.x + roi.width <= targetW && roi.y + roi.height <= targetH) {
        resized.copyTo(canvas(roi));
    }

    return canvas;
}

// ============================================================================
// 边界框解码
// ============================================================================

cv::Rect YoloSegInference::decodeBbox(
    float cx, float cy, float w, float h,
    int gridX, int gridY, int stride,
    int imgW, int imgH)
{
    // YOLOv8 anchor-free 解码公式：
    //   center_x = (gridX + sigmoid(cx)) * stride
    //   center_y = (gridY + sigmoid(cy)) * stride
    //   width    = exp(w) * stride
    //   height   = exp(h) * stride
    //
    // 再转换为左上角坐标 (x1, y1) 和宽高：
    //   x = center_x - width / 2
    //   y = center_y - height / 2

    // Sigmoid 函数: 1 / (1 + exp(-x))
    auto sigmoid = [](float x) -> float {
        return 1.0f / (1.0f + std::exp(-x));
    };

    float centerX = (gridX + sigmoid(cx)) * stride;
    float centerY = (gridY + sigmoid(cy)) * stride;
    float boxW = std::exp(w) * stride;
    float boxH = std::exp(h) * stride;

    // 转换为左上角坐标
    int x = static_cast<int>(centerX - boxW / 2.0f);
    int y = static_cast<int>(centerY - boxH / 2.0f);
    int bw = static_cast<int>(boxW);
    int bh = static_cast<int>(boxH);

    // 裁剪到输入图片范围内
    x = std::max(0, std::min(x, imgW - 1));
    y = std::max(0, std::min(y, imgH - 1));
    bw = std::max(1, std::min(bw, imgW - x));
    bh = std::max(1, std::min(bh, imgH - y));

    return cv::Rect(x, y, bw, bh);
}

// ============================================================================
// IoU 计算
// ============================================================================

float YoloSegInference::computeIoU(const cv::Rect& a, const cv::Rect& b)
{
    // IoU = 交集面积 / 并集面积
    // 并集面积 = A面积 + B面积 - 交集面积

    // 计算交集矩形的左上角和右下角
    int interLeft   = std::max(a.x, b.x);
    int interTop    = std::max(a.y, b.y);
    int interRight  = std::min(a.x + a.width, b.x + b.width);
    int interBottom = std::min(a.y + a.height, b.y + b.height);

    if (interLeft >= interRight || interTop >= interBottom) {
        return 0.0f;  // 无交集
    }

    float interArea = static_cast<float>(
        (interRight - interLeft) * (interBottom - interTop)
    );
    float areaA = static_cast<float>(a.width * a.height);
    float areaB = static_cast<float>(b.width * b.height);
    float unionArea = areaA + areaB - interArea;

    return interArea / unionArea;
}

// ============================================================================
// 非极大值抑制 (NMS)
// ============================================================================

std::vector<int> YoloSegInference::nms(
    const std::vector<cv::Rect>& boxes,
    const std::vector<float>& scores,
    float iouThreshold)
{
    // ── 按分数降序排列 ──────────────────────────────────────────────────
    std::vector<int> indices(boxes.size());
    std::iota(indices.begin(), indices.end(), 0);  // 0, 1, 2, ...

    std::sort(indices.begin(), indices.end(),
              [&scores](int a, int b) {
                  return scores[a] > scores[b];  // 降序
              });

    // ── 贪心 NMS 算法 ──────────────────────────────────────────────────
    // 1. 从最高分开始，将该框加入保留列表
    // 2. 移除所有与其 IoU 超过阈值的框
    // 3. 重复直到没有候选框

    std::vector<int> keep;

    while (!indices.empty()) {
        int bestIdx = indices[0];  // 当前最高分
        keep.push_back(bestIdx);

        // 筛选出与 bestIdx 框 IoU 低于阈值的框
        std::vector<int> remaining;
        for (size_t i = 1; i < indices.size(); ++i) {
            if (computeIoU(boxes[bestIdx], boxes[indices[i]]) <= iouThreshold) {
                remaining.push_back(indices[i]);
            }
        }

        indices = std::move(remaining);
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
    // ── 为每个类别预定义颜色（BGR 格式） ─────────────────────────────
    // 使用鲜艳的颜色便于区分不同实例
    static const std::vector<cv::Scalar> COLORS = {
        cv::Scalar(56,  56,  255),   // 红色      (person)
        cv::Scalar(151, 157, 255),   // 浅红色
        cv::Scalar(31,  112, 255),   // 橙色
        cv::Scalar(29,  178, 255),   // 黄色
        cv::Scalar(49,  210, 207),   // 青绿色
        cv::Scalar(10,  249, 72),    // 绿色
        cv::Scalar(23,  204, 146),   // 浅绿
        cv::Scalar(134, 219, 61),    // 橄榄绿
        cv::Scalar(52,  147, 26),    // 深绿
        cv::Scalar(187, 212, 0),     // 黄绿
        cv::Scalar(168, 115, 22),    // 金黄色
        cv::Scalar(179, 115, 198),   // 浅紫
    };

    // ── 创建输出图像（拷贝原图） ────────────────────────────────────
    cv::Mat output = image.clone();

    for (size_t i = 0; i < detections.size(); ++i) {
        const auto& det = detections[i];
        cv::Scalar color = COLORS[det.classId % COLORS.size()];

        // ── 绘制半透明 Mask ───────────────────────────────────────────
        if (drawMask && !det.mask.empty()) {
            // 创建彩色 Mask 叠加层
            cv::Mat coloredMask(image.rows, image.cols, CV_8UC3, cv::Scalar(0, 0, 0));

            // 在彩色层上填充 Mask 颜色
            coloredMask.setTo(color, det.mask);

            // Alpha 混合: output = output * (1 - alpha) + coloredMask * alpha
            // alpha = 0.35 表示 35% 透明度的 Mask 叠加
            const float alpha = 0.35f;
            cv::addWeighted(output, 1.0f, coloredMask, alpha, 0.0, output);

            // 绘制 Mask 轮廓（白色细线，增强边界的可见性）
            std::vector<std::vector<cv::Point>> contours;
            cv::findContours(det.mask.clone(), contours,
                             cv::RETR_EXTERNAL, cv::CHAIN_APPROX_SIMPLE);
            cv::drawContours(output, contours, -1,
                             cv::Scalar(255, 255, 255), 2);
        }

        // ── 绘制边界框 ───────────────────────────────────────────────
        cv::rectangle(output, det.bbox, color, 2);

        // ── 绘制标签（类别名 + 置信度） ──────────────────────────────
        std::string label = det.className + " "
                           + std::to_string(static_cast<int>(det.confidence * 100))
                           + "%";

        // 计算标签文字尺寸
        int baseline = 0;
        cv::Size textSize = cv::getTextSize(
            label,
            cv::FONT_HERSHEY_SIMPLEX,
            0.5,     // 字体缩放
            1,       // 线宽
            &baseline
        );

        // 标签背景（实心矩形，位于检测框上方）
        cv::Point labelTopLeft(
            det.bbox.x,
            std::max(det.bbox.y - textSize.height - 8, 0)
        );

        cv::rectangle(
            output,
            labelTopLeft,
            cv::Point(labelTopLeft.x + textSize.width + 4,
                      labelTopLeft.y + textSize.height + 4),
            color,
            cv::FILLED
        );

        // 标签文字（白色）
        cv::putText(
            output,
            label,
            cv::Point(labelTopLeft.x + 2,
                      labelTopLeft.y + textSize.height + 1),
            cv::FONT_HERSHEY_SIMPLEX,
            0.5,
            cv::Scalar(255, 255, 255),
            1,
            cv::LINE_AA   // 抗锯齿
        );
    }

    return output;
}
