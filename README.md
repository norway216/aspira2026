# aspira2026

> C++ · Go · Qt/QML · 嵌入式 Linux · AI 推理 · 网络系统

个人技术仓库，涵盖嵌入式 Linux 系统开发、C++ 底层编程、Qt/QML 桌面应用、Go 后端服务、AI 模型推理、局域网监控、安全网关等领域。

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | C++ (C++17/20), Go, Python, Shell |
| 系统 | Linux (Ubuntu/Debian), Yocto, Buildroot |
| 平台 | RK3568 / RK3588, Jetson, ARM, x86-64 |
| 框架 | Qt 5.15 + QML, Gin, Vue 3, ONNX Runtime |
| AI/ML | YOLO (v5/v8), ONNX, OpenCV, dlib, PyTorch |
| 存储 | PostgreSQL, SQLite, Redis |
| 工具 | Docker, CMake, Git, GTest, spdlog, OpenSSL |

## 项目清单

### Go 后端服务

#### 🔌 [lan-monitor](projects/lan-monitor/) — 局域网设备监控管理系统
局域网资产发现 + 在线状态监控 + 流量可视化 Web 运维平台。周期性 ARP/Ping/TCP 扫描局域网设备，WebSocket 实时推送设备状态、告警、流量数据，支持用户 RBAC 权限管理和审计日志。
- `Go` `Gin` `SQLite` `JWT` `WebSocket` `ECharts` `Docker`

#### 📡 [embedded-ops-platform](projects/embedded-ops-platform/) — 分布式嵌入式设备运维平台
分布式嵌入式设备 Web 运维管理系统。Agent 零配置自动注册、心跳监控、CPU/内存/GPU/温度指标采集、白名单远程命令执行、USB/WiFi/音频设备检测、WebSocket 实时状态推送。
- `Go` `Gin` `PostgreSQL` `Redis` `WebSocket` `JWT+HMAC` `Vue 3` `Naive UI` `Docker`

#### 🛡️ [secure-gateway](projects/secure-gateway/) — 分布式安全流量网关
控制面/数据面分离的多节点安全流量网关。TLS 加密传输、负载均衡、用户管理、节点监控、Prometheus 指标采集、流量统计。
- `Go` `Gin` `PostgreSQL` `Redis` `JWT` `Vue 3` `Element Plus` `Docker Compose`

#### 🚦 [traffic_lab_proxy](projects/traffic_lab_proxy/) — 多节点代理流量调度平台
实验性质的多节点代理流量调度系统。令牌桶速率限制、多策略负载均衡（轮询/最少连接/最少带宽/加权/组合）、节点心跳检测、Prometheus + Grafana 监控。
- `Go` `chi` `SQLite` `Prometheus` `Grafana` `Docker Compose`

### C++ 嵌入式 & 桌面应用

#### 💰 [hardware-wallet](projects/hardware-wallet/) — RK3568 硬件加密钱包
功能完整的 Qt/QML 硬件加密货币钱包。Ed25519 密钥对管理、KEK/DEK 分层加密、BIP39 助记词、AES-GCM 加密备份与恢复、交易签名验证、审计哈希链、3 次输错密码自毁机制、Repository+Service+ViewModel 分层架构。
- `C++20` `Qt 5.15` `QML` `OpenSSL` `libsodium` `SQLite` `spdlog` `CMake`

#### 🔑 [embedded_hardware_wallet](projects/embedded_hardware_wallet/) — ARM 嵌入式硬件钱包
ARM 嵌入式平台硬件钱包。ECC 密钥管理、加密安全存储、分布式备份（Shamir 秘密共享）、TrustZone/Secure Enclave 安全隔离、QML UI。
- `C++20` `Qt 5.15` `QML` `OpenSSL` `CMake`

#### 👁️ [iris_recognition_system](projects/iris_recognition_system/) — 虹膜识别系统
完整的虹膜识别流程：采集 → 人眼检测 → 质量检查 → 分割 → 归一化 → 特征提取 → 活体检测 → 匹配 → 决策。支持传统 CV（Gabor + IrisCode）和 ONNX/RKNN/TensorRT 神经网络推理，CLI + GUI 双模式。
- `C++20` `OpenCV` `ONNX Runtime` `OpenSSL` `CMake`

#### 😊 [embedded_face_recognition](projects/embedded_face_recognition/) — 嵌入式人脸识别
基于 OpenCV HOG/CNN 检测 + dlib 128 维特征向量的人脸识别系统。多线程流水线（RingBuffer + ThreadPool），支持摄像头/视频/图片输入。
- `C++20` `OpenCV` `dlib` `CMake`

#### 🩻 [ultrasound_system](projects/ultrasound_system/) — 超声成像演示系统
Qt/QML 超声成像工程骨架。仿真 B/M/Color/PW/Triplex 模式实时成像、探头插拔检测、冻结/解冻、Cine 回放、参数调节、测量标注、患者注册、DICOM/PACS/USB/打印模拟。
- `C++17` `Qt 5.15` `QML` `Quick Controls 2` `CMake`

