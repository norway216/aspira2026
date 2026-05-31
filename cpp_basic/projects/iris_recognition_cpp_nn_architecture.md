# C++ + 神经网络虹膜识别系统完整架构设计说明书

> 文档版本：v1.0  
> 目标项目：Iris Recognition System with C++ and Neural Networks  
> 适用平台：Ubuntu / Debian / RK3568 / RK3588 / Jetson / x86_64 PC  
> 技术栈：C++20 + OpenCV + Qt/QML + ONNX Runtime / OpenVINO / RKNN / TensorRT  
> 核心任务：虹膜采集、质量评估、虹膜分割、归一化、特征提取、活体检测、身份注册、身份识别、安全存储

---

## 1. 项目目标

本项目目标是设计并实现一个可落地的虹膜识别系统，使用 C++ 作为核心开发语言，OpenCV 负责图像采集与预处理，神经网络负责虹膜分割、特征提取与活体检测，最终完成用户注册、虹膜模板生成、身份识别和安全校验。

系统既可以运行在普通 Ubuntu PC 上，也可以部署到 RK3568、RK3588、Jetson 等嵌入式设备上。

### 1.1 核心能力

- 支持摄像头实时采集眼部图像。
- 支持图片 / 视频文件离线识别。
- 支持虹膜区域自动检测与分割。
- 支持瞳孔、虹膜边界定位。
- 支持虹膜归一化展开。
- 支持神经网络特征提取。
- 支持虹膜模板注册与身份识别。
- 支持活体检测，降低照片、屏幕、打印攻击风险。
- 支持本地加密存储虹膜模板。
- 支持嵌入式硬件加速部署。

---

## 2. 总体架构

```text
┌──────────────────────────────────────────────────────────────┐
│                         QML / UI Layer                        │
│  用户注册 / 实时预览 / 识别结果 / 设备状态 / 日志显示            │
└───────────────────────────────┬──────────────────────────────┘
                                │
┌───────────────────────────────▼──────────────────────────────┐
│                      Application Service Layer                 │
│  EnrollmentService / RecognitionService / LivenessService      │
│  UserService / DeviceService / AuditService                    │
└───────────────────────────────┬──────────────────────────────┘
                                │
┌───────────────────────────────▼──────────────────────────────┐
│                         Pipeline Layer                        │
│  Capture → QualityCheck → Segmentation → Normalization         │
│  → FeatureExtraction → LivenessCheck → Matching → Decision     │
└───────────────────────────────┬──────────────────────────────┘
                                │
┌───────────────┬───────────────▼───────────────┬──────────────┐
│ Image Module  │        Neural Network Module   │ Crypto Module│
│ OpenCV        │ U-Net / MobileNet / Siamese    │ AES / SHA256 │
│ Preprocess    │ ONNX / RKNN / TensorRT         │ SecureStore  │
└───────────────┴───────────────┬───────────────┴──────────────┘
                                │
┌───────────────────────────────▼──────────────────────────────┐
│                       Persistence Layer                       │
│  SQLite / UserRepository / TemplateRepository / AuditLog       │
└──────────────────────────────────────────────────────────────┘
```

---

## 3. 推荐神经网络组合

虹膜识别系统不建议只使用一个神经网络完成所有任务，而是采用多个轻量模型组成流水线。

### 3.1 推荐组合一：工程实用版

适合 Ubuntu PC、RK3588、Jetson：

| 模块 | 推荐网络 | 作用 |
|---|---|---|
| 眼部检测 | YOLOv5n-eye / SCRFD-eye | 定位眼部 ROI |
| 虹膜分割 | U-Net / Attention U-Net | 输出虹膜 mask |
| 虹膜特征提取 | ResNet18-Embedding / MobileNetV3-Embedding | 输出 128/256/512 维向量 |
| 活体检测 | MiniFASNet / MobileNetV3-Liveness | 判断是否真人虹膜 |
| 匹配 | Cosine Similarity / Siamese Head | 计算模板相似度 |

### 3.2 推荐组合二：嵌入式轻量版

适合 RK3568、低算力 ARM：

