#include "Common.h"
#include "camera/CameraManager.h"
#include "detection/FaceDetector.h"
#include "recognition/FaceRecognizer.h"
#include "pipeline/ThreadPool.h"
#include "pipeline/RingBuffer.h"
#include "ui/DisplayManager.h"

#include <csignal>
#include <atomic>
#include <iostream>
#include <thread>
#include <getopt.h>

using namespace efr;

// 全局信号处理
static std::atomic<bool> g_running{true};
static std::atomic<bool> g_processing_done{false};

static void signalHandler(int signum) {
    Logger::info("收到信号 {}, 正在退出...", signum);
    g_running = false;
}

// ---- Logger 静态成员初始化 ----

LogLevel Logger::min_level = LogLevel::Info;
bool Logger::use_color = true;

// ---- SystemConfig 命令行解析 ----

void SystemConfig::printUsage(const char* prog) {
    std::cout << R"(
嵌入式人脸检测与识别系统 v1.0
================================

用法: )" << prog << R"( [选项]

输入源:
  -c, --camera <id>       使用摄像头设备 (默认: 0)
  -v, --video <path>      使用视频文件
  -i, --images <dir>      使用图片目录

检测参数:
  --detector <backend>    检测后端: hog (默认), haar, cnn
  --hog-threshold <val>   HOG 检测阈值 (-1.0 ~ 1.0, 默认: 0.0)
  --haar-scale <val>      Haar 缩放因子 (默认: 1.1)
  --haar-neighbors <n>    Haar 最少邻接数 (默认: 3)

识别参数:
  --recognize             启用人脸识别
  --recog-threshold <val> 识别阈值 (0.0 ~ 1.0, 默认: 0.6)
  --shape-predictor <p>   Shape predictor 模型路径
  --face-model <p>        Face recognition 模型路径
  --known-faces <dir>     已知人脸目录

流水线参数:
  --threads <n>           处理线程数 (默认: 4)
  --buffer-size <n>       Ring buffer 大小 (默认: 8)
  --no-display            不显示 GUI (headless 模式)

其他:
  -h, --help              显示帮助
  --verbose               详细输出
  --fps <n>               目标帧率 (默认: 30)

示例:
  # 使用默认摄像头检测人脸
  )" << prog << R"(

  # 使用视频文件并启用人脸识别
  )" << prog << R"( -v test.mp4 --recognize --known-faces ./known_people/

  # 使用 Haar Cascade 检测器
  )" << prog << R"( --detector haar

按键:
  ESC / q  退出程序
)" << std::endl;
}

