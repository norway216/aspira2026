#pragma once

#include "Common.h"
#include <opencv2/highgui.hpp>
#include <string>
#include <functional>
#include <atomic>

namespace efr {

/// 用于 UI 参数调整的回调
using TrackbarCallback = std::function<void(int, void*)>;

/**
 * 显示管理器
 *
 * 职责：
 * - 使用 OpenCV highgui 创建显示窗口
 * - 在图像上叠加人脸检测框和识别标签
 * - 提供 Trackbar 用于参数实时调节
 * - 显示 FPS 和性能统计
 */
class DisplayManager {
public:
    DisplayManager();
    ~DisplayManager();

    DisplayManager(const DisplayManager&) = delete;
    DisplayManager& operator=(const DisplayManager&) = delete;

    /// 初始化显示窗口
    bool initialize(const SystemConfig& config);

    /**
     * 渲染帧到窗口
     * @param frame_data  待显示的帧数据
     */
    void render(const FrameData& frame_data);

    /// 检查是否应该退出（ESC 按下 / 窗口关闭）
    bool shouldQuit() const;

    /// 关闭窗口
    void close();

    /// FPS 信息
    int currentFPS() const;
    int processingFPS() const { return processing_fps_; }
    void setProcessingFPS(int fps) { processing_fps_ = fps; }

    /// 获取 UI 调整的参数
    float getDetectionThreshold() const { return detection_threshold_; }
    float getRecognitionThreshold() const { return recognition_threshold_; }

private:
    /// 绘制检测框
    void drawDetections(cv::Mat& canvas, const FrameData& frame_data);

    /// 绘制识别结果
    void drawRecognitions(cv::Mat& canvas, const FrameData& frame_data);

    /// 绘制 FPS 和状态信息
    void drawHUD(cv::Mat& canvas, const FrameData& frame_data);

    std::string window_name_;
    cv::Mat display_buffer_;
    FPSCounter fps_counter_;

    // UI 参数
    std::atomic<float> detection_threshold_{0.0f};
    std::atomic<float> recognition_threshold_{0.6f};
    std::atomic<int> processing_fps_{0};

    // 绘制颜色（cv::Scalar 非 literal 类型，不能用 constexpr）
    static inline const cv::Scalar COLOR_DETECT{0, 255, 0};      // 绿色检测框
    static inline const cv::Scalar COLOR_RECOGNIZE{255, 0, 0};   // 蓝色识别框
    static inline const cv::Scalar COLOR_TEXT{255, 255, 255};    // 白色文字
    static inline const cv::Scalar COLOR_UNKNOWN{0, 165, 255};   // 橙色未知
    static inline const cv::Scalar COLOR_HUD_BG{0, 0, 0};        // 黑色 HUD 背景

    // Trackbar 回调
    static void onDetectionThreshold(int val, void* userdata);
    static void onRecognitionThreshold(int val, void* userdata);
};

} // namespace efr
