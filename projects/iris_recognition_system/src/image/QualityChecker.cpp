#include "image/QualityChecker.h"
#include "image/ImagePreprocessor.h"
#include <opencv2/imgproc.hpp>
#include <cmath>
#include <iomanip>
#include <sstream>

namespace iris {

QualityChecker::QualityChecker(float threshold)
    : m_threshold(threshold) {}

QualityResult QualityChecker::check(const cv::Mat& eyeRoi) {
    QualityResult result{};

    cv::Mat gray = ImagePreprocessor::toGrayscale(eyeRoi);

    result.sharpness  = computeSharpness(gray);
    result.brightness = computeBrightness(gray);
    result.occlusion  = estimateOcclusion(gray);
    result.overall    = computeOverall(result.sharpness, result.brightness, result.occlusion);

    result.passed = (result.overall >= m_threshold);

    if (!result.passed) {
        std::ostringstream oss;
        if (result.sharpness < 0.3f)
            oss << "Low sharpness(" << std::fixed << std::setprecision(2) << result.sharpness << "); ";
        if (result.brightness < 0.2f || result.brightness > 0.9f)
            oss << "Bad brightness(" << result.brightness << "); ";
        if (result.occlusion > 0.5f)
            oss << "High occlusion(" << result.occlusion << "); ";
        result.reason = oss.str();
        if (result.reason.empty()) result.reason = "Overall score below threshold";
    }

    return result;
}

float QualityChecker::computeSharpness(const cv::Mat& gray) {
    cv::Mat lap;
    cv::Laplacian(gray, lap, CV_64F);
    cv::Scalar mean, stddev;
    cv::meanStdDev(lap, mean, stddev);
    // Normalize: typical good iris images have variance 50-500
    float variance = static_cast<float>(stddev[0] * stddev[0]);
    return std::min(1.0f, variance / 200.0f);
}

float QualityChecker::computeBrightness(const cv::Mat& gray) {
    cv::Scalar mean = cv::mean(gray);
    return static_cast<float>(mean[0] / 255.0);
}

float QualityChecker::estimateOcclusion(const cv::Mat& gray) {
    // Count very dark pixels (potential eyelashes/eyelid occlusion)
    // Optimized: use cv::threshold + countNonZero (SIMD-accelerated)
    cv::Mat binary;
    cv::threshold(gray, binary, 40, 255, cv::THRESH_BINARY_INV);
    int darkPixels = cv::countNonZero(binary);
    int totalPixels = gray.rows * gray.cols;

    return std::min(1.0f, static_cast<float>(darkPixels) / (totalPixels * 0.3f));
}

float QualityChecker::computeOverall(float sharpness, float brightness, float occlusion) {
    // Weighted combination
    float brightnessScore = 1.0f - std::abs(brightness - 0.5f) * 2.0f; // prefer mid-brightness
    float occlusionInv    = 1.0f - occlusion;

    return 0.50f * sharpness
         + 0.25f * brightnessScore
         + 0.25f * occlusionInv;
}

} // namespace iris