SystemConfig SystemConfig::fromArgs(int argc, char* argv[]) {
    SystemConfig cfg;

    static struct option long_options[] = {
        {"camera",          required_argument, nullptr, 'c'},
        {"video",           required_argument, nullptr, 'v'},
        {"images",          required_argument, nullptr, 'i'},
        {"detector",        required_argument, nullptr, 'd'},
        {"hog-threshold",   required_argument, nullptr, 't'},
        {"haar-scale",      required_argument, nullptr, 's'},
        {"haar-neighbors",  required_argument, nullptr, 'n'},
        {"recognize",       no_argument,       nullptr, 'r'},
        {"recog-threshold", required_argument, nullptr, 'T'},
        {"shape-predictor", required_argument, nullptr, 'p'},
        {"face-model",      required_argument, nullptr, 'm'},
        {"known-faces",     required_argument, nullptr, 'k'},
        {"threads",         required_argument, nullptr, 'j'},
        {"buffer-size",     required_argument, nullptr, 'b'},
        {"no-display",      no_argument,       nullptr, 'D'},
        {"help",            no_argument,       nullptr, 'h'},
        {"verbose",         no_argument,       nullptr, 'V'},
        {"fps",             required_argument, nullptr, 'f'},
        {nullptr, 0, nullptr, 0}
    };

    int opt;
    int option_index = 0;
    while ((opt = getopt_long(argc, argv, "c:v:i:hrVD", long_options, &option_index)) != -1) {
        switch (opt) {
        case 'c':
            cfg.input_source = InputSource::Camera;
            cfg.camera_id = std::stoi(optarg);
            break;
        case 'v':
            cfg.input_source = InputSource::VideoFile;
            cfg.video_path = optarg;
            break;
        case 'i':
            cfg.input_source = InputSource::ImageDirectory;
            cfg.image_dir = optarg;
            break;
        case 'd':
            if (std::string(optarg) == "hog") cfg.detector_backend = DetectorBackend::DlibHOG;
            else if (std::string(optarg) == "haar") cfg.detector_backend = DetectorBackend::OpenCVHaar;
            else if (std::string(optarg) == "cnn") cfg.detector_backend = DetectorBackend::DlibCNN;
            else Logger::warn("未知检测后端: {}, 使用 HOG", optarg);
            break;
        case 't':
            cfg.hog_detection_threshold = std::stof(optarg);
            break;
        case 's':
            cfg.haar_scale_factor = std::stof(optarg);
            break;
        case 'n':
            cfg.haar_min_neighbors = std::stoi(optarg);
            break;
        case 'r':
            cfg.enable_recognition = true;
            break;
        case 'T':
            cfg.recognition_threshold = std::stof(optarg);
            break;
        case 'p':
            cfg.shape_predictor_path = optarg;
            break;
        case 'm':
            cfg.face_recognition_model_path = optarg;
            break;
        case 'k':
            cfg.known_faces_dir = optarg;
            break;
        case 'j':
            cfg.thread_pool_size = std::stoi(optarg);
            break;
        case 'b':
            cfg.ring_buffer_capacity = std::stoi(optarg);
            break;
        case 'D':
            cfg.enable_display = false;
            break;
        case 'f':
            cfg.target_fps = std::stoi(optarg);
            break;
        case 'V':
            cfg.verbose = true;
            Logger::min_level = LogLevel::Debug;
            break;
        case 'h':
        default:
            printUsage(argv[0]);
            exit(0);
        }
    }

    return cfg;
}

// ============================================================
// 主流水线
// ============================================================

