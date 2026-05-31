#pragma once

#include "nn/InferenceEngine.h"
#include <memory>
#include <vector>
#include <string>

// Forward-declare ONNX Runtime types (avoids leaking ONNX headers)
namespace Ort {
class Env;
class Session;
class MemoryInfo;
class AllocatorWithDefaultOptions;
struct Value;
} // namespace Ort

namespace iris {

/// Real ONNX Runtime inference engine.
/// Loads .onnx models and runs inference using CPU (or CUDA/ROCM if available).
class OnnxRuntimeEngine : public InferenceEngine {
public:
    OnnxRuntimeEngine();
    ~OnnxRuntimeEngine() override;

    OnnxRuntimeEngine(const OnnxRuntimeEngine&)            = delete;
    OnnxRuntimeEngine& operator=(const OnnxRuntimeEngine&) = delete;

    /// Load an ONNX model from file
    bool loadModel(const std::string& modelPath) override;

    /// Run inference
    /// @param input  flattened float input data (NCHW order)
    /// @param inputShape  shape [batch, channels, height, width]
    /// @return flattened float output data
    std::vector<float> infer(const std::vector<float>& input,
                              const std::vector<int64_t>& inputShape) override;

    bool isLoaded() const override;
    std::string backendName() const override { return "onnxruntime"; }

    /// Get number of input/output nodes
    size_t numInputs() const;
    size_t numOutputs() const;

    /// Get input/output names
    std::vector<std::string> inputNames() const;
    std::vector<std::string> outputNames() const;

    /// Get input shape from loaded model
    std::vector<int64_t> getInputShape(size_t index = 0) const;

    /// Get output shape from loaded model (for multi-output models like YOLOv5)
    std::vector<int64_t> getOutputShape(size_t index = 0) const;

private:
    struct Impl;
    std::unique_ptr<Impl> m_impl;
};

} // namespace iris
