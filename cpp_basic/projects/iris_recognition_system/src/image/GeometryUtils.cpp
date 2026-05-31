#include "image/GeometryUtils.h"
#include <opencv2/imgproc.hpp>
#include <cmath>

namespace iris {
namespace GeometryUtils {

bool fitCircle(const std::vector<cv::Point>& points,
               cv::Point2f& center, float& radius) {
    if (points.size() < 3) return false;

    // Use algebraic circle fitting (Taubin method simplified)
    // Build linear system for least-squares circle fit
    // (x^2 + y^2) = 2*cx*x + 2*cy*y + (r^2 - cx^2 - cy^2)
    // Let a = 2*cx, b = 2*cy, c = r^2 - cx^2 - cy^2

    double sumX = 0, sumY = 0, sumX2 = 0, sumY2 = 0;
    double sumXY = 0, sumX3 = 0, sumY3 = 0;
    double sumX2Y = 0, sumXY2 = 0;
    double sumZ = 0, sumZX = 0, sumZY = 0;
    size_t n = points.size();

    for (const auto& p : points) {
        double x = p.x, y = p.y;
        double z = x * x + y * y;
        sumX += x;      sumY += y;
        sumX2 += x*x;   sumY2 += y*y;
        sumXY += x*y;
        sumX3 += x*x*x; sumY3 += y*y*y;
        sumX2Y += x*x*y; sumXY2 += x*y*y;
        sumZ += z;
        sumZX += z*x;   sumZY += z*y;
    }

    // Solve 3x3 system using Cramer's rule (simplified)
    double A[3][3] = {
        {sumX2,     sumXY,      sumX},
        {sumXY,     sumY2,      sumY},
        {sumX,      sumY,       static_cast<double>(n)}
    };
    double B[3] = {sumZX, sumZY, sumZ};

    // Compute determinant
    double det = A[0][0] * (A[1][1]*A[2][2] - A[1][2]*A[2][1])
               - A[0][1] * (A[1][0]*A[2][2] - A[1][2]*A[2][0])
               + A[0][2] * (A[1][0]*A[2][1] - A[1][1]*A[2][0]);

    if (std::abs(det) < 1e-10) return false;

    double invDet = 1.0 / det;

    double a = invDet * (
        B[0] * (A[1][1]*A[2][2] - A[1][2]*A[2][1])
      - A[0][1] * (B[1]*A[2][2] - A[1][2]*B[2])
      + A[0][2] * (B[1]*A[2][1] - A[1][1]*B[2])
    );

    center.x = static_cast<float>(a / 2.0);
    center.y = static_cast<float>(
        invDet * (
            A[0][0] * (B[1]*A[2][2] - A[1][2]*B[2])
          - B[0] * (A[1][0]*A[2][2] - A[1][2]*A[2][0])
          + A[0][2] * (A[1][0]*B[2] - B[1]*A[2][0])
        ) / 2.0
    );

    double c = invDet * (
        A[0][0] * (A[1][1]*B[2] - B[1]*A[2][1])
      - A[0][1] * (A[1][0]*B[2] - B[1]*A[2][0])
      + B[0] * (A[1][0]*A[2][1] - A[1][1]*A[2][0])
    );

    radius = std::sqrt(static_cast<float>(c + center.x*center.x + center.y*center.y));
    return radius > 0;
}

bool fitEllipse(const std::vector<cv::Point>& points, cv::RotatedRect& ellipse) {
    if (points.size() < 5) return false;
    if (points.empty()) return false;
    cv::Mat pointsMat(static_cast<int>(points.size()), 1, CV_32FC2);
    for (size_t i = 0; i < points.size(); ++i) {
        pointsMat.at<cv::Vec2f>(static_cast<int>(i)) = cv::Vec2f(
            static_cast<float>(points[i].x),
            static_cast<float>(points[i].y));
    }
    ellipse = cv::fitEllipse(pointsMat);
    return true;
}

cv::Point2f polarToCartesian(const cv::Point2f& pupilCenter, float pupilRadius,
                              const cv::Point2f& irisCenter, float irisRadius,
                              float r, float theta) {
    // r: 0=pupil boundary, 1=iris boundary
    // Interpolate between pupil and iris boundaries
    float px = pupilCenter.x + pupilRadius * std::cos(theta);
    float py = pupilCenter.y + pupilRadius * std::sin(theta);
    float ix = irisCenter.x + irisRadius * std::cos(theta);
    float iy = irisCenter.y + irisRadius * std::sin(theta);

    return cv::Point2f(
        px + r * (ix - px),
        py + r * (iy - py)
    );
}

bool isInsideCircle(const cv::Point2f& pt, const cv::Point2f& center, float radius) {
    return distance(pt, center) <= radius;
}

} // namespace GeometryUtils
} // namespace iris
