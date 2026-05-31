#include "service/RecognitionService.h"
#include "model/EyeDetector.h"
#include "image/QualityChecker.h"
#include "model/IrisSegmenter.h"
#include "image/IrisNormalizer.h"
#include "model/IrisFeatureExtractor.h"
#include "model/IrisLivenessDetector.h"
#include "matching/IrisMatcher.h"
#include "matching/DecisionEngine.h"
#include "persistence/Database.h"
#include <iostream>

namespace iris {

RecognitionService::RecognitionService(
    EyeDetector& eyeDetector,
    QualityChecker& qualityChecker,
    IrisSegmenter& segmenter,
    IrisNormalizer& normalizer,
    IrisFeatureExtractor& featureExtractor,
    IrisLivenessDetector& livenessDetector,
    IrisMatcher& matcher,
    DecisionEngine& decisionEngine,
    Database& database)
    : m_eyeDetector(eyeDetector)
    , m_qualityChecker(qualityChecker)
    , m_segmenter(segmenter)
    , m_normalizer(normalizer)
    , m_featureExtractor(featureExtractor)
    , m_livenessDetector(livenessDetector)
    , m_matcher(matcher)
    , m_decisionEngine(decisionEngine)
    , m_database(database) {}

FinalDecision RecognitionService::processFrame(const cv::Mat& frame) {
    FinalDecision decision;

    if (frame.empty()) {
        decision.reason = "Empty frame";
        return decision;
    }

    // Step 1: Eye detection
    cv::Mat eyeRoi = m_eyeDetector.getBestEyeRoi(frame);

    // Step 2: Quality check
    m_lastQuality = m_qualityChecker.check(eyeRoi);
    if (!m_lastQuality.passed) {
        decision.reason = "Quality: " + m_lastQuality.reason;
        decision.quality_score = m_lastQuality.overall;
        return decision;
    }

    // Step 3: Iris segmentation
    auto segResult = m_segmenter.segment(eyeRoi);
    if (!segResult.boundaries.valid()) {
        decision.reason = "Segmentation failed";
        decision.quality_score = m_lastQuality.overall;
        return decision;
    }

    // Step 4: Normalization
    auto normalized = m_normalizer.normalize(eyeRoi, segResult.boundaries);
    if (normalized.image.empty()) {
        decision.reason = "Normalization failed";
        return decision;
    }

    // Step 5: Feature extraction
    auto embedding = m_featureExtractor.extract(normalized);

    // Step 6: Liveness check
    m_recentFrames.push_back(eyeRoi.clone());
    if (m_recentFrames.size() > 10) {
        m_recentFrames.pop_front();
    }
    m_lastLiveness = m_livenessDetector.check(eyeRoi, m_recentFrames);

    // Step 7: Match against database
    auto allTemplates = m_database.getAllTemplates();
    m_lastMatch = m_matcher.search(embedding, allTemplates);

    // Step 8: Final decision
    decision = m_decisionEngine.decide(m_lastMatch, m_lastLiveness, m_lastQuality);

    // Log
    if (decision.accepted) {
        auto user = m_database.getUser(decision.user_id);
        decision.username = user.username;
        m_database.logAction("identify_success", decision.user_id,
                              "Matched user: " + user.username);
    }

    return decision;
}

} // namespace iris
