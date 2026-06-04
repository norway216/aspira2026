# Android 手机共享摄像头虹膜识别实验系统架构设计

## 1. 项目目标

本项目目标是设计一个基于 **Android 手机摄像头 + C++ 网络视频流 + 虹膜识别客户端** 的实验系统。

系统使用授权的个人 Android 手机作为网络摄像头，通过局域网向 PC、Linux 主机或嵌入式设备推送视频流，并在接收端使用 C++、OpenCV、FFmpeg/GStreamer、ONNX Runtime 等技术完成眼部检测、虹膜定位、虹膜分割、归一化、特征提取与匹配。

> 注意：普通 Android 手机摄像头主要是可见光摄像头，只适合虹膜识别算法实验与工程验证。高可靠虹膜识别通常需要近红外 NIR 摄像头和近红外补光。

---

## 2. 系统总体架构

```text
Android 手机端
    ↓
摄像头采集 Camera2 / NDK Camera
    ↓
预处理：自动对焦、曝光、眼部 ROI 裁剪
    ↓
编码：H.264 / MJPEG
    ↓
网络推流：RTSP / HTTP MJPEG / WebSocket
    ↓
PC / RK3588 / Linux C++ 接收端
    ↓
解码取帧
    ↓
人脸 / 眼睛检测
    ↓
虹膜定位
    ↓
虹膜分割
    ↓
虹膜归一化
    ↓
特征提取
    ↓
模板匹配
    ↓
识别结果输出
```

系统建议拆分为两个独立程序：

```text
1. AndroidIrisCamera
   Android 手机摄像头共享 App

2. IrisRecognitionClient
   Linux / Windows / RK3588 C++ 虹膜识别客户端
```

推荐第一阶段不要将虹膜识别算法直接放到 Android 端，而是让 Android 端只负责采集和推流，PC 或 RK3588 端负责算法处理。

---

## 3. 推荐技术路线

### 3.1 路线 A：快速实验版 MJPEG over HTTP

```text
Android App
    ↓
Camera2 / CameraX
    ↓
YUV420 帧
    ↓
JPEG 编码
    ↓
HTTP Multipart MJPEG 推流
    ↓
PC C++ OpenCV VideoCapture 读取
```

优点：

- 实现简单
- 调试方便
- OpenCV 端容易接入
- 适合第一版快速验证

缺点：

- 带宽占用较大
- 帧率一般
- 1080p 下可能只有 15–30 FPS
- 不适合长期工程化使用

适合验证内容：

```text
1. 手机摄像头能否稳定推流
2. PC 能否拿到视频帧
3. 是否可以检测眼睛 ROI
4. 是否可以进行虹膜分割实验
```

---

### 3.2 路线 B：工程化实验版 RTSP / H.264

```text
Android App
    ↓
NDK Camera / Camera2
    ↓
Surface / ImageReader
    ↓
MediaCodec H.264 硬编码
    ↓
RTSP Server
    ↓
PC / RK3588 C++ 接收
    ↓
FFmpeg / GStreamer / OpenCV 解码
```

优点：

- 带宽低
- 延迟低
- 帧率高
- 适合嵌入式设备接收
- 更接近真实网络摄像头架构

缺点：

- 开发复杂度较高
- RTSP 协议调试较麻烦
- 需要处理 H.264 NALU、RTP 打包、客户端连接管理等问题

建议最终采用此方案作为主线。

---

### 3.3 路线 C：低延迟高级版 WebRTC

```text
Android App
    ↓
Camera2 / CameraX
    ↓
H.264 / VP8 / VP9
    ↓
WebRTC
    ↓
C++ / Browser / Server 接收
```

WebRTC 适合低延迟远程视频通信，但第一阶段不建议使用。

原因：

```text
1. 需要 Signaling Server
2. 需要 STUN / TURN
3. 需要 ICE 连接协商
4. NAT 穿透复杂
5. 工程复杂度明显高于 RTSP 和 MJPEG
```

