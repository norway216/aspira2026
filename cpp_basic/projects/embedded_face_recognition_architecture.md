嵌入式人脸检测与识别系统架构设计文档

---

## 1. 系统总体架构

```
+-----------------------------------------------------------+
|                          UI 层 (QML)                      |
|-----------------------------------------------------------|
| 主界面显示      | 参数调节面板                               |
| - 视频显示      | - 阈值调整                                 |
| - 检测框叠加   | - 模型选择                                 |
| - 识别结果叠加 | - FPS显示                                  |
+-----------------------------------------------------------+
           ↑               ↑
           |               |
+-----------------------------------------------------------+
|                   应用逻辑层 (C++20)                     |
|-----------------------------------------------------------|
| CameraManager / VideoManager                                |
| - 摄像头检测/切换/视频文件加载                              |
| FaceDetectionManager (OpenCV + dlib)                        |
| - 人脸检测 (HOG/CNN)                                        |
| - 多线程人脸裁剪 / 预处理                                    |
| FaceRecognitionManager (dlib)                               |
| - 特征提取 / 人脸比对                                        |
| - 多线程批处理                                              |
| DataPipeline / ThreadPool                                    |
| - CameraThread -> PreprocessThread -> AIInferenceThread -> RenderThread |
| - RingBuffer/MemoryPool优化                                  |
| - 高性能锁 / 条件变量控制线程                                |
+-----------------------------------------------------------+
           ↑               ↑
           |               |
+-----------------------------------------------------------+
|                       底层支持层                             |
|-----------------------------------------------------------|
| FFmpeg 解码视频文件                                          |
| OpenGL / EGL / GPU 加速                                      |
| Embedded Linux / Board BSP                                    |
| SystemMonitor / ResourceManager                               |
+-----------------------------------------------------------+
```

## 2. 关键模块说明

1. **摄像头与视频管理**
   - `CameraManager`：检测设备摄像头，如果存在则使用摄像头，否则调用 `VideoManager` 读取指定目录的视频文件。
   - 视频读取支持 FFmpeg 解码，保证不同编码格式（H.264/H.265/VP9）可用。
   - 提供帧率控制和环形缓冲，避免 UI 阻塞。

2. **人脸检测 (FaceDetectionManager)**
   - 使用 OpenCV 捕获图像，dlib 提供 HOG/CNN 检测。
   - 多线程处理每一帧：
     - **CameraThread**：抓取视频帧。
     - **PreprocessThread**：灰度化、缩放、对齐等预处理。
   - 使用内存池 + RingBuffer 减少内存分配开销。

3. **人脸识别 (FaceRecognitionManager)**
   - dlib 提供 128 维特征向量。
   - 特征比对使用多线程批处理，保证高并发识别。
   - 支持动态库加载不同模型（可替换或更新）。

4. **渲染与 UI**
   - QML + Qt SceneGraph + OpenGL ES 渲染。
   - GPU 加速显示检测框与识别结果。
   - UI 支持参数调节（阈值、模型选择）和实时 FPS 显示。
   - 采用异步信号槽刷新，确保 UI 60 FPS 流畅显示。

5. **多线程与性能优化**
   - **ThreadPool** 管理所有处理线程。
   - **RingBuffer + MemoryPool** 解决帧数据拷贝问题。
   - 条件变量 (`QWaitCondition`) 控制线程执行顺序。
   - 性能统计和 FPS 监控模块，用于优化嵌入式资源。

6. **构建与工程管理**
   - 使用 CMake 管理工程，支持跨平台编译。
   - 目录结构推荐：
     ```
     embedded_face_recognition/
     ├── src/
     │   ├── camera/
     │   ├── detection/
     │   ├── recognition/
     │   ├── pipeline/
     │   └── ui/
     ├── include/
     ├── resources/
     ├── cmake/
     └── main.cpp
     ```
   - 支持模块化编译、单元测试和日志系统集成。

