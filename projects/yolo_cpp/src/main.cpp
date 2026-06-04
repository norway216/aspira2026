/**
 * @file    main.cpp
 * @brief   YOLO 分割模型 ONNX Runtime C++ 推理 —— 主程序入口
 *
 * ============================================================================
 * 功能说明
 * ============================================================================
 * 本程序演示如何使用 YoloSegInference 类在 C++ 中完成 YOLOv8 分割模型的推理。
 *
 * 运行方式：
 *   ./yolo_seg_inference [图片路径] [模型路径]
 *
 * 默认参数：
 *   - 图片: ../images/road.jpg
 *   - 模型: ../models/yolov8n-seg.onnx
 *
 * 输出：
 *   - 控制台：每步耗时统计
 *   - 文件: ../images/road_result.jpg（带分割Mask和检测框的可视化结果）
 *
 * 依赖：
 *   - OpenCV 4.x (core, imgproc, imgcodecs)
 *   - ONNX Runtime 1.18.0+
 * ============================================================================
 */

#include "yolo_seg_inference.h"

#include <opencv2/imgcodecs.hpp>  // cv::imread, cv::imwrite

#include <iostream>    // std::cout, std::cerr
#include <chrono>      // 总耗时计时
#include <cstdlib>     // std::getenv
#include <iomanip>     // std::fixed, std::setprecision

// ============================================================================
// 辅助函数：打印配置信息
// ============================================================================

void printConfig(const InferenceConfig& config)
{
    std::cout << "╔════════════════════════════════════════════════════════╗\n";
    std::cout << "║       YOLOv8-seg ONNX Runtime C++ 推理引擎            ║\n";
    std::cout << "╠════════════════════════════════════════════════════════╣\n";
    std::cout << "║  模型输入尺寸 : " << config.inputWidth << " × "
              << config.inputHeight << "                               ║\n";
    std::cout << "║  类别数       : " << config.numClasses
              << " (COCO)                                    ║\n";
    std::cout << "║  Mask通道数   : " << config.maskChannels
              << "                                           ║\n";
    std::cout << "║  置信度阈值   : " << std::fixed << std::setprecision(2)
              << config.confThreshold << "                                     ║\n";
    std::cout << "║  NMS IoU阈值  : " << config.iouThreshold << "                                     ║\n";
    std::cout << "║  Mask二值化   : " << config.maskThreshold << "                                     ║\n";
    std::cout << "║  ONNX RT线程  : " << config.numThreads << "                                           ║\n";
    std::cout << "╚════════════════════════════════════════════════════════╝\n";
}

// ============================================================================
// 辅助函数：打印推理结果
// ============================================================================

void printResults(const InferenceResult& result)
{
    std::cout << "\n┌────────────────────────────────────────────────────────┐\n";
    std::cout << "│                   推理结果                             │\n";
    std::cout << "├────────────────────────────────────────────────────────┤\n";

    if (result.detections.empty()) {
        std::cout << "│  ⚠ 未检测到任何目标                                  │\n";
    } else {
        std::cout << "│  检测到 " << result.detections.size()
                  << " 个目标:                                           │\n";
        std::cout << "├──────┬──────────────────┬──────────┬───────────────────┤\n";
        std::cout << "│ 序号 │ 类别             │ 置信度   │ 边界框 (x,y,w,h) │\n";
        std::cout << "├──────┼──────────────────┼──────────┼───────────────────┤\n";

        for (size_t i = 0; i < result.detections.size(); ++i) {
            const auto& det = result.detections[i];
            std::cout << "│ " << std::setw(4) << (i + 1)
                      << " │ " << std::setw(16) << std::left << det.className
                      << " │ " << std::setw(8) << std::fixed << std::setprecision(2)
                      << det.confidence
                      << " │ (" << det.bbox.x << "," << det.bbox.y
                      << "," << det.bbox.width << "," << det.bbox.height << ")"
                      << "     │\n";
        }
        std::cout << "└──────┴──────────────────┴──────────┴───────────────────┘\n";
    }

    // ── 性能统计 ───────────────────────────────────────────────────────
    std::cout << "\n┌────────────────────────────────────────────────────────┐\n";
    std::cout << "│                   性能统计                             │\n";
    std::cout << "├────────────────────────────────────────────────────────┤\n";
    std::cout << "│  预处理耗时 : " << std::setw(8) << std::fixed
              << std::setprecision(2) << result.preprocessMs << " ms               │\n";
    std::cout << "│  推理耗时   : " << std::setw(8) << std::fixed
              << std::setprecision(2) << result.inferenceMs << " ms               │\n";
    std::cout << "│  后处理耗时 : " << std::setw(8) << std::fixed
              << std::setprecision(2) << result.postprocessMs << " ms               │\n";
    std::cout << "│  总耗时     : " << std::setw(8) << std::fixed
              << std::setprecision(2)
              << (result.preprocessMs + result.inferenceMs + result.postprocessMs)
              << " ms               │\n";
    std::cout << "└────────────────────────────────────────────────────────┘\n";
}

// ============================================================================
// 主函数
// ============================================================================