---

## 4. Android 端 App 架构

### 4.1 工程目录结构

```text
AndroidIrisCameraApp
├── app/
│   ├── MainActivity.kt
│   ├── CameraPermissionManager.kt
│   ├── CameraPreviewView.kt
│   ├── SettingsView.kt
│   └── JNI Bridge
│
├── native/
│   ├── camera/
│   │   ├── NdkCameraManager.cpp
│   │   ├── CameraDevice.cpp
│   │   ├── CaptureSession.cpp
│   │   └── FrameBuffer.cpp
│   │
│   ├── image/
│   │   ├── YuvConverter.cpp
│   │   ├── EyeRoiPreprocessor.cpp
│   │   └── FrameQualityEvaluator.cpp
│   │
│   ├── encoder/
│   │   ├── H264Encoder.cpp
│   │   ├── MjpegEncoder.cpp
│   │   └── EncoderConfig.cpp
│   │
│   ├── network/
│   │   ├── RtspServer.cpp
│   │   ├── HttpMjpegServer.cpp
│   │   ├── WebSocketServer.cpp
│   │   └── AuthManager.cpp
│   │
│   ├── security/
│   │   ├── TokenAuth.cpp
│   │   ├── LocalNetworkGuard.cpp
│   │   └── AccessLogger.cpp
│   │
│   └── jni/
│       ├── NativeBridge.cpp
│       └── NativeBridge.h
│
├── CMakeLists.txt
└── AndroidManifest.xml
```

---

### 4.2 Android 端模块职责

| 模块 | 职责 |
|---|---|
| MainActivity | UI、权限申请、启动/停止推流 |
| CameraPermissionManager | 管理 CAMERA、INTERNET、WAKE_LOCK 权限 |
| NdkCameraManager | 枚举前后摄像头、选择分辨率和 FPS |
| CaptureSession | 创建相机采集会话 |
| FrameBuffer | 环形缓冲区，缓存最新帧 |
| YuvConverter | YUV420 转 RGB / 灰度 |
| EyeRoiPreprocessor | 可选：裁剪眼部区域，降低传输压力 |
| H264Encoder | 硬件 H.264 编码 |
| MjpegEncoder | 实验版 MJPEG 编码 |
| RtspServer | 提供 rtsp://phone_ip:8554/iris |
| HttpMjpegServer | 提供 http://phone_ip:8080/video |
| AuthManager | Token 验证，避免未授权访问 |
| AccessLogger | 记录访问 IP、时间、连接状态 |

---

## 5. Android 端数据流设计

### 5.1 MJPEG 数据流

```text
Camera Frame YUV420
    ↓
YUV → RGB / BGR
    ↓
JPEG Encode
    ↓
HTTP Multipart Stream
    ↓
Client
```

HTTP MJPEG 响应格式示例：

```http
HTTP/1.1 200 OK
Content-Type: multipart/x-mixed-replace; boundary=frame

--frame
Content-Type: image/jpeg

<JPEG_BINARY>
--frame
Content-Type: image/jpeg

<JPEG_BINARY>
```

PC 端 OpenCV 读取方式：

```cpp
cv::VideoCapture cap("http://192.168.1.100:8080/video");
```

---

### 5.2 RTSP / H.264 数据流

```text
Camera Surface
    ↓
MediaCodec Input Surface
    ↓
H.264 NALU
    ↓
RTP Packetizer
    ↓
RTSP Server
    ↓
Client
```

RTSP 地址设计：

```text
rtsp://192.168.1.100:8554/iris
```

推荐编码参数：

| 参数 | 建议值 |
|---|---|
| 编码 | H.264 Baseline / Main |
| 分辨率 | 1280x720 或 1920x1080 |
| FPS | 30 |
| Bitrate | 4–8 Mbps |
| GOP | 1–2 秒 |
| 传输 | RTP over UDP 或 TCP |
| 延迟目标 | 80–200 ms |

