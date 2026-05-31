#include "model/IrisFeatureExtractor.h"
#include <opencv2/imgproc.hpp>
#include <cmath>
#include <iostream>
#include <algorithm>

namespace iris {

IrisFeatureExtractor::IrisFeatureExtractor() = default;

bool IrisFeatureExtractor::initialize(const std::string& /*modelPath*/) {
    m_initialized = true;
    return true;
}

IrisEmbedding IrisFeatureExtractor::extract(const NormalizedIris& normalized) {
    IrisEmbedding result;

    if (normalized.image.empty()) {
        result.quality = 0.0f;
        return result;
    }

    // Ensure image is grayscale float
    cv::Mat irisFloat;
    if (normalized.image.type() == CV_8U) {
        normalized.image.convertTo(irisFloat, CV_32F, 1.0 / 255.0);
    } else {
        irisFloat = normalized.image.clone();
    }

    // Extract IrisCode using Gabor filters
    extractIrisCode(irisFloat, normalized.mask,
                    result.iris_code, result.mask_code);

    // Also generate a float embedding from the IrisCode for API compatibility
    result.vector.resize(result.iris_code.size());
    for (size_t i = 0; i < result.iris_code.size(); ++i) {
        result.vector[i] = static_cast<float>(result.iris_code[i]) / 255.0f;
    }

    // Estimate quality by checking code entropy
    int ones = 0, total = 0;
    for (size_t i = 0; i < result.iris_code.size(); ++i) {
        for (int bit = 0; bit < 8; ++bit) {
            if (result.iris_code[i] & (1 << bit)) ++ones;
            ++total;
        }
    }
    // Good IrisCode should have ~50% ones
    float fraction = static_cast<float>(ones) / static_cast<float>(total);
    result.quality = 1.0f - 2.0f * std::abs(fraction - 0.5f);

    return result;
}

cv::Mat IrisFeatureExtractor::createGaborKernel(int kernelSize,
                                                  float sigma, float theta,
                                                  float lambda, float gamma, float psi) {
    int halfSize = kernelSize / 2;
    cv::Mat kernel(kernelSize, kernelSize, CV_32F);

    float sigmaX = sigma;
    float sigmaY = sigma / gamma;

    for (int y = -halfSize; y <= halfSize; ++y) {
        for (int x = -halfSize; x <= halfSize; ++x) {
            // Rotate coordinates
            float xTheta = x * std::cos(theta) + y * std::sin(theta);
            float yTheta = -x * std::sin(theta) + y * std::cos(theta);

            // Gabor function
            float gauss = std::exp(-0.5f * (
                (xTheta * xTheta) / (sigmaX * sigmaX) +
                (yTheta * yTheta) / (sigmaY * sigmaY)
            ));

            float sinusoid = std::cos(2.0f * static_cast<float>(M_PI) * xTheta / lambda + psi);

            kernel.at<float>(y + halfSize, x + halfSize) = gauss * sinusoid;
        }
    }

    // Remove DC component
    cv::Scalar mean = cv::mean(kernel);
    kernel -= mean[0];

    return kernel;
}

void IrisFeatureExtractor::extractIrisCode(const cv::Mat& normalizedIris,
                                            const cv::Mat& mask,
                                            std::vector<uint8_t>& irisCode,
                                            std::vector<uint8_t>& maskCode) {
    // Use Gabor filter bank with multiple scales and orientations
    // Phase quantization: [0, π/2, π, 3π/2] -> 2 bits per filter response

    int rows = normalizedIris.rows;
    int cols = normalizedIris.cols;

    // Downsample factor to get manageable code size (~2048 bits = 256 bytes)
    int dsRow = std::max(1, rows / 16);  // 4 rows  for 64-row input
    int dsCol = std::max(1, cols / 16);  // 32 cols for 512-col input
    int codeRows = rows / dsRow;
    int codeCols = cols / dsCol;

    // Total bits needed
    size_t totalBits = codeRows * codeCols * m_numOrientations * 2; // 2 bits per orientation
    size_t totalBytes = (totalBits + 7) / 8;

    irisCode.assign(totalBytes, 0);
    maskCode.assign(totalBytes, 0);

    // Gabor parameters
    float lambda  = 8.0f;   // wavelength
    float sigma   = 5.0f;   // Gaussian envelope
    float gamma   = 0.5f;   // spatial aspect ratio
    float psi     = 0.0f;   // phase offset (0 for even, π/2 for odd)

    int kernelSize = 21;
    size_t bitIndex = 0;

    for (int orient = 0; orient < m_numOrientations; ++orient) {
        float theta = static_cast<float>(orient) * static_cast<float>(M_PI)
                      / static_cast<float>(m_numOrientations);

        // Even-symmetric Gabor filter (real part, psi=0)
        cv::Mat gaborEven = createGaborKernel(kernelSize, sigma, theta, lambda, gamma, 0.0f);
        // Odd-symmetric Gabor filter (imaginary part, psi=π/2)
        cv::Mat gaborOdd  = createGaborKernel(kernelSize, sigma, theta, lambda, gamma,
                                               static_cast<float>(M_PI_2));

        // Convolve the normalized iris with both filters
        cv::Mat responseEven, responseOdd;
        cv::filter2D(normalizedIris, responseEven, CV_32F, gaborEven);
        cv::filter2D(normalizedIris, responseOdd,  CV_32F, gaborOdd);

        // Phase quantization at downsampled positions
        for (int r = 0; r < codeRows; ++r) {
            for (int c = 0; c < codeCols; ++c) {
                int srcR = r * dsRow + dsRow / 2;
                int srcC = c * dsCol + dsCol / 2;
                srcR = std::min(srcR, rows - 1);
                srcC = std::min(srcC, cols - 1);

                float re = responseEven.at<float>(srcR, srcC);
                float im = responseOdd.at<float>(srcR, srcC);

                // Phase quantization to 2 bits based on quadrant
                // Bit 0: sign of real part (re > 0 = 1)
                // Bit 1: sign of imaginary part (im > 0 = 1)
                uint8_t bit1 = (re > 0) ? 1 : 0;
                uint8_t bit2 = (im > 0) ? 1 : 0;

                // Store bits
                if (bitIndex / 8 < irisCode.size()) {
                    irisCode[bitIndex / 8] |= (bit1 << (bitIndex % 8));
                }
                ++bitIndex;

                if (bitIndex / 8 < irisCode.size()) {
                    irisCode[bitIndex / 8] |= (bit2 << (bitIndex % 8));
                }
                ++bitIndex;

                // Mask bit: check if the point is in valid iris region
                if (!mask.empty() && mask.at<uint8_t>(srcR, srcC) > 0) {
                    size_t maskBitIdx = bitIndex - 2;
                    maskCode[maskBitIdx / 8] |= (1 << (maskBitIdx % 8));
                    maskCode[(maskBitIdx + 1) / 8] |= (1 << ((maskBitIdx + 1) % 8));
                }
            }
        }
    }

    // Truncate to exactly the bits we generated
    size_t actualBytes = (bitIndex + 7) / 8;
    irisCode.resize(actualBytes);
    maskCode.resize(actualBytes);

    // If no mask was set (all valid), set all mask bits
    if (mask.empty()) {
        std::fill(maskCode.begin(), maskCode.end(), 0xFF);
    }
}

std::vector<uint8_t> IrisFeatureExtractor::embeddingToCode(
    const std::vector<float>& embedding) {
    std::vector<uint8_t> code;
    code.reserve(embedding.size());

    for (float v : embedding) {
        code.push_back(static_cast<uint8_t>(
            std::clamp(v * 255.0f, 0.0f, 255.0f)
        ));
    }
    return code;
}

} // namespace iris
