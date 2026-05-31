#pragma once

#include "image/IrisNormalizer.h"  // NormalizedIris
#include <opencv2/core.hpp>
#include <string>
#include <vector>

namespace iris {

/// Result of feature extraction
struct IrisEmbedding {
    std::vector<float> vector;     // float embedding (for NN/compatibility)
    std::vector<uint8_t> iris_code; // binary IrisCode (classical approach)
    std::vector<uint8_t> mask_code; // valid bits mask
    float quality = 0.0f;
};

/// Iris feature extractor using 2D Gabor wavelet filters (classical CV).
/// Generates IrisCode by phase quantization of Gabor filter responses.
/// In production, can be replaced with ResNet18/MobileNetV3 ONNX inference.
class IrisFeatureExtractor {
public:
    IrisFeatureExtractor();

    /// Initialize with optional model path (for NN mode)
    bool initialize(const std::string& modelPath = "");

    /// Extract features from a normalized iris image
    IrisEmbedding extract(const NormalizedIris& normalized);

    /// Set Gabor filter parameters
    void setGaborParams(int scales, int orientations) {
        m_numScales = scales;
        m_numOrientations = orientations;
    }

private:
    /// Generate a 2D Gabor kernel
    static cv::Mat createGaborKernel(int kernelSize,
                                      float sigma, float theta,
                                      float lambda, float gamma, float psi);

    /// Apply Gabor filter bank and quantize phase responses
    void extractIrisCode(const cv::Mat& normalizedIris,
                          const cv::Mat& mask,
                          std::vector<uint8_t>& irisCode,
                          std::vector<uint8_t>& maskCode);

    /// Convert float embedding to binary IrisCode for storage
    static std::vector<uint8_t> embeddingToCode(const std::vector<float>& embedding);

    int m_numScales       = 4;
    int m_numOrientations = 8;
    bool m_initialized    = true;  // classical CV needs no model
};

} // namespace iris