| 模块 | 推荐网络 | 输入尺寸 | 说明 |
|---|---|---|---|
| 眼部检测 | YOLOv5n / NanoDet | 320×320 | 轻量检测 |
| 虹膜分割 | Mobile-UNet | 128×128 或 160×160 | 小模型分割 |
| 特征提取 | MobileNetV2 / MobileFaceNet 改造 | 64×512 或 128×512 | 输出 embedding |
| 活体检测 | MiniFASNet-small | 80×80 或 128×128 | 低延迟 |
| 匹配 | Cosine Similarity | 向量维度 128/256 | CPU 即可完成 |

### 3.3 推荐组合三：研究增强版

适合 PC / GPU 训练和高精度验证：

| 模块 | 推荐网络 | 特点 |
|---|---|---|
| 虹膜分割 | DeepLabV3+ / HRNet-Seg | 分割精度高 |
| 特征提取 | ResNet34 / EfficientNet-B0 | 特征表达更强 |
| 匹配训练 | Siamese Network / Triplet Network | 适合身份匹配 |
| 活体检测 | CNN + LSTM / Temporal CNN | 利用多帧动态信息 |

---

## 4. 神经网络模块详细设计

---

# 4.1 眼部检测网络 Eye Detection Network

## 4.1.1 目标

从摄像头完整画面中定位人眼区域，减少后续虹膜分割的输入尺寸和计算量。

输入：

```text
Camera Frame: 640×480 / 1280×720 / 1920×1080
```

输出：

```text
left_eye_bbox  = [x, y, w, h]
right_eye_bbox = [x, y, w, h]
confidence
```

## 4.1.2 推荐网络

### YOLOv5n-eye

优点：

- 部署简单。
- 支持 ONNX / RKNN / TensorRT。
- 检测速度快。
- 可以同时检测左眼、右眼、眼部遮挡状态。

缺点：

- 需要标注眼部框数据。
- 对极近距离虹膜图像可能需要重新训练。

### SCRFD-eye

优点：

- 人脸/眼部检测鲁棒性较好。
- 适合做人脸到眼部 ROI 的定位。

缺点：

- 如果只做虹膜识别，模型可能略重。

## 4.1.3 C++ 模块接口

```cpp
struct EyeBox {
    cv::Rect bbox;
    float confidence;
    int label; // 0 = left_eye, 1 = right_eye
};

class EyeDetector {
public:
    virtual ~EyeDetector() = default;
    virtual bool initialize(const std::string& modelPath) = 0;
    virtual std::vector<EyeBox> detect(const cv::Mat& frame) = 0;
};
```

---

# 4.2 虹膜分割网络 Iris Segmentation Network

## 4.2.1 目标

将眼部 ROI 中的虹膜区域、瞳孔区域、眼睑、睫毛、背景区分出来。

输入：

```text
Eye ROI: 128×128 / 160×160 / 224×224 RGB or Gray
```

输出：

```text
Segmentation Mask:
0 = background
1 = sclera / eye white
2 = iris
3 = pupil
4 = eyelid / eyelash / occlusion
```

最小可用版本也可以只输出二分类：

```text
0 = non-iris
1 = iris
```

## 4.2.2 推荐网络

### U-Net

适合原型和 PC 部署。

优点：

- 结构清晰。
- 分割效果稳定。
- 训练资料丰富。
- 适合医学影像和生物识别分割任务。

缺点：

- 原始 U-Net 对嵌入式设备偏重。

### Mobile-UNet

适合 RK3568 / RK3588。

设计：

```text
Encoder: MobileNetV2 / MobileNetV3 blocks
Decoder: Lightweight upsample + depthwise separable conv
Output: 1-channel or 4-channel mask
```

优点：

- 参数少。
- 推理快。
- 适合 INT8 量化。

### DeepLabV3+

适合高精度版本。

优点：

- 边界分割效果较好。
- 对复杂光照和遮挡更强。

缺点：

- 模型更重。
- 嵌入式部署成本更高。

## 4.2.3 输出后处理

分割 mask 之后，需要使用 OpenCV 后处理：

```text
1. 二值化 iris mask
2. 形态学开运算去噪
3. 查找最大连通区域
4. 拟合虹膜外圆 / 椭圆
5. 拟合瞳孔圆 / 椭圆
6. 生成有效区域 mask
```

## 4.2.4 C++ 模块接口

