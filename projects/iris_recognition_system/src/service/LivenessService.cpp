#include "service/LivenessService.h"
#include "model/IrisLivenessDetector.h"

namespace iris {

LivenessService::LivenessService(IrisLivenessDetector& detector)
    : m_detector(detector) {}

LivenessResult LivenessService::check(const cv::Mat& eyeRoi) {
    if (!eyeRoi.empty()) {
        m_recentFrames.push_back(eyeRoi.clone());
        if (m_recentFrames.size() > MAX_HISTORY) {
            m_recentFrames.pop_front();
        }
    }
    return m_detector.check(eyeRoi, m_recentFrames);
}

void LivenessService::reset() {
    m_recentFrames.clear();
}

} // namespace iris