int main(int argc, char* argv[])
{
    // ── 解析命令行参数 ──────────────────────────────────────────────────
    std::string imagePath;
    std::string modelPath;

    if (argc >= 2) {
        imagePath = argv[1];
    } else {
        // 默认图片路径：优先使用 PH2 测试图，回退到 COCO 演示图
        imagePath = "../ph2_data/images/test/0003.png";
    }

    if (argc >= 3) {
        modelPath = argv[2];
    } else {
        // 默认模型路径：优先使用 PH2 模型，回退到 COCO 模型
        modelPath = "../models/yolov8n-seg-ph2.onnx";
    }

    // 自动检测数据集类型
    bool isMedical = (modelPath.find("ph2") != std::string::npos ||
                      modelPath.find("PH2") != std::string::npos ||
                      modelPath.find("isic") != std::string::npos ||
                      modelPath.find("ISIC") != std::string::npos);

    std::cout << "═══════════════════════════════════════════════════════\n";
    std::cout << "  YOLOv8 分割模型 ONNX Runtime C++ 推理引擎\n";
    if (isMedical) {
        std::cout << "  模式: 皮肤镜医学图像分割 (PH2/ISIC)\n";
    }
    std::cout << "═══════════════════════════════════════════════════════\n\n";

    // ── 步骤1：初始化推理引擎 ──────────────────────────────────────────
    std::cout << "[步骤1] 初始化推理引擎...\n";
    std::cout << "  模型路径: " << modelPath << "\n";
    std::cout << "  图片路径: " << imagePath << "\n\n";

    // 配置推理参数（根据数据集自动调整）
    InferenceConfig config;
    if (isMedical) {
        config.numClasses   = 1;          // 单类：病灶
        config.classNames   = {"lesion"}; // 类别名称
        config.confThreshold = 0.25f;     // 置信度阈值
        config.iouThreshold  = 0.45f;     // NMS IoU 阈值
        config.maskThreshold = 0.5f;      // Mask 二值化阈值
    } else {
        config.confThreshold = 0.25f;
        config.iouThreshold  = 0.45f;
        config.maskThreshold = 0.5f;
    }
    config.numThreads    = 4;
    config.enableProfiling = true;

    printConfig(config);

    YoloSegInference engine(config);

    auto tStart = std::chrono::high_resolution_clock::now();

    if (!engine.initialize(modelPath, config.numThreads)) {
        std::cerr << "\n[致命错误] 无法初始化推理引擎，程序退出\n";
        return -1;
    }

    auto tInit = std::chrono::high_resolution_clock::now();
    double initMs = std::chrono::duration<double, std::milli>(tInit - tStart).count();
    std::cout << "  初始化耗时: " << std::fixed << std::setprecision(2)
              << initMs << " ms\n\n";

    // ── 步骤2：读取图片 ────────────────────────────────────────────────
    std::cout << "[步骤2] 读取测试图片...\n";

    cv::Mat image = cv::imread(imagePath, cv::IMREAD_COLOR);
    if (image.empty()) {
        std::cerr << "[错误] 无法读取图片: " << imagePath << "\n";
        std::cerr << "  请检查图片路径是否正确。\n";
        return -1;
    }

    std::cout << "  图片尺寸: " << image.cols << " × "
              << image.rows << " (W × H)\n";
    std::cout << "  通道数: " << image.channels() << "\n\n";

    // ── 步骤3：执行推理 ────────────────────────────────────────────────
    std::cout << "[步骤3] 执行推理...\n";

    auto tInferStart = std::chrono::high_resolution_clock::now();
    InferenceResult result = engine.infer(image);
    auto tInferEnd = std::chrono::high_resolution_clock::now();
    double totalMs = std::chrono::duration<double, std::milli>(tInferEnd - tInferStart).count();

    // ── 打印结果 ───────────────────────────────────────────────────────
    printResults(result);

    std::cout << "  (注意: 以上耗时来自 C++ 内部计时，总耗时含调度开销: "
              << std::fixed << std::setprecision(2) << totalMs << " ms)\n";

    // ── 步骤4：可视化并保存 ────────────────────────────────────────────
    std::cout << "\n[步骤4] 生成可视化结果...\n";

    cv::Mat visualized = engine.visualize(image, result.detections, true);

    // 生成输出路径（在输入文件名后加 _result）
    std::string outputPath;
    size_t dotPos = imagePath.find_last_of('.');
    if (dotPos != std::string::npos) {
        outputPath = imagePath.substr(0, dotPos) + "_result"
                     + imagePath.substr(dotPos);
    } else {
        outputPath = imagePath + "_result.jpg";
    }

    cv::imwrite(outputPath, visualized);
    std::cout << "  结果已保存到: " << outputPath << "\n";

    // ── 完成 ────────────────────────────────────────────────────────────
    std::cout << "\n═══════════════════════════════════════════════════════\n";
    std::cout << "  推理完成！\n";

    if (!result.detections.empty()) {
        std::cout << "  检测到 " << result.detections.size() << " 个目标。\n";
        std::cout << "  打开 " << outputPath << " 查看带分割Mask的结果图。\n";
    } else {
        std::cout << "  未检测到目标，可尝试降低置信度阈值。\n";
    }

    std::cout << "═══════════════════════════════════════════════════════\n";

    return 0;
}
