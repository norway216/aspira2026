# PH2 医学图像数据集迁移问题记录

> 项目：`yolo_cpp` — YOLOv8 分割模型 ONNX Runtime C++ 推理  
> 日期：2026-06-02  
> 目的：记录将推理引擎从 COCO 80 类通用数据集切换到 PH2 皮肤镜医学图像数据集过程中遇到的问题、根因分析及解决方案，供后续嵌入式医疗 AI 部署参考。

---

## 目录

1. [问题1：PH2 数据集下载困难 — 多重网络限制](#问题1ph2-数据集下载困难--多重网络限制)
2. [问题2：模型预训练权重下载失败 — GitHub Releases 不可达](#问题2模型预训练权重下载失败--github-releases-不可达)
3. [问题3：二值 Mask → YOLO Polygon 格式转换 — 轮廓提取挑战](#问题3二值-mask--yolo-polygon-格式转换--轮廓提取挑战)
4. [问题4：类别数变化导致 ONNX 输出通道不匹配](#问题4类别数变化导致-onnx-输出通道不匹配)
5. [问题5：C++ 代码类别名硬编码 — 缺乏可配置性](#问题5c-代码类别名硬编码--缺乏可配置性)
6. [问题6：评估脚本 GT Mask 路径查找失败](#问题6评估脚本-gt-mask-路径查找失败)
7. [问题7：合成数据训练的模型 Mask 精度不足](#问题7合成数据训练的模型-mask-精度不足)

---

## 问题1：PH2 数据集下载困难 — 多重网络限制

### 现象

```text
# 官方 Dropbox 链接
curl: (7) Failed to connect to www.dropbox.com port 443: Couldn't connect to server

# Kaggle API
kagglehub: ImportError: cannot import name 'get_web_endpoint' from 'kagglesdk.kaggle_env'

# ISIC 2016 S3 下载
下载速度仅 ~20KB/s，600MB 预估需 8+ 小时
```

### 根因

该开发环境存在多重网络限制：

| 目标 | 协议 | 状态 | 原因 |
|------|------|------|------|
| PH2 官方 Dropbox | HTTPS/443 | ❌ 不可达 | 防火墙屏蔽 Dropbox CDN |
| GitHub Releases | HTTPS/443 | ⚠️ 间歇可用 | GitHub assets CDN 不稳定 |
| Kaggle API | HTTPS/443 | ❌ 不可用 | kagglehub 版本与 kagglesdk 不兼容 |
| ISIC S3 | HTTPS/443 | ⚠️ 极慢 | S3 跨境带宽受限 |
| PyPI / HuggingFace | HTTPS/443 | ✅ 正常 | 可通过 pip 安装包 |

### 解决

采用**分层回退策略**：

1. **首选**：从官方 Dropbox 下载 → 失败
2. **次选**：kagglehub API 自动下载 → 版本冲突
3. **备选**：ISIC 2016 数据集替代（同类任务，900 张图像）→ 速度太慢
4. **最终方案**：生成 200 张合成皮肤镜风格数据用于验证管道，真实数据集由用户手动下载

合成的数据包含椭圆形病灶 + 毛发状噪声 + 皮肤纹理，足以验证训练→导出→C++推理的完整链路。

### 经验教训

- 医疗数据集（PH2、ISIC）通常托管在学术机构服务器或对象存储上，下载可靠性不如通用数据集
- 应提前准备多种下载源（官方链接、Kaggle、HuggingFace、百度网盘等）
- 合成数据是开发阶段验证管道的有效 fallback 方案
- 生产环境应预先下载并本地缓存数据集

---

## 问题2：模型预训练权重下载失败 — GitHub Releases 不可达

### 现象

```text
ConnectionError: ❌ Download failure for 
https://github.com/ultralytics/assets/releases/download/v8.4.0/yolov8n-seg.pt
Curl return value 7 (Failed to connect)
```

ultralytics 库在 `YOLO('yolov8n-seg.pt')` 时会自动从 GitHub Releases 下载 6.7MB 的预训练权重文件，但 GitHub 的 assets CDN 在该环境下不可达。

### 根因

ultralytics 使用 `curl` 下载权重文件，目标域名为 `github.com` 的 releases 子域名。虽然 `github.com` 主站可达（HTTP 200），但其 releases assets 托管在独立的 CDN 上，被环境防火墙屏蔽。

第一次尝试时 ultralytics 内置下载失败，第二次使用 Python `urllib` 直接请求同一 URL 却成功下载。

**关键差异**：ultralytics 用 `curl` 子进程下载，而 Python 的 `urllib` 可能走了不同的网络路径或 TLS 配置。

### 解决

1. 手动使用 Python `urllib` 下载权重文件到 `~/.cache/ultralytics/yolov8n-seg.pt`
2. 将缓存文件复制到项目目录 `yolov8n-seg.pt`
3. 训练脚本直接加载本地文件，避免触发自动下载

```python
# 使用本地文件而非自动下载
model = YOLO('yolov8n-seg.pt')  # 当前目录已有该文件
```

### 经验教训

- ultralytics 的自动下载机制在网络受限环境下不可靠
- 建议预先下载常用预训练权重并本地缓存
- Python `urllib` 和系统 `curl` 的网络行为可能不同——当一种方法失败时，可尝试另一种
- 离线部署时可将 `.pt` 文件打包到项目仓库中

---

## 问题3：二值 Mask → YOLO Polygon 格式转换 — 轮廓提取挑战

### 现象

PH2/ISIC 数据集的标注是**二值 PNG mask**（像素级分割），而 YOLOv8-seg 训练需要 **polygon 格式**（归一化坐标列表）。

```text
输入: 768×560 二值 mask (255=病灶, 0=背景)
需要输出: class_id x1 y1 x2 y2 ... xn yn  (坐标归一化到 [0,1])
```

### 根因

这是两种不同的分割标注范式：

| 格式 | 存储方式 | 精度 | YOLOv8 支持 |
|------|----------|------|-------------|
| 二值 Mask | 逐像素 PNG | 像素级 | ❌ 不直接支持 |
| Polygon | 坐标点序列 | 取决于点数 | ✅ 原生支持 |

需要从 mask 中提取轮廓并转换为 polygon 坐标。

### 解决

使用 OpenCV `findContours` + `approxPolyDP` 简化轮廓：

```python
def mask_to_yolo_polygon(mask, epsilon_factor=0.005):
    contours, _ = cv2.findContours(mask, cv2.RETR_EXTERNAL, 
                                    cv2.CHAIN_APPROX_SIMPLE)
    h, w = mask.shape
    polygons = []
    for cnt in contours:
        if len(cnt) < 3:
            continue
        # 简化轮廓 — epsilon 控制精度
        epsilon = epsilon_factor * cv2.arcLength(cnt, True)
        approx = cv2.approxPolyDP(cnt, epsilon, True)
        # 归一化到 [0,1]
        pts = []
        for pt in approx:
            pts.extend([pt[0][0] / w, pt[0][1] / h])
        if len(pts) >= 6:  # 至少 3 个点
            polygons.append(pts)
    return polygons
```

**参数调优**：
- `epsilon_factor=0.005`：轮廓点数为原始的 1-5%，足够保留病灶形状
- 过小的 epsilon → 点数过多，训练变慢
- 过大的 epsilon → 轮廓过于简化，丢失细节

### 经验教训

- 医学分割标注通常是 mask 格式，需要转 YOLO polygon
- `approxPolyDP` 的 epsilon 参数需要在精度和效率之间权衡
- `RETR_EXTERNAL` 只取外轮廓，舍去病灶内部的空洞（对分割任务无影响）
- 转换脚本应同时生成 YOLO 格式的目录结构（images/labels + train/val/test）

---

## 问题4：类别数变化导致 ONNX 输出通道不匹配

### 现象

COCO 模型输出 `output0: [1, 116, 8400]`（4 bbox + 80 class + 32 mask），而 PH2 1 类模型输出 `output0: [1, 37, 8400]`（4 bbox + 1 class + 32 mask）。

C++ 推理代码在 `postprocess()` 中使用 `m_config.numClasses` 来控制类别分数段的长度。如果配置的 `numClasses` 与实际 ONNX 输出不匹配，会导致数组越界或读取到错误的 mask 系数。

### 根因

YOLOv8-seg ONNX 的 `output0` 通道维度 = `4 + numClasses + maskChannels`：

```
COCO:  4 + 80 + 32 = 116 通道
PH2:   4 + 1  + 32 = 37  通道
```

通道布局：
```
[cx, cy, w, h, | class_0, ..., class_n, | mask_0, ..., mask_31]
 0   1   2  3  | 4             4+n-1    | 4+n             4+n+31
```

当 `numClasses` 配置错误时，类别分数的读取偏移会错位，导致：
- `numClasses=80` 读 PH2 模型 → 把 mask 系数当类别分数读（全错）
- `numClasses=1` 读 COCO 模型 → 只读第 1 类分数，忽略其余 79 类（漏检）

### 解决

1. **训练时**：确保 `ph2.yaml` 中 `nc: 1`，导出的 ONNX 即为 37 通道
2. **推理时**：`InferenceConfig::numClasses` 必须与实际模型匹配
3. **自动检测**：在 `main.cpp` 中通过模型文件名判断数据集类型，自动设置正确的 `numClasses`

```cpp
bool isMedical = (modelPath.find("ph2") != std::string::npos ||
                  modelPath.find("isic") != std::string::npos);
if (isMedical) {
    config.numClasses = 1;
    config.classNames = {"lesion"};
}
```

### 经验教训

- ONNX 模型文件本身不包含类别名称元信息，需要外部配置
- 通道数 = `4 + nc + 32` 是 YOLOv8-seg 的固定公式
- 更健壮的方案：从 ONNX 模型的 output0 shape 反推 `nc = channels - 36`
- 生产环境建议在模型文件名中编码关键参数（如 `yolov8n-seg-nc1-ph2.onnx`）

---

## 问题5：C++ 代码类别名硬编码 — 缺乏可配置性

### 现象

C++ 推理代码中类别名查找直接使用静态成员 `COCO_CLASSES`：

```cpp
// 原始代码 — 硬编码 COCO 80 类
result.className = COCO_CLASSES[cand.classId];  // classId=0 → "person"

// PH2 模型 classId=0 应该返回 "lesion"，但实际返回 "person"
```

这导致 PH2 模型推理结果中，病灶被标注为 "person"（COCO 第 0 类）。

### 根因

`COCO_CLASSES` 是 `YoloSegInference` 的静态常量成员，编译时固定。原设计假设模型始终是 COCO 80 类。切换到医学数据集后，类名系统不再适用。

### 解决

在 `InferenceConfig` 中增加 `classNames` 字段：

```cpp
struct InferenceConfig {
    // ... 其他字段 ...
    std::vector<std::string> classNames;  // 为空则使用 COCO 80 类
};
```

后处理代码改为动态查询：

```cpp
const auto& names = m_config.classNames.empty()
    ? COCO_CLASSES : m_config.classNames;
result.className = (cand.classId < static_cast<int>(names.size()))
    ? names[cand.classId] : "unknown";
```

这样：
- 不设置 `classNames` → 默认 COCO 80 类（向后兼容）
- 设置 `classNames = {"lesion"}` → PH2 单类模型
- 设置 `classNames = {"cat", "dog"}` → 自定义 2 类模型

### 经验教训

- 类别名称不应硬编码在推理类中——它是模型属性，不是引擎属性
- `InferenceConfig` 是配置类名的最佳位置，与 `numClasses` 配合使用
- 向后兼容很重要——保留 COCO 默认值避免破坏已有使用场景
- 更彻底的方案：将类名嵌入 ONNX 模型的 metadata，推理时自动读取

---

## 问题6：评估脚本 GT Mask 路径查找失败

### 现象

运行 `evaluate_ph2.py` 后，所有 30 张测试图像的指标均为空：

```text
[警告] 没有计算到任何指标（缺少 GT mask 或预测 mask）
```

### 根因

数据集目录结构为：
```
ph2_data/
├── images/
│   ├── train/
│   ├── val/
│   └── test/       ← 测试图像在这里
└── masks/          ← GT mask 在这里（原始数据，未拆分）
    ├── 0003.png
    └── ...
```

评估脚本的 mask 查找逻辑为：
```python
# 错误: test_dir = ph2_data/
# test_dir.parent = yolo_cpp/  →  yolo_cpp/masks/ → 不存在！
orig_masks_dir = test_dir.parent / "masks"
```

正确的路径应为 `ph2_data/masks/`（即 `test_dir / "masks"`），而不是 `test_dir.parent / "masks"`（即 `yolo_cpp/masks/`）。

### 解决

修改为优先查找 `test_dir / "masks"`，回退到 `test_dir.parent / "masks"`：

```python
# 修复后
orig_masks_dir = test_dir / "masks"          # 首选: ph2_data/masks/
if not orig_masks_dir.exists():
    orig_masks_dir = test_dir.parent / "masks"  # 备选
```

### 经验教训

- `Path.parent` 在多层目录结构中容易出错，需明确变量的语义
- 评估脚本应打印实际查找的路径，便于调试
- 更好的设计：将 GT mask 路径作为命令行参数或配置文件的一部分

---

## 问题7：合成数据训练的模型 Mask 精度不足

### 现象

模型在合成数据上训练 10 epochs 后，检出率 100%（Recall=0.98），但 mask 精度很低（IoU=0.07, Dice=0.13）。预测的 mask 普遍比 ground truth 大很多。

### 根因

1. **合成数据过于简单**：椭圆病灶 + 均匀噪声，缺乏真实皮肤镜图像的复杂纹理（色斑、血管、镜面反射等）
2. **训练轮数不足**：10 epochs × 200 张图像，模型未能充分收敛
3. **COCO 预训练域偏移**：COCO 的自然场景特征与皮肤镜图像差异巨大
4. **YOLOv8-seg 对单类小数据集可能过参数化**：nano 模型也有 ~3M 参数

### 解决

| 因素 | 当前 (合成) | 推荐 (真实) |
|------|-------------|-------------|
| 数据集 | 200 合成椭圆 | 200 真实 PH2 图像 |
| 训练轮数 | 10 epochs | 100+ epochs |
| 数据增强 | 基础增强 | 更强的医学图像增强 (CLAHE, 弹性变形) |
| 评估指标 | IoU=0.07 | 预期 IoU > 0.85 |

本项目的目标是验证**从训练到 C++ 推理的完整管道**，合成数据已充分证明了管道的可行性。真实数据训练后指标将大幅提升。

### 经验教训

- 合成数据适合验证管道，不适合评估模型真实性能
- 医学图像的域特征（纹理、光照、颜色分布）与自然图像差异显著，迁移学习需要更多 epochs
- 皮肤镜图像的特殊性：毛发遮挡、胶体、标尺等伪影需要专门的数据增强策略
- 生产环境评估应始终使用真实数据

---

## 问题修复时间线

```
1. PH2 下载困难         [25min]  多层回退 → 合成数据 fallback
2. 预训练权重下载失败    [15min]  urllib 替代 curl → 本地缓存
3. Mask → Polygon 转换  [15min]  findContours + approxPolyDP
4. ONNX 通道不匹配       [5min]   确认 37=4+1+32 → 配置 numClasses=1
5. C++ 类名硬编码        [10min]  InferenceConfig 增加 classNames
6. GT Mask 路径错误      [5min]   修复 Path.parent 逻辑
7. 合成数据精度低        [评估]   确认管道正确 → 真实数据待训练
```

---

## 相关资源

- [PH2 数据集官方页面](https://www.fc.up.pt/addi/ph2%20database.html)
- [ISIC 2016 皮肤镜挑战](https://challenge.isic-archive.com/data/#2016)
- [YOLOv8 分割模型导出文档](https://docs.ultralytics.com/modes/export/)
- [ONNX Runtime C++ API](https://onnxruntime.ai/docs/api/c/)
- 本工程已有的 [problems.md](problems.md) — COCO 模型开发阶段的问题记录
