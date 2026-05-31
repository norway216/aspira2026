# =============================================================================
# GDB breakpoints for Iris Recognition System pipeline
#
# Usage from within GDB:
#   (gdb) source gdb_breakpoints.gdb
#
# Usage from command line:
#   gdb -x .gdbinit -x gdb_breakpoints.gdb ./build/iris_app
# =============================================================================

# ── Entry point ──────────────────────────────────────────────────────────────
# break main

# ── Pipeline stage breakpoints ───────────────────────────────────────────────
# To enable a breakpoint, uncomment the corresponding line.

# Eye Detection (Haar cascade)
# b iris::EyeDetector::detectEyes(cv::Mat const&)
# b iris::EyeDetector::getBestEyeRoi(cv::Mat const&)

# Quality Checker
# b iris::QualityChecker::evaluate(cv::Mat const&)
# b iris::QualityChecker::evaluateSharpness(cv::Mat const&)
# b iris::QualityChecker::evaluateBrightness(cv::Mat const&)
# b iris::QualityChecker::evaluateOcclusion(cv::Mat const&)

# Iris Segmentation (Hough Circles)
# b iris::IrisSegmenter::segment(cv::Mat const&)

# Iris Normalization (Daugman Rubber Sheet)
# b iris::IrisNormalizer::normalize(cv::Mat const&, iris::IrisBoundaries const&)

# Feature Extraction (Gabor filter bank)
# b iris::IrisFeatureExtractor::extractIrisCode(cv::Mat const&, cv::Mat const&, ...)
# b iris::IrisFeatureExtractor::extractEmbedding(cv::Mat const&, cv::Mat const&)

# Liveness Detection
# b iris::IrisLivenessDetector::detect(cv::Mat const&, cv::Mat const&)
# b iris::IrisLivenessDetector::analyzeTexture(cv::Mat const&)
# b iris::IrisLivenessDetector::detectMoirePattern(cv::Mat const&)
# b iris::IrisLivenessDetector::analyzeLBP(cv::Mat const&)
# b iris::IrisLivenessDetector::analyzeSpecularReflections(cv::Mat const&)

# Matching (Hamming distance)
# b iris::IrisMatcher::match(iris::IrisTemplate const&, iris::IrisTemplate const&)
# b iris::IrisMatcher::search(iris::IrisTemplate const&, ...)

# Decision Engine
# b iris::DecisionEngine::makeDecision(iris::MatchResult const&, ...)

# ── Service breakpoints ──────────────────────────────────────────────────────
# b iris::RecognitionService::processFrame(cv::Mat const&)
# b iris::EnrollmentService::processFrame(cv::Mat const&)

# ── Camera breakpoints ───────────────────────────────────────────────────────
# b iris::CameraDevice::getFrame(cv::Mat&)

# ── Database / Persistence ───────────────────────────────────────────────────
# b iris::Database::storeTemplate(...)
# b iris::Database::getAllTemplates(...)

# ── Conditional breakpoint example ───────────────────────────────────────────
# Break when quality score is low (below threshold):
# b iris::QualityChecker::evaluate
# condition $bpnum qualityScore < 0.35

echo ===== Breakpoints configured =====\n
echo Use 'info breakpoints' to see active breakpoints.\n
