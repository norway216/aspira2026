#include "DisplayManager.h"
#include <opencv2/imgproc.hpp>

namespace efr {

DisplayManager::DisplayManager() {}

DisplayManager::~DisplayManager() {
    close();
}

bool DisplayManager::initialize(const SystemConfig& config) {
    if (!config.enable_display) return true;

    window_name_ = "Embedded Face Recognition System";
    cv::namedWindow(window_name_, cv::WINDOW_NORMAL | cv::WINDOW_GUI_EXPANDED);
    cv::resizeWindow(window_name_, 960, 640);

    // 创建参数调节 Trackbar
    cv::createTrackbar("Detect Thresh (x10)", window_name_, nullptr, 20,
                       onDetectionThreshold, this);
    cv::createTrackbar("Recog Thresh (x100)", window_name_, nullptr, 100,
                       onRecognitionThreshold, this);

    // 设置默认值
    cv::setTrackbarPos("Detect Thresh (x10)", window_name_,
                       static_cast<int>(detection_threshold_.load() * 10 + 10));
    cv::setTrackbarPos("Recog Thresh (x100)", window_name_,
                       static_cast<int>(recognition_threshold_.load() * 100));

    Logger::info("显示窗口已创建: {}", window_name_);
    return true;
}

void DisplayManager::render(const FrameData& frame_data) {
    if (window_name_.empty()) return;

    cv::Mat canvas;
    if (frame_data.frame.channels() == 1) {
        cv::cvtColor(frame_data.frame, canvas, cv::COLOR_GRAY2BGR);
    } else {
        canvas = frame_data.frame.clone();
    }

    // 绘制各层叠加信息
    drawDetections(canvas, frame_data);
    drawRecognitions(canvas, frame_data);
    drawHUD(canvas, frame_data);

    // 显示
    display_buffer_ = canvas;
    cv::imshow(window_name_, display_buffer_);

    fps_counter_.tick();
}

bool DisplayManager::shouldQuit() const {
    if (window_name_.empty()) return false;
    int key = cv::waitKey(1);
    return (key == 27 || key == 'q' || key == 'Q'); // ESC 或 q 退出
}

void DisplayManager::close() {
    if (!window_name_.empty()) {
        cv::destroyWindow(window_name_);
        window_name_.clear();
    }
}

int DisplayManager::currentFPS() const {
    return fps_counter_.fps();
}

void DisplayManager::drawDetections(cv::Mat& canvas, const FrameData& frame_data) {
    for (const auto& face : frame_data.faces) {
        // 绘制矩形框
        cv::rectangle(canvas, face.rect, COLOR_DETECT, 2);

        // 绘制角标增强可视效果
        int x = face.rect.x, y = face.rect.y;
        int w = face.rect.width, h = face.rect.height;
        int corner_len = std::max(10, w / 6);

        cv::line(canvas, cv::Point(x, y), cv::Point(x + corner_len, y), COLOR_DETECT, 3);
        cv::line(canvas, cv::Point(x, y), cv::Point(x, y + corner_len), COLOR_DETECT, 3);
        cv::line(canvas, cv::Point(x + w, y), cv::Point(x + w - corner_len, y), COLOR_DETECT, 3);
        cv::line(canvas, cv::Point(x + w, y), cv::Point(x + w, y + corner_len), COLOR_DETECT, 3);
        cv::line(canvas, cv::Point(x, y + h), cv::Point(x + corner_len, y + h), COLOR_DETECT, 3);
        cv::line(canvas, cv::Point(x, y + h), cv::Point(x, y + h - corner_len), COLOR_DETECT, 3);
        cv::line(canvas, cv::Point(x + w, y + h), cv::Point(x + w - corner_len, y + h), COLOR_DETECT, 3);
        cv::line(canvas, cv::Point(x + w, y + h), cv::Point(x + w, y + h - corner_len), COLOR_DETECT, 3);
    }
}

void DisplayManager::drawRecognitions(cv::Mat& canvas, const FrameData& frame_data) {
    for (const auto& face : frame_data.recognized) {
        cv::Scalar color = (face.name == "Unknown") ? COLOR_UNKNOWN : COLOR_RECOGNIZE;

        // 标签背景
        std::string label = face.name;
        if (!face.name.empty() && face.name != "Unknown") {
            std::ostringstream oss;
            oss << face.name << " (" << std::fixed << std::setprecision(2) << face.distance << ")";
            label = oss.str();
        }

        int baseline = 0;
        cv::Size text_size = cv::getTextSize(label, cv::FONT_HERSHEY_SIMPLEX, 0.5, 1, &baseline);

        int label_y = face.rect.y - 10;
        if (label_y < text_size.height + 5) {
            label_y = face.rect.y + face.rect.height + text_size.height + 5;
        }

        cv::Rect bg_rect(face.rect.x, label_y - text_size.height - 4,
                         text_size.width + 8, text_size.height + 6);
        cv::rectangle(canvas, bg_rect, color, -1); // 填充背景

        cv::putText(canvas, label,
                    cv::Point(face.rect.x + 4, label_y - 2),
                    cv::FONT_HERSHEY_SIMPLEX, 0.5, COLOR_TEXT, 1);
    }
}

void DisplayManager::drawHUD(cv::Mat& canvas, const FrameData& frame_data) {
    int y_offset = 25;
    const int line_height = 20;
    const int x_margin = 10;

    // 半透明 HUD 背景
    cv::Rect hud_bg(0, 0, 280, 120);
    cv::Mat roi = canvas(hud_bg & cv::Rect(0, 0, canvas.cols, canvas.rows));
    cv::Mat overlay;
    cv::addWeighted(roi, 0.3, roi, 0.0, 0, overlay);
    overlay.copyTo(roi);

    auto drawLine = [&](const std::string& text, const cv::Scalar& color = COLOR_TEXT) {
        cv::putText(canvas, text, cv::Point(x_margin, y_offset),
                    cv::FONT_HERSHEY_SIMPLEX, 0.5, color, 1);
        y_offset += line_height;
    };

    // 系统名称
    cv::putText(canvas, "Face Recognition System v1.0",
                cv::Point(x_margin, 18), cv::FONT_HERSHEY_SIMPLEX, 0.5,
                cv::Scalar(0, 255, 255), 1);

    y_offset = 42;

    // FPS
    {
        std::ostringstream oss;
        oss << "Display FPS: " << currentFPS()
            << " | Process FPS: " << processing_fps_;
        drawLine(oss.str(), cv::Scalar(0, 255, 0));
    }

    // 帧信息
    {
        std::ostringstream oss;
        oss << "Frame: #" << frame_data.frame_index
            << " | Faces: " << frame_data.faces.size()
            << " | Known: " << frame_data.recognized.size();
        drawLine(oss.str());
    }

    // 检测/识别状态
    {
        std::ostringstream oss;
        oss << "Detect: " << frame_data.faces.size()
            << " | Recognized: ";
        int known_count = 0;
        for (const auto& r : frame_data.recognized) {
            if (r.name != "Unknown" && !r.name.empty()) known_count++;
        }
        oss << known_count;
        drawLine(oss.str());
    }
}

// ---- Trackbar 回调 ----

void DisplayManager::onDetectionThreshold(int val, void* userdata) {
    auto* self = static_cast<DisplayManager*>(userdata);
    // val: 0~20, 映射到 -1.0 ~ 1.0
    self->detection_threshold_ = (val - 10) / 10.0f;
}

void DisplayManager::onRecognitionThreshold(int val, void* userdata) {
    auto* self = static_cast<DisplayManager*>(userdata);
    // val: 0~100, 映射到 0.0 ~ 1.0
    self->recognition_threshold_ = val / 100.0f;
}

} // namespace efr