---

## 6. PC / 嵌入式 C++ 接收端架构

### 6.1 工程目录结构

```text
IrisRecognitionClient
├── src/
│   ├── main.cpp
│   ├── stream/
│   │   ├── RtspStreamReader.cpp
│   │   ├── MjpegStreamReader.cpp
│   │   └── FrameQueue.cpp
│   │
│   ├── detect/
│   │   ├── FaceDetector.cpp
│   │   ├── EyeDetector.cpp
│   │   └── IrisLocator.cpp
│   │
│   ├── iris/
│   │   ├── IrisSegmenter.cpp
│   │   ├── IrisNormalizer.cpp
│   │   ├── IrisFeatureExtractor.cpp
│   │   ├── IrisMatcher.cpp
│   │   └── TemplateDatabase.cpp
│   │
│   ├── ui/
│   │   ├── QtPreviewWindow.cpp
│   │   └── ResultPanel.cpp
│   │
│   └── security/
│       ├── TemplateEncryptor.cpp
│       └── UserDatabase.cpp
│
├── models/
│   ├── eye_detector.onnx
│   ├── iris_segment_unet.onnx
│   └── liveness_model.onnx
│
├── CMakeLists.txt
└── config.yaml
```

---

### 6.2 接收端模块职责

| 模块 | 职责 |
|---|---|
| RtspStreamReader | 读取 RTSP/H.264 视频流 |
| MjpegStreamReader | 读取 HTTP MJPEG 视频流 |
| FrameQueue | 帧队列，只缓存最新帧 |
| FaceDetector | 人脸检测 |
| EyeDetector | 眼睛区域检测 |
| IrisLocator | 虹膜和瞳孔圆定位 |
| IrisSegmenter | 虹膜分割与 Mask 生成 |
| IrisNormalizer | 虹膜橡皮片模型归一化 |
| IrisFeatureExtractor | 特征提取，生成 IrisCode 或 embedding |
| IrisMatcher | 模板匹配，计算相似度 |
| TemplateDatabase | 用户模板管理 |
| TemplateEncryptor | 虹膜模板加密存储 |
| QtPreviewWindow | 视频预览与调试 UI |

---

## 7. 虹膜识别算法链路

完整虹膜识别流程如下：

```text
视频帧
  ↓
人脸检测 / 眼睛检测
  ↓
左眼 / 右眼 ROI 提取
  ↓
虹膜定位
  ↓
虹膜分割
  ↓
瞳孔边界检测
  ↓
虹膜外边界检测
  ↓
眼睑 / 睫毛 / 反光区域 Mask
  ↓
橡皮片模型归一化
  ↓
特征提取
  ↓
模板编码
  ↓
Hamming Distance 匹配
  ↓
识别结果
```

---

### 7.1 眼睛检测

第一版可使用：

```text
1. OpenCV Haar Cascade Eye Detector
2. OpenCV DNN Face Detector
3. MediaPipe Face Mesh
4. YOLOv5n-face / YOLOv8n-face
```

推荐路线：

```text
第一版：OpenCV Haar / DNN
第二版：YOLOv8n-face + landmark
第三版：专用 eye / iris detector
```

---

### 7.2 虹膜定位

传统虹膜定位方法：

```text
1. 灰度化
2. 高斯滤波
3. Canny 边缘检测
4. Hough Circle 检测瞳孔圆
5. Hough Circle 检测虹膜外圆
```

优点：

- 实现简单
- 适合作为 baseline
- 便于调试和可视化

缺点：

- 对反光敏感
- 对眼睑遮挡敏感
- 对光照变化敏感
- 对普通手机可见光图像效果有限

---

### 7.3 虹膜分割

建议保留两套方案：

