/**
 * @file    yolo_seg_inference.h
 * @brief   YOLOv8 分割模型 ONNX Runtime C++ 推理类（多线程优化版）
 * @author  yolo_cpp project
 * @date    2026-06-01
 *
 * ============================================================================
 * 优化记录 (v3.0)
 * ============================================================================
 * 多线程优化：
 *   - 新增轻量级 ThreadPool，基于 std::thread 实现，无需外部依赖
 *   - Anchor 扫描并行化：8400 个 anchor 按线程数分块并行处理
 *   - Mask 生成并行化：每个检测结果的 mask 独立并行生成
 *   - 每线程独立 mask 工作缓冲区，避免锁竞争
 *
 * 历史优化 (v2.0)：
 *   - 精度修复：Mask 系数添加 sigmoid 激活
 *   - 速度优化：缓存 ONNX Runtime name pointers 和 MemoryInfo
 *   - 速度优化：启用 inter-op 并行 + 内存模式优化
 *   - 速度优化：预分配 mask/postprocess 缓冲区
 *   - 速度优化：Anchor 扫描循环优化（class score stride 预计算）
 *   - 速度优化：NMS 原地筛选，减少 vector 分配
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
#include <thread>
#include <mutex>
#include <condition_variable>
#include <queue>
#include <functional>
#include <atomic>

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
    int numClasses   = 80;          ///< 类别数 (COCO=80, PH2/ISIC=1)
    int maskChannels = 32;
    int maskProtoH   = 160;
    int maskProtoW   = 160;

    float confThreshold = 0.25f;
    float iouThreshold  = 0.45f;
    float maskThreshold = 0.5f;

    int  numThreads     = 4;        ///< ONNX intra-op 线程数 + 后处理线程池大小
    int  interOpThreads = 2;        ///< inter-op 并行线程数
    bool enableProfiling = false;
    bool enableMemoryPattern = true;  ///< 启用内存模式优化

    /// 自定义类别名称（为空则使用内置 COCO 80 类）
    std::vector<std::string> classNames;
};

// ============================================================================
// 轻量级线程池
// ============================================================================

/**
 * @brief 固定大小线程池，用于后处理并行化
 *
 * 设计要点：
 *   - 使用 std::thread + 任务队列，无需外部依赖
 *   - 在 YoloSegInference::initialize() 时创建，析构时销毁
 *   - parallelFor(begin, end, fn) 将范围分块后分发到工作线程
 *   - 线程数由 InferenceConfig::numThreads 控制
 */
class ThreadPool {
public:
    /**
     * @brief 构造并启动 numThreads 个工作线程
     * @param numThreads 工作线程数（若为 0 则回退到 1）
     */
    explicit ThreadPool(size_t numThreads);

    ~ThreadPool();

    // 禁止拷贝和移动
    ThreadPool(const ThreadPool&) = delete;
    ThreadPool& operator=(const ThreadPool&) = delete;
    ThreadPool(ThreadPool&&) = delete;
    ThreadPool& operator=(ThreadPool&&) = delete;

    /**
     * @brief 并行执行 for 循环
     *
     * 将 [begin, end) 范围内的整数按线程数分块，每个线程处理一个分块。
     * 调用线程（主线程）也参与计算，不会闲置。
     *
     * @tparam Func 可调用对象，签名为 void(int index)
     * @param begin 起始索引（包含）
     * @param end   结束索引（不包含）
     * @param func  对每个索引调用的函数
     */
    template<typename Func>
    void parallelFor(int begin, int end, Func&& func);

    /** @brief 返回工作线程数 */
    size_t size() const { return m_workers.size(); }

private:
    struct Task {
        int begin;
        int end;
        std::function<void(int)> func;
        std::atomic<int>* counter;   ///< 完成计数
        std::atomic<bool>* hasError; ///< 错误标记（共享）
    };

    void workerLoop();

    std::vector<std::thread> m_workers;
    std::queue<Task> m_taskQueue;
    std::mutex m_queueMutex;
    std::condition_variable m_cv;
    std::condition_variable m_doneCv;
    bool m_stop = false;
};

// ============================================================================
// YoloSegInference 类（多线程优化版）
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

    // ── 后处理子步骤（并行化） ─────────────────────────────────────────
    /// 并行 Anchor 扫描：返回所有通过置信度阈值的候选检测
    std::vector<struct RawDetection> scanAnchors(
        const float* output0Tensor,
        int numAnchors, int stride,
        int bboxOffset, int classOffset, int maskOffset
    );

    /// 并行 Mask 生成：为每个保留的检测生成实例分割 mask
    void generateMasks(
        const std::vector<int>& keepIndices,
        const std::vector<struct RawDetection>& candidates,
        const float* output1Tensor,
        const cv::Size& originalSize,
        std::vector<SegmentationResult>& results
    );

    /// 为单个检测生成 mask（由工作线程调用）
    SegmentationResult generateSingleMask(
        const RawDetection& cand,
        const float* output1Tensor,
        const cv::Size& originalSize,
        int threadIdx
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

    // 线程池（后处理并行化）
    std::unique_ptr<ThreadPool> m_threadPool;

    // 每线程独立 mask 工作缓冲区（避免锁竞争）
    // 每个缓冲区大小为 maskProtoH * maskProtoW = 25600 floats ≈ 100KB
    std::vector<std::vector<float>> m_maskWorkBuffers;

    // Letterbox 变换参数
    float m_letterboxScale = 1.0f;
    int m_letterboxDx = 0;
    int m_letterboxDy = 0;

    // COCO 类别名称
    static const std::vector<std::string> COCO_CLASSES;
};

// ============================================================================
// ThreadPool::parallelFor 模板实现（必须在头文件中）
// ============================================================================

template<typename Func>
void ThreadPool::parallelFor(int begin, int end, Func&& func)
{
    const int total = end - begin;
    if (total <= 0) return;

    const int numWorkers = static_cast<int>(m_workers.size());
    const int chunkSize = (total + numWorkers - 1) / numWorkers;

    std::atomic<int> counter(0);
    std::atomic<bool> hasError(false);

    {
        std::lock_guard<std::mutex> lock(m_queueMutex);

        for (int t = 0; t < numWorkers; ++t) {
            int chunkBegin = begin + t * chunkSize;
            int chunkEnd = std::min(chunkBegin + chunkSize, end);
            if (chunkBegin >= chunkEnd) break;

            m_taskQueue.push({
                chunkBegin,
                chunkEnd,
                std::forward<Func>(func),
                &counter,
                &hasError
            });
        }

        // 更新 counter 为任务总数（原子写入）
        int numTasks = static_cast<int>(m_taskQueue.size());
        counter.store(numTasks, std::memory_order_release);
    }

    m_cv.notify_all();

    // 主线程也参与计算：从队列中取任务执行
    while (true) {
        Task task;
        {
            std::unique_lock<std::mutex> lock(m_queueMutex);
            if (m_taskQueue.empty()) break;
            task = std::move(m_taskQueue.front());
            m_taskQueue.pop();
        }
        for (int i = task.begin; i < task.end; ++i) {
            task.func(i);
        }
        counter.fetch_sub(1, std::memory_order_acq_rel);
    }

    // 等待所有工作线程完成任务
    {
        std::unique_lock<std::mutex> lock(m_queueMutex);
        m_doneCv.wait(lock, [&counter] {
            return counter.load(std::memory_order_acquire) == 0;
        });
    }
}