#### 🔬 [ultrasound_software_project](projects/ultrasound_software_project/) — 超声软件项目
Qt/QML 超声软件早期迭代版本。包含图像处理、内存池、线程池、B 模式图像生成、参数管理、数据库设置等模块。
- `C++20` `Qt 5.15` `QML` `CMake`

### AI 模型推理

#### 🤖 [yolo_cpp](projects/yolo_cpp/) — YOLOv8 C++ 分割推理
使用 ONNX Runtime 在 C++ 中运行 YOLOv8 分割模型。包含 PH2 皮肤病变数据集训练、ONNX 模型导出、多线程后处理优化、OpenCV 图像预处理。
- `C++20` `ONNX Runtime 1.18` `OpenCV` `YOLOv8` `Python/PyTorch` `CMake`

### 嵌入式 Linux

#### 📶 [aic8800-driver](embedded/linux/aic8800-driver/) — AIC8800 WiFi 6 驱动
AIC8800 芯片 (AX300 WiFi 6 USB) Linux 驱动编译与部署。预编译内核模块、固件、udev 规则，适配 Debian 12 + LB-LINK BL-WN300AX。
- `Linux Kernel` `ARM` `Cross-compile` `Shell`

#### 🐳 [aic8800-deploy](embedded/linux/aic8800-deploy/) — AIC8800 驱动部署包
预编译的 .ko 内核模块、固件文件、udev 规则及一键部署脚本。
- `Shell` `Linux`

## 架构设计文档

以下为独立的设计文档，部分已有对应代码实现：

| 文档 | 对应代码 | 说明 |
|------|----------|------|
| [go_lan_device_web_system_architecture.md](projects/go_lan_device_web_system_architecture.md) | [lan-monitor](projects/lan-monitor/) | 局域网设备检测与运维 Web 系统设计 |
| [distributed_embedded_web_ops_platform_design.md](projects/distributed_embedded_web_ops_platform_design.md) | [embedded-ops-platform](projects/embedded-ops-platform/) | 分布式嵌入式设备运维平台设计 |
| [distributed_secure_traffic_gateway_design.md](projects/distributed_secure_traffic_gateway_design.md) | [secure-gateway](projects/secure-gateway/) | 分布式安全流量网关设计 |
| [traffic_lab_proxy_architecture.md](projects/traffic_lab_proxy_architecture.md) | [traffic_lab_proxy](projects/traffic_lab_proxy/) | 多节点代理流量调度平台设计 |
| [hardware_wallet_architecture.md](projects/hardware_wallet_architecture.md) | [hardware-wallet](projects/hardware-wallet/) | RK3568 硬件钱包架构设计 |
| [embedded_hardware_wallet_architecture.md](projects/embedded_hardware_wallet_architecture.md) | [embedded_hardware_wallet](projects/embedded_hardware_wallet/) | ARM 嵌入式硬件钱包架构设计 |
| [iris_recognition_cpp_nn_architecture.md](projects/iris_recognition_cpp_nn_architecture.md) | [iris_recognition_system](projects/iris_recognition_system/) | 虹膜识别 NN 推理系统架构设计 |
| [embedded_face_recognition_architecture.md](projects/embedded_face_recognition_architecture.md) | [embedded_face_recognition](projects/embedded_face_recognition/) | 嵌入式人脸识别架构设计 |
| [android_iris_camera_architecture.md](projects/android_iris_camera_architecture.md) | — | Android 手机作为虹膜采集相机的方案设计 |
| [minimal_portfolio_architecture.md](projects/minimal_portfolio_architecture.md) | — | 极简风格个人作品集网站 (Aspira Studio) 设计 |

## 基础学习

| 目录 | 内容 |
|------|------|
| [cpp_basic/algorithm](cpp_basic/algorithm/) | 常用算法实现 (排序、DP 等) |
| [cpp_basic/basic](cpp_basic/basic/) | C++ 基础练习 (智能指针、模板、线程池、内存池等) |
| [cpp_basic/notes](cpp_basic/notes/) | C/C++ 学习笔记 (结构体、内存布局) |
| [dl_basic](dl_basic/) | 深度学习入门 (PyTorch、FCN、U-Net) |
| [imaging/basic](imaging/basic/) | 图像处理基础 (OpenCV 人脸检测、分割、QR 码等) |
| [embedded/kernel](embedded/kernel/) | 内核构建笔记 |

## 仓库统计

```
项目总数:     ~18 个项目 (含代码实现 + 设计文档)
Go 项目:      4 个 (lan-monitor, embedded-ops-platform, secure-gateway, traffic_lab_proxy)
C++ 项目:     7 个 (hardware-wallet, embedded_hardware_wallet, iris_recognition_system,
                     embedded_face_recognition, ultrasound_system, ultrasound_software_project, yolo_cpp)
设计文档:     10 个 (.md 架构设计文档)
```

## 联系方式

- GitHub: [@norway216](https://github.com/norway216)

---

<p align="center"><sub>持续更新中 · 2024–2026</sub></p>