```cpp
struct IrisSegmentationResult {
    cv::Mat irisMask;
    cv::Mat pupilMask;
    cv::RotatedRect irisEllipse;
    cv::RotatedRect pupilEllipse;
    float qualityScore;
};

class IrisSegmenter {
public:
    virtual ~IrisSegmenter() = default;
    virtual bool initialize(const std::string& modelPath) = 0;
    virtual IrisSegmentationResult segment(const cv::Mat& eyeRoi) = 0;
};
```

---

# 4.3 虹膜归一化 Iris Normalization

## 4.3.1 目标

由于不同人的虹膜大小、瞳孔收缩程度、摄像头距离不同，直接比较原图不稳定。需要将环形虹膜区域展开成固定尺寸的矩形图。

经典方法是 Daugman Rubber Sheet Model。

输入：

```text
Eye ROI + iris boundary + pupil boundary + iris mask
```

输出：

```text
Normalized Iris Image: 64×512 or 128×512
Normalized Valid Mask: 64×512 or 128×512
```

## 4.3.2 是否使用神经网络？

这个阶段通常不使用神经网络，而是使用几何变换：

```text
极坐标展开
圆环采样
亮度归一化
遮挡 mask 同步展开
```

## 4.3.3 C++ 接口

```cpp
struct NormalizedIris {
    cv::Mat image; // 64×512 grayscale
    cv::Mat mask;  // valid region mask
};

class IrisNormalizer {
public:
    NormalizedIris normalize(const cv::Mat& eyeRoi,
                             const IrisSegmentationResult& segResult);
};
```

---

# 4.4 虹膜特征提取网络 Iris Feature Extraction Network

## 4.4.1 目标

将归一化虹膜图转换成稳定的特征向量 embedding。

输入：

```text
Normalized Iris Image: 64×512 or 128×512
```

输出：

```text
Embedding Vector: 128 / 256 / 512 dimensions
```

## 4.4.2 推荐网络

### ResNet18-Embedding

适合 PC 和 RK3588。

结构：

```text
Input: 1×64×512
Conv Stem
ResNet18 Blocks
Global Average Pooling
FC 512
L2 Normalize
Output: 512-dim embedding
```

优点：

- 特征表达强。
- 易训练。
- ONNX 部署成熟。

缺点：

- 对 RK3568 偏重。

### MobileNetV3-Embedding

适合嵌入式。

结构：

```text
Input: 1×64×512
MobileNetV3 Small Backbone
Global Average Pooling
FC 256
L2 Normalize
Output: 256-dim embedding
```

优点：

- 参数少。
- 推理快。
- 适合 RKNN INT8 量化。

### Siamese Network

适合训练阶段。

训练时输入两张虹膜图：

```text
image_a → shared backbone → embedding_a
image_b → shared backbone → embedding_b
distance = cosine / euclidean
loss = contrastive loss / triplet loss
```

部署时只保留单分支 backbone：

```text
normalized_iris → backbone → embedding
```

## 4.4.3 损失函数设计

推荐组合：

```text
Classification Loss + Triplet Loss
```

或者：

```text
ArcFace Loss / CosFace Loss
```

适合身份特征学习。

## 4.4.4 C++ 接口

```cpp
struct IrisEmbedding {
    std::vector<float> vector;
    float quality;
};

class IrisFeatureExtractor {
public:
    virtual ~IrisFeatureExtractor() = default;
    virtual bool initialize(const std::string& modelPath) = 0;
    virtual IrisEmbedding extract(const NormalizedIris& normalized) = 0;
};
```

---

# 4.5 活体检测网络 Iris Liveness Detection Network

## 4.5.1 目标

判断当前虹膜图像是否来自真实活体，而不是打印照片、手机屏幕、视频回放、隐形眼镜或合成图像。

输入可以是：

```text
单帧 eye ROI
多帧 eye ROI sequence
虹膜 ROI + 反光特征
```

输出：

```text
liveness_score: 0.0 ~ 1.0
attack_type: real / print / screen / replay / unknown
```

## 4.5.2 推荐网络

### MiniFASNet-Iris

适合嵌入式单帧活体检测。

输入：

```text
128×128 eye ROI
```

输出：

```text
real_score
spoof_score
```

优点：

- 模型小。
- 推理快。
- 适合 RK3568 / RK3588。

