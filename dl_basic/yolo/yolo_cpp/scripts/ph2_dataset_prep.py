#!/usr/bin/env python3
"""
PH2 数据集准备脚本
  1. 下载/获取 PH2 皮肤镜数据集
  2. 将二值 mask 转为 YOLOv8-seg polygon 格式
  3. 划分 train/val/test 并生成 ph2.yaml

用法：
  # 尝试自动下载
  python3 scripts/ph2_dataset_prep.py

  # 使用已有 PH2 数据
  python3 scripts/ph2_dataset_prep.py --input-dir /path/to/PH2

  # 生成合成数据用于测试管道
  python3 scripts/ph2_dataset_prep.py --synthetic
"""

import os, sys, glob, shutil, argparse, hashlib, zipfile, io, urllib.request
from pathlib import Path
import numpy as np
import cv2

PROJECT_ROOT = Path(__file__).resolve().parent.parent


def download_ph2_direct(output_dir: Path) -> bool:
    """尝试直接从多个镜像源下载 PH2 数据集"""
    urls = [
        # ADDI 官方下载页（可能需要手动在浏览器中下载）
        # 尝试一些已知的镜像
    ]

    # 尝试使用 Kaggle API
    try:
        import kagglehub
        print("[下载] 通过 kagglehub 下载 PH2...")
        path = kagglehub.dataset_download("eswarbalu/processed-dataset")
        for f in Path(path).rglob("*"):
            if f.suffix.lower() in (".png", ".jpg", ".bmp"):
                dest = output_dir / "images" / f.name
                dest.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(f, dest)
        print(f"[下载] 完成: {path}")
        return True
    except Exception:
        pass

    return False


def create_synthetic_ph2(output_dir: Path) -> bool:
    """生成 200 张合成皮肤镜风格图像 + mask，用于开发测试"""
    print("[生成] 创建合成 PH2 数据集...")
    img_dir = output_dir / "images"
    msk_dir = output_dir / "masks"
    img_dir.mkdir(parents=True, exist_ok=True)
    msk_dir.mkdir(parents=True, exist_ok=True)
    np.random.seed(42)

    for i in range(200):
        # 皮肤色背景
        img = np.random.randint(25, 70, (560, 768, 3), dtype=np.uint8)
        # 毛发噪声
        for _ in range(np.random.randint(5, 20)):
            x1, y1 = np.random.randint(0, 768), np.random.randint(0, 560)
            a, L = np.random.uniform(0, np.pi), np.random.randint(20, 150)
            x2 = int(x1 + L * np.cos(a))
            y2 = int(y1 + L * np.sin(a))
            cv2.line(img, (x1, y1), (x2, y2),
                     tuple(np.random.randint(0, 40, 3).tolist()), 1)

        # 椭圆形病灶 mask
        mask = np.zeros((560, 768), dtype=np.uint8)
        cx, cy = np.random.randint(200, 568), np.random.randint(150, 410)
        rx, ry = np.random.randint(40, 180), np.random.randint(35, 160)
        cv2.ellipse(mask, (cx, cy), (rx, ry),
                     np.random.uniform(0, 180), 0, 360, 255, -1)

        # 病灶区域纹理
        m = mask > 0
        img[m] = np.clip(img[m].astype(np.int16) +
                         np.random.randint(50, 130) +
                         np.random.randint(-25, 25, img[m].shape), 0, 255).astype(np.uint8)

        cv2.imwrite(str(img_dir / f"{i:04d}.png"), img)
        cv2.imwrite(str(msk_dir / f"{i:04d}.png"), mask)

    print(f"[生成] {img_dir}/ 和 {msk_dir}/ 各 200 张")
    return True


def mask_to_yolo_polygon(mask: np.ndarray, eps: float = 0.005) -> list:
    """二值 mask → YOLO polygon 归一化坐标列表"""
    contours, _ = cv2.findContours(mask.astype(np.uint8),
                                    cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)
    h, w = mask.shape[:2]
    polygons = []
    for cnt in contours:
        if len(cnt) < 3:
            continue
        approx = cv2.approxPolyDP(cnt, eps * cv2.arcLength(cnt, True), True)
        pts = []
        for pt in approx:
            pts.extend([pt[0][0] / w, pt[0][1] / h])
        if len(pts) >= 6:
            polygons.append(pts)
    return polygons


