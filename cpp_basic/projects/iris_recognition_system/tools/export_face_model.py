#!/usr/bin/env python3
"""
Export a YOLO face detection model to ONNX format for the Iris Recognition System.

Supports YOLOv8n-face, YOLOv8s-face, YOLO11n-face, and custom models.
The exported ONNX model is optimized for ONNX Runtime inference.

Usage:
    # Export YOLOv8n-face (recommended, ~6MB ONNX)
    python export_face_model.py --model yolov8n-face --output models/face_detector.onnx

    # Export YOLO11n-face (newer, better accuracy)
    python export_face_model.py --model yolo11n-face --output models/face_detector.onnx

    # Export custom trained model
    python export_face_model.py --model path/to/best.pt --output models/face_detector.onnx

    # Test the exported model
    python export_face_model.py --model yolov8n-face --output models/face_detector.onnx --test

Requirements:
    pip install ultralytics onnx onnxruntime
"""

import argparse
import os
import sys
import numpy as np


def export_model(model_path: str, output_path: str, opset: int = 11, simplify: bool = True):
    """Export a YOLO model to ONNX format."""
    try:
        from ultralytics import YOLO
    except ImportError:
        print("ERROR: ultralytics not installed. Run: pip install ultralytics")
        return False

    print(f"Loading model: {model_path}")

    # Check if it's a named model (yolov8n-face) or a file path
    model_name = os.path.basename(model_path)
    if not os.path.exists(model_path):
        # Named model - will be downloaded from ultralytics
        print(f"  Downloading {model_path} from Ultralytics...")

    model = YOLO(model_path)

    print(f"Exporting to ONNX...")
    print(f"  Output: {output_path}")
    print(f"  Opset:  {opset}")
    print(f"  Simplify: {simplify}")

    # Export to ONNX
    success = model.export(
        format="onnx",
        imgsz=640,
        opset=opset,
        simplify=simplify,
        dynamic=False,  # Fixed batch size for faster inference
        half=False,     # FP32 for CPU compatibility
    )

    if success:
        # The export creates a file next to the source
        # If output_path differs, move it
        src = model_path.replace('.pt', '.onnx')
        if os.path.exists(src) and os.path.abspath(src) != os.path.abspath(output_path):
            os.makedirs(os.path.dirname(output_path) or '.', exist_ok=True)
            os.rename(src, output_path)
            print(f"  Moved to: {output_path}")

        file_size_mb = os.path.getsize(output_path) / (1024 * 1024)
        print(f"✅ Export successful! ({file_size_mb:.1f} MB)")
        return True
    else:
        print("❌ Export failed")
        return False


def test_model(onnx_path: str):
    """Test the exported ONNX model with random input."""
    try:
        import onnxruntime as ort
    except ImportError:
        print("ERROR: onnxruntime not installed. Run: pip install onnxruntime")
        return

    print(f"\nTesting ONNX model: {onnx_path}")

    # Load session
    session = ort.InferenceSession(onnx_path)

    # Print model info
    print(f"  Inputs:")
    for inp in session.get_inputs():
        print(f"    {inp.name}: shape={inp.shape}, type={inp.type}")

    print(f"  Outputs:")
    for out in session.get_outputs():
        print(f"    {out.name}: shape={out.shape}, type={out.type}")

    # Create random input
    input_shape = session.get_inputs()[0].shape
    # Replace dynamic dims with concrete values
    concrete_shape = []
    for d in input_shape:
        if isinstance(d, str) or d is None or d <= 0:
            concrete_shape.append(1)
        else:
            concrete_shape.append(d)

    print(f"  Running inference with shape: {concrete_shape}")

    input_data = np.random.randn(*concrete_shape).astype(np.float32)
    input_name = session.get_inputs()[0].name

    outputs = session.run(None, {input_name: input_data})

    for i, out in enumerate(outputs):
        print(f"  Output[{i}]: shape={out.shape}, dtype={out.dtype}, "
              f"range=[{out.min():.3f}, {out.max():.3f}]")

    # Detect YOLO version
    num_outputs = len(session.get_outputs())
    if num_outputs == 1:
        print(f"\n  ✅ YOLOv8/11 format detected (single output, anchor-free)")
    elif num_outputs == 3:
        print(f"\n  ✅ YOLOv5 format detected (3 outputs, anchor-based)")
    else:
        print(f"\n  ⚠️  Unknown format ({num_outputs} outputs)")

    print(f"  ✅ Model ready for Iris Recognition System!")
    print(f"\n  To use with iris_app:")
    print(f"    ./iris_app --detector-mode yolo --yolo-model {onnx_path}")


def main():
    parser = argparse.ArgumentParser(
        description="Export YOLO face detection model to ONNX")
    parser.add_argument("--model", required=True,
                        help="Model name (yolov8n-face) or path to .pt file")
    parser.add_argument("--output", default="models/face_detector.onnx",
                        help="Output ONNX file path")
    parser.add_argument("--opset", type=int, default=11,
                        help="ONNX opset version (default: 11 for OpenCV 4.6 compat)")
    parser.add_argument("--no-simplify", action="store_true",
                        help="Skip ONNX simplification")
    parser.add_argument("--test", action="store_true",
                        help="Test the exported model with random input")

    args = parser.parse_args()

    # Export
    if not export_model(args.model, args.output, args.opset, not args.no_simplify):
        sys.exit(1)

    # Test
    if args.test:
        test_model(args.output)

    print(f"\nNext steps:")
    print(f"  1. Move the model to the iris_app models directory if needed")
    print(f"  2. Run: ./iris_app --detector-mode yolo --yolo-model {args.output}")


if __name__ == "__main__":
    main()
