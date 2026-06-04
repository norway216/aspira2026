#pragma once

#include <opencv2/opencv.hpp>
#include <dlib/image_processing.h>
#include <dlib/image_io.h>
#include <dlib/opencv.h>

#include <string>
#include <vector>
#include <memory>
#include <atomic>
#include <chrono>
#include <mutex>
#include <condition_variable>
#include <thread>
#include <queue>
#include <functional>
#include <future>
#include <iostream>
#include <sstream>
#include <iomanip>
#include <filesystem>
#include <format>

namespace efr {

// ============================================================
// 基础数据结构
// ============================================================

/// 人脸检测结果矩形
struct FaceRect {
    cv::Rect rect;
    double confidence;          // 检测置信度 (dlib HOG 不提供，保留用于 CNN)
    int track_id = -1;          // 跟踪 ID（可选）

    FaceRect() : confidence(0.0) {}
    FaceRect(const cv::Rect& r, double c = 0.0) : rect(r), confidence(c) {}
};

/// 识别后的人脸信息
struct RecognizedFace {
    cv::Rect rect;
    std::string name;
    float distance;             // 欧氏距离，越小表示越匹配
    std::vector<float> embedding; // 128 维特征向量

    RecognizedFace() : distance(1.0f) {}
};

/// 单帧完整数据，在流水线各阶段间传递
struct FrameData {
    cv::Mat frame;                          // 原始/处理后图像
    int64_t timestamp_us;                   // 时间戳 (微秒)
    int frame_index;                        // 帧序号
    std::vector<FaceRect> faces;           // 检测到的人脸
    std::vector<RecognizedFace> recognized;// 识别结果

    FrameData() : timestamp_us(0), frame_index(0) {}

    // 深拷贝（图像数据独立）
    std::shared_ptr<FrameData> clone() const {
        auto copy = std::make_shared<FrameData>();
        copy->frame = frame.clone();
        copy->timestamp_us = timestamp_us;
        copy->frame_index = frame_index;
        copy->faces = faces;
        copy->recognized = recognized;
        return copy;
    }
};

using FramePtr = std::shared_ptr<FrameData>;

// ============================================================
// 系统配置
// ============================================================

enum class InputSource {
    Camera,
    VideoFile,
    ImageDirectory
};

enum class DetectorBackend {
    DlibHOG,        // dlib HOG 检测器（默认，无需外部模型）
    OpenCVHaar,     // OpenCV Haar Cascade（轻量备选）
    DlibCNN         // dlib CNN 检测器（需 GPU 加速模型）
};

struct SystemConfig {
    // ---- 输入源 ----
    InputSource input_source = InputSource::Camera;
    int camera_id = 0;
    std::string video_path;
    std::string image_dir;

    // ---- 检测参数 ----
    DetectorBackend detector_backend = DetectorBackend::DlibHOG;
    float hog_detection_threshold = 0.0f;   // dlib HOG: 越小越宽松 (-1.0 ~ 1.0)
    float haar_scale_factor = 1.1f;          // Haar: 缩放因子
    int haar_min_neighbors = 3;              // Haar: 最少邻接数
    cv::Size haar_min_size{30, 30};         // Haar: 最小检测尺寸

    // ---- 识别参数 ----
    bool enable_recognition = false;
    float recognition_threshold = 0.6f;      // 最大欧氏距离（小于此值认为匹配）
    std::string shape_predictor_path;        // 68 点 landmark 模型路径
    std::string face_recognition_model_path; // ResNet 128D 模型路径
    std::string known_faces_dir;             // 已知人脸图片目录

    // ---- 流水线 ----
    int thread_pool_size = 4;                // 处理线程数
    int ring_buffer_capacity = 8;            // RingBuffer 容量
    int target_fps = 30;                     // 目标帧率
    bool enable_display = true;              // 是否显示 GUI
    bool show_fps = true;                    // 显示 FPS
    bool verbose = false;                    // 详细日志

    // ---- 性能 ----
    bool enable_gpu = false;                 // GPU 加速（预留）

    // 工厂方法：从命令行参数构建
    static SystemConfig fromArgs(int argc, char* argv[]);
    static void printUsage(const char* prog);
};

// ============================================================
// 日志工具
// ============================================================

enum class LogLevel {
    Debug,
    Info,
    Warning,
    Error
};

class Logger {
public:
    static LogLevel min_level;
    static bool use_color;

    template<typename... Args>
    static void debug(const std::string& fmt, Args&&... args) {
        log(LogLevel::Debug, "[DEBUG] ", fmt, std::forward<Args>(args)...);
    }

    template<typename... Args>
    static void info(const std::string& fmt, Args&&... args) {
        log(LogLevel::Info, "[INFO]  ", fmt, std::forward<Args>(args)...);
    }

    template<typename... Args>
    static void warn(const std::string& fmt, Args&&... args) {
        log(LogLevel::Warning, "[WARN]  ", fmt, std::forward<Args>(args)...);
    }

    template<typename... Args>
    static void error(const std::string& fmt, Args&&... args) {
        log(LogLevel::Error, "[ERROR] ", fmt, std::forward<Args>(args)...);
    }

private:
    template<typename... Args>
    static void log(LogLevel level, const char* prefix, const std::string& fmt, Args&&... args) {
        if (level < min_level) return;
        try {
            std::string msg = std::vformat(fmt, std::make_format_args(args...));
            std::cerr << prefix << msg << std::endl;
        } catch (const std::exception& e) {
            std::cerr << prefix << fmt << " [format error: " << e.what() << "]" << std::endl;
        }
    }
};

// ============================================================
// FPS 计数器
// ============================================================

class FPSCounter {
public:
    void tick() {
        std::lock_guard<std::mutex> lock(mtx_);
        frame_count_++;
        auto now = std::chrono::steady_clock::now();
        auto elapsed = std::chrono::duration<double>(now - last_time_).count();
        if (elapsed >= 1.0) {
            fps_ = static_cast<int>(frame_count_ / elapsed);
            frame_count_ = 0;
            last_time_ = now;
        }
    }

    int fps() const { return fps_; }

private:
    mutable std::mutex mtx_;
    std::chrono::steady_clock::time_point last_time_{std::chrono::steady_clock::now()};
    int frame_count_ = 0;
    int fps_ = 0;
};

} // namespace efr