| 路线 | 方法 | 适合阶段 |
|---|---|---|
| 传统算法 | Hough Circle + 边缘检测 + Mask | 第一版 |
| 深度学习 | U-Net / MobileNetV3-UNet / BiSeNet | 第二版 |

第二版建议使用轻量 U-Net 或 MobileNetV3-UNet，并导出 ONNX 模型，在 C++ 接收端使用 ONNX Runtime、OpenVINO、NCNN 或 MNN 推理。

---

### 7.4 虹膜归一化

使用经典 Daugman Rubber Sheet Model：

```text
将环形虹膜区域展开成固定尺寸矩形图
例如：64 x 512
```

输出结果：

```text
normalized_iris.png
iris_mask.png
```

---

### 7.5 特征提取

第一版建议使用传统方法：

```text
1. Gabor Filter
2. Log-Gabor Filter
3. LBP
4. ORB / SIFT 实验
```

第二版可尝试深度学习 embedding：

```text
1. 轻量 CNN embedding
2. MobileFaceNet 类似结构改造
3. ONNX Runtime 推理
```

注意：虹膜识别不是普通人脸识别。传统 IrisCode + Hamming Distance 仍然是非常重要的 baseline。

---

### 7.6 模板匹配

匹配过程：

```text
IrisCode A
IrisCode B
Mask A
Mask B
    ↓
有效区域异或
    ↓
Hamming Distance
```

判断逻辑：

```text
distance < threshold  → same user
distance >= threshold → different user
```

实验阈值可以先设置：

```text
0.30 ~ 0.38
```

最终阈值必须通过自己的样本集测试决定。

---

## 8. 线程模型设计

### 8.1 Android 端线程模型

```text
UI Thread
    └── 负责界面显示，不做重计算

Camera Thread
    └── 从 Camera2 / NDK Camera 获取帧

Preprocess Thread
    └── YUV 转换、裁剪、质量检测

Encoder Thread
    └── JPEG / H.264 编码

Network Thread
    └── RTSP / HTTP 发送

Control Thread
    └── 处理启动、停止、配置更新、认证
```

关键原则：

```text
1. UI 线程不做图像处理
2. 相机采集和编码分离
3. 网络发送不能阻塞采集线程
4. 帧队列只保留最新几帧
5. 宁可丢帧，不要延迟堆积
```

---

### 8.2 PC / RK3588 接收端线程模型

```text
Stream Thread
    └── 拉流、解码

Frame Queue
    └── 只保留最新 2~3 帧，避免延迟堆积

Detection Thread
    └── 人脸 / 眼睛 / 虹膜检测

Recognition Thread
    └── 特征提取和模板匹配

UI Thread
    └── 显示结果
```

对于虹膜识别，清晰度比帧率更重要。稳定 15–30 FPS 加清晰对焦，比追求 60 FPS 更有价值。

---

## 9. 网络接口设计

### 9.1 视频流接口

MJPEG：

```text
http://phone_ip:8080/video?token=xxxx
```

RTSP：

```text
rtsp://phone_ip:8554/iris?token=xxxx
```

---

### 9.2 控制接口

```http
GET  /api/status
POST /api/start
POST /api/stop
POST /api/config
GET  /api/snapshot
```

配置示例：

```json
{
  "camera": "back",
  "width": 1280,
  "height": 720,
  "fps": 30,
  "codec": "h264",
  "bitrate": 6000000,
  "roi_mode": "eye",
  "auth": true
}
```

状态接口示例：

```json
{
  "running": true,
  "ip": "192.168.1.100",
  "stream_url": "rtsp://192.168.1.100:8554/iris",
  "width": 1280,
  "height": 720,
  "fps": 30,
  "clients": 1,
  "battery": 82,
  "temperature": 39.5
}
```

---

## 10. 安全设计

由于项目涉及摄像头与生物特征数据，即使是实验项目，也必须加入基本安全机制。

### 10.1 必须实现

