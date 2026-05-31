#pragma once

#include "Common.h"
#include <dlib/image_processing.h>
#include <dlib/dnn.h>
#include <dlib/image_transforms.h>
#include <unordered_map>
#include <memory>

namespace efr {

// ============================================================
// dlib 人脸识别网络定义 (ResNet-34 风格)
// 兼容 dlib_face_recognition_resnet_model_v1.dat 预训练权重
// ============================================================

template <template <int,template<typename>class,int,typename> class block,
          int N, template<typename>class BN, typename SUBNET>
using residual = dlib::add_prev1<block<N,BN,1,dlib::tag1<SUBNET>>>;

template <template <int,template<typename>class,int,typename> class block,
          int N, template<typename>class BN, typename SUBNET>
using residual_down = dlib::add_prev2<dlib::avg_pool<2,2,2,2,
    dlib::skip1<dlib::tag2<block<N,BN,2,dlib::tag1<SUBNET>>>>>>;

template <int N, template <typename> class BN, int stride, typename SUBNET>
using block = BN<dlib::con<N,3,3,1,1,
    dlib::relu<BN<dlib::con<N,3,3,stride,stride,SUBNET>>>>>;

template <int N, typename SUBNET>
using ares = dlib::relu<residual<block,N,dlib::affine,SUBNET>>;

template <int N, typename SUBNET>
using ares_down = dlib::relu<residual_down<block,N,dlib::affine,SUBNET>>;

template <typename SUBNET> using alevel0 = ares_down<256,SUBNET>;
template <typename SUBNET> using alevel1 = ares<256,ares<256,ares_down<256,SUBNET>>>;
template <typename SUBNET> using alevel2 = ares<128,ares<128,ares_down<128,SUBNET>>>;
template <typename SUBNET> using alevel3 = ares<64,ares<64,ares<64,ares_down<64,SUBNET>>>>;
template <typename SUBNET> using alevel4 = ares<32,ares<32,ares<32,SUBNET>>>;

/// 人脸识别网络：输入 150x150 RGB，输出 128 维特征向量
using FaceRecognitionNetType = dlib::loss_metric<dlib::fc_no_bias<128,
    dlib::avg_pool_everything<
    alevel0<
    alevel1<
    alevel2<
    alevel3<
    alevel4<
    dlib::max_pool<3,3,2,2,dlib::relu<dlib::affine<dlib::con<32,7,7,2,2,
    dlib::input_rgb_image_sized<150>
    >>>>>>>>>>>>;

/// 已知人脸库中的一条记录
struct KnownFace {
    std::string name;
    std::vector<float> embedding; // 128 维特征向量
};

/**
 * 人脸识别管理器
 *
 * 职责：
 * - 加载 dlib ResNet 模型，提取 128 维人脸特征
 * - 从目录加载已知人脸，建立特征数据库
 * - 将检测到的人脸与数据库比对，输出识别结果
 */
class FaceRecognizer {
public:
    FaceRecognizer();
    ~FaceRecognizer();

    FaceRecognizer(const FaceRecognizer&) = delete;
    FaceRecognizer& operator=(const FaceRecognizer&) = delete;

    /**
     * 初始化识别器
     * @param config  系统配置
     * @return 是否初始化成功
     */
    bool initialize(const SystemConfig& config);

    /**
     * 对检测到的人脸执行识别
     * @param frame_data  帧数据（faces 已填充，识别结果回填到 recognized）
     * @return 识别到的人脸数
     */
    int recognize(FrameData& frame_data);

    /**
     * 从目录加载已知人脸
     * @param dir_path  图片目录，每个子目录以人名命名，内含该人的人脸图片
     * @return 加载的人数
     */
    int loadKnownFaces(const std::string& dir_path);

    /**
     * 注册单张人脸
     * @param name  人名
     * @param image  人脸图片
     * @return 是否注册成功
     */
    bool registerFace(const std::string& name, const cv::Mat& image);

    /// 识别器是否可用
    bool isAvailable() const { return available_; }

    /// 已知人脸数量
    size_t knownFaceCount() const { return known_faces_.size(); }

    /// 设置识别阈值
    void setThreshold(float threshold) { threshold_ = threshold; }

private:
    /// 提取单张人脸的 128 维特征向量
    bool extractEmbedding(const cv::Mat& image,
                          const cv::Rect& face_rect,
                          std::vector<float>& embedding);

    /// 在已知人脸库中搜索最匹配的人
    std::pair<std::string, float> findBestMatch(const std::vector<float>& embedding);

    // dlib 模型
    dlib::shape_predictor shape_predictor_;
    std::unique_ptr<FaceRecognitionNetType> face_net_;

    // 已知人脸数据库
    std::vector<KnownFace> known_faces_;

    bool available_ = false;
    bool shape_predictor_loaded_ = false;
    bool face_net_loaded_ = false;
    float threshold_ = 0.6f;

    // 线程安全锁
    mutable std::mutex recog_mutex_;
};

} // namespace efr
