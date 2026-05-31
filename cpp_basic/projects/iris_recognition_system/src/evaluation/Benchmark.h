#pragma once

#include "domain/RecognitionResult.h"
#include "model/IrisFeatureExtractor.h" // IrisEmbedding
#include <string>
#include <vector>
#include <map>
#include <functional>
#include <chrono>
#include <fstream>

namespace iris {

// ── Benchmark data structures ──────────────────────────────────

/// A single enrollment/query sample
struct IrisSample {
    std::string user_id;       // which user this belongs to
    std::string image_path;    // source image path
    IrisEmbedding embedding;   // extracted features
    float quality_score = 0.0f;
    int64_t extraction_time_us = 0;  // extraction latency
};

/// Genuine or imposter comparison pair
struct ComparisonPair {
    std::string user_a;
    std::string user_b;
    std::string image_a;
    std::string image_b;
    float similarity  = 0.0f;
    float hamming_dist = 1.0f;
    bool same_user    = false;  // genuine (true) or imposter (false)
};

/// ROC curve point
struct RocPoint {
    float threshold    = 0.0f;
    float far          = 0.0f;  // False Accept Rate
    float frr          = 0.0f;  // False Reject Rate
    float tar          = 0.0f;  // True Accept Rate (= 1 - FRR)
    int true_positives  = 0;
    int false_positives = 0;
    int true_negatives  = 0;
    int false_negatives = 0;
};

/// Overall benchmark results
struct BenchmarkResult {
    // Distribution statistics
    struct DistributionStats {
        float mean    = 0.0f;
        float stddev  = 0.0f;
        float min_val = 0.0f;
        float max_val = 0.0f;
        std::vector<float> histogram;  // 20-bin histogram [0, 1]
    };

    DistributionStats genuine_dist;    // genuine pair similarity distribution
    DistributionStats imposter_dist;   // imposter pair similarity distribution

    float eer                    = 1.0f;   // Equal Error Rate
    float eer_threshold          = 0.0f;   // threshold at EER
    float far_at_1_percent_frr   = 1.0f;   // FAR when FRR=1%
    float frr_at_1_percent_far   = 1.0f;   // FRR when FAR=1%
    float d_prime                = 0.0f;   // d' = |mu_gen - mu_imp| / sqrt(0.5*(σ²gen + σ²imp))

    std::vector<RocPoint> roc_curve;  // ROC data points
    std::vector<RocPoint> det_curve;  // DET curve (FAR vs FRR)

    // Threshold sweep results
    std::vector<RocPoint> threshold_sweep;

    // Metadata
    int total_users       = 0;
    int total_samples     = 0;
    int genuine_pairs     = 0;
    int imposter_pairs    = 0;
    float avg_extraction_time_ms = 0.0f;
    float avg_quality_score      = 0.0f;

    // Report
    std::string toReport() const;
};

// ── Benchmark runner ───────────────────────────────────────────

class EyeDetector;
class QualityChecker;
class IrisSegmenter;
class IrisNormalizer;
class IrisFeatureExtractor;
class IrisMatcher;
class Database;

class Benchmark {
public:
    Benchmark(EyeDetector& eyeDetector,
              QualityChecker& qualityChecker,
              IrisSegmenter& segmenter,
              IrisNormalizer& normalizer,
              IrisFeatureExtractor& featureExtractor,
              IrisMatcher& matcher);

    /// Run benchmark on a directory of iris images
    /// Expected structure: benchmark_dir/user_XXX/image_YYY.jpg
    BenchmarkResult run(const std::string& datasetPath);

    /// Run synthetic benchmark: apply transforms to a single iris image
    /// to generate controlled genuine/imposter pairs
    BenchmarkResult runSynthetic(const std::string& imagePath, int numVariations = 10);

    /// Export ROC data to CSV for plotting
    static void exportRocCsv(const BenchmarkResult& result, const std::string& path);

    /// Export DET data to CSV
    static void exportDetCsv(const BenchmarkResult& result, const std::string& path);

private:
    /// Load all images from the dataset directory
    std::vector<IrisSample> loadDataset(const std::string& datasetPath);

    /// Extract features from a single image
    IrisSample processImage(const std::string& imagePath, const std::string& userId);

    /// Generate genuine and imposter comparison pairs
    std::pair<std::vector<ComparisonPair>, std::vector<ComparisonPair>>
    generatePairs(const std::vector<IrisSample>& samples);

    /// Compute all comparison scores
    void computeScores(std::vector<ComparisonPair>& genuinePairs,
                       std::vector<ComparisonPair>& imposterPairs);

    /// Calculate distribution statistics
    static BenchmarkResult::DistributionStats
    computeDistribution(const std::vector<float>& scores);

    /// Compute ROC and DET curves
    static void computeCurves(BenchmarkResult& result,
                               const std::vector<float>& genuineScores,
                               const std::vector<float>& imposterScores);

    /// Find EER
    static float findEER(const std::vector<RocPoint>& detCurve,
                          float& threshold);

    /// Apply image variations for synthetic benchmark
    static std::vector<cv::Mat> generateVariations(const cv::Mat& image, int count);

    EyeDetector& m_eyeDetector;
    QualityChecker& m_qualityChecker;
    IrisSegmenter& m_segmenter;
    IrisNormalizer& m_normalizer;
    IrisFeatureExtractor& m_featureExtractor;
    IrisMatcher& m_matcher;

    static constexpr int NUM_THRESHOLD_STEPS = 200;
};

} // namespace iris
