#pragma once

#include "domain/User.h"
#include "domain/IrisTemplate.h"
#include "domain/RecognitionResult.h"
#include <memory>
#include <string>
#include <vector>
#include <opencv2/core.hpp>

namespace iris {

class EyeDetector;
class QualityChecker;
class IrisSegmenter;
class IrisNormalizer;
class IrisFeatureExtractor;
class IrisLivenessDetector;
class Database;
class TemplateEncryptor;

/// Orchestrates the iris enrollment pipeline
class EnrollmentService {
public:
    EnrollmentService(EyeDetector& eyeDetector,
                      QualityChecker& qualityChecker,
                      IrisSegmenter& segmenter,
                      IrisNormalizer& normalizer,
                      IrisFeatureExtractor& featureExtractor,
                      IrisLivenessDetector& livenessDetector,
                      Database& database,
                      TemplateEncryptor& encryptor);

    /// Process one frame for enrollment. Returns true if enough good frames collected.
    bool processFrame(const cv::Mat& frame);

    /// Complete enrollment and save template
    bool finalizeEnrollment(const std::string& username);

    /// Get enrollment progress
    int getProgress() const { return static_cast<int>(m_collectedCodes.size()); }
    int getRequiredFrames() const { return m_requiredFrames; }

    /// Reset enrollment state
    void reset();

    /// Get latest quality info
    QualityResult getLastQuality() const { return m_lastQuality; }

private:
    EyeDetector& m_eyeDetector;
    QualityChecker& m_qualityChecker;
    IrisSegmenter& m_segmenter;
    IrisNormalizer& m_normalizer;
    IrisFeatureExtractor& m_featureExtractor;
    IrisLivenessDetector& m_livenessDetector;
    Database& m_database;
    TemplateEncryptor& m_encryptor;

    std::vector<std::vector<uint8_t>> m_collectedCodes;
    std::vector<std::vector<uint8_t>> m_collectedMasks;
    std::vector<float> m_collectedQualities;
    std::vector<cv::Mat> m_recentFrames;

    QualityResult m_lastQuality;
    int m_requiredFrames = 5;
};

} // namespace iris
