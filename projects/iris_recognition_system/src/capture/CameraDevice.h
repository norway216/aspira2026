#pragma once

#include <opencv2/core.hpp>
#include <opencv2/videoio.hpp>
#include <atomic>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

namespace iris {

/// Thread-safe camera wrapper with double-buffering
class CameraDevice {
public:
    CameraDevice();
    ~CameraDevice();

    CameraDevice(const CameraDevice&)            = delete;
    CameraDevice& operator=(const CameraDevice&) = delete;

    /// Open a camera by device index
    bool open(int deviceId, int width = 640, int height = 480, int fps = 30);

    /// Open a video file
    bool openFile(const std::string& path);

    /// Start continuous capture in a background thread
    void startCapture();

    /// Stop background capture
    void stopCapture();

    /// Get the latest frame (thread-safe)
    bool getFrame(cv::Mat& frame);

    /// Check if camera is opened
    bool isOpened() const;

    /// Get camera properties
    int getWidth() const  { return m_width; }
    int getHeight() const { return m_height; }

private:
    void captureLoop();

    cv::VideoCapture m_cap;
    std::atomic<bool> m_running{false};
    std::atomic<bool> m_opened{false};
    std::thread m_thread;
    std::mutex m_frameMutex;
    cv::Mat m_latestFrame;
    int m_width   = 640;
    int m_height  = 480;
};

} // namespace iris
