#pragma once

#include <cstdint>
#include <string>
#include <vector>

namespace iris {

/// Abstract interface for neural network inference backends.
/// Implementations: ONNX Runtime, RKNN, TensorRT, OpenVINO.
/// For MVP without actual model files, a classical CV fallback is used.
class InferenceEngine {
public:
    virtual ~InferenceEngine() = default;

    /// Load a model from file
    virtual bool loadModel(const std::string& modelPath) = 0;

    /// Run inference with float input tensor
    /// @param input  flattened input data
    /// @param inputShape  shape [batch, channels, height, width]
    /// @return flattened output data
    virtual std::vector<float> infer(
        const std::vector<float>& input,
        const std::vector<int64_t>& inputShape) = 0;

    /// Check if model is loaded
    virtual bool isLoaded() const = 0;

    /// Backend name for logging
    virtual std::string backendName() const = 0;
};

/// A no-op engine for when no real NN backend is available
class DummyInferenceEngine : public InferenceEngine {
public:
    bool loadModel(const std::string& /*modelPath*/) override { return false; }
    std::vector<float> infer(const std::vector<float>& /*input*/,
                              const std::vector<int64_t>& /*inputShape*/) override {
        return {};
    }
    bool isLoaded() const override { return false; }
    std::string backendName() const override { return "dummy"; }
};

} // namespace iris
