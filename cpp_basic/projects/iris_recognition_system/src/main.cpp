/**
 * Iris Recognition System - Main Entry Point
 *
 * C++20 implementation using classical computer vision techniques.
 * Follows the architecture defined in iris_recognition_cpp_nn_architecture.md
 *
 * Pipeline: Capture → Eye Detection → Quality → Segmentation →
 *           Normalization → Feature Extraction → Liveness → Matching → Decision
 */

#include "app/Constants.h"
#include "capture/CameraDevice.h"
#include "image/ImagePreprocessor.h"
#include "image/QualityChecker.h"
#include "image/IrisNormalizer.h"
#include "model/EyeDetector.h"
#include "model/IrisSegmenter.h"
#include "model/IrisFeatureExtractor.h"
#include "model/IrisLivenessDetector.h"
#include "matching/IrisMatcher.h"
#include "matching/DecisionEngine.h"
#include "persistence/Database.h"
#include "security/CryptoProvider.h"
#include "security/TemplateEncryptor.h"
#include "service/EnrollmentService.h"
#include "service/RecognitionService.h"
#include "service/LivenessService.h"
#include "service/AuditService.h"
#include "evaluation/Benchmark.h"
#include "evaluation/FpsBenchmark.h"
#ifdef HAS_ONNXRUNTIME
#include "nn/OnnxRuntimeEngine.h"
#endif
#include "domain/RecognitionResult.h"

#include <opencv2/core.hpp>
#include <opencv2/highgui.hpp>
#include <opencv2/imgproc.hpp>

#include <iostream>
#include <string>
#include <cstring>
#include <csignal>
#include <atomic>
#include <thread>
#include <chrono>
#include <iomanip>

using namespace iris;

// Global flag for signal handling
std::atomic<bool> g_running{true};

// Debug mode flag (set via --debug)
bool g_debug = false;

// Debug logging macro
#define DEBUG_LOG(msg) do { if (g_debug) { \
    std::cout << "[DEBUG] " << __FILE__ << ":" << __LINE__ << " | " << msg << std::endl; \
} } while(0)

// Trace log to track pipeline stages per frame
static int g_debugFrameCount = 0;
#define DEBUG_TRACE(msg) do { if (g_debug) { \
    std::cout << "[TRACE] frame=" << g_debugFrameCount << " | " << msg << std::endl; \
} } while(0)

void signalHandler(int /*sig*/) {
    g_running = false;
}

// ── Display helpers ────────────────────────────────────────────

void drawIrisOverlay(cv::Mat& display, const cv::Mat& eyeRoi,
                     int eyeX, int eyeY, int roiDisplaySize,
                     const IrisSegmentationResult& seg) {
    // Resize eye ROI for display
    cv::Mat eyeDisplay;
    cv::resize(eyeRoi, eyeDisplay, cv::Size(roiDisplaySize, roiDisplaySize));

    // Convert to color for overlay
    if (eyeDisplay.channels() == 1) {
        cv::cvtColor(eyeDisplay, eyeDisplay, cv::COLOR_GRAY2BGR);
    }

    // Draw iris circle
    if (seg.boundaries.valid()) {
        cv::Point ic(static_cast<int>(seg.boundaries.iris_center.x * roiDisplaySize / eyeRoi.cols),
                      static_cast<int>(seg.boundaries.iris_center.y * roiDisplaySize / eyeRoi.rows));
        int ir = static_cast<int>(seg.boundaries.iris_radius * roiDisplaySize / eyeRoi.cols);
        cv::circle(eyeDisplay, ic, ir, cv::Scalar(0, 255, 0), 2);

        cv::Point pc(static_cast<int>(seg.boundaries.pupil_center.x * roiDisplaySize / eyeRoi.cols),
                      static_cast<int>(seg.boundaries.pupil_center.y * roiDisplaySize / eyeRoi.rows));
        int pr = static_cast<int>(seg.boundaries.pupil_radius * roiDisplaySize / eyeRoi.cols);
        cv::circle(eyeDisplay, pc, pr, cv::Scalar(0, 0, 255), 2);
    }

    // Place in the display
    if (eyeX + roiDisplaySize <= display.cols &&
        eyeY + roiDisplaySize <= display.rows) {
        eyeDisplay.copyTo(display(cv::Rect(eyeX, eyeY, roiDisplaySize, roiDisplaySize)));
    }
}