缺点：

- 单帧对高级攻击仍有限。

### MobileNetV3-Liveness

适合轻量多类别攻击检测。

分类：

```text
0 = live
1 = printed_photo
2 = screen_replay
3 = cosmetic_contact_lens
4 = synthetic
```

### CNN + LSTM / Temporal CNN

适合多帧活体检测。

输入：

```text
N frames of iris ROI, for example 8 or 16 frames
```

可以识别：

```text
自然瞳孔微动
眨眼动态
屏幕刷新纹理
视频回放异常
```

缺点：

- 计算更重。
- 延迟更高。
- 需要时间序列数据。

## 4.5.3 活体检测建议融合策略

不要只靠一个神经网络分数，建议融合多个信号：

```text
final_liveness_score =
    0.60 * nn_liveness_score
  + 0.15 * blink_score
  + 0.10 * pupil_reflection_score
  + 0.10 * image_quality_score
  + 0.05 * motion_consistency_score
```

判定规则：

```text
score >= 0.75 → Live
0.50 <= score < 0.75 → Need Retry
score < 0.50 → Spoof / Attack
```

## 4.5.4 C++ 接口

```cpp
struct LivenessResult {
    float liveScore;
    float spoofScore;
    QString attackType;
    bool passed;
};

class IrisLivenessDetector {
public:
    virtual ~IrisLivenessDetector() = default;
    virtual bool initialize(const std::string& modelPath) = 0;
    virtual LivenessResult check(const cv::Mat& eyeRoi,
                                 const std::vector<cv::Mat>& recentFrames) = 0;
};
```

---

# 4.6 匹配模块 Iris Matching

## 4.6.1 目标

比较当前虹膜 embedding 与数据库中已注册模板，判断是否为同一人。

输入：

```text
current_embedding
registered_template_embedding
```

输出：

```text
similarity_score
match / not_match
user_id
```

## 4.6.2 匹配方法

### Cosine Similarity

推荐作为 MVP 默认方案。

```cpp
float cosineSimilarity(const std::vector<float>& a,
                       const std::vector<float>& b);
```

判定：

```text
similarity >= threshold → same user
similarity < threshold  → different user
```

阈值需要通过验证集确定：

```text
threshold = 0.65 / 0.70 / 0.75
```

### Euclidean Distance

如果 embedding 已经 L2 normalize，则欧氏距离也可用。

```text
distance <= threshold → same user
```

### Hamming Distance

如果采用传统 IrisCode，则使用 Hamming Distance；但本设计以神经网络 embedding 为主。

---

## 5. 完整识别流水线

### 5.1 注册流程 Enrollment Pipeline

```text
用户进入注册页面
  ↓
摄像头采集多帧眼部图像
  ↓
EyeDetector 定位眼部 ROI
  ↓
QualityChecker 检查清晰度、亮度、遮挡、角度
  ↓
IrisSegmenter 分割虹膜和瞳孔
  ↓
IrisNormalizer 归一化展开虹膜
  ↓
IrisFeatureExtractor 提取 embedding
  ↓
LivenessDetector 检查活体
  ↓
多帧 embedding 聚合，生成稳定模板
  ↓
TemplateEncryptor 加密模板
  ↓
TemplateRepository 存储模板
  ↓
注册完成
```

### 5.2 识别流程 Recognition Pipeline

```text
摄像头实时采集
  ↓
EyeDetector 定位眼部 ROI
  ↓
QualityChecker 过滤低质量帧
  ↓
IrisSegmenter 分割虹膜
  ↓
IrisNormalizer 展开归一化
  ↓
IrisFeatureExtractor 提取 embedding
  ↓
LivenessDetector 检查活体
  ↓
TemplateRepository 加载候选模板
  ↓
Matcher 计算相似度
  ↓
DecisionEngine 综合判定
  ↓
输出识别结果
```

---

## 6. C++ 项目目录结构

