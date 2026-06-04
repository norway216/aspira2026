#include "image/IrisNormalizer.h"
#include "image/GeometryUtils.h"
#include <cmath>
#include <algorithm>

namespace iris {

IrisNormalizer::IrisNormalizer(int outputWidth, int outputHeight)
    : m_outputWidth(outputWidth)
    , m_outputHeight(outputHeight) {}

NormalizedIris IrisNormalizer::normalize(const cv::Mat& eyeRoi,
                                          const IrisBoundaries& b) {
    NormalizedIris result;
    result.image = cv::Mat(m_outputHeight, m_outputWidth, CV_32F, cv::Scalar(0));
    result.mask  = cv::Mat(m_outputHeight, m_outputWidth, CV_8U, cv::Scalar(0));

    if (!b.valid()) return result;

    const float TWO_PI = 2.0f * static_cast<float>(M_PI);

    for (int row = 0; row < m_outputHeight; ++row) {
        // r goes from 0 (pupil boundary) to 1 (iris boundary)
        float r = static_cast<float>(row) / static_cast<float>(m_outputHeight - 1);

        for (int col = 0; col < m_outputWidth; ++col) {
            // theta goes from 0 to 2*pi
            float theta = static_cast<float>(col) / static_cast<float>(m_outputWidth) * TWO_PI;

            // Map polar (r, theta) back to Cartesian in the eye ROI
            cv::Point2f pt = GeometryUtils::polarToCartesian(
                b.pupil_center, b.pupil_radius,
                b.iris_center,  b.iris_radius,
                r, theta);

            // Sample the original image with bilinear interpolation
            if (pt.x >= 0 && pt.x < eyeRoi.cols - 1 &&
                pt.y >= 0 && pt.y < eyeRoi.rows - 1) {
                result.image.at<float>(row, col) = sampleBilinear(eyeRoi, pt.x, pt.y);

                // Check if this point is within valid iris region
                float distToIris = GeometryUtils::distance(pt, b.iris_center);
                float distToPupil = GeometryUtils::distance(pt, b.pupil_center);

                if (distToPupil >= b.pupil_radius * 0.98f &&
                    distToIris  <= b.iris_radius  * 1.02f) {
                    result.mask.at<uint8_t>(row, col) = 255;
                }
            }
        }
    }

    return result;
}

float IrisNormalizer::sampleBilinear(const cv::Mat& src, float x, float y) {
    // Clamp coordinates to safe range
    x = std::clamp(x, 0.0f, static_cast<float>(src.cols - 1));
    y = std::clamp(y, 0.0f, static_cast<float>(src.rows - 1));

    int x0 = static_cast<int>(x);
    int y0 = static_cast<int>(y);
    int x1 = std::min(x0 + 1, src.cols - 1);
    int y1 = std::min(y0 + 1, src.rows - 1);

    float dx = x - static_cast<float>(x0);
    float dy = y - static_cast<float>(y0);

    float v00, v01, v10, v11;

    if (src.type() == CV_8U || src.type() == CV_8UC1) {
        v00 = src.at<uint8_t>(y0, x0);
        v01 = src.at<uint8_t>(y1, x0);
        v10 = src.at<uint8_t>(y0, x1);
        v11 = src.at<uint8_t>(y1, x1);
    } else if (src.type() == CV_32F) {
        v00 = src.at<float>(y0, x0);
        v01 = src.at<float>(y1, x0);
        v10 = src.at<float>(y0, x1);
        v11 = src.at<float>(y1, x1);
    } else {
        return 0.0f;
    }

    float top    = v00 * (1.0f - dx) + v10 * dx;
    float bottom = v01 * (1.0f - dx) + v11 * dx;
    return top * (1.0f - dy) + bottom * dy;
}

} // namespace iris
