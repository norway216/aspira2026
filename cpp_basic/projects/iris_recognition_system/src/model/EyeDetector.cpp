#include "model/EyeDetector.h"
#include <opencv2/imgproc.hpp>
#include <iostream>
#include <algorithm>

namespace iris {

EyeDetector::EyeDetector() = default;

bool EyeDetector::initialize(const std::string& cascadePath) {
    std::string path = cascadePath;

    if (path.empty()) {
        // Try to find OpenCV's built-in eye cascade
        // Common locations
        std::vector<std::string> candidates = {
            "/usr/share/opencv4/haarcascades/haarcascade_eye.xml",
            "/usr/share/opencv/haarcascades/haarcascade_eye.xml",
            "/usr/local/share/opencv4/haarcascades/haarcascade_eye.xml",
            "haarcascade_eye.xml"
        };

        for (const auto& p : candidates) {
            if (m_cascade.load(p)) {
                path = p;
                break;
            }
        }
    } else {
        m_cascade.load(path);
    }

    if (m_cascade.empty()) {
        std::cout << "[EyeDetector] Haar cascade not found. "
                  << "Using center-fallback ROI mode.\n";
        m_initialized = false;
        return false;
    }

    m_initialized = true;
    std::cout << "[EyeDetector] Loaded cascade: " << path << "\n";
    return true;
}

std::vector<EyeBox> EyeDetector::detect(const cv::Mat& frame) {
    std::vector<EyeBox> results;

    if (!m_initialized || frame.empty()) return results;
    // Guard against very small frames
    if (frame.rows < 60 || frame.cols < 60) return results;

    cv::Mat gray;
    if (frame.channels() == 3) {
        cv::cvtColor(frame, gray, cv::COLOR_BGR2GRAY);
    } else {
        gray = frame.clone();
    }

    cv::equalizeHist(gray, gray);

    std::vector<cv::Rect> eyes;
    // Detect eyes in the upper half of the frame (where eyes typically are)
    int upperHalfHeight = std::max(30, frame.rows / 2);
    cv::Rect upperHalf(0, 0, frame.cols, upperHalfHeight);
    cv::Mat roi = gray(upperHalf);

    m_cascade.detectMultiScale(roi, eyes,
                               1.1,   // scale factor
                               3,     // min neighbors
                               0,     // flags
                               cv::Size(30, 30)); // min size

    // Also search lower half with smaller expectation
    int lowerHalfY = std::max(0, frame.rows / 4);
    int lowerHalfHeight = std::min(frame.rows - lowerHalfY, frame.rows * 3 / 4);
    if (lowerHalfHeight >= 30) {
        cv::Rect lowerHalf(0, lowerHalfY, frame.cols, lowerHalfHeight);
        cv::Mat roi2 = gray(lowerHalf);
        std::vector<cv::Rect> eyes2;
        m_cascade.detectMultiScale(roi2, eyes2, 1.1, 4, 0, cv::Size(25, 25));

        for (auto& e : eyes2) {
            EyeBox box;
            box.bbox = e;
            box.bbox.y += lowerHalfY;
            box.confidence = 0.8f;
            box.label = static_cast<int>(results.size());
            results.push_back(box);
        }
    }

    // Merge upper half results
    for (size_t i = 0; i < eyes.size(); ++i) {
        EyeBox box;
        box.bbox = eyes[i];
        box.bbox.y += upperHalf.y;
        box.confidence = 1.0f;
        box.label = static_cast<int>(results.size());
        results.push_back(box);
    }

    // Sort by x position so left eye is first
    std::sort(results.begin(), results.end(),
              [](const EyeBox& a, const EyeBox& b) {
                  return a.bbox.x < b.bbox.x;
              });

    // Re-label based on position
    for (size_t i = 0; i < results.size(); ++i) {
        results[i].label = static_cast<int>(i);
    }

    return results;
}

cv::Mat EyeDetector::extractEyeRoi(const cv::Mat& frame,
                                    const EyeBox& box, int roiSize) {
    // Guard against invalid inputs
    if (frame.empty() || roiSize <= 0) {
        return cv::Mat(roiSize, roiSize, frame.type(), cv::Scalar(0));
    }

    // Expand the detected box to ensure full eye region
    int margin = roiSize / 4;
    int halfSize = roiSize / 2;

    // Compute center of detection, clamped to frame bounds
    int cx = std::clamp(box.bbox.x + box.bbox.width / 2, 0, frame.cols - 1);
    int cy = std::clamp(box.bbox.y + box.bbox.height / 2, 0, frame.rows - 1);

    int x0 = std::max(0, cx - halfSize - margin);
    int y0 = std::max(0, cy - halfSize - margin);
    int x1 = std::min(frame.cols, cx + halfSize + margin);
    int y1 = std::min(frame.rows, cy + halfSize + margin);

    // Ensure valid dimensions
    int width  = x1 - x0;
    int height = y1 - y0;
    if (width <= 0 || height <= 0) {
        // Fallback: use frame center
        x0 = std::max(0, frame.cols / 2 - halfSize);
        y0 = std::max(0, frame.rows / 3 - halfSize);
        x1 = std::min(frame.cols, x0 + roiSize);
        y1 = std::min(frame.rows, y0 + roiSize);
        // Re-clamp if still invalid
        if (x1 <= x0) { x0 = 0; x1 = std::min(frame.cols, roiSize); }
        if (y1 <= y0) { y0 = 0; y1 = std::min(frame.rows, roiSize); }
        width  = x1 - x0;
        height = y1 - y0;
    }

    if (width <= 0 || height <= 0) {
        return cv::Mat(roiSize, roiSize, frame.type(), cv::Scalar(0));
    }

    cv::Rect roi(x0, y0, width, height);
    cv::Mat eyeRoi = frame(roi).clone();

    // Resize to target size
    cv::resize(eyeRoi, eyeRoi, cv::Size(roiSize, roiSize), 0, 0, cv::INTER_LANCZOS4);
    return eyeRoi;
}

cv::Mat EyeDetector::getBestEyeRoi(const cv::Mat& frame) {
    if (frame.empty()) {
        return cv::Mat(256, 256, CV_8UC3, cv::Scalar(0));
    }

    auto detections = detect(frame);

    if (!detections.empty()) {
        // Return first (leftmost) eye with reasonable confidence
        for (const auto& d : detections) {
            if (d.confidence > 0.5f) {
                return extractEyeRoi(frame, d);
            }
        }
        return extractEyeRoi(frame, detections[0]);
    }

    // Fallback: assume the eye is roughly in the center-upper part of the frame
    int roiSize = 256;
    int halfSize = roiSize / 2;

    int centerX = std::max(halfSize, std::min(frame.cols - halfSize, frame.cols / 2));
    int centerY = std::max(halfSize, std::min(frame.rows - halfSize, frame.rows / 3));

    int x0 = centerX - halfSize;
    int y0 = centerY - halfSize;
    int x1 = x0 + roiSize;
    int y1 = y0 + roiSize;

    // Final clamp to frame bounds
    x0 = std::max(0, x0);
    y0 = std::max(0, y0);
    x1 = std::min(frame.cols, x1);
    y1 = std::min(frame.rows, y1);

    // Ensure we have at least some pixels
    if (x1 <= x0) { x0 = 0; x1 = std::min(frame.cols, roiSize); }
    if (y1 <= y0) { y0 = 0; y1 = std::min(frame.rows, roiSize); }

    int width  = std::max(1, x1 - x0);
    int height = std::max(1, y1 - y0);

    cv::Rect roi(x0, y0, width, height);
    cv::Mat eyeRoi = frame(roi).clone();
    cv::resize(eyeRoi, eyeRoi, cv::Size(roiSize, roiSize), 0, 0, cv::INTER_LANCZOS4);
    return eyeRoi;
}

} // namespace iris