```text
iris-recognition-system/
├── CMakeLists.txt
├── README.md
├── docs/
│   ├── ARCHITECTURE.md
│   ├── MODEL_DESIGN.md
│   └── DEPLOYMENT.md
├── models/
│   ├── eye_detector.onnx
│   ├── iris_segmenter.onnx
│   ├── iris_feature.onnx
│   └── iris_liveness.onnx
├── config/
│   └── app_config.json
├── resources/
│   └── qml.qrc
├── src/
│   ├── main.cpp
│   ├── app/
│   │   ├── AppContext.h
│   │   └── Constants.h
│   ├── capture/
│   │   ├── CameraDevice.h/cpp
│   │   ├── VideoFileSource.h/cpp
│   │   └── FrameQueue.h/cpp
│   ├── image/
│   │   ├── ImagePreprocessor.h/cpp
│   │   ├── QualityChecker.h/cpp
│   │   ├── IrisNormalizer.h/cpp
│   │   └── GeometryUtils.h/cpp
│   ├── nn/
│   │   ├── InferenceEngine.h
│   │   ├── OnnxRuntimeEngine.h/cpp
│   │   ├── RknnEngine.h/cpp
│   │   ├── TensorRtEngine.h/cpp
│   │   └── TensorUtils.h/cpp
│   ├── model/
│   │   ├── EyeDetector.h/cpp
│   │   ├── IrisSegmenter.h/cpp
│   │   ├── IrisFeatureExtractor.h/cpp
│   │   └── IrisLivenessDetector.h/cpp
│   ├── domain/
│   │   ├── User.h
│   │   ├── IrisTemplate.h
│   │   ├── RecognitionResult.h
│   │   └── AuditLog.h
│   ├── service/
│   │   ├── EnrollmentService.h/cpp
│   │   ├── RecognitionService.h/cpp
│   │   ├── TemplateService.h/cpp
│   │   ├── LivenessService.h/cpp
│   │   └── AuditService.h/cpp
│   ├── matching/
│   │   ├── IrisMatcher.h/cpp
│   │   └── DecisionEngine.h/cpp
│   ├── persistence/
│   │   ├── Database.h/cpp
│   │   ├── UserRepository.h/cpp
│   │   ├── TemplateRepository.h/cpp
│   │   └── AuditRepository.h/cpp
│   ├── security/
│   │   ├── CryptoProvider.h/cpp
│   │   ├── TemplateEncryptor.h/cpp
│   │   └── SecureBuffer.h/cpp
│   ├── viewmodel/
│   │   ├── EnrollmentViewModel.h/cpp
│   │   ├── RecognitionViewModel.h/cpp
│   │   └── SettingsViewModel.h/cpp
│   └── qml/
│       ├── main.qml
│       ├── pages/
│       │   ├── EnrollmentPage.qml
│       │   ├── RecognitionPage.qml
│       │   └── SettingsPage.qml
│       └── components/
│           ├── CameraPreview.qml
│           ├── ResultCard.qml
│           └── StatusBar.qml
└── tests/
    ├── test_matcher.cpp
    ├── test_normalizer.cpp
    └── test_crypto.cpp
```

---

## 7. 核心 C++ 类设计

## 7.1 InferenceEngine 抽象层

用于屏蔽 ONNX Runtime、RKNN、TensorRT、OpenVINO 的差异。

```cpp
class InferenceEngine {
public:
    virtual ~InferenceEngine() = default;

    virtual bool loadModel(const std::string& modelPath) = 0;

    virtual std::vector<float> infer(const std::vector<float>& input,
                                     const std::vector<int64_t>& inputShape) = 0;

    virtual std::string backendName() const = 0;
};
```

具体实现：

```text
OnnxRuntimeEngine   → PC / 通用 Linux
RknnEngine          → RK3568 / RK3588
TensorRtEngine      → NVIDIA Jetson
OpenVinoEngine      → Intel CPU / NPU
```

---

## 7.2 QualityChecker

质量评估非常重要。低质量虹膜图像会导致识别错误。

```cpp
struct QualityResult {
    bool passed;
    float sharpness;
    float brightness;
    float occlusionRatio;
    float irisAreaRatio;
    QString reason;
};

class QualityChecker {
public:
    QualityResult checkEyeRoi(const cv::Mat& eyeRoi,
                              const IrisSegmentationResult& seg);
};
```

质量指标：

| 指标 | 方法 | 说明 |
|---|---|---|
| 清晰度 | Laplacian variance | 判断模糊 |
| 亮度 | Mean intensity | 判断过暗/过曝 |
| 遮挡比例 | mask 统计 | 睫毛/眼睑遮挡 |
| 虹膜面积 | iris mask area | 判断距离是否合适 |
| 姿态 | ellipse ratio | 判断倾斜角度 |