def convert_to_yolo(input_dir: Path, output_dir: Path) -> int:
    """将 PH2 数据转换为 YOLO 格式并划分 train/val/test"""
    img_dir = input_dir / "images"
    msk_dir = input_dir / "masks"

    if not img_dir.exists() or not msk_dir.exists():
        print(f"[错误] 缺少 images/ 或 masks/ 目录")
        return 0

    # 配对
    pairs = []
    for ip in sorted(img_dir.glob("*")):
        if ip.suffix.lower() not in (".png", ".jpg", ".bmp", ".jpeg"):
            continue
        mp = msk_dir / f"{ip.stem}.png"
        if mp.exists():
            pairs.append((ip, mp))

    if not pairs:
        print("[错误] 没有找到 image-mask 对")
        return 0

    print(f"[转换] {len(pairs)} 对 image-mask")

    # 确定性划分
    pairs.sort(key=lambda x: hashlib.md5(x[0].name.encode()).hexdigest())
    n = len(pairs)
    splits = {"train": pairs[:int(n*0.7)], "val": pairs[int(n*0.7):int(n*0.85)],
              "test": pairs[int(n*0.85):]}

    total = 0
    for split, items in splits.items():
        oi = output_dir / "images" / split
        ol = output_dir / "labels" / split
        oi.mkdir(parents=True, exist_ok=True)
        ol.mkdir(parents=True, exist_ok=True)

        for ip, mp in items:
            img = cv2.imread(str(ip))
            mask = cv2.imread(str(mp), cv2.IMREAD_GRAYSCALE)
            if img is None or mask is None:
                continue
            polys = mask_to_yolo_polygon(mask)
            if not polys:
                continue
            cv2.imwrite(str(oi / ip.name), img)
            with open(ol / f"{ip.stem}.txt", "w") as f:
                for p in polys:
                    f.write(f"0 {' '.join(f'{c:.6f}' for c in p)}\n")
            total += 1

    for s, items in splits.items():
        print(f"  {s}: {len(items)} 张")
    print(f"[转换] 成功转换 {total} 对")
    return total


def main():
    parser = argparse.ArgumentParser(description="PH2 数据集准备")
    parser.add_argument("--output-dir", default=str(PROJECT_ROOT / "ph2_data"))
    parser.add_argument("--input-dir", default=None, help="已有 PH2 数据目录")
    parser.add_argument("--synthetic", action="store_true", help="生成合成测试数据")
    parser.add_argument("--epsilon", type=float, default=0.005)
    args = parser.parse_args()

    out = Path(args.output_dir)
    out.mkdir(parents=True, exist_ok=True)

    print("=" * 55)
    print("  PH2 数据集准备")
    print("=" * 55)

    # Step 1: 获取数据
    if args.synthetic:
        print("\n[Step 1] 生成合成数据...")
        inp = out
        create_synthetic_ph2(out)
    elif args.input_dir:
        inp = Path(args.input_dir)
        print(f"\n[Step 1] 使用已有数据: {inp}")
    else:
        print("\n[Step 1] 尝试下载...")
        if not download_ph2_direct(out):
            print("[提示] 自动下载失败。请使用以下方式之一：")
            print(f"  1. 合成数据: python3 {__file__} --synthetic")
            print(f"  2. 已有数据: python3 {__file__} --input-dir /path/to/PH2")
            print(f"\n  PH2 官方下载: https://www.fc.up.pt/addi/ph2%20database.html")
            print(f"  解压后结构: images/ (200 PNG) + masks/ (200 PNG)")
            sys.exit(0)

    # Step 2: 转换格式
    print(f"\n[Step 2] mask → YOLO polygon...")
    n = convert_to_yolo(inp, out)
    if n == 0:
        print("[错误] 转换失败")
        sys.exit(1)

    # Step 3: 生成配置
    yaml_path = out / "ph2.yaml"
    yaml_path.write_text(f"""# PH2 Dataset Config
path: {out.resolve()}
train: images/train
val: images/val
test: images/test
nc: 1
names: ['lesion']
""")
    print(f"\n[Step 3] 配置已生成: {yaml_path}")
    print(f"\n{'=' * 55}")
    print(f"  完成！{n} 张图像 → {out.resolve()}")
    print(f"{'=' * 55}")


if __name__ == "__main__":
    main()
