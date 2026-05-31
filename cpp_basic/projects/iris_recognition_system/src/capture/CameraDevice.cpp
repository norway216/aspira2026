#include "capture/CameraDevice.h"
#include <iostream>
#include <chrono>

namespace iris {

CameraDevice::CameraDevice() = default;

CameraDevice::~CameraDevice() {
    stopCapture();
    if (m_cap.isOpened()) {
        m_cap.release();
    }
}

bool CameraDevice::open(int deviceId, int width, int height, int fps) {
    if (!m_cap.open(deviceId, cv::CAP_V4L2)) {
        // Fallback: try any backend
        if (!m_cap.open(deviceId)) {
            std::cerr << "[CameraDevice] Failed to open camera device " << deviceId << "\n";
            return false;
        }
    }

    m_cap.set(cv::CAP_PROP_FRAME_WIDTH, width);
    m_cap.set(cv::CAP_PROP_FRAME_HEIGHT, height);
    m_cap.set(cv::CAP_PROP_FPS, fps);

    m_width  = static_cast<int>(m_cap.get(cv::CAP_PROP_FRAME_WIDTH));
    m_height = static_cast<int>(m_cap.get(cv::CAP_PROP_FRAME_HEIGHT));

    m_opened = true;
    std::cout << "[CameraDevice] Opened camera " << deviceId
              << " (" << m_width << "x" << m_height << ")\n";
    return true;
}

bool CameraDevice::openFile(const std::string& path) {
    if (!m_cap.open(path)) {
        std::cerr << "[CameraDevice] Failed to open video file: " << path << "\n";
        return false;
    }

    m_width  = static_cast<int>(m_cap.get(cv::CAP_PROP_FRAME_WIDTH));
    m_height = static_cast<int>(m_cap.get(cv::CAP_PROP_FRAME_HEIGHT));
    m_opened = true;
    std::cout << "[CameraDevice] Opened video: " << path << "\n";
    return true;
}

void CameraDevice::startCapture() {
    if (m_running) return;
    m_running = true;
    m_thread = std::thread(&CameraDevice::captureLoop, this);
}

void CameraDevice::stopCapture() {
    m_running = false;
    if (m_thread.joinable()) {
        m_thread.join();
    }
}

bool CameraDevice::getFrame(cv::Mat& frame) {
    std::lock_guard<std::mutex> lock(m_frameMutex);
    if (m_latestFrame.empty()) return false;
    frame = m_latestFrame.clone();
    return true;
}

bool CameraDevice::isOpened() const {
    return m_opened;
}

void CameraDevice::captureLoop() {
    cv::Mat frame;
    using namespace std::chrono_literals;

    while (m_running) {
        if (!m_cap.read(frame) || frame.empty()) {
            // For video files, loop or stop
            if (m_cap.get(cv::CAP_PROP_FRAME_COUNT) > 0) {
                // It's a video file that ended
                break;
            }
            std::this_thread::sleep_for(10ms);
            continue;
        }

        {
            std::lock_guard<std::mutex> lock(m_frameMutex);
            m_latestFrame = frame.clone();
        }
    }

    if (!m_opened) {
        // File playback ended, signal that we're done
    }
}

} // namespace iris
