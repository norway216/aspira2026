#pragma once

#include <opencv2/core.hpp>
#include <vector>

namespace iris {

/// Geometric utilities for iris processing
namespace GeometryUtils {

/// Fit a circle to a set of points using least squares
bool fitCircle(const std::vector<cv::Point>& points,
               cv::Point2f& center, float& radius);

/// Fit an ellipse to a set of points
bool fitEllipse(const std::vector<cv::Point>& points, cv::RotatedRect& ellipse);

/// Convert polar coordinates (relative to iris annulus) to Cartesian
/// r goes from 0 (pupil boundary) to 1 (iris boundary)
/// theta is in radians
cv::Point2f polarToCartesian(const cv::Point2f& pupilCenter, float pupilRadius,
                              const cv::Point2f& irisCenter, float irisRadius,
                              float r, float theta);

/// Check if a point is within a circle
bool isInsideCircle(const cv::Point2f& pt, const cv::Point2f& center, float radius);

/// Compute the Euclidean distance between two points
inline float distance(const cv::Point2f& a, const cv::Point2f& b) {
    return std::sqrt((a.x - b.x) * (a.x - b.x) + (a.y - b.y) * (a.y - b.y));
}

} // namespace GeometryUtils

} // namespace iris