void drawHud(cv::Mat& display, const FinalDecision& decision,
             const QualityResult& quality, const LivenessResult& liveness,
             const MatchResult& match, const std::string& mode,
             int fps) {
    int y = 25;
    int lineHeight = 22;
    auto putText = [&](const std::string& text, cv::Scalar color = cv::Scalar(0, 255, 0)) {
        cv::putText(display, text, cv::Point(10, y),
                    cv::FONT_HERSHEY_SIMPLEX, 0.5, color, 1, cv::LINE_AA);
        y += lineHeight;
    };

    putText("Iris Recognition System v" + std::string(APP_VERSION) +
            " | Mode: " + mode + " | FPS: " + std::to_string(fps),
            cv::Scalar(255, 255, 255));

    // Quality bar
    std::string qText = "Quality:  " + std::to_string(quality.overall).substr(0, 4);
    cv::Scalar qColor = quality.passed ? cv::Scalar(0, 255, 0) : cv::Scalar(0, 165, 255);
    putText(qText, qColor);

    // Liveness bar
    std::string lText = "Liveness: " + std::to_string(liveness.live_score).substr(0, 4) +
                        " [" + liveness.attack_type + "]";
    cv::Scalar lColor = liveness.passed ? cv::Scalar(0, 255, 0) : cv::Scalar(0, 0, 255);
    putText(lText, lColor);

    // Match bar
    if (!match.user_id.empty()) {
        std::string mText = "Match:    " + std::to_string(match.similarity).substr(0, 4) +
                            " (HD=" + std::to_string(match.hamming_dist).substr(0, 4) + ")";
        cv::Scalar mColor = match.matched ? cv::Scalar(0, 255, 0) : cv::Scalar(0, 165, 255);
        putText(mText, mColor);
    }

    // Decision
    if (!decision.reason.empty()) {
        cv::Scalar dColor = decision.accepted ? cv::Scalar(0, 255, 0) : cv::Scalar(0, 0, 255);
        std::string dText = decision.accepted
            ? "ACCEPTED: " + decision.username
            : "REJECTED: " + decision.reason.substr(0, 60);
        putText(dText, dColor);
    }
}

void printHelp(const char* progName) {
    std::cout << "Iris Recognition System v" << APP_VERSION << "\n\n"
              << "Usage: " << progName << " [options]\n\n"
              << "Options:\n"
              << "  --mode <enroll|identify>   Operating mode (default: identify)\n"
              << "  --camera <id>              Camera device ID (default: 0)\n"
              << "  --image <path>             Process a single image file\n"
              << "  --video <path>             Process a video file\n"
              << "  --user <name>              Username for enrollment\n"
              << "  --config <path>            Configuration file path\n"
              << "  --db <path>                Database file path\n"
              << "  --threshold <value>        Matching threshold (default: 0.35)\n"
              << "  --no-display               Run without GUI display\n"
              << "  --list-users               List all enrolled users\n"
              << "  --benchmark <dir>          Run accuracy benchmark on dataset\n"
              << "  --benchmark-synthetic <img> Run synthetic benchmark\n"
              << "  --export-roc <path>        Export ROC curve data to CSV\n"
              << "  --test-onnx <model.onnx>    Test ONNX Runtime inference\n"
              << "  --yolo-model <model.onnx>   Path to YOLO face detection ONNX model\n"
              << "  --detector-mode <haar|yolo> Eye detection backend (default: haar)\n"
              << "  --benchmark-fps             Enable real-time FPS benchmarking\n"
              << "  --debug                     Enable verbose debug output\n"
              << "  --help                      Show this help\n\n"
              << "Examples:\n"
              << "  " << progName << " --detector-mode yolo --yolo-model models/face.onnx\n"
              << "  " << progName << " --benchmark-fps --no-display\n"
              << "  " << progName << " --debug --detector-mode yolo\n\n"
              << "Controls (in GUI window):\n"
              << "  q / ESC     Quit\n"
              << "  e           Switch to enrollment mode\n"
              << "  i           Switch to identification mode\n"
              << "  s           Save current frame\n"
              << std::endl;
}

