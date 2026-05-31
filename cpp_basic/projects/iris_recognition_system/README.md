# Iris Recognition System (C++20)

基于 C++20 + OpenCV 的虹膜识别系统，遵循 `iris_recognition_cpp_nn_architecture.md` 架构设计。

## 架构概览

```
Capture → EyeDetection → QualityCheck → Segmentation
    → Normalization → FeatureExtraction → LivenessCheck
    → Matching → Decision
```

## 技术栈

- **语言**: C++20
- **图像处理**: OpenCV 4.x
- **推理后端**: 经典计算机视觉（Gabor滤波器 + IrisCode），预留 ONNX/RKNN/TensorRT 接口
- **数据库**: SQLite3（可选，支持 JSON 文件回退）
- **加密**: OpenSSL AES-256-GCM（可选，内置简化加密回退）

## 构建

### 依赖

```bash
# Ubuntu/Debian
sudo apt install build-essential cmake \
    libopencv-dev libsqlite3-dev libssl-dev
```

### 编译

```bash
cd iris_recognition_system
mkdir -p build && cd build
cmake ..
make -j$(nproc)
```

## 使用

### 帮助信息
```bash
./iris_app --help
```

### 实时识别（摄像头）
```bash
./iris_app --mode identify --camera 0
```

### 用户注册
```bash
./iris_app --mode enroll --user "张三"
```

### 图像文件识别
```bash
./iris_app --image /path/to/eye.jpg
```

### 查看已注册用户
```bash
./iris_app --list-users
```

### 完整参数
| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--mode <enroll\|identify>` | 运行模式 | identify |
| `--camera <id>` | 摄像头设备 ID | 0 |
| `--image <path>` | 处理单张图片 | - |
| `--video <path>` | 处理视频文件 | - |
| `--user <name>` | 注册时用户名 | user |
| `--db <path>` | 数据库文件路径 | iris_data.db |
| `--threshold <value>` | 匹配阈值 | 0.35 |
| `--no-display` | 无 GUI 运行 | - |

### GUI 快捷键
| 键 | 功能 |
|----|------|
| q / ESC | 退出 |
| e | 切换到注册模式 |
| i | 切换到识别模式 |
| s | 保存当前帧 |

## 项目结构

```
iris_recognition_system/
├── CMakeLists.txt
├── README.md
├── config/
│   └── app_config.json
├── src/
│   ├── main.cpp                    # 入口点
│   ├── app/Constants.h             # 全局常量
│   ├── capture/CameraDevice.*      # 摄像头采集
│   ├── image/
│   │   ├── ImagePreprocessor.*     # 图像预处理
│   │   ├── QualityChecker.*        # 质量评估
│   │   ├── IrisNormalizer.*        # Daugman 归一化
│   │   └── GeometryUtils.*         # 几何工具
│   ├── nn/InferenceEngine.h        # NN推理抽象接口
│   ├── model/
│   │   ├── EyeDetector.*           # 眼部检测 (Haar/CNN)
│   │   ├── IrisSegmenter.*         # 虹膜分割 (Hough/UNet)
│   │   ├── IrisFeatureExtractor.*  # 特征提取 (Gabor/ResNet)
│   │   └── IrisLivenessDetector.*  # 活体检测
│   ├── matching/
│   │   ├── IrisMatcher.*           # Hamming距离匹配
│   │   └── DecisionEngine.*        # 综合决策
│   ├── persistence/Database.*      # SQLite/JSON存储
│   ├── security/
│   │   ├── CryptoProvider.*        # AES-256-GCM
│   │   └── TemplateEncryptor.*     # 模板加密
│   ├── service/
│   │   ├── EnrollmentService.*     # 注册服务
│   │   ├── RecognitionService.*    # 识别服务
│   │   ├── LivenessService.*       # 活体检测服务
│   │   └── AuditService.*          # 审计日志
│   └── domain/                     # 领域模型
│       ├── User.h
│       ├── IrisTemplate.h
│       ├── RecognitionResult.h
│       └── AuditLog.h
└── tests/
```

## 核心算法

### 虹膜分割
- Hough 圆检测定位虹膜/瞳孔边界
- 边界精修（径向梯度分析）

### 虹膜归一化 (Daugman Rubber Sheet Model)
- 极坐标展开：环形虹膜区域 → 矩形归一化图像
- 双线性插值采样

### 特征提取 (Gabor IrisCode)
- 多尺度、多方向 2D Gabor 滤波器组
- 相位量化（2 bits/像素/方向）
- 生成 ~2048-bit IrisCode

### 匹配
- Hamming 距离 + 旋转补偿
- 有效位掩码支持

### 活体检测
- 纹理统计（Laplacian 方差）
- Moiré 图案检测（FFT 高频分析）
- LBP 纹理分布
- 多帧运动分析

## 生产部署路线

1. **PC 原型**: OpenCV + 经典CV（当前实现）
2. **NN 增强**: 替换为 ONNX Runtime + 预训练模型
3. **嵌入式迁移**: RK3588 + RKNN / Jetson + TensorRT

见 `iris_recognition_cpp_nn_architecture.md` 完整架构文档。
