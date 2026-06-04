#pragma once

#include <string>
#include <vector>
#include <chrono>
#include <mutex>

namespace iris {

/// Per-stage timing record
struct StageTiming {
    std::string name;
    double time_ms;
};

/// Per-frame timing record (all stages)
struct FrameTiming {
    int frame_index;
    double total_ms;
    std::vector<StageTiming> stages;
};

/// FPS benchmarking utility for pipeline performance measurement.
/// Thread-safe for use in multi-threaded pipelines.
class FpsBenchmark {
public:
    explicit FpsBenchmark(const std::string& name = "IrisRecognition");

    /// Start timing the overall frame
    void startFrame();

    /// Start timing a pipeline stage
    void startStage(const std::string& stageName);

    /// End current stage and record duration
    void endStage();

    /// End the frame, compute statistics
    void endFrame();

    /// Get average FPS over recorded frames
    double avgFps() const;

    /// Get average time for a specific stage
    double avgStageMs(const std::string& stageName) const;

    /// Get min/avg/max for a stage
    void stageStats(const std::string& stageName,
                    double& minMs, double& avgMs, double& maxMs) const;

    /// Total elapsed time across recorded frames
    double totalElapsedMs() const;

    /// Number of recorded frames
    int totalFrames() const { return static_cast<int>(m_frames.size()); }

    /// Print summary report to stdout
    void report() const;

    /// Save detailed per-frame data to CSV
    void saveCsv(const std::string& path) const;

    /// Reset all recorded data
    void reset();

    /// Get last N frame timings for live display
    std::vector<FrameTiming> lastFrames(int n = 10) const;

private:
    std::string m_name;
    std::vector<FrameTiming> m_frames;
    int m_frameIndex = 0;

    // Live timing (per-frame, cleared at end of frame)
    std::chrono::steady_clock::time_point m_frameStart;
    std::chrono::steady_clock::time_point m_stageStart;
    std::string m_currentStage;
    std::vector<StageTiming> m_currentStages;

    mutable std::mutex m_mutex;
};

} // namespace iris