// ── Main ───────────────────────────────────────────────────────

int main(int argc, char* argv[]) {
    // Install signal handlers
    std::signal(SIGINT, signalHandler);
    std::signal(SIGTERM, signalHandler);

    // ── Parse arguments ────────────────────────────────────────
    std::string mode          = "identify";
    std::string username      = "user";
    std::string imagePath;
    std::string videoPath;
    std::string configPath    = "config/app_config.json";
    std::string dbPath        = "iris_data.db";
    std::string benchmarkPath;
    std::string benchmarkSyntheticPath;
    std::string exportRocPath;
    std::string testOnnxPath;
    std::string yoloModelPath;
    std::string detectorMode   = "haar";  // "haar" or "yolo"
    int cameraId              = 0;
    float matchThreshold      = DEFAULT_MATCHING_THRESHOLD;
    bool showDisplay          = true;
    bool listUsers            = false;
    bool debugMode            = false;
    bool benchmarkFps         = false;

    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--mode" && i + 1 < argc)        mode = argv[++i];
        else if (arg == "--camera" && i + 1 < argc) cameraId = std::stoi(argv[++i]);
        else if (arg == "--image" && i + 1 < argc)  imagePath = argv[++i];
        else if (arg == "--video" && i + 1 < argc)  videoPath = argv[++i];
        else if (arg == "--user" && i + 1 < argc)   username = argv[++i];
        else if (arg == "--config" && i + 1 < argc) configPath = argv[++i];
        else if (arg == "--db" && i + 1 < argc)     dbPath = argv[++i];
        else if (arg == "--threshold" && i + 1 < argc) matchThreshold = std::stof(argv[++i]);
        else if (arg == "--benchmark" && i + 1 < argc) benchmarkPath = argv[++i];
        else if (arg == "--benchmark-synthetic" && i + 1 < argc) benchmarkSyntheticPath = argv[++i];
        else if (arg == "--export-roc" && i + 1 < argc) exportRocPath = argv[++i];
        else if (arg == "--test-onnx" && i + 1 < argc) testOnnxPath = argv[++i];
        else if (arg == "--yolo-model" && i + 1 < argc) yoloModelPath = argv[++i];
        else if (arg == "--detector-mode" && i + 1 < argc) detectorMode = argv[++i];
        else if (arg == "--no-display")             showDisplay = false;
        else if (arg == "--list-users")             listUsers = true;
        else if (arg == "--debug")                  debugMode = true;
        else if (arg == "--benchmark-fps")          benchmarkFps = true;
        else if (arg == "--help")                   { printHelp(argv[0]); return 0; }
        else {
            std::cerr << "Unknown option: " << arg << "\n";
            printHelp(argv[0]);
            return 1;
        }
    }

    std::cout << "╔══════════════════════════════════════════════════╗\n"
              << "║     Iris Recognition System v" << APP_VERSION << "                  ║\n"
              << "║     C++20 + OpenCV Classical CV Pipeline        ║\n"
              << "╚══════════════════════════════════════════════════╝\n\n";

    // Set global debug flag
    g_debug = debugMode;
    if (g_debug) {
        std::cout << "[Debug] Debug mode enabled! Use this with GDB.\n"
                  << "[Debug] Set breakpoints, then 'continue' to run.\n"
                  << "[Debug] Key variables: g_debug, g_running, g_debugFrameCount\n\n";
    }

    // ── Initialize modules ─────────────────────────────────────
    std::cout << "[Init] Initializing modules...\n";

    // Security
    CryptoProvider crypto;
    crypto.initialize("iris_recognition_device_key_v1");
    TemplateEncryptor encryptor(crypto);

    // Database
    Database database;
    database.open(dbPath);

    if (listUsers) {
        auto users = database.getAllUsers();
        std::cout << "\nEnrolled users (" << users.size() << "):\n";
        for (const auto& u : users) {
            auto templates = database.getTemplatesForUser(u.id);
            std::cout << "  - " << u.username << " (id=" << u.id
                      << ", templates=" << templates.size() << ")\n";
        }
        return 0;
    }

    // Eye detector
    EyeDetector eyeDetector;
    if (detectorMode == "yolo" && !yoloModelPath.empty()) {
        if (eyeDetector.enableYolo(yoloModelPath)) {
            std::cout << "[Init] YOLO detector loaded: " << yoloModelPath << "\n";
        } else {
            std::cerr << "[Init] YOLO failed, falling back to Haar cascade\n";
            eyeDetector.initialize();
        }
    } else {
        eyeDetector.initialize();
    }

    // Quality checker
    QualityChecker qualityChecker(DEFAULT_QUALITY_THRESHOLD);

    // Iris segmenter
    IrisSegmenter segmenter;

    // Iris normalizer
    IrisNormalizer normalizer(NORMALIZED_WIDTH, NORMALIZED_HEIGHT);

    // Feature extractor
    IrisFeatureExtractor featureExtractor;

    // Liveness detector
    IrisLivenessDetector livenessDetector;

    // Matcher
    IrisMatcher matcher(matchThreshold);

    // Decision engine
    DecisionEngine decisionEngine;

    // Services
    EnrollmentService enrollmentService(
        eyeDetector, qualityChecker, segmenter, normalizer,
        featureExtractor, livenessDetector, database, encryptor);

    RecognitionService recognitionService(
        eyeDetector, qualityChecker, segmenter, normalizer,
        featureExtractor, livenessDetector, matcher, decisionEngine, database);

    AuditService auditService(database);

    std::cout << "[Init] All modules initialized.\n";
    std::cout << "[Init] Mode: " << mode << "\n";
    std::cout << "[Init] Backend: Classical CV (Gabor filter IrisCode)\n\n";

    // ── Benchmark mode ─────────────────────────────────────────
    if (!benchmarkPath.empty() || !benchmarkSyntheticPath.empty()) {
        Benchmark benchmark(eyeDetector, qualityChecker, segmenter,
                            normalizer, featureExtractor, matcher);

        BenchmarkResult benchResult;

        if (!benchmarkPath.empty()) {
            // Run on dataset
            benchResult = benchmark.run(benchmarkPath);
        } else {
            // Run synthetic benchmark
            benchResult = benchmark.runSynthetic(benchmarkSyntheticPath, 15);
        }

        // Export ROC data if requested
        if (!exportRocPath.empty()) {
            benchmark.exportRocCsv(benchResult, exportRocPath);
            benchmark.exportDetCsv(benchResult,
                exportRocPath.substr(0, exportRocPath.find_last_of('.')) + "_det.csv");
        }

        database.close();
        return 0;
    }

    // ── ONNX Runtime test ──────────────────────────────────────
    if (!testOnnxPath.empty()) {
#ifdef HAS_ONNXRUNTIME
        std::cout << "[Test] ONNX Runtime test mode\n"
                  << "[Test] Loading model: " << testOnnxPath << "\n";

        OnnxRuntimeEngine engine;
        if (engine.loadModel(testOnnxPath)) {
            std::cout << "[Test] ✅ Model loaded successfully!\n"
                      << "[Test] Backend: " << engine.backendName() << "\n"
                      << "[Test] Inputs:  " << engine.numInputs() << "\n"
                      << "[Test] Outputs: " << engine.numOutputs() << "\n";

            // Print input names and shapes
            auto inNames = engine.inputNames();
            for (size_t i = 0; i < inNames.size(); ++i) {
                auto shape = engine.getInputShape(i);
                std::cout << "[Test]   Input[" << i << "]: " << inNames[i]
                          << " shape=[";
                for (size_t j = 0; j < shape.size(); ++j) {
                    if (j > 0) std::cout << ",";
                    std::cout << shape[j];
                }
                std::cout << "]\n";
            }

            // Run a test inference with random data
            auto inputShape = engine.getInputShape(0);
            // Replace dynamic dims with concrete values
            size_t totalElements = 1;
            for (auto& d : inputShape) {
                if (d <= 0) d = 1;  // replace dynamic dims with 1
                totalElements *= static_cast<size_t>(d);
            }

            std::vector<float> testInput(totalElements, 0.5f);
            auto output = engine.infer(testInput, inputShape);

            if (!output.empty()) {
                std::cout << "[Test] ✅ Inference successful!\n"
                          << "[Test] Output size: " << output.size() << " elements\n"
                          << "[Test] Output[0..4]: ";
                for (size_t i = 0; i < std::min(size_t(5), output.size()); ++i) {
                    std::cout << output[i] << " ";
                }
                std::cout << "\n";
            } else {
                std::cout << "[Test] ❌ Inference returned no output\n";
            }
        } else {
            std::cout << "[Test] ❌ Failed to load model: " << testOnnxPath << "\n";
        }
#else
        std::cout << "[Test] ❌ ONNX Runtime support not compiled in.\n"
                  << "       Rebuild with ONNX Runtime headers available.\n";
#endif
        database.close();
        return 0;
    }

    // ── Open camera ────────────────────────────────────────────
    CameraDevice camera;

    if (!imagePath.empty()) {
        // Single image mode
        std::cout << "[Main] Processing image: " << imagePath << "\n";
        cv::Mat frame = cv::imread(imagePath);
        if (frame.empty()) {
            std::cerr << "Failed to load image: " << imagePath << "\n";
            return 1;
        }

        FinalDecision decision;
        if (mode == "enroll") {
            enrollmentService.processFrame(frame);
            enrollmentService.finalizeEnrollment(username);
        } else {
            decision = recognitionService.processFrame(frame);
        }

        std::cout << "\n=== Results ===\n"
                  << "Quality:  " << recognitionService.getLastQuality().overall << "\n"
                  << "Liveness: " << recognitionService.getLastLiveness().live_score
                  << " (" << recognitionService.getLastLiveness().attack_type << ")\n"
                  << "Match:    " << recognitionService.getLastMatch().similarity << "\n"
                  << "Decision: " << (decision.accepted ? "ACCEPTED" : "REJECTED") << "\n"
                  << "Reason:   " << decision.reason << "\n";

        if (showDisplay) {
            cv::Mat display = frame.clone();
            if (display.cols > 800) {
                cv::resize(display, display, cv::Size(800, 600));
            }
            drawHud(display, decision,
                    recognitionService.getLastQuality(),
                    recognitionService.getLastLiveness(),
                    recognitionService.getLastMatch(),
                    mode, 0);
            cv::imshow("Iris Recognition", display);
            std::cout << "\nPress any key to exit...\n";
            cv::waitKey(0);
        }

        auditService.log("image_process", "", "Processed image: " + imagePath);
        return 0;
    }

    if (!videoPath.empty()) {
        camera.openFile(videoPath);
    } else {
        if (!camera.open(cameraId, DEFAULT_CAMERA_WIDTH, DEFAULT_CAMERA_HEIGHT)) {
            std::cerr << "[Main] Failed to open camera. Trying default...\n";
            if (!camera.open(0)) {
                std::cerr << "[Main] No camera available.\n";
                return 1;
            }
        }
    }
    camera.startCapture();

    // ── Create display window ──────────────────────────────────
    if (showDisplay) {
        cv::namedWindow("Iris Recognition System", cv::WINDOW_NORMAL);
        cv::resizeWindow("Iris Recognition System", 1024, 680);
    }

    // ── Main loop ──────────────────────────────────────────────
    cv::Mat frame;
    int frameCount = 0;
    auto lastTime = std::chrono::steady_clock::now();
    int fps = 0;
    std::string currentUser = username;

    // FPS Benchmark
    FpsBenchmark fpsBenchmark("IrisRecognition");
    int benchmarkReportInterval = 100;  // Report every 100 frames

    std::cout << "[Main] Starting main loop. Press 'q' or ESC to quit.\n";
    std::cout << "[Main] Press 'e' for enrollment, 'i' for identification.\n\n";

    while (g_running) {
        if (!camera.getFrame(frame) || frame.empty()) {
            if (!videoPath.empty()) break;  // video ended
            std::this_thread::sleep_for(std::chrono::milliseconds(10));
            continue;
        }

        ++frameCount;
        g_debugFrameCount = frameCount;
        DEBUG_TRACE("--- Frame " << frameCount << " START ---");
        DEBUG_TRACE("Frame acquired: " << frame.cols << "x" << frame.rows
                     << " channels=" << frame.channels() << " depth=" << frame.depth());

        // FPS calculation
        auto now = std::chrono::steady_clock::now();
        auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(
            now - lastTime).count();
        if (elapsed >= 1000) {
            fps = static_cast<int>(frameCount * 1000.0 / elapsed);
            frameCount = 0;
            lastTime = now;
        }

        // Process frame
        FinalDecision decision;
        cv::Mat eyeRoi;

        DEBUG_TRACE("Processing mode=" << mode);

        if (benchmarkFps) fpsBenchmark.startFrame();

        if (mode == "enroll") {
            if (benchmarkFps) fpsBenchmark.startStage("enrollment");
            bool completed = enrollmentService.processFrame(frame);
            if (benchmarkFps) fpsBenchmark.endStage();
            DEBUG_TRACE("Enrollment processFrame returned, completed=" << completed
                         << " progress=" << enrollmentService.getProgress()
                         << "/" << enrollmentService.getRequiredFrames());

            if (completed) {
                enrollmentService.finalizeEnrollment(currentUser);
                std::cout << "[Main] Enrollment completed for user: "
                          << currentUser << "\n";
                mode = "identify";  // Switch to identify mode
            }
        } else {
            if (benchmarkFps) fpsBenchmark.startStage("recognition");
            decision = recognitionService.processFrame(frame);
            if (benchmarkFps) fpsBenchmark.endStage();
            DEBUG_TRACE("Recognition decision: accepted=" << decision.accepted
                         << " username=" << decision.username
                         << " reason=" << decision.reason);
        }

        if (benchmarkFps) fpsBenchmark.endFrame();

        // Get eye ROI for display
        eyeRoi = eyeDetector.getBestEyeRoi(frame);
        DEBUG_TRACE("Eye ROI: " << eyeRoi.cols << "x" << eyeRoi.rows
                     << (eyeRoi.empty() ? " (EMPTY)" : ""));

        auto segResult = segmenter.segment(eyeRoi);
        DEBUG_TRACE("Segmentation: valid=" << segResult.boundaries.valid()
                     << " iris_center=(" << segResult.boundaries.iris_center.x
                     << "," << segResult.boundaries.iris_center.y
                     << ") iris_r=" << segResult.boundaries.iris_radius
                     << " pupil_r=" << segResult.boundaries.pupil_radius);

        // ── Display ────────────────────────────────────────────
        if (showDisplay) {
            cv::Mat display(680, 1024, CV_8UC3, cv::Scalar(30, 30, 30));

            // Main frame (resized) - fit within left panel area
            cv::Mat frameDisplay;
            double maxDisplayW = 600.0;
            double maxDisplayH = 440.0;
            double scale = std::min({maxDisplayW / frame.cols, maxDisplayH / frame.rows, 1.0});
            cv::resize(frame, frameDisplay,
                       cv::Size(static_cast<int>(frame.cols * scale),
                                static_cast<int>(frame.rows * scale)));

            int fx = 10, fy = 180;
            // Ensure frame fits in display
            int fw = std::min(frameDisplay.cols, display.cols - fx);
            int fh = std::min(frameDisplay.rows, display.rows - fy);
            if (fw > 0 && fh > 0) {
                cv::Mat subDisplay = display(cv::Rect(fx, fy, fw, fh));
                frameDisplay(cv::Rect(0, 0, fw, fh)).copyTo(subDisplay);
            }

            // Eye ROI with overlay (right side of frame)
            int roiSize = 200;
            int eyeX = fx + fw + 20;
            // Clamp eye ROI to fit in display
            if (eyeX + roiSize > display.cols) {
                eyeX = display.cols - roiSize - 10;
            }
            if (eyeX < fx + fw + 5) {
                // Not enough space on the right, place below frame
                eyeX = fx;
                fy = fy + fh + 10;
            }
            drawIrisOverlay(display, eyeRoi, eyeX, fy, roiSize, segResult);

            // Normalized iris display
            if (mode != "enroll") {
                auto normResult = normalizer.normalize(eyeRoi, segResult.boundaries);
                if (!normResult.image.empty()) {
                    cv::Mat normDisplay;
                    normResult.image.convertTo(normDisplay, CV_8U, 255.0);
                    int normW = std::min(400, display.cols - eyeX - 10);
                    int normH = 50;
                    cv::resize(normDisplay, normDisplay, cv::Size(normW, normH));
                    cv::cvtColor(normDisplay, normDisplay, cv::COLOR_GRAY2BGR);
                    int ny = fy + roiSize + 20;
                    // Clamp to display bounds
                    if (ny + normH > display.rows) {
                        ny = display.rows - normH - 10;
                    }
                    if (eyeX + normW <= display.cols && ny >= 0) {
                        normDisplay.copyTo(display(cv::Rect(eyeX, ny, normW, normH)));
                    }
                }
            }

            // HUD overlay
            auto lastQuality   = (mode == "enroll")
                ? enrollmentService.getLastQuality()
                : recognitionService.getLastQuality();
            auto lastLiveness  = (mode != "enroll")
                ? recognitionService.getLastLiveness()
                : LivenessResult{};
            auto lastMatch     = (mode != "enroll")
                ? recognitionService.getLastMatch()
                : MatchResult{};

            drawHud(display, decision, lastQuality, lastLiveness, lastMatch, mode, fps);

            // Enrollment progress
            if (mode == "enroll") {
                int prog = enrollmentService.getProgress();
                int req  = enrollmentService.getRequiredFrames();
                std::string progText = "Enrollment: " + std::to_string(prog) +
                                       "/" + std::to_string(req) + " frames";
                cv::putText(display, progText, cv::Point(10, display.rows - 20),
                            cv::FONT_HERSHEY_SIMPLEX, 0.6, cv::Scalar(255, 200, 0), 1, cv::LINE_AA);

                // Progress bar
                int barY = display.rows - 10;
                int barW = 200;
                cv::rectangle(display, cv::Rect(10, barY - 12, barW, 8),
                              cv::Scalar(80, 80, 80), -1);
                int filled = static_cast<int>(static_cast<float>(barW) * prog / req);
                if (filled > 0) {
                    cv::rectangle(display,
                        cv::Rect(10, barY - 12, filled, 8),
                        cv::Scalar(0, 200, 0), -1);
                }
            }

            cv::imshow("Iris Recognition System", display);
        }

        // Handle keyboard input
        int key = cv::waitKey(1) & 0xFF;
        if (key != 0xFF) {
            DEBUG_TRACE("Key pressed: 0x" << std::hex << key << std::dec);
        }
        if (key == 'q' || key == 27) {  // q or ESC
            DEBUG_LOG("Quit key pressed");
            g_running = false;
        } else if (key == 'e') {
            DEBUG_LOG("Switching to enrollment mode");
            mode = "enroll";
            enrollmentService.reset();
            std::cout << "[Main] Switched to enrollment mode.\n"
                      << "       Enter username: " << std::flush;
            // Non-blocking: use default or previously set username
            std::cout << currentUser << "\n";
        } else if (key == 'i') {
            DEBUG_LOG("Switching to identification mode");
            mode = "identify";
            std::cout << "[Main] Switched to identification mode.\n";
        } else if (key == 's') {
            std::string filename = "iris_capture_" +
                std::to_string(std::chrono::system_clock::now().time_since_epoch().count()) +
                ".png";
            cv::imwrite(filename, frame);
            DEBUG_LOG("Frame saved to " << filename);
            std::cout << "[Main] Saved frame: " << filename << "\n";
        }
        DEBUG_TRACE("--- Frame " << frameCount << " END ---");

        // Periodic FPS benchmark report
        if (benchmarkFps && fpsBenchmark.totalFrames() > 0 &&
            fpsBenchmark.totalFrames() % benchmarkReportInterval == 0) {
            std::cout << "[FPS] " << std::fixed << std::setprecision(1)
                      << fpsBenchmark.avgFps() << " FPS (avg over "
                      << fpsBenchmark.totalFrames() << " frames)\n";
        }
    }

    // ── FPS Benchmark final report ──────────────────────────────
    if (benchmarkFps && fpsBenchmark.totalFrames() > 0) {
        fpsBenchmark.report();
    }

    // ── Cleanup ────────────────────────────────────────────────
    std::cout << "\n[Main] Shutting down...\n";
    camera.stopCapture();
    database.close();
    cv::destroyAllWindows();

    std::cout << "[Main] Iris Recognition System terminated.\n";
    return 0;
}