---

## 7.3 IrisMatcher

```cpp
struct MatchResult {
    QString userId;
    float similarity;
    bool matched;
};

class IrisMatcher {
public:
    explicit IrisMatcher(float threshold = 0.72f);

    MatchResult matchOne(const IrisEmbedding& query,
                         const IrisTemplate& registeredTemplate);

    MatchResult search(const IrisEmbedding& query,
                       const std::vector<IrisTemplate>& templates);

private:
    float cosineSimilarity(const std::vector<float>& a,
                           const std::vector<float>& b);

    float m_threshold;
};
```

---

## 7.4 DecisionEngine

综合识别、活体、质量分数。

```cpp
struct FinalDecision {
    bool accepted;
    QString userId;
    float matchScore;
    float livenessScore;
    float qualityScore;
    QString reason;
};

class DecisionEngine {
public:
    FinalDecision decide(const MatchResult& match,
                         const LivenessResult& live,
                         const QualityResult& quality);
};
```

判定规则示例：

```text
quality.passed == true
livenessScore >= 0.75
matchScore >= 0.72
→ accepted
```

---

## 8. 数据库设计

使用 SQLite 存储用户信息、虹膜模板和审计日志。

## 8.1 users 表

```sql
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
```

## 8.2 iris_templates 表

```sql
CREATE TABLE IF NOT EXISTS iris_templates (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    eye_side TEXT NOT NULL,
    embedding_encrypted BLOB NOT NULL,
    embedding_dim INTEGER NOT NULL,
    model_version TEXT NOT NULL,
    quality_score REAL NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

## 8.3 audit_logs 表

```sql
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    action TEXT NOT NULL,
    user_id TEXT,
    details TEXT,
    created_at INTEGER NOT NULL
);
```

---

## 9. 虹膜模板安全存储

虹膜模板属于敏感生物识别数据，不能明文存储。

### 9.1 加密策略

```text
设备主密钥 Device Master Key
  ↓
HKDF 派生 Template Encryption Key
  ↓
AES-256-GCM 加密 iris embedding
  ↓
SQLite 存储 encrypted embedding
```

### 9.2 TemplateEncryptor 接口

```cpp
class TemplateEncryptor {
public:
    QByteArray encryptEmbedding(const std::vector<float>& embedding,
                                const QByteArray& userId);

    std::vector<float> decryptEmbedding(const QByteArray& encryptedBlob,
                                        const QByteArray& userId);
};
```

### 9.3 安全要求

- 不保存原始虹膜图片。
- 不保存明文 embedding。
- 数据库文件应设置权限为 600。
- 模型版本和模板版本要绑定。
- 识别日志不记录完整图像。

---

## 10. 多线程流水线设计

为了在嵌入式设备上实时运行，必须做流水线并发。

```text
Capture Thread
  ↓ FrameQueue
Preprocess Thread
  ↓ ROIQueue
Segmentation Thread / NPU
  ↓ SegResultQueue
Feature Thread / NPU
  ↓ EmbeddingQueue
Recognition Thread
  ↓
UI Thread
```

### 10.1 FrameQueue

```cpp
class FrameQueue {
public:
    void push(const cv::Mat& frame);
    bool pop(cv::Mat& frame);
    void clear();
    size_t size() const;
};
```

建议：

```text
队列长度限制为 2~5 帧。
如果队列满，丢弃旧帧，保证实时性。
```

### 10.2 性能目标

| 平台 | 目标帧率 | 说明 |
|---|---:|---|
| x86 PC CPU | 20~30 FPS | ONNX Runtime CPU |
| x86 PC GPU | 30~60 FPS | CUDA / TensorRT |
| RK3568 | 5~15 FPS | 轻量模型 + RKNN |
| RK3588 | 15~30 FPS | 多模型 NPU 流水线 |
| Jetson TX2 | 15~30 FPS | TensorRT FP16 |

---

## 11. 模型训练与导出流程

### 11.1 训练阶段

```text
PyTorch Training
  ↓
验证集评估
  ↓
导出 ONNX
  ↓
