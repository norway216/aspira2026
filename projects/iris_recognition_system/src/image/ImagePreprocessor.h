#pragma once

#include <opencv2/core.hpp>

namespace iris {

/// Image preprocessing utilities for iris recognition pipeline
class ImagePreprocessor {
public:
    /// Convert to grayscale if needed
    static cv::Mat toGrayscale(const cv::Mat& src);

    /// Apply CLAHE (Contrast Limited Adaptive Histogram Equalization)
    static cv::Mat enhanceContrast(const cv::Mat& gray);

    /// Remove high-frequency noise with median blur
    static cv::Mat denoise(const cv::Mat& src, int kernelSize = 5);

    /// Resize with aspect ratio preservation
    static cv::Mat resizeTo(const cv::Mat& src, int width, int height);

    /// Normalize pixel values to [0, 1] float
    static cv::Mat normalizeToFloat(const cv::Mat& src);

    /// Full preprocessing pipeline for eye ROI
    static cv::Mat preprocessEyeRoi(const cv::Mat& eyeRoi);
};

} // namespace iris
