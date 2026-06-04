#include "service/EnrollmentService.h"
#include "model/EyeDetector.h"
#include "image/QualityChecker.h"
#include "model/IrisSegmenter.h"
#include "image/IrisNormalizer.h"
#include "model/IrisFeatureExtractor.h"
#include "model/IrisLivenessDetector.h"
#include "persistence/Database.h"
#include "security/TemplateEncryptor.h"
#include <chrono>
#include <iostream>
#include <sstream>
#include <random>

namespace iris {

EnrollmentService::EnrollmentService(
    EyeDetector& eyeDetector,
    QualityChecker& qualityChecker,
    IrisSegmenter& segmenter,
    IrisNormalizer& normalizer,
    IrisFeatureExtractor& featureExtractor,
    IrisLivenessDetector& livenessDetector,
    Database& database,
    TemplateEncryptor& encryptor)
    : m_eyeDetector(eyeDetector)
    , m_qualityChecker(qualityChecker)
    , m_segmenter(segmenter)
    , m_normalizer(normalizer)
    , m_featureExtractor(featureExtractor)
    , m_livenessDetector(livenessDetector)
    , m_database(database)
    , m_encryptor(encryptor) {}

bool EnrollmentService::processFrame(const cv::Mat& frame) {
    if (frame.empty()) return false;

    // Step 1: Detect eye region
    cv::Mat eyeRoi = m_eyeDetector.getBestEyeRoi(frame);

    // Step 2: Quality check
    m_lastQuality = m_qualityChecker.check(eyeRoi);
    if (!m_lastQuality.passed) {
        std::cout << "[Enrollment] Frame rejected: " << m_lastQuality.reason << "\n";
        return false;
    }

    // Step 3: Segment iris
    auto segResult = m_segmenter.segment(eyeRoi);
    if (!segResult.boundaries.valid()) {
        std::cout << "[Enrollment] Segmentation failed\n";
        return false;
    }

    // Step 4: Normalize
    auto normalized = m_normalizer.normalize(eyeRoi, segResult.boundaries);
    if (normalized.image.empty()) {
        std::cout << "[Enrollment] Normalization failed\n";
        return false;
    }

    // Step 5: Extract features
    auto embedding = m_featureExtractor.extract(normalized);

    // Step 6: Liveness check
    m_recentFrames.push_back(eyeRoi.clone());
    if (m_recentFrames.size() > 10) {
        m_recentFrames.pop_front();
    }
    auto livenessResult = m_livenessDetector.check(eyeRoi, m_recentFrames);

    if (!livenessResult.passed) {
        std::cout << "[Enrollment] Liveness check failed: "
                  << livenessResult.attack_type << "\n";
        return false;
    }

    // Collect good frame
    m_collectedCodes.push_back(embedding.iris_code);
    m_collectedMasks.push_back(embedding.mask_code);
    m_collectedQualities.push_back(embedding.quality);

    std::cout << "[Enrollment] Frame " << m_collectedCodes.size()
              << "/" << m_requiredFrames << " accepted "
              << "(quality=" << m_lastQuality.overall
              << ", liveness=" << livenessResult.live_score << ")\n";

    return m_collectedCodes.size() >= static_cast<size_t>(m_requiredFrames);
}

bool EnrollmentService::finalizeEnrollment(const std::string& username) {
    if (m_collectedCodes.empty()) {
        std::cerr << "[Enrollment] No frames collected\n";
        return false;
    }

    // Generate user ID
    std::random_device rd;
    std::mt19937_64 gen(rd());
    std::uniform_int_distribution<uint64_t> dist;
    std::ostringstream userIdStream;
    userIdStream << std::hex << dist(gen) << dist(gen);
    std::string userId = userIdStream.str();

    // Create user
    User user;
    user.id         = userId;
    user.username   = username;
    user.created_at = std::chrono::duration_cast<std::chrono::milliseconds>(
        std::chrono::system_clock::now().time_since_epoch()).count();
    user.updated_at = user.created_at;
    m_database.insertUser(user);

    // Create template - use the best quality frame
    size_t bestIdx = 0;
    float bestQuality = -1.0f;
    for (size_t i = 0; i < m_collectedQualities.size(); ++i) {
        if (m_collectedQualities[i] > bestQuality) {
            bestQuality = m_collectedQualities[i];
            bestIdx = i;
        }
    }

    IrisTemplate tmpl;
    tmpl.id            = userId + "_iris_0";
    tmpl.user_id       = userId;
    tmpl.eye_side      = "right";
    tmpl.iris_code     = m_collectedCodes[bestIdx];
    tmpl.mask_code     = m_collectedMasks[bestIdx];
    tmpl.embedding_dim = static_cast<int>(tmpl.iris_code.size());
    tmpl.model_version = "iris_gabor_v1.0";
    tmpl.quality_score = bestQuality;
    tmpl.created_at    = user.created_at;

    m_database.insertTemplate(tmpl);

    // Log
    m_database.logAction("enroll", userId, "Enrolled user: " + username);

    std::cout << "[Enrollment] User '" << username << "' (id=" << userId
              << ") enrolled successfully with "
              << m_collectedCodes.size() << " frames\n";

    reset();
    return true;
}

void EnrollmentService::reset() {
    m_collectedCodes.clear();
    m_collectedMasks.clear();
    m_collectedQualities.clear();
    m_recentFrames.clear();
    m_lastQuality = QualityResult{};
}

} // namespace iris
