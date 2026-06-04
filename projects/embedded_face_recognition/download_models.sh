#!/bin/bash
# ============================================================
# 下载 dlib 人脸识别预训练模型
# ============================================================

set -e

MODEL_DIR="$(dirname "$0")/models"
mkdir -p "$MODEL_DIR"

echo "=========================================="
echo "  下载 dlib 人脸识别模型文件"
echo "=========================================="
echo ""

# 1. shape_predictor_68_face_landmarks.dat (已通过系统包安装，可选)
SP_PATH="$MODEL_DIR/shape_predictor_68_face_landmarks.dat"
if [ -f "$SP_PATH" ]; then
    echo "[✓] shape_predictor_68_face_landmarks.dat 已存在"
elif [ -f "/usr/share/dlib/shape_predictor_68_face_landmarks.dat" ]; then
    echo "[i] 系统已安装 shape predictor, 位于 /usr/share/dlib/"
else
    echo "[→] 下载 shape_predictor_68_face_landmarks.dat (~100MB)..."
    wget -q --show-progress -O "$SP_PATH" \
        "https://github.com/davisking/dlib-models/raw/master/shape_predictor_68_face_landmarks.dat.bz2" 2>/dev/null || \
    wget -q --show-progress -O "${SP_PATH}.bz2" \
        "http://dlib.net/files/shape_predictor_68_face_landmarks.dat.bz2"
    if [ -f "${SP_PATH}.bz2" ]; then
        bunzip2 "${SP_PATH}.bz2"
    fi
    echo "[✓] shape_predictor_68_face_landmarks.dat 下载完成"
fi

# 2. dlib_face_recognition_resnet_model_v1.dat
FR_PATH="$MODEL_DIR/dlib_face_recognition_resnet_model_v1.dat"
if [ -f "$FR_PATH" ]; then
    echo "[✓] dlib_face_recognition_resnet_model_v1.dat 已存在"
else
    echo "[→] 下载 dlib_face_recognition_resnet_model_v1.dat (~21MB)..."
    wget -q --show-progress -O "${FR_PATH}.bz2" \
        "http://dlib.net/files/dlib_face_recognition_resnet_model_v1.dat.bz2"
    if [ -f "${FR_PATH}.bz2" ]; then
        bunzip2 "${FR_PATH}.bz2"
        echo "[✓] dlib_face_recognition_resnet_model_v1.dat 下载完成"
    else
        echo "[✗] 下载失败，请手动下载:"
        echo "    http://dlib.net/files/dlib_face_recognition_resnet_model_v1.dat.bz2"
        echo "    解压后放置到: $MODEL_DIR/"
    fi
fi

echo ""
echo "=========================================="
echo "  模型文件检查完成"
echo "=========================================="
ls -lh "$MODEL_DIR/" 2>/dev/null || echo "  (无模型文件)"
echo ""
echo "运行程序时指定模型路径："
echo "  ./build/embedded_face_recognition \\"
echo "    --shape-predictor models/shape_predictor_68_face_landmarks.dat \\"
echo "    --face-model models/dlib_face_recognition_resnet_model_v1.dat \\"
echo "    --recognize --known-faces <人脸目录>"
