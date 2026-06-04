#include "image/ImagePreprocessor.h"
#include <opencv2/imgproc.hpp>

namespace iris {

cv::Mat ImagePreprocessor::toGrayscale(const cv::Mat& src) {
    if (src.channels() == 1) return src.clone();
    cv::Mat gray;
    cv::cvtColor(src, gray, cv::COLOR_BGR2GRAY);
    return gray;
}

cv::Mat ImagePreprocessor::enhanceContrast(const cv::Mat& gray) {
    cv::Mat result;
    auto clahe = cv::createCLAHE(2.0, cv::Size(8, 8));
    clahe->apply(gray, result);
    return result;
}

cv::Mat ImagePreprocessor::denoise(const cv::Mat& src, int kernelSize) {
    cv::Mat result;
    cv::medianBlur(src, result, kernelSize);
    return result;
}

cv::Mat ImagePreprocessor::resizeTo(const cv::Mat& src, int width, int height) {
    cv::Mat result;
    cv::resize(src, result, cv::Size(width, height), 0, 0, cv::INTER_LINEAR);
    return result;
}

cv::Mat ImagePreprocessor::normalizeToFloat(const cv::Mat& src) {
    cv::Mat result;
    src.convertTo(result, CV_32F, 1.0 / 255.0);
    return result;
}

cv::Mat ImagePreprocessor::preprocessEyeRoi(const cv::Mat& eyeRoi) {
    cv::Mat gray = toGrayscale(eyeRoi);
    cv::Mat enhanced = enhanceContrast(gray);
    cv::Mat denoised = denoise(enhanced, 3);
    return denoised;
}

} // namespace iris
