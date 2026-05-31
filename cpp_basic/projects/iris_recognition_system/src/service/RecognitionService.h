#pragma once

#include "domain/RecognitionResult.h"
#include <deque>
#include <memory>
#include <string>
#include <opencv2/core.hpp>

namespace iris {

class EyeDetector;
class QualityChecker;
class IrisSegmenter;
class IrisNormalizer;
class IrisFeatureExtractor;
class IrisLivenessDetector;
class IrisMatcher;
class DecisionEngine;
class Database;

/// Orchestrates the iris recognition pipeline
class RecognitionService {
public:
    RecognitionService(EyeDetector& eyeDetector,
                       QualityChecker& qualityChecker,
                       IrisSegmenter& segmenter,
                       IrisNormalizer& normalizer,
                       IrisFeatureExtractor& featureExtractor,
                       IrisLivenessDetector& livenessDetector,
                       IrisMatcher& matcher,
                       DecisionEngine& decisionEngine,
                       Database& database);

    /// Process one frame for identification
    FinalDecision processFrame(const cv::Mat& frame);

    /// Get latest intermediate results
    QualityResult getLastQuality() const   { return m_lastQuality; }
    LivenessResult getLastLiveness() const { return m_lastLiveness; }
    MatchResult getLastMatch() const       { return m_lastMatch; }

private:
    EyeDetector& m_eyeDetector;
    QualityChecker& m_qualityChecker;
    IrisSegmenter& m_segmenter;
    IrisNormalizer& m_normalizer;
    IrisFeatureExtractor& m_featureExtractor;
    IrisLivenessDetector& m_livenessDetector;
    IrisMatcher& m_matcher;
    DecisionEngine& m_decisionEngine;
    Database& m_database;

    QualityResult m_lastQuality;
    LivenessResult m_lastLiveness;
    MatchResult m_lastMatch;

    std::deque<cv::Mat> m_recentFrames;
};

} // namespace iris
