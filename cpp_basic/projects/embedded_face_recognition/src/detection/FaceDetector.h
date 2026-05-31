#pragma once

#include "Common.h"
#include <dlib/image_processing/frontal_face_detector.h>
#include <dlib/opencv/cv_image.h>
#include <opencv2/objdetect.hpp>

namespace efr {

/**
 * 人脸检测管理器
 *
 * 职责：
 * - 封装 dlib HOG / CNN 检测器 和 OpenCV Haar Cascade
 * - 提供统一检测接口
 * - 支持检测置信度阈值调节
 */
class FaceDetector {
public:
    FaceDetector();
    ~FaceDetector() = default;

    FaceDetector(const FaceDetector&) = delete;
    FaceDetector& operator=(const FaceDetector&) = delete;

    /**
     * 初始化检测器
     * @param config  系统配置
     * @return 是否初始化成功
     */
    bool initialize(const SystemConfig& config);

    /**
     * 对单帧执行人脸检测
     * @param frame_data  输入帧数据（检测结果回填到 frame_data->faces）
     * @return 检测到的人脸数
     */
    int detect(FrameData& frame_data);

    /// 更新检测阈值
    void setDetectionThreshold(float threshold);

    /// 切换检测后端
    bool switchBackend(DetectorBackend backend);

    /// 当前使用的后端
    DetectorBackend currentBackend() const { return backend_; }

private:
    /// 初始化 Haar Cascade
    bool initHaarCascade(const SystemConfig& config);

    // dlib HOG 检测
    int detectHOG(FrameData& frame_data);

    // OpenCV Haar Cascade 检测
    int detectHaar(FrameData& frame_data);

    DetectorBackend backend_ = DetectorBackend::DlibHOG;

    // dlib 检测器
    dlib::frontal_face_detector hog_detector_;

    // OpenCV Haar Cascade
    cv::CascadeClassifier haar_cascade_;
    float haar_scale_factor_ = 1.1f;
    int haar_min_neighbors_ = 3;
    cv::Size haar_min_size_{30, 30};

    // 线程安全锁
    mutable std::mutex detect_mutex_;
};

} // namespace efr