```text
1. App 前台运行时才允许开启摄像头
2. UI 明确显示“摄像头正在共享”
3. 每次启动生成随机 token
4. 默认只允许局域网访问
5. 禁止公网裸露端口
6. 访问日志记录 IP 和时间
7. 断开连接后自动停止推流，可选
8. 后台运行需要常驻通知
9. 虹膜模板加密保存
10. 原始眼部图像默认不长期保存
```

### 10.2 不建议实现

```text
1. 隐藏式后台调用摄像头
2. 无提示远程开启摄像头
3. 公网无密码访问
4. 长期保存他人虹膜数据
5. 未经授权采集生物特征
6. 将虹膜模板上传到不可信云端
```

虹膜属于高敏感生物特征数据。实验阶段建议只采集自己的数据，模板本地加密保存，不上传云端。

---

## 11. 数据存储设计

### 11.1 实验数据目录

实验阶段可以保存中间结果：

```text
data/
├── raw_frames/
├── eye_roi/
├── normalized_iris/
├── masks/
└── logs/
```

正式使用时，不建议长期保存原始眼部图像。

---

### 11.2 虹膜模板格式

```json
{
  "user_id": "user_001",
  "eye": "left",
  "template_version": "v1",
  "feature_type": "iris_code_log_gabor",
  "template": "base64_encoded_binary",
  "mask": "base64_encoded_binary",
  "created_at": "2026-05-31T10:00:00Z"
}
```

模板文件建议：

```text
1. AES-256-GCM 加密
2. 使用 PBKDF2 / Argon2 派生密钥
3. 不要明文保存
4. 不要直接保存可逆的原始虹膜图像
5. 模板文件应绑定用户 ID 和版本号
```

---

## 12. 性能目标

### 12.1 Android 端性能目标

| 项目 | 目标 |
|---|---|
| 分辨率 | 1280x720 第一版，1920x1080 第二版 |
| 帧率 | 15–30 FPS |
| 编码 | H.264 优先 |
| 延迟 | MJPEG 150–300 ms，RTSP 80–200 ms |
| CPU 占用 | 尽量 < 40% |
| 温度 | 避免长时间超过 45°C |
| 网络 | 局域网 WiFi 5GHz 优先 |

---

### 12.2 PC / RK3588 端性能目标

| 项目 | 目标 |
|---|---|
| 解码 | FFmpeg / GStreamer 硬解优先 |
| 识别频率 | 不必每帧识别，可每 5–10 帧识别一次 |
| UI 显示 | 30 FPS |
| 虹膜识别 | 3–10 FPS 足够 |
| 延迟控制 | 队列长度不超过 3 帧 |

---

## 13. MVP 版本规划

### 第 1 阶段：Android MJPEG 摄像头共享

```text
1. Android 端打开摄像头
2. 获取 YUV 帧
3. 转 JPEG
4. 通过 HTTP MJPEG 推流
5. PC OpenCV 读取视频流
```

目标：快速验证手机摄像头共享能力。

---

### 第 2 阶段：PC 端虹膜识别 Demo

```text
1. OpenCV 读取视频流
2. 检测人脸 / 眼睛
3. 截取眼部 ROI
4. Hough Circle 定位瞳孔和虹膜
5. 展开归一化
6. 提取简单特征
7. 做注册 / 验证流程
```

目标：跑通虹膜识别算法主链路。

---

### 第 3 阶段：升级 RTSP / H.264

```text
1. Android 端接入 MediaCodec
2. 输出 H.264
3. 实现 RTSP Server
4. PC / RK3588 端低延迟接收
```

目标：降低带宽，提高帧率和工程可用性。

---

### 第 4 阶段：深度学习增强

```text
1. YOLO / landmark 检测眼睛
2. U-Net 分割虹膜
3. ONNX Runtime / NCNN / MNN 推理
4. 加入活体检测
5. 加入质量评分机制
```

