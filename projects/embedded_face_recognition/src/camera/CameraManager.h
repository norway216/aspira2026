#pragma once

#include "Common.h"
#include <opencv2/videoio.hpp>
#include <string>
#include <atomic>
#include <memory>

namespace efr {

/**
 * 摄像头与视频文件管理器
 *
 * 职责：
 * - 检测并打开摄像头设备
 * - 若摄像头不可用，退回视频文件读取
 * - 提供统一帧捕获接口
 * - 帧率控制与统计
 */
class CameraManager {
public:
    CameraManager();
    ~CameraManager();

    CameraManager(const CameraManager&) = delete;
    CameraManager& operator=(const CameraManager&) = delete;

    /**
     * 初始化捕获设备
     * @param config  系统配置（输入源、摄像头 ID、视频路径）
     * @return 是否成功打开
     */
    bool initialize(const SystemConfig& config);

    /**
     * 捕获一帧
     * @param frame  输出帧数据（包含图像和时间戳）
     * @return 是否成功捕获（视频结束返回 false）
     */
    bool captureFrame(FramePtr& frame);

    /// 释放资源
    void release();

    /// 是否已初始化
    bool isOpened() const { return opened_; }

    /// 获取视频属性
    double getFPS() const;
    int getWidth() const;
    int getHeight() const;
    int getTotalFrames() const;

    /// 当前帧序号
    int currentFrameIndex() const { return frame_index_.load(); }

private:
    /// 初始化视频文件
    bool initializeVideoFile(const std::string& path);

    cv::VideoCapture capture_;
    std::atomic<bool> opened_{false};
    std::atomic<int> frame_index_{0};
    InputSource source_type_;
};

} // namespace efr
