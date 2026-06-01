#!/usr/bin/env python3
"""
PH2/ISIC 皮肤镜病灶分割模型训练脚本
======================================
基于 YOLOv8n-seg COCO 预训练权重做迁移学习，训练单类（lesion）分割模型。
导出 ONNX 格式供 C++ 推理引擎使用。

用法：
  python3 scripts/train_ph2_seg.py [--data ph2_data/ph2.yaml] [--epochs 100]
"""

import os, sys, argparse, time
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(PROJECT_ROOT))


def main():
    parser = argparse.ArgumentParser(description="Train YOLOv8-seg on PH2/ISIC")
    parser.add_argument("--data", default=str(PROJECT_ROOT / "ph2_data" / "ph2.yaml"),
                        help="YOLO 数据集配置文件路径")
    parser.add_argument("--epochs", type=int, default=100, help="训练轮数")
    parser.add_argument("--imgsz", type=int, default=640, help="输入图像尺寸")
    parser.add_argument("--batch", type=int, default=8, help="Batch size")
    parser.add_argument("--lr", type=float, default=0.001, help="初始学习率")
    parser.add_argument("--model", default="yolov8n-seg.pt",
                        help="预训练权重 (yolov8n-seg.pt / yolov8s-seg.pt)")
    parser.add_argument("--device", default="cpu", help="训练设备 (cpu / cuda:0)")
    parser.add_argument("--export", action="store_true", default=True,
                        help="训练后导出 ONNX")
    parser.add_argument("--no-export", action="store_true", help="不导出 ONNX")
    args = parser.parse_args()

    from ultralytics import YOLO

    data_path = Path(args.data)
    if not data_path.exists():
        print(f"[错误] 数据集配置文件不存在: {data_path}")
        print("  请先运行: python3 scripts/ph2_dataset_prep.py --synthetic")
        sys.exit(1)

    print("=" * 55)
    print("  YOLOv8-seg 皮肤镜病灶分割 — 模型训练")
    print("=" * 55)
    print(f"  数据集配置: {data_path}")
    print(f"  训练轮数:   {args.epochs}")
    print(f"  图像尺寸:   {args.imgsz}×{args.imgsz}")
    print(f"  Batch:      {args.batch}")
    print(f"  学习率:     {args.lr}")
    print(f"  设备:       {args.device}")
    print(f"  预训练:     {args.model}")
    print("=" * 55)

    # ── 加载模型 ──────────────────────────────────────────────────────
    print("\n[1/4] 加载预训练模型...")
    model = YOLO(args.model)
    print(f"  模型已加载: {args.model}")

    # ── 训练 ──────────────────────────────────────────────────────────
    print(f"\n[2/4] 开始训练 ({args.epochs} epochs)...")
    t0 = time.time()

    results = model.train(
        data=str(data_path),
        epochs=args.epochs,
        imgsz=args.imgsz,
        batch=args.batch,
        lr0=args.lr,
        device=args.device,
        verbose=True,
        # 数据增强（医学图像适用）
        hsv_h=0.015,      # 轻微色调抖动
        hsv_s=0.3,        # 饱和度增强
        hsv_v=0.2,        # 亮度增强
        degrees=30.0,     # 随机旋转 ±30°
        translate=0.1,    # 随机平移 ±10%
        scale=0.3,        # 随机缩放
        shear=2.0,        # 剪切
        flipud=0.2,       # 上下翻转概率
        fliplr=0.5,       # 左右翻转概率
        mosaic=0.0,       # 禁用 mosaic（小数据集不适用）
        # 优化器
        optimizer="AdamW",
        cos_lr=True,      # 余弦退火学习率
    )

    elapsed = time.time() - t0
    print(f"\n  训练完成！耗时: {elapsed/60:.1f} 分钟")

    # ── 验证 ──────────────────────────────────────────────────────────
    print(f"\n[3/4] 验证模型...")
    val_results = model.val()
    print(f"  验证结果: {val_results}")

    # ── 导出 ONNX ─────────────────────────────────────────────────────
    if args.export and not args.no_export:
        print(f"\n[4/4] 导出 ONNX 模型...")
        onnx_path = model.export(
            format="onnx",
            imgsz=args.imgsz,
            opset=12,
            simplify=True,
            dynamic=False,
        )
        print(f"  ONNX 模型已导出: {onnx_path}")

        # 复制到 models/ 目录
        dest = PROJECT_ROOT / "models" / "yolov8n-seg-ph2.onnx"
        import shutil
        if Path(onnx_path) != dest:
            shutil.copy2(onnx_path, dest)
            print(f"  已复制到: {dest}")

    print(f"\n{'=' * 55}")
    print(f"  训练流程完成！")
    print(f"  模型: {PROJECT_ROOT / 'models' / 'yolov8n-seg-ph2.onnx'}")
    print(f"{'=' * 55}")


if __name__ == "__main__":
    main()