目标：提高可见光场景下的鲁棒性。

---

## 14. Android Native CMake 示例

```cmake
cmake_minimum_required(VERSION 3.22)
project(AndroidIrisCameraNative)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

add_library(
    iris_camera_native
    SHARED
    jni/NativeBridge.cpp
    camera/NdkCameraManager.cpp
    camera/CameraDevice.cpp
    camera/CaptureSession.cpp
    image/YuvConverter.cpp
    image/EyeRoiPreprocessor.cpp
    encoder/MjpegEncoder.cpp
    encoder/H264Encoder.cpp
    network/HttpMjpegServer.cpp
    network/RtspServer.cpp
    security/TokenAuth.cpp
)

find_library(log-lib log)
find_library(android-lib android)
find_library(camera2ndk-lib camera2ndk)
find_library(mediandk-lib mediandk)

target_link_libraries(
    iris_camera_native
    ${log-lib}
    ${android-lib}
    ${camera2ndk-lib}
    ${mediandk-lib}
)
```

---

## 15. 关键类设计

### 15.1 Android 端 NdkCameraManager

```cpp
class NdkCameraManager {
public:
    bool initialize();
    std::vector<CameraInfo> listCameras();
    bool openCamera(const std::string& cameraId);
    bool startCapture(int width, int height, int fps);
    void stopCapture();

private:
    ACameraManager* manager_ = nullptr;
    ACameraDevice* device_ = nullptr;
};
```

---

### 15.2 Android 端 FrameBuffer

```cpp
class FrameBuffer {
public:
    void push(Frame&& frame);
    bool popLatest(Frame& frame);
    void clear();

private:
    std::mutex mutex_;
    std::deque<Frame> queue_;
    size_t maxSize_ = 3;
};
```

---

### 15.3 Android 端 HttpMjpegServer

```cpp
class HttpMjpegServer {
public:
    bool start(int port);
    void stop();
    void publishJpegFrame(const std::vector<uint8_t>& jpeg);

private:
    std::atomic<bool> running_;
    std::vector<ClientConnection> clients_;
};
```

---

### 15.4 Android 端 H264Encoder

```cpp
class H264Encoder {
public:
    bool configure(int width, int height, int fps, int bitrate);
    bool start();
    bool encode(const Frame& frame, EncodedPacket& packet);
    void stop();
};
```

---

### 15.5 PC 端 RtspStreamReader

```cpp
class RtspStreamReader {
public:
    bool open(const std::string& url);
    bool read(cv::Mat& frame);
    void close();

private:
    cv::VideoCapture cap_;
};
```

---

### 15.6 PC 端 IrisRecognitionPipeline

```cpp
class IrisRecognitionPipeline {
public:
    RecognitionResult process(const cv::Mat& frame);

private:
    EyeDetector eyeDetector_;
    IrisSegmenter segmenter_;
    IrisNormalizer normalizer_;
    IrisFeatureExtractor extractor_;
    IrisMatcher matcher_;
};
```

---

### 15.7 PC 端 IrisMatcher

```cpp
class IrisMatcher {
public:
    double hammingDistance(
        const IrisTemplate& a,
        const IrisTemplate& b
    );

    bool verify(
        const IrisTemplate& input,
        const IrisTemplate& enrolled,
        double threshold
    );
};
```

---

## 16. UI 设计

### 16.1 Android 端 UI

```text
主界面
├── 摄像头预览
├── Start Sharing
├── Stop Sharing
├── 当前 IP 地址
├── MJPEG URL
├── RTSP URL
├── 当前分辨率 / FPS
├── 当前连接客户端数量
├── Token 显示 / 复制
└── 安全提示：Only local network access
```

必须明确显示摄像头共享状态，避免用户误以为摄像头已经关闭。

---

### 16.2 PC 端 UI