int main(int argc, char* argv[]) {
    // 信号处理
    std::signal(SIGINT, signalHandler);
    std::signal(SIGTERM, signalHandler);

    // 解析配置
    SystemConfig config = SystemConfig::fromArgs(argc, argv);

    Logger::info("========================================");
    Logger::info("  嵌入式人脸检测与识别系统 v1.0");
    Logger::info("========================================");
    Logger::info("检测后端: {}",
                 config.detector_backend == DetectorBackend::DlibHOG ? "dlib HOG" :
                 config.detector_backend == DetectorBackend::OpenCVHaar ? "OpenCV Haar" : "dlib CNN");
    Logger::info("识别功能: {}", config.enable_recognition ? "启用" : "禁用");
    Logger::info("线程池大小: {}", config.thread_pool_size);
    Logger::info("Ring Buffer 容量: {}", config.ring_buffer_capacity);

    // ---- 模块初始化 ----

    // 1. 摄像头/视频管理器
    CameraManager camera;
    if (!camera.initialize(config)) {
        Logger::error("无法打开输入源");
        Logger::info("提示: 使用 -v <视频文件> 指定视频, 或 -c <id> 指定摄像头");
        return 1;
    }

    // 2. 人脸检测器
    FaceDetector detector;
    if (!detector.initialize(config)) {
        Logger::error("人脸检测器初始化失败");
        return 1;
    }

    // 3. 人脸识别器（可选）
    FaceRecognizer recognizer;
    if (config.enable_recognition) {
        recognizer.initialize(config);
        if (recognizer.isAvailable()) {
            Logger::info("人脸识别器已就绪, 已知 {} 人", recognizer.knownFaceCount());
        } else {
            Logger::warn("人脸识别不可用, 仅执行检测");
        }
    }

    // 4. 显示管理器
    DisplayManager display;
    if (config.enable_display) {
        display.initialize(config);
    }

    // 5. 线程池
    ThreadPool thread_pool(config.thread_pool_size);

    // 6. 流水线 Ring Buffers
    RingBuffer<FramePtr> raw_buffer(config.ring_buffer_capacity);     // Camera -> Processing
    RingBuffer<FramePtr> result_buffer(config.ring_buffer_capacity);  // Processing -> Display

    Logger::info("流水线已就绪, 开始处理...");
    Logger::info("按 ESC 或 q 退出");

    // ---- 摄像头采集线程 ----
    std::thread capture_thread([&]() {
        FPSCounter capture_fps;
        while (g_running && camera.isOpened()) {
            FramePtr frame;
            if (!camera.captureFrame(frame)) {
                // 视频结束
                if (camera.getTotalFrames() > 0) {
                    Logger::info("视频播放完毕");
                }
                g_running = false;
                break;
            }

            capture_fps.tick();

            // 推入原始帧缓冲区（阻塞，超时 100ms）
            if (!raw_buffer.push(frame, std::chrono::milliseconds(100))) {
                Logger::warn("帧缓冲已满, 丢弃帧 #{}", frame ? frame->frame_index : -1);
            }
        }

        // 唤醒等待线程
        raw_buffer.notifyAll();
        result_buffer.notifyAll();
        Logger::info("采集线程已退出");
    });

    // ---- 处理线程：调度检测和识别任务到线程池 ----
    FPSCounter process_fps;

    std::thread processing_thread([&]() {
        bool capture_done = false;

        while (!capture_done) {
            // 非阻塞轮询，同时检查退出条件
            auto frame_opt = raw_buffer.pop(std::chrono::milliseconds(50));
            if (!frame_opt.has_value()) {
                if (!g_running) capture_done = true;
                continue;
            }

            auto frame = frame_opt.value();

            // 提交到线程池：检测 + 识别
            thread_pool.submit([&detector, &recognizer, &result_buffer, &process_fps, frame]() {
                // 人脸检测
                int num_faces = detector.detect(*frame);

                // 人脸识别（如果启用且可用）
                if (num_faces > 0 && recognizer.isAvailable()) {
                    recognizer.recognize(*frame);
                }

                process_fps.tick();

                // 推入结果缓冲
                if (!result_buffer.push(frame, std::chrono::milliseconds(10))) {
                    // 显示端来不及消费，丢弃
                }
            });
        }

        // 等待所有提交的任务完成
        thread_pool.waitAll();
        g_processing_done.store(true);

        result_buffer.notifyAll();
        Logger::info("处理线程已退出");
    });

    // ---- 主线程：显示 ----
    int total_faces_detected = 0;
    int displayed_frames = 0;
    bool draining = false;

    while (true) {
        // 检查是否退出（ESC / q）
        if (config.enable_display && display.shouldQuit()) {
            g_running = false;
            break;
        }

        auto result_opt = result_buffer.pop(std::chrono::milliseconds(draining ? 20 : 33));
        if (!result_opt.has_value()) {
            if (draining && g_processing_done.load()) {
                break; // 排空完成且处理已结束
            }
            if (!draining && !g_running) {
                draining = true;
                Logger::info("输入结束，排空流水线...");
                continue;
            }
            continue;
        }

        auto frame = result_opt.value();
        displayed_frames++;
        total_faces_detected += frame->faces.size();

        display.setProcessingFPS(process_fps.fps());

        if (config.enable_display) {
            display.render(*frame);
        } else {
            static auto last_report = std::chrono::steady_clock::now();
            auto now = std::chrono::steady_clock::now();
            if (std::chrono::duration<double>(now - last_report).count() >= 1.0) {
                Logger::info("Frame #{} | 检测: {} 人脸 | 累计: {} 帧, {} 人脸 | 处理 FPS: {}",
                             frame->frame_index, frame->faces.size(),
                             displayed_frames, total_faces_detected,
                             process_fps.fps());
                last_report = now;
            }
        }
    }

    Logger::info("========================================");
    Logger::info("  处理完成统计");
    Logger::info("  显示帧数: {}", displayed_frames);
    Logger::info("  检测人脸总数: {}", total_faces_detected);
    Logger::info("  平均处理 FPS: {}", process_fps.fps());
    Logger::info("========================================");

    // ---- 清理 ----
    Logger::info("正在关闭...");

    g_running = false;
    raw_buffer.notifyAll();
    result_buffer.notifyAll();

    thread_pool.shutdown();

    if (capture_thread.joinable()) capture_thread.join();
    if (processing_thread.joinable()) processing_thread.join();

    display.close();
    camera.release();

    Logger::info("系统已退出");
    return 0;
}