ONNX Simplifier
  ↓
量化 INT8 / FP16
  ↓
平台转换
```

### 11.2 模型导出

```bash
python export_iris_segmenter.py --weights iris_unet.pth --output iris_segmenter.onnx
python export_feature.py --weights iris_mobilenet.pth --output iris_feature.onnx
python export_liveness.py --weights iris_liveness.pth --output iris_liveness.onnx
```

### 11.3 嵌入式转换

#### RK3568 / RK3588

```bash
rknn-toolkit2 convert iris_feature.onnx iris_feature.rknn
```

#### Jetson

```bash
trtexec --onnx=iris_feature.onnx --saveEngine=iris_feature.engine --fp16
```

#### Intel / OpenVINO

```bash
ovc iris_feature.onnx --output_model iris_feature.xml
```

---

## 12. 推荐模型输入输出规范

## 12.1 eye_detector.onnx

```text
Input:
  name: images
  shape: [1, 3, 320, 320]
  dtype: float32

Output:
  boxes: [N, 4]
  scores: [N]
  labels: [N]
```

## 12.2 iris_segmenter.onnx

```text
Input:
  name: eye_roi
  shape: [1, 1, 128, 128]
  dtype: float32

Output:
  name: mask
  shape: [1, 4, 128, 128]
  dtype: float32
```

## 12.3 iris_feature.onnx

```text
Input:
  name: normalized_iris
  shape: [1, 1, 64, 512]
  dtype: float32

Output:
  name: embedding
  shape: [1, 256]
  dtype: float32
```

## 12.4 iris_liveness.onnx

```text
Input:
  name: eye_roi
  shape: [1, 3, 128, 128]
  dtype: float32

Output:
  name: scores
  shape: [1, 2]
  dtype: float32
