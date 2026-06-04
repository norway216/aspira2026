#include "model/IrisSegmenter.h"
#include "image/ImagePreprocessor.h"
#include "image/GeometryUtils.h"
#include <opencv2/imgproc.hpp>
#include <iostream>
#include <algorithm>
#include <cmath>

namespace iris {

IrisSegmenter::IrisSegmenter() = default;

IrisSegmentationResult IrisSegmenter::segment(const cv::Mat& eyeRoi) {
    IrisSegmentationResult result;

    if (eyeRoi.empty()) return result;

    // Preprocess
    cv::Mat gray = ImagePreprocessor::toGrayscale(eyeRoi);
    cv::Mat enhanced = ImagePreprocessor::enhanceContrast(gray);
    cv::Mat denoised = ImagePreprocessor::denoise(enhanced, 3);

    // Step 1: Detect iris outer boundary
    cv::Point2f irisCenter;
    float irisRadius = 0;
    bool irisFound = detectIrisBoundary(denoised, irisCenter, irisRadius);

    if (!irisFound) {
        // Fallback: assume iris is centered and takes up ~70% of the ROI
        irisCenter = cv::Point2f(eyeRoi.cols / 2.0f, eyeRoi.rows / 2.0f);
        irisRadius = std::min(eyeRoi.cols, eyeRoi.rows) * 0.35f;
    }

    // Step 2: Detect pupil boundary
    cv::Point2f pupilCenter = irisCenter;
    float pupilRadius = irisRadius * 0.35f;
    bool pupilFound = detectPupilBoundary(denoised, irisCenter, irisRadius,
                                           pupilCenter, pupilRadius);

    if (!pupilFound) {
        pupilCenter = irisCenter;
        pupilRadius = irisRadius * 0.3f;
    }

    // Fill boundaries
    result.boundaries.pupil_center = pupilCenter;
    result.boundaries.pupil_radius = pupilRadius;
    result.boundaries.iris_center  = irisCenter;
    result.boundaries.iris_radius  = irisRadius;

    // Refine boundaries
    refineBoundaries(denoised, result.boundaries);

    // Generate masks
    generateMasks(eyeRoi.size(), result.boundaries,
                  result.iris_mask, result.pupil_mask);

    // Estimate segmentation quality
    float pupilIrisRatio = result.boundaries.pupil_radius / result.boundaries.iris_radius;
    bool ratioOk = (pupilIrisRatio > 0.15f && pupilIrisRatio < 0.65f);
    float centerDist = GeometryUtils::distance(
        result.boundaries.pupil_center, result.boundaries.iris_center);
    bool centersClose = (centerDist < result.boundaries.iris_radius * 0.3f);

    result.quality_score = ratioOk ? (centersClose ? 0.9f : 0.6f) : 0.3f;

    return result;
}

bool IrisSegmenter::detectIrisBoundary(const cv::Mat& gray,
                                        cv::Point2f& center, float& radius) {
    std::vector<cv::Vec3f> circles;
    double minDist = std::max(1.0, static_cast<double>(gray.rows) / 4.0);
    cv::HoughCircles(gray, circles, cv::HOUGH_GRADIENT,
                     1.2,                    // dp (reduced resolution for speed)
                     minDist,                // minDist
                     100.0,                  // param1 (Canny high threshold)
                     30.0,                   // param2 (accumulator threshold)
                     m_minIrisRadius,        // minRadius
                     m_maxIrisRadius);       // maxRadius

    if (circles.empty()) {
        // Try again with lower threshold
        cv::HoughCircles(gray, circles, cv::HOUGH_GRADIENT,
                         1.2, minDist,
                         80.0, 20.0,
                         m_minIrisRadius, m_maxIrisRadius);
    }

    if (circles.empty()) return false;

    // Pick the best circle (largest that's reasonably centered)
    float imgCx = gray.cols / 2.0f;
    float imgCy = gray.rows / 2.0f;
    float bestScore = -1.0f;
    int bestIdx = 0;

    for (size_t i = 0; i < circles.size(); ++i) {
        float cx = circles[i][0], cy = circles[i][1], r = circles[i][2];

        // Score: prefer circles near center, larger radius
        float centerness = 1.0f - std::min(1.0f,
            std::sqrt((cx - imgCx)*(cx - imgCx) + (cy - imgCy)*(cy - imgCy))
            / (gray.cols * 0.3f));
        float sizeScore = r / static_cast<float>(m_maxIrisRadius);
        float score = 0.6f * centerness + 0.4f * sizeScore;

        if (score > bestScore) {
            bestScore = score;
            bestIdx = static_cast<int>(i);
        }
    }

    center = cv::Point2f(circles[bestIdx][0], circles[bestIdx][1]);
    radius = circles[bestIdx][2];

    return true;
}

bool IrisSegmenter::detectPupilBoundary(const cv::Mat& gray,
                                         const cv::Point2f& irisCenter, float irisRadius,
                                         cv::Point2f& pupilCenter, float& pupilRadius) {
    // Search for pupil within the iris region
    int margin = 5;
    int x0 = std::max(0, static_cast<int>(irisCenter.x - irisRadius) - margin);
    int y0 = std::max(0, static_cast<int>(irisCenter.y - irisRadius) - margin);
    int x1 = std::min(gray.cols, static_cast<int>(irisCenter.x + irisRadius) + margin);
    int y1 = std::min(gray.rows, static_cast<int>(irisCenter.y + irisRadius) + margin);

    cv::Rect irisRoi(x0, y0, x1 - x0, y1 - y0);
    if (irisRoi.width <= 5 || irisRoi.height <= 5) return false;

    cv::Mat irisRegion = gray(irisRoi);

    // Invert for pupil detection (pupil is dark)
    cv::Mat inverted;
    cv::bitwise_not(irisRegion, inverted);

    std::vector<cv::Vec3f> circles;
    int maxPupilR = std::min(m_maxPupilRadius,
                              static_cast<int>(irisRadius * 0.55f));
    int minPupilR = std::max(m_minPupilRadius, 5);

    // Ensure minDist is at least 1
    double minDist = std::max(1.0, static_cast<double>(irisRegion.rows) / 3.0);
    cv::HoughCircles(inverted, circles, cv::HOUGH_GRADIENT,
                     1.5,       // dp
                     minDist,   // minDist
                     120.0,     // param1
                     25.0,      // param2
                     minPupilR, maxPupilR);

    if (circles.empty()) {
        // Second try: use original (non-inverted) with dark circle detection
        cv::HoughCircles(irisRegion, circles, cv::HOUGH_GRADIENT,
                         1.5, minDist,
                         100.0, 20.0,
                         minPupilR, maxPupilR);
    }

    if (circles.empty()) return false;

    // Pick circle closest to iris center
    float bestDist = std::numeric_limits<float>::max();
    int bestIdx = 0;

    for (size_t i = 0; i < circles.size(); ++i) {
        float cx = circles[i][0] + x0;
        float cy = circles[i][1] + y0;
        float dist = std::sqrt((cx - irisCenter.x)*(cx - irisCenter.x) +
                                (cy - irisCenter.y)*(cy - irisCenter.y));
        if (dist < bestDist) {
            bestDist = dist;
            bestIdx = static_cast<int>(i);
        }
    }

    pupilCenter = cv::Point2f(circles[bestIdx][0] + x0,
                               circles[bestIdx][1] + y0);
    pupilRadius = circles[bestIdx][2];

    return true;
}

void IrisSegmenter::generateMasks(const cv::Size& size,
                                   const IrisBoundaries& boundaries,
                                   cv::Mat& irisMask, cv::Mat& pupilMask) {
    // Optimized: use cv::circle (SIMD-accelerated) instead of per-pixel distance loop
    // Draw pupil mask (filled white circle)
    pupilMask = cv::Mat::zeros(size, CV_8U);
    cv::circle(pupilMask,
               cv::Point(static_cast<int>(boundaries.pupil_center.x),
                         static_cast<int>(boundaries.pupil_center.y)),
               static_cast<int>(boundaries.pupil_radius),
               cv::Scalar(255), -1);

    // Draw iris mask: filled iris circle, then erase pupil area
    irisMask = cv::Mat::zeros(size, CV_8U);
    cv::circle(irisMask,
               cv::Point(static_cast<int>(boundaries.iris_center.x),
                         static_cast<int>(boundaries.iris_center.y)),
               static_cast<int>(boundaries.iris_radius * 1.02f),
               cv::Scalar(255), -1);
    // Erase pupil from iris mask (iris = annulus between pupil and iris boundary)
    cv::circle(irisMask,
               cv::Point(static_cast<int>(boundaries.pupil_center.x),
                         static_cast<int>(boundaries.pupil_center.y)),
               static_cast<int>(boundaries.pupil_radius * 0.95f),
               cv::Scalar(0), -1);
}

void IrisSegmenter::refineBoundaries(const cv::Mat& gray,
                                      IrisBoundaries& boundaries) {
    if (!boundaries.valid()) return;

    // Refine iris boundary using Daugman's integro-differential operator (simplified)
    // Sample radial gradients at multiple angles and adjust radius

    const int numAngles = 36;
    const float TWO_PI = 2.0f * static_cast<float>(M_PI);
    float radiusAdjust = 0.0f;

    for (int i = 0; i < numAngles; ++i) {
        float theta = static_cast<float>(i) / numAngles * TWO_PI;

        // Sample along the radial direction near the boundary
        float r0 = boundaries.iris_radius;
        for (float dr = -5.0f; dr <= 5.0f; dr += 1.0f) {
            float r = r0 + dr;
            cv::Point2f pt(
                boundaries.iris_center.x + r * std::cos(theta),
                boundaries.iris_center.y + r * std::sin(theta)
            );

            // Check gradient at this point
            if (pt.x >= 1 && pt.x < gray.cols - 1 &&
                pt.y >= 1 && pt.y < gray.rows - 1) {
                int x = static_cast<int>(pt.x);
                int y = static_cast<int>(pt.y);
                float grad = std::abs(
                    static_cast<float>(gray.at<uint8_t>(y, x+1)) -
                    static_cast<float>(gray.at<uint8_t>(y, x-1))
                );
                radiusAdjust += dr * grad;
            }
        }
    }

    // Small refinement only
    float totalAdjust = radiusAdjust / (numAngles * 11.0f * 100.0f);
    if (std::abs(totalAdjust) < boundaries.iris_radius * 0.1f) {
        boundaries.iris_radius += totalAdjust;
    }

    // Ensure pupil is inside iris
    if (boundaries.pupil_radius >= boundaries.iris_radius * 0.7f) {
        boundaries.pupil_radius = boundaries.iris_radius * 0.4f;
    }
}

} // namespace iris
