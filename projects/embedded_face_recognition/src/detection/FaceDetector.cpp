#include "FaceDetector.h"
#include <opencv2/imgproc.hpp>

namespace efr {

FaceDetector::FaceDetector() {
    // HOG 检测器是静态工厂方法，不需要在这里初始化
}

bool FaceDetector::initialize(const SystemConfig& config) {
    backend_ = config.detector_backend;

    switch (backend_) {
    case DetectorBackend::DlibHOG:
        try {
            hog_detector_ = dlib::get_frontal_face_detector();
            Logger::info("dlib HOG 人脸检测器已初始化");
        } catch (const std::exception& e) {
            Logger::error("dlib HOG 初始化失败: {}", e.what());
            return false;
        }
        break;

    case DetectorBackend::OpenCVHaar:
        return initHaarCascade(config);

    case DetectorBackend::DlibCNN:
        Logger::warn("dlib CNN 检测器需要额外模型文件，退回 HOG 检测器");
        hog_detector_ = dlib::get_frontal_face_detector();
        backend_ = DetectorBackend::DlibHOG;
        break;
    }

    return true;
}

bool FaceDetector::initHaarCascade(const SystemConfig& config) {
    // 尝试多个可能的路径
    std::vector<std::string> cascade_paths = {
        "/usr/share/opencv4/haarcascades/haarcascade_frontalface_default.xml",
        "/usr/share/opencv4/haarcascades/haarcascade_frontalface_alt2.xml",
        "/usr/share/opencv/haarcascades/haarcascade_frontalface_default.xml",
        "haarcascade_frontalface_default.xml"
    };

    for (const auto& path : cascade_paths) {
        if (haar_cascade_.load(path)) {
            Logger::info("OpenCV Haar Cascade 已加载: {}", path);
            haar_scale_factor_ = config.haar_scale_factor;
            haar_min_neighbors_ = config.haar_min_neighbors;
            haar_min_size_ = config.haar_min_size;
            return true;
        }
    }

    Logger::error("无法加载 Haar Cascade 模型文件");
    return false;
}

int FaceDetector::detect(FrameData& frame_data) {
    std::lock_guard<std::mutex> lock(detect_mutex_);
    switch (backend_) {
    case DetectorBackend::DlibHOG:
        return detectHOG(frame_data);
    case DetectorBackend::OpenCVHaar:
        return detectHaar(frame_data);
    default:
        return detectHOG(frame_data);
    }
}

int FaceDetector::detectHOG(FrameData& frame_data) {
    if (frame_data.frame.empty()) return 0;

    cv::Mat& img = frame_data.frame;

    // 转换为灰度图（dlib HOG 在灰度上运行）
    cv::Mat gray;
    if (img.channels() == 3) {
        cv::cvtColor(img, gray, cv::COLOR_BGR2GRAY);
    } else if (img.channels() == 4) {
        cv::cvtColor(img, gray, cv::COLOR_BGRA2GRAY);
    } else {
        gray = img;
    }

    // 包装为 dlib 图像（不拷贝数据）
    dlib::cv_image<unsigned char> dlib_img(gray);

    // 检测人脸
    std::vector<dlib::rectangle> dets;
    try {
        dets = hog_detector_(dlib_img, 0); // threshold=0 表示默认
    } catch (const std::exception& e) {
        Logger::error("HOG 检测异常: {}", e.what());
        return 0;
    }

    // 转换为内部格式
    frame_data.faces.clear();
    frame_data.faces.reserve(dets.size());
    for (const auto& d : dets) {
        cv::Rect r(d.left(), d.top(), d.width(), d.height());
        // 裁剪到图像范围
        r &= cv::Rect(0, 0, img.cols, img.rows);
        if (r.area() > 0) {
            frame_data.faces.emplace_back(r, 1.0);
        }
    }

    return static_cast<int>(frame_data.faces.size());
}

int FaceDetector::detectHaar(FrameData& frame_data) {
    if (frame_data.frame.empty()) return 0;

    cv::Mat& img = frame_data.frame;

    // 转灰度
    cv::Mat gray;
    if (img.channels() >= 3) {
        cv::cvtColor(img, gray, cv::COLOR_BGR2GRAY);
    } else {
        gray = img;
    }

    // 直方图均衡化，提高检测鲁棒性
    cv::equalizeHist(gray, gray);

    std::vector<cv::Rect> faces;
    haar_cascade_.detectMultiScale(gray, faces,
                                   haar_scale_factor_,
                                   haar_min_neighbors_,
                                   0, // flags
                                   haar_min_size_);

    frame_data.faces.clear();
    frame_data.faces.reserve(faces.size());
    for (const auto& r : faces) {
        frame_data.faces.emplace_back(r, 1.0);
    }

    return static_cast<int>(frame_data.faces.size());
}

void FaceDetector::setDetectionThreshold(float threshold) {
    // dlib HOG 阈值在 detect() 调用时传入
}

bool FaceDetector::switchBackend(DetectorBackend backend) {
    if (backend == backend_) return true;

    if (backend == DetectorBackend::DlibHOG) {
        try {
            hog_detector_ = dlib::get_frontal_face_detector();
            backend_ = backend;
            Logger::info("已切换到 dlib HOG 检测器");
            return true;
        } catch (...) {
            return false;
        }
    }

    if (backend == DetectorBackend::OpenCVHaar) {
        SystemConfig cfg;
        if (initHaarCascade(cfg)) {
            backend_ = backend;
            Logger::info("已切换到 OpenCV Haar Cascade 检测器");
            return true;
        }
    }

    return false;
}

} // namespace efr