```

---

## 13. QML UI 设计

### 13.1 页面

| 页面 | 功能 |
|---|---|
| EnrollmentPage | 用户注册、采集虹膜、生成模板 |
| RecognitionPage | 实时识别、显示匹配结果 |
| UserListPage | 用户管理 |
| SettingsPage | 摄像头、阈值、模型路径配置 |
| LogsPage | 查看审计日志 |

### 13.2 CameraPreview 组件

```qml
Item {
    property alias source: videoOutput.source
    property string statusText
    property real qualityScore
    property real livenessScore
    property real matchScore
}
```

UI 需要显示：

```text
实时摄像头画面
眼部 ROI 框
虹膜 mask 叠加
质量评分
活体评分
识别结果
```

---

## 14. 配置文件设计

`config/app_config.json`：

```json
{
  "camera": {
    "device": 0,
    "width": 640,
    "height": 480,
    "fps": 30
  },
  "models": {
    "backend": "onnxruntime",
    "eye_detector": "models/eye_detector.onnx",
    "iris_segmenter": "models/iris_segmenter.onnx",
    "iris_feature": "models/iris_feature.onnx",
    "iris_liveness": "models/iris_liveness.onnx"
  },
  "thresholds": {
    "quality": 0.65,
    "liveness": 0.75,
    "matching": 0.72
  },
  "security": {
    "encrypt_templates": true,
    "store_raw_images": false
  }
}
```

---

## 15. CMake 构建设计

```cmake
cmake_minimum_required(VERSION 3.18)
project(IrisRecognitionSystem LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

find_package(OpenCV REQUIRED)
find_package(Qt5 REQUIRED COMPONENTS Core Gui Qml Quick QuickControls2 Sql)
find_package(OpenSSL REQUIRED)
find_package(SQLite3 REQUIRED)

option(USE_ONNXRUNTIME "Use ONNX Runtime backend" ON)
option(USE_RKNN "Use RKNN backend" OFF)
option(USE_TENSORRT "Use TensorRT backend" OFF)

add_executable(iris_app
    src/main.cpp
    src/capture/CameraDevice.cpp
    src/image/ImagePreprocessor.cpp
    src/image/QualityChecker.cpp
    src/image/IrisNormalizer.cpp
    src/model/EyeDetector.cpp
    src/model/IrisSegmenter.cpp
    src/model/IrisFeatureExtractor.cpp
    src/model/IrisLivenessDetector.cpp
    src/matching/IrisMatcher.cpp
    src/matching/DecisionEngine.cpp
    src/persistence/Database.cpp
    src/security/TemplateEncryptor.cpp
)

target_link_libraries(iris_app
    PRIVATE
        ${OpenCV_LIBS}
        Qt5::Core
        Qt5::Gui
        Qt5::Qml
        Qt5::Quick
        Qt5::QuickControls2
        Qt5::Sql
        OpenSSL::SSL
        OpenSSL::Crypto
        SQLite::SQLite3
)
```

---

## 16. MVP 实施路线

### 第一阶段：基础原型

- OpenCV 打开摄像头。
- 截取眼部 ROI。
- 加载虹膜分割 ONNX 模型。
- 输出虹膜 mask。
- 显示 QML 实时画面。

### 第二阶段：注册与识别

- 实现虹膜归一化。
- 实现特征提取模型推理。
- 实现 embedding 存储。
- 实现 cosine similarity 匹配。

### 第三阶段：活体检测

- 加入 MiniFASNet / MobileNetV3 活体检测模型。
- 加入图像质量评估。
- 加入多帧融合。

### 第四阶段：嵌入式部署

- 模型 INT8 量化。
- ONNX 转 RKNN / TensorRT。
- C++ 推理后端切换。
- 多线程流水线优化。

### 第五阶段：安全增强

- 模板加密。
- 审计日志。
- 防重放检测。
- 权限控制。

---

## 17. 推荐技术路线总结

最终推荐架构：

```text
C++20 + OpenCV + Qt/QML
  ↓
ONNX Runtime 先完成 PC 原型
  ↓
虹膜分割：Mobile-UNet / U-Net
  ↓
虹膜特征：MobileNetV3-Embedding / ResNet18-Embedding
  ↓
活体检测：MiniFASNet / MobileNetV3-Liveness
  ↓
匹配：Cosine Similarity + 阈值判定
  ↓
嵌入式部署：RKNN / TensorRT / OpenVINO
```

对于你的技术栈，最建议先做：

```text
Ubuntu PC 原型：OpenCV + ONNX Runtime + Qt/QML
然后迁移到：RK3588 + RKNN + QML
```

如果目标是 RK3568，需要控制模型尺寸：

```text
Eye Detector: YOLOv5n / NanoDet
Segmenter: Mobile-UNet 128×128
Feature: MobileNetV3 Small 256-dim
Liveness: MiniFASNet-small
```

---

## 18. 关键风险与注意事项

### 18.1 硬件风险

虹膜识别对摄像头要求比普通人脸识别高很多。普通 RGB 摄像头可能无法获得足够清晰的虹膜纹理。

建议硬件：

```text
高清近距离摄像头
自动对焦
近红外 NIR 摄像头更佳
红外补光
固定焦距结构
防反光镜片设计
```

### 18.2 数据风险

神经网络虹膜识别需要高质量数据集。如果训练集不足，模型泛化能力会较差。

### 18.3 安全风险

虹膜模板属于敏感生物识别数据，一旦泄露不可更改。因此必须加密存储，避免保存原始眼部图像。

### 18.4 部署风险

不同推理后端对算子支持不同，模型训练时应尽量使用部署友好的算子：

```text
Conv
BatchNorm
ReLU / ReLU6 / SiLU
Pooling
Resize
Concat
Gemm
Softmax
```

避免复杂动态 shape、特殊自定义算子。

---

## 19. 最终结论

一个可用的 C++ 神经网络虹膜识别系统，不应只依赖单一模型，而应采用模块化流水线：

```text
摄像头采集
  → 眼部检测
  → 图像质量评估
  → 虹膜分割
  → 虹膜归一化
  → 特征提取
  → 活体检测
  → 模板匹配
  → 安全决策
```

推荐神经网络组合：

```text
眼部检测：YOLOv5n-eye / SCRFD-eye
虹膜分割：Mobile-UNet / U-Net / DeepLabV3+
特征提取：MobileNetV3-Embedding / ResNet18-Embedding / Siamese Network
活体检测：MiniFASNet / MobileNetV3-Liveness / CNN-LSTM
匹配算法：Cosine Similarity / Euclidean Distance
```

该架构既适合做 PC 原型，也适合后续迁移到 RK3568 / RK3588 / Jetson 等嵌入式平台。
