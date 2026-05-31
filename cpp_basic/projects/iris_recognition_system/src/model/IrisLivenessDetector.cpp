#include "model/IrisLivenessDetector.h"
#include "image/ImagePreprocessor.h"
#include <opencv2/imgproc.hpp>
#include <cmath>
#include <iostream>

namespace iris {

IrisLivenessDetector::IrisLivenessDetector() = default;

bool IrisLivenessDetector::initialize(const std::string& /*modelPath*/) {
    m_initialized = true;
    return true;
}

LivenessResult IrisLivenessDetector::check(const cv::Mat& eyeRoi,
                                            const std::vector<cv::Mat>& recentFrames) {
    LivenessResult result;

    if (eyeRoi.empty()) {
        result.passed = false;
        result.attack_type = "no_input";
        return result;
    }

    cv::Mat gray = ImagePreprocessor::toGrayscale(eyeRoi);

    // Multiple heuristic signals
    float textureScore  = analyzeTexture(gray);
    float moireScore    = detectMoirePatterns(gray);
    float lbpScore      = analyzeLBP(gray);
    float reflectionScore = detectSpecularReflection(gray);

    // Moire patterns indicate screen attack (invert score)
    float antiScreenScore = 1.0f - moireScore;

    // Motion analysis from recent frames
    float motionScore = 0.5f; // neutral if no frames provided
    if (!recentFrames.empty()) {
        motionScore = analyzeMotion(recentFrames);
    }

    // Weighted fusion
    result.live_score =
        0.35f * textureScore
      + 0.20f * antiScreenScore
      + 0.15f * lbpScore
      + 0.15f * reflectionScore
      + 0.15f * motionScore;

    result.spoof_score = 1.0f - result.live_score;
    result.passed = (result.live_score >= m_threshold);

    // Classify attack type
    if (result.passed) {
        result.attack_type = "real";
    } else if (moireScore > 0.6f) {
        result.attack_type = "screen_replay";
    } else if (textureScore < 0.3f) {
        result.attack_type = "printed_photo";
    } else if (motionScore < 0.2f) {
        result.attack_type = "replay";
    } else if (reflectionScore > 0.8f) {
        result.attack_type = "cosmetic_contact_lens";
    } else {
        result.attack_type = "unknown";
    }

    return result;
}

float IrisLivenessDetector::analyzeTexture(const cv::Mat& gray) {
    // Real iris has rich texture with natural gradient distribution
    // Print attacks often have compression artifacts and reduced texture

    cv::Mat lap;
    cv::Laplacian(gray, lap, CV_64F);
    cv::Scalar mean, stddev;
    cv::meanStdDev(lap, mean, stddev);
    float variance = static_cast<float>(stddev[0] * stddev[0]);

    // Normalize: good iris texture has moderate-high Laplacian variance
    float score = std::min(1.0f, variance / 150.0f);

    // Also check local variance consistency
    cv::Mat localVar;
    cv::Mat grayFloat;
    gray.convertTo(grayFloat, CV_32F);
    cv::Mat meanImg, sqrMeanImg;
    cv::boxFilter(grayFloat, meanImg, CV_32F, cv::Size(7, 7));
    cv::boxFilter(grayFloat.mul(grayFloat), sqrMeanImg, CV_32F, cv::Size(7, 7));
    cv::Mat localStd;
    cv::sqrt(sqrMeanImg - meanImg.mul(meanImg), localStd);

    cv::Scalar stdOfStd;
    cv::meanStdDev(localStd, mean, stdOfStd);
    float textureUniformity = 1.0f - std::min(1.0f,
        static_cast<float>(stdOfStd[0]) / 30.0f);

    return 0.7f * score + 0.3f * textureUniformity;
}

float IrisLivenessDetector::detectMoirePatterns(const cv::Mat& gray) {
    // Screen replay attacks often show Moire patterns
    // Detect using FFT: look for high-frequency peaks
    cv::Mat floatImg;
    gray.convertTo(floatImg, CV_32F);

    // Optimal FFT size
    int optRows = cv::getOptimalDFTSize(gray.rows);
    int optCols = cv::getOptimalDFTSize(gray.cols);

    cv::Mat padded;
    cv::copyMakeBorder(floatImg, padded, 0, optRows - gray.rows,
                        0, optCols - gray.cols, cv::BORDER_CONSTANT, cv::Scalar(0));

    cv::Mat planes[] = { padded, cv::Mat::zeros(padded.size(), CV_32F) };
    cv::Mat complexImg;
    cv::merge(planes, 2, complexImg);
    cv::dft(complexImg, complexImg);

    // Split and compute magnitude
    cv::split(complexImg, planes);
    cv::Mat magnitude;
    cv::magnitude(planes[0], planes[1], magnitude);

    // Shift quadrants
    int cx = magnitude.cols / 2;
    int cy = magnitude.rows / 2;
    cv::Mat q0(magnitude, cv::Rect(0, 0, cx, cy));
    cv::Mat q1(magnitude, cv::Rect(cx, 0, cx, cy));
    cv::Mat q2(magnitude, cv::Rect(0, cy, cx, cy));
    cv::Mat q3(magnitude, cv::Rect(cx, cy, cx, cy));
    cv::Mat tmp;
    q0.copyTo(tmp); q3.copyTo(q0); tmp.copyTo(q3);
    q1.copyTo(tmp); q2.copyTo(q1); tmp.copyTo(q2);

    // Analyze high-frequency energy (outside central region)
    cv::Mat highFreq = magnitude.clone();
    int marginX = cx / 4;
    int marginY = cy / 4;
    cv::rectangle(highFreq,
                  cv::Point(cx - marginX, cy - marginY),
                  cv::Point(cx + marginX, cy + marginY),
                  cv::Scalar(0), -1);

    cv::Scalar totalEnergy = cv::sum(magnitude);
    cv::Scalar highEnergy  = cv::sum(highFreq);

    float ratio = (totalEnergy[0] > 0)
        ? static_cast<float>(highEnergy[0] / totalEnergy[0])
        : 0.0f;

    // High ratio of high-frequency energy suggests Moire/screen patterns
    return std::min(1.0f, ratio * 5.0f);
}

float IrisLivenessDetector::analyzeLBP(const cv::Mat& gray) {
    // Simplified LBP analysis: check local pattern diversity
    // Real iris has natural pattern variation

    std::vector<int> patternHist(256, 0);
    int totalPatterns = 0;

    for (int y = 1; y < gray.rows - 1; ++y) {
        const uint8_t* row0 = gray.ptr<uint8_t>(y - 1);
        const uint8_t* row1 = gray.ptr<uint8_t>(y);
        const uint8_t* row2 = gray.ptr<uint8_t>(y + 1);

        for (int x = 1; x < gray.cols - 1; ++x) {
            uint8_t center = row1[x];
            uint8_t pattern = 0;

            pattern |= (row0[x-1] > center) << 7;
            pattern |= (row0[x]   > center) << 6;
            pattern |= (row0[x+1] > center) << 5;
            pattern |= (row1[x+1] > center) << 4;
            pattern |= (row2[x+1] > center) << 3;
            pattern |= (row2[x]   > center) << 2;
            pattern |= (row2[x-1] > center) << 1;
            pattern |= (row1[x-1] > center) << 0;

            patternHist[pattern]++;
            ++totalPatterns;
        }
    }

    // Entropy of pattern distribution
    float entropy = 0.0f;
    for (int i = 0; i < 256; ++i) {
        if (patternHist[i] > 0) {
            float p = static_cast<float>(patternHist[i]) / totalPatterns;
            entropy -= p * std::log2(p);
        }
    }

    // Normalized: max entropy for 256 patterns is 8.0
    return std::min(1.0f, entropy / 7.0f);
}

float IrisLivenessDetector::detectSpecularReflection(const cv::Mat& gray) {
    // Real eyes have natural corneal reflections (bright spots)
    // We want moderate reflection - too much or too little is suspicious

    int brightPixels = 0;
    int totalPixels = gray.rows * gray.cols;

    for (int y = 0; y < gray.rows; ++y) {
        const uint8_t* row = gray.ptr<uint8_t>(y);
        for (int x = 0; x < gray.cols; ++x) {
            if (row[x] > 220) ++brightPixels;
        }
    }

    float brightRatio = static_cast<float>(brightPixels) / totalPixels;

    // Ideal: some reflections but not too many (1-5% bright pixels)
    float ideal = 0.03f;
    return 1.0f - std::min(1.0f, std::abs(brightRatio - ideal) / ideal);
}

float IrisLivenessDetector::analyzeMotion(const std::vector<cv::Mat>& frames) {
    if (frames.size() < 2) return 0.5f;

    float totalMotion = 0.0f;
    int comparisons = 0;

    for (size_t i = 1; i < frames.size(); ++i) {
        if (frames[i].empty() || frames[i-1].empty()) continue;

        cv::Mat diff;
        cv::absdiff(frames[i], frames[i-1], diff);
        cv::Scalar meanDiff = cv::mean(diff);

        totalMotion += static_cast<float>(meanDiff[0]) / 255.0f;
        ++comparisons;
    }

    if (comparisons == 0) return 0.5f;

    float avgMotion = totalMotion / comparisons;

    // Real eyes have subtle natural motion (pupil hippus, micro-movements)
    // Too much motion = suspicious, too little = still image attack
    float idealMotion = 0.02f;
    float score = 1.0f - std::min(1.0f,
        std::abs(avgMotion - idealMotion) / (idealMotion * 3.0f));

    return score;
}

} // namespace iris
