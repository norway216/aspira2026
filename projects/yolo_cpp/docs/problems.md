# YOLOv8-seg ONNX Runtime C++ 部署问题记录

> 项目：`yolo_cpp` — YOLOv8 分割模型 ONNX Runtime C++ 推理  
> 日期：2026-05-31  
> 目的：记录从 Python 导出 ONNX 到 C++ 推理跑通过程中遇到的所有问题、根因分析及解决方案，供后续嵌入式部署参考。

---

## 目录

1. [问题1：NumPy 版本冲突 — Python 环境不兼容](#问题1numpy-版本冲突--python-环境不兼容)
2. [问题2：数据布局反转 — 多维数组索引错误](#问题2数据布局反转--多维数组索引错误)
3. [问题3：重复 Sigmoid — 激活函数二次应用](#问题3重复-sigmoid--激活函数二次应用)
4. [问题4：Bbox 格式误解 — Grid-Relative vs 绝对坐标](#问题4bbox-格式误解--grid-relative-vs-绝对坐标)
5. [问题5：悬空指针 — Ort::Value 不拷贝数据（致命Bug）](#问题5悬空指针--ortvalue-不拷贝数据致命bug)
6. [问题6：Letterbox 坐标逆变换不准确](#问题6letterbox-坐标逆变换不准确)

---

## 问题1：NumPy 版本冲突 — Python 环境不兼容

### 现象

```text
ImportError: A module that was compiled using NumPy 1.x cannot be run in
NumPy 2.4.6 as it may crash.
```

执行 `python3 scripts/export_yolo_seg_onnx.py` 时，`import ultralytics` 抛出 ImportError，程序无法启动。

### 根因

系统中存在两套 NumPy 相关包的冲突：

| 包 | 来源 | NumPy 版本 |
|---|---|---|
| `matplotlib` | 系统 apt 安装 (`/usr/lib/python3/dist-packages/`) | NumPy 1.x ABI |
| `ultralytics` + `torch` | `pip install --break-system-packages` | NumPy 2.4.6 |

`pip install ultralytics` 自动拉取了 NumPy 2.4.6 作为依赖，但系统预装的 matplotlib 是用 NumPy 1.x 的 C API 编译的，ABI 不兼容。

### 解决

将 matplotlib 也升级到与 NumPy 2.x 兼容的版本：

```bash
pip install --break-system-packages matplotlib --upgrade
```

升级到 `matplotlib-3.10.9` 后问题解决。

### 经验教训

- 在已有系统 Python 包的 Ubuntu 24.04 上，`pip install` 容易与系统包冲突
- 更干净的方案是使用 `python3 -m venv` 创建虚拟环境，但需要 `python3-venv` 包（需要 sudo）
- 次优方案：统一用 pip 管理所有包，确保版本一致

---

## 问题2：数据布局反转 — 多维数组索引错误

### 现象

```text
检测到 7705 个目标:
  [1] bicycle  635.60  (0,0,1,900)
  [2] person   635.21  (0,0,1,1)
  [3] spoon    634.95  (0,0,1200,900)
  ...
```

置信度显示为 600+（sigmoid 输出不可能 >1），bbox 坐标几乎全是 `(0,0,0,0)` 或全图，显然数据读取完全错位。

### 根因

ONNX 模型输出 `output0` 的形状是 `[1, 116, 8400]`，在 C 语言行优先（row-major）内存布局下：

```
正确的元素访问:  data[channel * 8400 + anchor]
错误的元素访问:  data[anchor * 116 + channel]   ← 代码中实际使用的方式
```

**图示说明：**

```
output0 形状: [batch=1, channels=116, anchors=8400]

内存布局 (row-major):
┌──────────────────────────────────────────────────┐
│ ch0_a0 │ ch0_a1 │ ch0_a2 │ ... │ ch0_a8399 │     │  ← channel 0 (bbox cx)
│ ch1_a0 │ ch1_a1 │ ch1_a2 │ ... │ ch1_a8399 │     │  ← channel 1 (bbox cy)
│ ch2_a0 │ ch2_a1 │ ...                            │  ← channel 2 (bbox w)
│ ch3_a0 │ ch3_a1 │ ...                            │  ← channel 3 (bbox h)
│ ch4_a0 │ ch4_a1 │ ...                            │  ← channel 4 (class 0 score)
│  ...                                              │
│ ch115_a0 │ ...                    │ ch115_a8399 │  ← channel 115 (mask coeff 31)
└──────────────────────────────────────────────────┘

通道内步长 (stride) = 8400 (相邻 anchor 之间的距离)
通道间步长 = 8400 (相邻 channel 同一个 anchor 的距离)

对于 anchor a, channel c:
  正确偏移: offset = c * 8400 + a
  错误偏移: offset = a * 116 + c   ← 跨通道错位！
```

### 解决

将数据访问改为以 `stride = numAnchors = 8400` 为步长：

```cpp
// 正确写法
const int stride = numAnchors;  // 8400

for (int a = 0; a < numAnchors; ++a) {
    // 边界框: channels 0-3
    float cx = output0Tensor[a + 0 * stride];
    float cy = output0Tensor[a + 1 * stride];
    float bw = output0Tensor[a + 2 * stride];
    float bh = output0Tensor[a + 3 * stride];

    // 类别分数: channels 4-83
    for (int c = 0; c < 80; ++c) {
        float score = output0Tensor[a + (4 + c) * stride];
        // ...
    }

    // Mask 系数: channels 84-115
    for (int k = 0; k < 32; ++k) {
        float coeff = output0Tensor[a + (4 + 80 + k) * stride];
        // ...
    }
}
```

### 经验教训

- 处理多维数组时必须明确内存布局（row-major vs column-major）
- ONNX Runtime 的 `GetTensorData<T>()` 返回的是 C 行优先的连续内存
- 不要凭直觉假设 stride — 手动计算并验证

---

## 问题3：重复 Sigmoid — 激活函数二次应用

### 现象

```text
[后处理] 候选数: 8397 → NMS后: 7705
```

8400 个 anchor 中 8397 个通过了置信度阈值 0.25，几乎全部通过，显然不正常。

### 根因

ultralytics 导出的 YOLOv8-seg ONNX 模型中，**类别分数已经过了 sigmoid 激活**（数值范围 `[0, 0.92]`，无负值）。但代码在后处理中又调用了一次 sigmoid：

```cpp
// 错误：对已经 sigmoid 过的值再次 sigmoid
float score = 1.0f / (1.0f + std::exp(-classScores[c]));
```

**数值分析：**

| 原始 ONNX 输出 | 实际含义 | 二次 sigmoid 结果 |
|---|---|---|
| 0.000 | 极低置信度 | `sigmoid(0) = 0.5` |
| 0.500 | 中等置信度 | `sigmoid(0.5) ≈ 0.62` |
| 0.920 | 高置信度 | `sigmoid(0.92) ≈ 0.71` |

原本 0.000 的 class score（几乎确定不是该类）经过二次 sigmoid 变成了 0.5（50% 概率），远超 0.25 阈值，导致几乎所有 anchor 都成为候选。

**验证方法：** 用 Python 直接检查 ONNX 输出的 class score 数值范围：

```python
out0, out1 = sess.run(None, {'images': inp})
cls_scores = out0[0, 4:84, :]  # [80, 8400]
print(f"min={cls_scores.min():.4f}, max={cls_scores.max():.4f}")
# 输出: min=0.0000, max=0.9205
# 非负 → 已 sigmoid
```

### 解决

直接使用 ONNX 输出的 class score，不做任何激活：

```cpp
// 正确：直接读取，无需 sigmoid
float maxScore = 0.0f;
for (int c = 0; c < classChannels; ++c) {
    if (classScores[c] > maxScore) {
        maxScore = classScores[c];
        bestClassId = c;
    }
}
```

### 经验教训

- 模型导出的 ONNX 图中可能已包含激活层，需要验证
- 判断方法：检查输出数值范围 — sigmoid 输出 ≥ 0；raw logit 通常有负值
- ultralytics 的 `Detect` 头在 `export=True` 时会预应用 sigmoid

---

## 问题4：Bbox 格式误解 — Grid-Relative vs 绝对坐标

### 现象

```text
检测到 22 个目标:
  [1] person        635.60  (594,444,603,456)
  [2] bottle        635.21  (254,112,571,586)
  [5] sink          634.90  (0,0,0,0)      ← 空框
```

部分 bbox 坐标为 `(0,0,0,0)`，部分覆盖整个图像，置信度超过 1.0。

### 根因

代码假设 ONNX 输出的 bbox 是 YOLO 传统的 **grid-relative 格式**（需要 sigmoid、exp、stride 解码），但 ultralytics 导出时已将 bbox **预解码为 640×640 空间的绝对坐标 `[cx, cy, w, h]`**。

**传统 YOLO 格式（训练时）：**
```
bbox = [cx_offset, cy_offset, w_raw, h_raw]
cx = (grid_x + sigmoid(cx_offset)) * stride
cy = (grid_y + sigmoid(cy_offset)) * stride
w  = exp(w_raw) * stride  (或基于 anchor)
h  = exp(h_raw) * stride
```

**ultralytics ONNX 导出格式（推理时）：**
```
bbox = [cx_abs, cy_abs, w_abs, h_abs]  ← 已经解码完毕的绝对坐标
```

**验证方法：** 用 Python 直接对比同一个 anchor 的 ONNX 输出和 ultralytics 最终结果：

```python
# ONNX 原始输出
cx, cy, w, h = out0[0, :4, 7339]  # [302.1, 376.5, 105.3, 92.4]

# 直接转 x1,y1,x2,y2 再映射回原图
x1 = cx - w/2; y1 = cy - h/2
# → [468, 470, 668, 643] 与 ultralytics 结果完全一致！
```

### 解决

移除所有 grid-relative 解码逻辑，直接使用 ONNX 输出的绝对坐标：

```cpp
// 正确：直接读取，已经是 640x640 空间的绝对坐标
float cx = output0Tensor[a + 0 * stride];
float cy = output0Tensor[a + 1 * stride];
float bw = output0Tensor[a + 2 * stride];
float bh = output0Tensor[a + 3 * stride];

// 转换为 [x1, y1, x2, y2]
float x1 = cx - bw / 2.0f;
float y1 = cy - bh / 2.0f;
```

### 经验教训

- 不同框架/不同导出方式的 ONNX 模型输出格式不同，必须验证
- 用 Python 对比 ONNX 原始输出和框架的最终后处理结果是验证格式的最快方法
- `ultralytics` 的 `export=True` 模式下，`Detect` 头会预执行 `dist2bbox` 解码

---

## 问题5：悬空指针 — Ort::Value 不拷贝数据（致命Bug）

### 现象

此 Bug 有三种不同表现，取决于编译模式和内存状态：

| 构建模式 | 表现 |
|---------|------|
| Release (`-O2`) | `段错误 (SIGSEGV)`，崩溃在模型初始化完成后 |
| Debug (`-g -O0`) | 程序正常运行，但推理结果全错（bbox 偏位，class score 全零） |
| Debug + ASAN | 程序正常运行，能检测到部分问题 |
| GDB 调试 | **间歇性正常运行**（最迷惑人的表现） |

崩溃位置在主函数中紧接 `engine.initialize()` 之后——而此时尚未调用 `infer()`，说明崩溃发生在函数返回后的某个栈展开/析构过程中。

### 根因

`Ort::Value::CreateTensor()` **不拷贝传入的数据**，只持有原始指针。

```cpp
// 文件: src/yolo_seg_inference.cpp — 原始错误代码
Ort::Value YoloSegInference::preprocess(const cv::Mat& image)
{
    // ...
    cv::Mat blob = cv::dnn::blobFromImage(...);  // ← 栈上的局部变量
    float* blobData = reinterpret_cast<float*>(blob.data);

    return Ort::Value::CreateTensor<float>(
        memoryInfo,
        blobData,                    // ← 只传了指针，没有拷贝数据!
        blob.total() * sizeof(float),
        inputShape.data(),
        inputShape.size()
    );
}   // ← 函数返回时 blob 析构，释放内存
    //   Ort::Value 内部的指针变成 悬空指针 (dangling pointer)
```

**ONNX Runtime API 行为确认：**

```cpp
// Ort::Value::CreateTensor 签名
static Value CreateTensor(
    const MemoryInfo& info,
    void* p_data,              // ← 用户负责管理此内存的生命周期
    size_t p_data_byte_count,
    const int64_t* shape,
    size_t shape_len
);
// 文档说明: The instance does NOT take ownership of the data.
// 数据必须保持有效直到 Ort::Value 被销毁
```

**为什么 GDB 下偶尔能正常运行：** 调试器会改变内存分配/释放的模式，已释放的内存可能尚未被覆盖，侥幸读到"看起来正确"的值。

**为什么 Release 模式崩溃：** 优化编译后，栈空间被立即重用，Ort::Value 持有的指针指向了完全不同的数据，导致 ONNX Runtime 内部访问越界。

### 解决

引入成员变量 `std::vector<float> m_inputBuffer`，在创建 Tensor 之前将数据**拷贝**到成员变量中，确保数据的生命周期与 `YoloSegInference` 对象一致：

```cpp
// 头文件: src/yolo_seg_inference.h
class YoloSegInference {
private:
    // 输入数据缓冲区 — Ort::Value 不拷贝数据，需要保持其生命周期
    std::vector<float> m_inputBuffer;
};

// 实现文件: src/yolo_seg_inference.cpp
Ort::Value YoloSegInference::preprocess(const cv::Mat& image)
{
    cv::Mat blob = cv::dnn::blobFromImage(...);
    float* blobData = reinterpret_cast<float*>(blob.data);
    const size_t totalElements = blob.total();

    // 关键修复：将数据拷贝到成员变量，确保 Ort::Value 使用时数据仍然有效
    m_inputBuffer.assign(blobData, blobData + totalElements);

    return Ort::Value::CreateTensor<float>(
        memoryInfo,
        m_inputBuffer.data(),    // 指向成员变量，不会随函数返回而失效
        m_inputBuffer.size() * sizeof(float),
        inputShape.data(),
        inputShape.size()
    );
}
```

### 经验教训

| 教训 | 说明 |
|------|------|
| **任何接受原始指针的 C API 封装必须确认拷贝语义** | `CreateTensor` 表面是现代 C++ API（`Ort::Value`），底层仍是零拷贝的 C 风格设计 |
| **局部变量 + 裸指针 + 异步/延迟使用 = 必崩** | 函数返回后局部变量销毁，但引用它的对象还在 |
| **GDB 下正常 ≠ 代码正确** | 调试器改变了内存复用模式，掩盖了悬空指针问题 |
| **ASAN 是排查此类问题的利器** | 即使不能直接检测到悬空指针（因为 Ort::Value 持有的指针不在 ASAN 追踪范围内），ASAN 改变了内存布局，让问题更容易暴露 |
| **三种表现同一根因** | Segfault / 零输出 / 偶尔正常——都来自同一个悬空指针 |

---

## 问题6：Letterbox 坐标逆变换不准确

### 现象

18 个候选检测通过了置信度阈值，但 bbox 映射回原图 1200×900 时，位置和尺寸有偏差。

### 根因

将 bbox 从 640×640 输入空间映射回原图时，最初使用了简单的等比缩放：

```cpp
// 错误：忽略了 letterbox 的 padding 偏移
float scaleX = (float)originalSize.width / inputWidth;
float scaleY = (float)originalSize.height / inputHeight;
int xOrig = bbox.x * scaleX;
int yOrig = bbox.y * scaleY;
```

但 letterbox 预处理包括了居中对齐的 padding（本例中上下各 80 像素）：

```text
┌──────────────────────────┐
│  padding (gray 114)      │  ← dy = 80px 上部填充
├──────────────────────────┤
│                          │
│  缩放后的图像             │  ← 640×480 实际图像区域
│  (保持宽高比)             │
│                          │
├──────────────────────────┤
│  padding (gray 114)      │  ← 下部填充
└──────────────────────────┘
        640×640
```

### 正确变换

```cpp
// 逆 letterbox 变换
x_orig = (x_640 - padding_dx) / scale
y_orig = (y_640 - padding_dy) / scale
w_orig = w_640 / scale
h_orig = h_640 / scale
```

### 解决

1. **预处理时**记录 letterbox 参数到成员变量：

```cpp
// 在 letterbox() 中计算并保存
m_letterboxScale = scale;
m_letterboxDx = dx;
m_letterboxDy = dy;
```

2. **后处理时**使用逆变换：

```cpp
float x1Orig = (cand.bbox.x - m_letterboxDx) / m_letterboxScale;
float y1Orig = (cand.bbox.y - m_letterboxDy) / m_letterboxScale;
float wOrig  = cand.bbox.width / m_letterboxScale;
float hOrig  = cand.bbox.height / m_letterboxScale;
```

### 经验教训

- 预处理和后处理是一对镜像操作，中间参数必须妥善保存
- 简单的等比缩放只在无需 padding（原图本身就是正方形）时有效
- 将预处理参数（scale, dx, dy）作为成员变量保存是最简洁的方案

---

## 问题修复时间线

```
1. NumPy 冲突       [10min]  pip 升级 matplotlib
2. 数据布局反转      [30min]  改 stride = 8400
3. 重复 Sigmoid      [15min]  移除 sigmoid
4. Bbox 格式误解     [20min]  移除 grid-relative 解码
5. 悬空指针 ← 最耗时  [90min]  ASAN 定位 → 发现 CreateTensor 不拷贝
6. Letterbox 变换    [10min]  保存参数 → 逆变换
```

---

## 相关资源

- [ONNX Runtime C++ API 文档](https://onnxruntime.ai/docs/api/c/)
- [ultralytics YOLOv8 导出文档](https://docs.ultralytics.com/modes/export/)
- [Ort::Value::CreateTensor 源码](https://github.com/microsoft/onnxruntime/blob/main/include/onnxruntime_cxx_api.h)
- 本工程 `scripts/export_yolo_seg_onnx.py` — 用于对比验证 ONNX 输出的 Python 脚本
