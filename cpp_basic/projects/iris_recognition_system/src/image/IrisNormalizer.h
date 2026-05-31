#pragma once

#include <opencv2/core.hpp>

namespace iris {

/// Result of iris normalization
struct NormalizedIris {
    cv::Mat image;   // normalized iris texture (64x512 grayscale float)
    cv::Mat mask;    // valid region mask (64x512 uint8), 1=valid, 0=invalid
};

/// Boundary parameters for iris normalization
struct IrisBoundaries {
    cv::Point2f pupil_center;
    float pupil_radius    = 0.0f;
    cv::Point2f iris_center;
    float iris_radius     = 0.0f;

    bool valid() const {
        return pupil_radius > 0 && iris_radius > 0
            && iris_radius > pupil_radius;
    }
};

/// Implements Daugman's Rubber Sheet Model for iris normalization.
/// Maps the annular iris region to a rectangular normalized image
/// using polar-to-Cartesian coordinate transformation.
class IrisNormalizer {
public:
    /// Configure output dimensions
    IrisNormalizer(int outputWidth = 512, int outputHeight = 64);

    /// Normalize an iris using detected boundaries
    NormalizedIris normalize(const cv::Mat& eyeRoi,
                              const IrisBoundaries& boundaries);

    /// Set output size
    void setOutputSize(int width, int height) {
        m_outputWidth  = width;
        m_outputHeight = height;
    }

private:
    /// Bilinear interpolation at a floating-point coordinate
    static float sampleBilinear(const cv::Mat& src, float x, float y);

    int m_outputWidth;
    int m_outputHeight;
};

} // namespace iris