```text
主界面
├── 视频预览
├── 眼睛 ROI 显示
├── 虹膜分割 Mask
├── 归一化虹膜图像
├── 当前匹配用户
├── 匹配距离
├── 注册按钮
├── 验证按钮
└── 日志窗口
```

PC 端 UI 可以使用 Qt 实现，便于你后续和 C++/Qt 工程经验结合。

---

## 17. 推荐依赖库

### 17.1 Android 端

| 功能 | 推荐技术 |
|---|---|
| Camera API | Camera2 / NDK Camera |
| 编码 | MediaCodec / Media NDK |
| C++ 构建 | CMake + Android NDK |
| JNI | JNI Bridge |
| HTTP Server | 自实现轻量 HTTP / cpp-httplib 移植 |
| RTSP Server | live555 / 自实现最小 RTSP / GStreamer |
| 日志 | Android logcat |

---

### 17.2 PC / Linux / RK3588 端

| 功能 | 推荐技术 |
|---|---|
| 视频读取 | OpenCV VideoCapture |
| 视频解码 | FFmpeg / GStreamer |
| 图像处理 | OpenCV |
| 深度学习推理 | ONNX Runtime / OpenVINO / NCNN / MNN |
| UI | Qt Widgets / Qt Quick |
| 配置文件 | YAML-CPP |
| 加密 | OpenSSL / libsodium |
| 日志 | spdlog |

---

## 18. 工程风险与解决方案

| 风险 | 说明 | 建议 |
|---|---|---|
| 手机摄像头对焦不稳定 | 虹膜需要清晰纹理 | 锁定焦距，增加质量评分 |
| 可见光虹膜纹理不足 | 普通手机对暗色虹膜效果差 | 后续使用 NIR 摄像头 |
| 网络延迟 | 帧队列堆积导致识别滞后 | 只保留最新 2–3 帧 |
| MJPEG 带宽大 | 高分辨率下压力明显 | 第二版改 H.264 |
| 反光干扰 | 眼镜、屏幕光源会影响分割 | 加入反光 Mask 和质量评分 |
| 生物特征数据泄露 | 虹膜模板敏感 | 加密模板，避免保存原图 |
| 手机发热 | 长时间编码会升温 | 降低分辨率、帧率、码率 |
| 公网暴露风险 | 摄像头接口可能被扫描 | 仅局域网，使用 token 和 VPN |

---

## 19. 推荐最终架构

```text
Android 手机端：
Camera2 / NDK Camera
+ MediaCodec H.264
+ RTSP Server
+ Token Auth
+ 前台 UI 显示

Linux C++ 接收端：
OpenCV / FFmpeg / GStreamer
+ Eye Detection
+ Iris Segmentation
+ Iris Normalization
+ IrisCode Feature
+ Hamming Distance Matching
+ Qt UI

长期优化：
可见光实验
→ 外接 NIR 摄像头
→ NIR 光源
→ 高质量虹膜识别
```

---

## 20. 总结

本项目建议按照以下路径推进：

```text
第一版：Android MJPEG 推流 + PC OpenCV 读取
第二版：PC 端跑通传统虹膜识别算法
第三版：Android 端升级 RTSP / H.264
第四版：加入 U-Net 虹膜分割和 ONNX 推理
第五版：加入模板加密、安全控制和质量评分
第六版：迁移到 RK3588 或其他嵌入式平台
```

核心原则：

```text
1. 手机端只负责采集和推流
2. 识别端负责算法处理和模板管理
3. 第一版优先跑通链路，不追求完美性能
4. 第二版重点优化清晰度、延迟和分割质量
5. 虹膜模板必须加密保存
6. 摄像头共享必须有明显 UI 提示和访问控制
```

一句话结论：

> 第一版用 MJPEG 快速跑通；第二版升级 RTSP/H.264；虹膜识别先放在 PC/RK3588 C++ 端；普通手机摄像头只适合实验，真正可靠的虹膜识别最好使用近红外摄像头和近红外补光。
