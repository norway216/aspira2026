#include "FaceRecognizer.h"
#include <dlib/image_processing/frontal_face_detector.h>
#include <dlib/opencv/cv_image.h>
#include <opencv2/imgproc.hpp>

#include <cmath>
#include <fstream>
#include <algorithm>
#include <filesystem>

namespace fs = std::filesystem;

namespace efr {

FaceRecognizer::FaceRecognizer() {}

FaceRecognizer::~FaceRecognizer() = default;

bool FaceRecognizer::initialize(const SystemConfig& config) {
    threshold_ = config.recognition_threshold;

    // 1. 加载 shape predictor (68 点 landmark)
    std::string sp_path = config.shape_predictor_path;
    if (sp_path.empty()) {
        sp_path = "/usr/share/dlib/shape_predictor_68_face_landmarks.dat";
    }

    try {
        dlib::deserialize(sp_path) >> shape_predictor_;
        shape_predictor_loaded_ = true;
        Logger::info("Shape predictor 已加载: {}", sp_path);
    } catch (const std::exception& e) {
        Logger::warn("无法加载 shape predictor ({}): {}", sp_path, e.what());
        Logger::warn("人脸识别功能将不可用");
    }

    // 2. 加载 face recognition model (ResNet, 128D)
    if (shape_predictor_loaded_) {
        std::string fr_path = config.face_recognition_model_path;
        if (fr_path.empty()) {
            fr_path = "models/dlib_face_recognition_resnet_model_v1.dat";
        }

        try {
            face_net_ = std::make_unique<FaceRecognitionNetType>();
            dlib::deserialize(fr_path) >> *face_net_;
            face_net_loaded_ = true;
            Logger::info("Face recognition model 已加载: {}", fr_path);
        } catch (const std::exception& e) {
            Logger::warn("无法加载 face recognition model ({}): {}", fr_path, e.what());
            Logger::warn("请运行 ./download_models.sh 下载模型文件");
        }
    }

    // 3. 加载已知人脸库
    if (face_net_loaded_ && !config.known_faces_dir.empty()) {
        int count = loadKnownFaces(config.known_faces_dir);
        Logger::info("已知人脸: {} 人已加载", count);
    }

    available_ = shape_predictor_loaded_ && face_net_loaded_;
    return available_;
}

int FaceRecognizer::recognize(FrameData& frame_data) {
    std::lock_guard<std::mutex> lock(recog_mutex_);
    if (!available_ || frame_data.faces.empty()) {
        frame_data.recognized.clear();
        return 0;
    }

    frame_data.recognized.clear();
    frame_data.recognized.reserve(frame_data.faces.size());

    for (const auto& face : frame_data.faces) {
        // 提取特征向量
        std::vector<float> embedding;
        if (!extractEmbedding(frame_data.frame, face.rect, embedding)) {
            continue;
        }

        // 在已知库中搜索
        auto [name, distance] = findBestMatch(embedding);

        RecognizedFace rf;
        rf.rect = face.rect;
        rf.embedding = std::move(embedding);

        if (distance < threshold_ && !name.empty()) {
            rf.name = name;
            rf.distance = distance;
        } else {
            rf.name = "Unknown";
            rf.distance = distance;
        }

        frame_data.recognized.push_back(std::move(rf));
    }

    return static_cast<int>(frame_data.recognized.size());
}

bool FaceRecognizer::extractEmbedding(const cv::Mat& image,
                                       const cv::Rect& face_rect,
                                       std::vector<float>& embedding) {
    if (!face_net_loaded_ || !shape_predictor_loaded_) return false;

    try {
        // 转换 OpenCV Mat 到 dlib 格式
        cv::Mat bgr;
        if (image.channels() == 4) {
            cv::cvtColor(image, bgr, cv::COLOR_BGRA2BGR);
        } else if (image.channels() == 1) {
            cv::cvtColor(image, bgr, cv::COLOR_GRAY2BGR);
        } else {
            bgr = image;
        }

        dlib::cv_image<dlib::bgr_pixel> dlib_img(bgr);

        // 将 cv::Rect 转换为 dlib::rectangle
        dlib::rectangle dlib_rect(
            face_rect.x, face_rect.y,
            face_rect.x + face_rect.width - 1,
            face_rect.y + face_rect.height - 1
        );

        // 提取 68 点 landmark
        auto shape = shape_predictor_(dlib_img, dlib_rect);

        // 提取对齐后的人脸图像芯片 (150x150, padding=0.25)
        dlib::matrix<dlib::rgb_pixel> face_chip;
        dlib::extract_image_chip(dlib_img,
                                 dlib::get_face_chip_details(shape, 150, 0.25),
                                 face_chip);

        // 计算 128 维特征向量
        dlib::matrix<float, 0, 1> face_descriptor = (*face_net_)(face_chip);

        // 转换为 std::vector<float>
        embedding.resize(128);
        for (long i = 0; i < face_descriptor.nr(); ++i) {
            embedding[i] = face_descriptor(i);
        }

        return true;

    } catch (const std::exception& e) {
        Logger::error("特征提取失败: {}", e.what());
        return false;
    }
}

std::pair<std::string, float> FaceRecognizer::findBestMatch(const std::vector<float>& embedding) {
    if (known_faces_.empty()) {
        return {"", 1.0f};
    }

    std::string best_name;
    float best_distance = std::numeric_limits<float>::max();

    for (const auto& kf : known_faces_) {
        // 计算欧氏距离
        float dist = 0.0f;
        for (size_t i = 0; i < 128; ++i) {
            float diff = embedding[i] - kf.embedding[i];
            dist += diff * diff;
        }
        dist = std::sqrt(dist);

        if (dist < best_distance) {
            best_distance = dist;
            best_name = kf.name;
        }
    }

    return {best_name, best_distance};
}

int FaceRecognizer::loadKnownFaces(const std::string& dir_path) {
    if (!fs::exists(dir_path) || !fs::is_directory(dir_path)) {
        Logger::warn("已知人脸目录不存在: {}", dir_path);
        return 0;
    }

    dlib::frontal_face_detector detector = dlib::get_frontal_face_detector();
    int loaded = 0;

    for (const auto& person_dir : fs::directory_iterator(dir_path)) {
        if (!person_dir.is_directory()) continue;

        std::string name = person_dir.path().filename().string();
        std::vector<std::vector<float>> embeddings;

        for (const auto& img_file : fs::directory_iterator(person_dir.path())) {
            if (!img_file.is_regular_file()) continue;

            std::string ext = img_file.path().extension().string();
            std::transform(ext.begin(), ext.end(), ext.begin(), ::tolower);
            if (ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".bmp") continue;

            cv::Mat img = cv::imread(img_file.path().string());
            if (img.empty()) continue;

            // 检测图片中的最大人脸
            cv::Mat gray;
            cv::cvtColor(img, gray, cv::COLOR_BGR2GRAY);
            dlib::cv_image<unsigned char> dlib_img(gray);
            auto dets = detector(dlib_img);

            if (dets.empty()) {
                Logger::warn("  在 {} 中未检测到人脸，跳过", img_file.path().filename().string());
                continue;
            }

            // 取最大的人脸
            auto& d = dets[0];
            for (size_t i = 1; i < dets.size(); ++i) {
                if (dets[i].area() > d.area()) d = dets[i];
            }

            cv::Rect face_rect(d.left(), d.top(), d.width(), d.height());
            std::vector<float> emb;
            if (extractEmbedding(img, face_rect, emb)) {
                embeddings.push_back(std::move(emb));
            }
        }

        if (!embeddings.empty()) {
            // 计算平均特征向量
            std::vector<float> avg_emb(128, 0.0f);
            for (const auto& emb : embeddings) {
                for (size_t i = 0; i < 128; ++i) {
                    avg_emb[i] += emb[i];
                }
            }
            for (size_t i = 0; i < 128; ++i) {
                avg_emb[i] /= static_cast<float>(embeddings.size());
            }

            known_faces_.push_back({name, std::move(avg_emb)});
            loaded++;
            Logger::info("  已注册: {} ({} 张图片)", name, embeddings.size());
        }
    }

    return loaded;
}

bool FaceRecognizer::registerFace(const std::string& name, const cv::Mat& image) {
    if (!face_net_loaded_ || !shape_predictor_loaded_) return false;

    // 检测人脸
    cv::Mat gray;
    cv::cvtColor(image, gray, cv::COLOR_BGR2GRAY);
    dlib::cv_image<unsigned char> dlib_img(gray);
    dlib::frontal_face_detector detector = dlib::get_frontal_face_detector();
    auto dets = detector(dlib_img);

    if (dets.empty()) return false;

    auto& d = dets[0];
    cv::Rect face_rect(d.left(), d.top(), d.width(), d.height());

    std::vector<float> emb;
    if (!extractEmbedding(image, face_rect, emb)) return false;

    known_faces_.push_back({name, std::move(emb)});
    Logger::info("已注册人脸: {}", name);
    return true;
}

} // namespace efr
