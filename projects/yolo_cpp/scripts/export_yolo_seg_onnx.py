#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
==============================================================================
步骤1：YOLOv8 分割模型导出 ONNX 脚本
==============================================================================

本脚本完成以下工作：
  1. 使用 ultralytics 库加载 YOLOv8n-seg 预训练分割模型
  2. 将 PyTorch 模型导出为 ONNX 格式（含检测头 + 分割头 + 原型Mask）
  3. 解析 ONNX 模型的输入/输出节点信息（名称、形状、数据类型）
  4. 使用 ONNX Runtime 跑一次实际推理，验证导出模型的正确性
  5. 保存 ONNX 模型到 models/yolov8n-seg.onnx

YOLOv8-seg ONNX 输出格式说明：
  - output0: [1, 116, 8400] 检测预测
      其中 116 = 4(边界框xywh) + 80(COCO类别分数) + 32(Mask系数)
      8400 = 80x80 + 40x40 + 20x20 三个特征图网格数
  - output1: [1, 32, 160, 160] 原型Mask（prototype masks）
      32个原型mask，每个160x160，用于与mask系数组合生成实例分割

运行方式：
  python3 scripts/export_yolo_seg_onnx.py

依赖：
  pip install ultralytics onnx onnxruntime
==============================================================================
"""

import os
import sys
import numpy as np

# ── 项目路径设置 ──────────────────────────────────────────────────────────
# 获取脚本所在目录的上级目录（即 yolo_cpp 工程根目录）
PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MODELS_DIR = os.path.join(PROJECT_ROOT, "models")
os.makedirs(MODELS_DIR, exist_ok=True)

# ── 第1步：加载 YOLOv8 分割模型 ──────────────────────────────────────────
print("=" * 70)
print("[步骤1] 加载 YOLOv8n-seg 预训练模型...")
print("=" * 70)

from ultralytics import YOLO

# YOLOv8n-seg 是 ultralytics 提供的轻量级分割模型
# 'n' = nano（最小），适合嵌入式设备部署实验
# 首次运行会自动从网络下载预训练权重（约 6.7MB）
model = YOLO("yolov8n-seg.pt")
print(f"  模型类别数: {model.model.model[-1].nc}")     # COCO: 80类
print(f"  分割头维度: {model.model.model[-1].nm}")     # Mask系数: 32
print(f"  模型参数量: {sum(p.numel() for p in model.model.parameters()) / 1e6:.2f}M")

# ── 第2步：导出为 ONNX 格式 ──────────────────────────────────────────────
print("\n" + "=" * 70)
print("[步骤2] 将 PyTorch 模型导出为 ONNX 格式...")
print("=" * 70)

onnx_path = os.path.join(MODELS_DIR, "yolov8n-seg.onnx")

# ultralytics 的 export 方法会：
#   - 自动追踪模型计算图（torch.onnx.export）
#   - 合并后处理（如需要）到 ONNX 图中
#   - 对于分割模型，输出包含检测预测 + 原型mask
# 参数说明：
#   format='onnx'      — 导出为 ONNX 格式
#   imgsz=640          — 输入图像尺寸 640x640
#   opset=12           — ONNX opset 版本（嵌入式设备通常支持 11-17）
#   simplify=True      — 使用 onnx-simplifier 简化计算图
#   dynamic=False      — 固定 batch size（嵌入式设备通常一次推理一张图）
# ultralytics export 会将文件保存到当前工作目录，而不是我们指定的路径
# 因此我们先导出，然后手动移动到目标位置
success = model.export(
    format="onnx",
    imgsz=640,
    opset=12,
    simplify=True,
    dynamic=False,
)

# 查找导出的 ONNX 文件并移动到 models/ 目录
import glob
exported_files = glob.glob("*.onnx")
if exported_files and not os.path.exists(onnx_path):
    for f in exported_files:
        dest = os.path.join(MODELS_DIR, f)
        if os.path.exists(dest):
            os.remove(dest)  # 覆盖旧文件
        import shutil
        shutil.move(f, dest)
        print(f"  已将 ONNX 文件移动到: {dest}")

# 确认文件已存在
if not os.path.exists(onnx_path):
    # 尝试在当前目录找
    current_dir_onnx = os.path.join(os.getcwd(), "yolov8n-seg.onnx")
    if os.path.exists(current_dir_onnx):
        import shutil
        shutil.move(current_dir_onnx, onnx_path)
        print(f"  已将 ONNX 文件移动到: {onnx_path}")

print(f"  ONNX 模型已保存到: {onnx_path}")
if os.path.exists(onnx_path):
    print(f"  文件大小: {os.path.getsize(onnx_path) / 1024 / 1024:.2f} MB")
else:
    print(f"  ⚠ 文件未找到，可能在当前目录下")
    onnx_path = os.path.join(os.getcwd(), "yolov8n-seg.onnx")
    if os.path.exists(onnx_path):
        print(f"  找到文件: {onnx_path}，大小: {os.path.getsize(onnx_path) / 1024 / 1024:.2f} MB")

# ── 第3步：使用 ONNX Runtime 解析模型信息 ────────────────────────────────
print("\n" + "=" * 70)
print("[步骤3] 使用 ONNX Runtime 解析模型输入/输出信息...")
print("=" * 70)

import onnxruntime as ort

# 创建 ONNX Runtime 推理会话
# providers=['CPUExecutionProvider'] 使用 CPU 推理
# 嵌入式设备上可替换为:
#   - 'CUDAExecutionProvider'      (NVIDIA Jetson)
#   - 'OpenVINOExecutionProvider'  (Intel)
#   - 'CoreMLExecutionProvider'    (Apple)
#   - 'QNNExecutionProvider'       (Qualcomm)
session = ort.InferenceSession(onnx_path, providers=['CPUExecutionProvider'])

# ── 解析输入信息 ──────────────────────────────────────────────────────────
print("\n[模型输入]")
for inp in session.get_inputs():
    print(f"  名称: {inp.name}")
    print(f"  形状: {inp.shape}")            # 动态维度用字符串表示，如 'batch'
    print(f"  数据类型: {inp.type}")          # tensor(float) 或 tensor(int64)
    print(f"  说明: BCHW 格式 — Batch=1, Channel=3(RGB), Height=640, Width=640")
    input_name = inp.name

# ── 解析输出信息 ──────────────────────────────────────────────────────────
print("\n[模型输出]")
for out in session.get_outputs():
    print(f"  名称: {out.name}")
    print(f"  形状: {out.shape}")
    print(f"  数据类型: {out.type}")

# 对两个输出做详细说明
outputs_info = session.get_outputs()
if len(outputs_info) >= 1:
    print(f"\n  输出0 (检测预测) 详细说明:")
    print(f"    output0 形状: {outputs_info[0].shape}")
    print(f"    维度含义: [Batch=1, Channels=116, Anchors=8400]")
    print(f"    Channels 组成: 4(bbox) + 80(class_scores) + 32(mask_coefficients)")
    print(f"    Anchors=8400 = (80×80 + 40×40 + 20×20) 三个特征图网格")

if len(outputs_info) >= 2:
    print(f"\n  输出1 (原型Mask) 详细说明:")
    print(f"    output1 形状: {outputs_info[1].shape}")
    print(f"    维度含义: [Batch=1, Channels=32, Height=160, Width=160]")
    print(f"    32 个原型 mask，每个 160×160")
    print(f"    使用方法: 将 output0 中的 mask 系数与原型 mask 做矩阵乘法")
    print(f"    公式: instance_mask = sigmoid(mask_coefficients @ proto_masks)")

# ── 第4步：使用 ONNX Runtime 进行推理验证 ────────────────────────────────
print("\n" + "=" * 70)
print("[步骤4] 使用 ONNX Runtime 进行推理验证...")
print("=" * 70)

# 构造一张虚拟输入图像（全零，仅验证模型能跑通）
# 实际使用时需要做预处理：读取图片 → Resize 640x640 → BGR→RGB → /255.0 → CHW
dummy_input = np.random.randn(1, 3, 640, 640).astype(np.float32)

print(f"  输入图像形状: {dummy_input.shape}")
print(f"  输入数值范围: [{dummy_input.min():.3f}, {dummy_input.max():.3f}]")

# 运行推理
outputs = session.run(None, {input_name: dummy_input})

for i, out in enumerate(outputs):
    print(f"\n  output{i} 形状: {out.shape}")
    print(f"  output{i} 数值范围: [{out.min():.4f}, {out.max():.4f}]")
    print(f"  output{i} 数据类型: {out.dtype}")

print("\n  ✓ ONNX Runtime 推理验证成功！")

# ── 第5步：用一张真实图片做端到端测试 ────────────────────────────────────
print("\n" + "=" * 70)
print("[步骤5] 用真实图片做端到端测试（Python侧验证完整管线）...")
print("=" * 70)

# 尝试用测试图片做推理
test_image_path = os.path.join(PROJECT_ROOT, "images", "road.jpg")
if os.path.exists(test_image_path):
    import cv2
    from ultralytics import YOLO

    # 读取图片
    img = cv2.imread(test_image_path)
    print(f"  测试图片: {test_image_path}")
    print(f"  图片尺寸: {img.shape} (H×W×C)")

    # 使用 ultralytics 原始模型做推理（作为对比基准）
    results = model(img, imgsz=640, conf=0.25, iou=0.45)
    result = results[0]

    # 打印检测结果
    if result.boxes is not None and len(result.boxes) > 0:
        print(f"\n  检测到 {len(result.boxes)} 个目标:")
        for i, (box, cls_id) in enumerate(zip(result.boxes.xyxy, result.boxes.cls)):
            class_name = result.names[int(cls_id)]
            conf = result.boxes.conf[i]
            print(f"    [{i+1}] {class_name:12s}  置信度: {conf:.3f}  框: {box.tolist()}")

        # 保存可视化结果（Python侧基准输出）
        annotated = result.plot()
        baseline_path = os.path.join(PROJECT_ROOT, "images", "baseline_result.jpg")
        cv2.imwrite(baseline_path, annotated)
        print(f"\n  基准结果已保存: {baseline_path}")
        print(f"  （后续C++推理结果将与此基准结果对比验证）")
    else:
        print("  未检测到目标（可能图片场景与COCO类别不匹配）")
else:
    print(f"  测试图片不存在: {test_image_path}，跳过真实图片测试")

# ── 完成 ──────────────────────────────────────────────────────────────────
print("\n" + "=" * 70)
print("[完成] ONNX 模型已成功导出并验证！")
print(f"  模型位置: {onnx_path}")
print(f"  下一步: 使用 C++ 加载此 ONNX 模型进行推理")
print("=" * 70)
