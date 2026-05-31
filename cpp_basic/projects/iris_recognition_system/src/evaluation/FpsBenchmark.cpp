#include "evaluation/FpsBenchmark.h"
#include <iostream>
#include <fstream>
#include <iomanip>
#include <algorithm>
#include <numeric>

namespace iris {

FpsBenchmark::FpsBenchmark(const std::string& name)
    : m_name(name) {}

void FpsBenchmark::startFrame() {
    std::lock_guard<std::mutex> lock(m_mutex);
    m_currentStages.clear();
    m_frameStart = std::chrono::steady_clock::now();
}

void FpsBenchmark::startStage(const std::string& stageName) {
    std::lock_guard<std::mutex> lock(m_mutex);
    m_currentStage = stageName;
    m_stageStart = std::chrono::steady_clock::now();
}

void FpsBenchmark::endStage() {
    std::lock_guard<std::mutex> lock(m_mutex);
    auto now = std::chrono::steady_clock::now();
    double elapsed = std::chrono::duration<double, std::milli>(now - m_stageStart).count();
    StageTiming t;
    t.name    = m_currentStage;
    t.time_ms = elapsed;
    m_currentStages.push_back(t);
}

void FpsBenchmark::endFrame() {
    std::lock_guard<std::mutex> lock(m_mutex);
    auto now = std::chrono::steady_clock::now();
    double total = std::chrono::duration<double, std::milli>(now - m_frameStart).count();

    FrameTiming ft;
    ft.frame_index = m_frameIndex++;
    ft.total_ms    = total;
    ft.stages      = m_currentStages;
    m_frames.push_back(ft);

    // Keep only last 1000 frames to bound memory
    if (m_frames.size() > 1000) {
        m_frames.erase(m_frames.begin());
    }
}

double FpsBenchmark::avgFps() const {
    std::lock_guard<std::mutex> lock(m_mutex);
    if (m_frames.empty()) return 0.0;
    double totalMs = 0.0;
    for (const auto& f : m_frames) {
        totalMs += f.total_ms;
    }
    return 1000.0 * m_frames.size() / totalMs;
}

double FpsBenchmark::avgStageMs(const std::string& stageName) const {
    std::lock_guard<std::mutex> lock(m_mutex);
    double total = 0.0;
    int count = 0;
    for (const auto& frame : m_frames) {
        for (const auto& stage : frame.stages) {
            if (stage.name == stageName) {
                total += stage.time_ms;
                ++count;
            }
        }
    }
    return count > 0 ? total / count : 0.0;
}

void FpsBenchmark::stageStats(const std::string& stageName,
                               double& minMs, double& avgMs, double& maxMs) const {
    std::lock_guard<std::mutex> lock(m_mutex);
    minMs = std::numeric_limits<double>::max();
    maxMs = 0.0;
    double total = 0.0;
    int count = 0;
    for (const auto& frame : m_frames) {
        for (const auto& stage : frame.stages) {
            if (stage.name == stageName) {
                minMs = std::min(minMs, stage.time_ms);
                maxMs = std::max(maxMs, stage.time_ms);
                total += stage.time_ms;
                ++count;
            }
        }
    }
    avgMs = count > 0 ? total / count : 0.0;
    if (count == 0) minMs = 0.0;
}

double FpsBenchmark::totalElapsedMs() const {
    std::lock_guard<std::mutex> lock(m_mutex);
    double total = 0.0;
    for (const auto& f : m_frames) {
        total += f.total_ms;
    }
    return total;
}

void FpsBenchmark::report() const {
    std::lock_guard<std::mutex> lock(m_mutex);
    if (m_frames.empty()) {
        std::cout << "[FpsBenchmark] No frames recorded\n";
        return;
    }

    std::cout << "\n╔══════════════════════════════════════════════════╗\n"
              << "║  " << m_name << " FPS Benchmark Report          ║\n"
              << "╚══════════════════════════════════════════════════╝\n\n";

    std::cout << "Frames recorded: " << m_frames.size() << "\n";
    std::cout << "Average FPS:     " << std::fixed << std::setprecision(1) << avgFps() << "\n\n";

    // Collect all unique stage names
    std::vector<std::string> stageNames;
    for (const auto& frame : m_frames) {
        for (const auto& stage : frame.stages) {
            if (std::find(stageNames.begin(), stageNames.end(), stage.name) == stageNames.end()) {
                stageNames.push_back(stage.name);
            }
        }
    }

    // Print per-stage breakdown
    std::cout << std::left << std::setw(22) << "Stage"
              << std::right << std::setw(8) << "Min(ms)"
              << std::setw(8) << "Avg(ms)"
              << std::setw(8) << "Max(ms)" << "\n";
    std::cout << std::string(46, '-') << "\n";

    double sumAvg = 0.0;
    for (const auto& name : stageNames) {
        double minMs, avgMs, maxMs;
        stageStats(name, minMs, avgMs, maxMs);
        sumAvg += avgMs;
        std::cout << std::left << std::setw(22) << name
                  << std::right << std::fixed << std::setprecision(2)
                  << std::setw(8) << minMs
                  << std::setw(8) << avgMs
                  << std::setw(8) << maxMs << "\n";
    }
    std::cout << std::string(46, '-') << "\n"
              << std::left << std::setw(22) << "TOTAL (avg)"
              << std::right << std::setw(8) << "-"
              << std::setw(8) << std::fixed << std::setprecision(2) << sumAvg
              << std::setw(8) << "-" << "\n\n";
}

void FpsBenchmark::saveCsv(const std::string& path) const {
    std::lock_guard<std::mutex> lock(m_mutex);
    std::ofstream file(path);
    if (!file.is_open()) {
        std::cerr << "[FpsBenchmark] Cannot open CSV file: " << path << "\n";
        return;
    }

    // Collect stage names for CSV header
    std::vector<std::string> stageNames;
    for (const auto& frame : m_frames) {
        for (const auto& stage : frame.stages) {
            if (std::find(stageNames.begin(), stageNames.end(), stage.name) == stageNames.end()) {
                stageNames.push_back(stage.name);
            }
        }
    }

    // Header
    file << "frame,total_ms";
    for (const auto& name : stageNames) {
        file << "," << name << "_ms";
    }
    file << "\n";

    // Data rows
    for (const auto& frame : m_frames) {
        file << frame.frame_index << "," << frame.total_ms;
        for (const auto& name : stageNames) {
            double stageTime = 0.0;
            for (const auto& stage : frame.stages) {
                if (stage.name == name) {
                    stageTime = stage.time_ms;
                    break;
                }
            }
            file << "," << stageTime;
        }
        file << "\n";
    }

    std::cout << "[FpsBenchmark] CSV saved to " << path << "\n";
}

void FpsBenchmark::reset() {
    std::lock_guard<std::mutex> lock(m_mutex);
    m_frames.clear();
    m_currentStages.clear();
    m_frameIndex = 0;
}

std::vector<FrameTiming> FpsBenchmark::lastFrames(int n) const {
    std::lock_guard<std::mutex> lock(m_mutex);
    std::vector<FrameTiming> result;
    int start = std::max(0, static_cast<int>(m_frames.size()) - n);
    for (int i = start; i < static_cast<int>(m_frames.size()); ++i) {
        result.push_back(m_frames[i]);
    }
    return result;
}

} // namespace iris
