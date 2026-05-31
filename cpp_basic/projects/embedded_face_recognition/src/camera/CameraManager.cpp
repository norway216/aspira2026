#include "CameraManager.h"
#include <chrono>

namespace efr {

CameraManager::CameraManager() {}

CameraManager::~CameraManager() {
    release();
}

bool CameraManager::initialize(const SystemConfig& config) {
    release();
    source_type_ = config.input_source;

    if (config.input_source == InputSource::Camera) {
        // 优先尝试打开摄像头
        int api = cv::CAP_ANY; // Linux: V4L2
        if (!capture_.open(config.camera_id, api)) {
            Logger::warn("无法打开摄像头 /dev/video{}, 尝试 V4L2 后端...", config.camera_id);
            if (!capture_.open(config.camera_id, cv::CAP_V4L2)) {
                Logger::warn("摄像头不可用, 退回视频文件模式");
                // 退回到视频文件
                if (!config.video_path.empty()) {
                    return initializeVideoFile(config.video_path);
                }
                return false;
            }
        }

        // 设置摄像头参数
        capture_.set(cv::CAP_PROP_FRAME_WIDTH, 640);
        capture_.set(cv::CAP_PROP_FRAME_HEIGHT, 480);
        capture_.set(cv::CAP_PROP_FPS, config.target_fps);
        capture_.set(cv::CAP_PROP_BUFFERSIZE, 3); // 减小缓冲延迟

        Logger::info("摄像头已打开 ({}x{} @ {}fps)",
                     (int)capture_.get(cv::CAP_PROP_FRAME_WIDTH),
                     (int)capture_.get(cv::CAP_PROP_FRAME_HEIGHT),
                     capture_.get(cv::CAP_PROP_FPS));

    } else if (config.input_source == InputSource::VideoFile) {
        return initializeVideoFile(config.video_path);
    }

    opened_ = capture_.isOpened();
    return opened_;
}

bool CameraManager::initializeVideoFile(const std::string& path) {
    if (path.empty()) {
        Logger::error("未指定视频文件路径");
        return false;
    }

    if (!capture_.open(path)) {
        Logger::error("无法打开视频文件: {}", path);
        return false;
    }

    Logger::info("视频文件已打开: {} ({}x{}, {:.1f}fps, {} 帧)",
                 path,
                 (int)capture_.get(cv::CAP_PROP_FRAME_WIDTH),
                 (int)capture_.get(cv::CAP_PROP_FRAME_HEIGHT),
                 capture_.get(cv::CAP_PROP_FPS),
                 (int)capture_.get(cv::CAP_PROP_FRAME_COUNT));

    source_type_ = InputSource::VideoFile;
    opened_ = true;
    return true;
}

bool CameraManager::captureFrame(FramePtr& frame) {
    if (!opened_) return false;

    cv::Mat img;
    if (!capture_.read(img)) {
        return false; // 视频结束或设备断连
    }

    if (img.empty()) {
        return false;
    }

    frame = std::make_shared<FrameData>();
    frame->frame = img;
    frame->frame_index = frame_index_.fetch_add(1) + 1;
    frame->timestamp_us = std::chrono::duration_cast<std::chrono::microseconds>(
        std::chrono::steady_clock::now().time_since_epoch()
    ).count();

    return true;
}

void CameraManager::release() {
    if (capture_.isOpened()) {
        capture_.release();
    }
    opened_ = false;
    frame_index_ = 0;
}

double CameraManager::getFPS() const {
    if (!opened_) return 0.0;
    double fps = capture_.get(cv::CAP_PROP_FPS);
    return fps > 0 ? fps : 30.0; // 默认 30fps
}

int CameraManager::getWidth() const {
    return opened_ ? (int)capture_.get(cv::CAP_PROP_FRAME_WIDTH) : 0;
}

int CameraManager::getHeight() const {
    return opened_ ? (int)capture_.get(cv::CAP_PROP_FRAME_HEIGHT) : 0;
}

int CameraManager::getTotalFrames() const {
    return opened_ ? (int)capture_.get(cv::CAP_PROP_FRAME_COUNT) : 0;
}

} // namespace efr
