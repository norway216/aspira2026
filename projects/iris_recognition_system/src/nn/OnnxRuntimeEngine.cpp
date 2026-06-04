#include "nn/OnnxRuntimeEngine.h"

// ONNX Runtime C++ API
#include <onnxruntime_cxx_api.h>

#include <iostream>
#include <cstring>
#include <algorithm>
#include <numeric>

namespace iris {

// ── PIMPL to avoid leaking ONNX Runtime headers ───────────────

struct OnnxRuntimeEngine::Impl {
    std::unique_ptr<Ort::Env> env;
    std::unique_ptr<Ort::Session> session;
    std::unique_ptr<Ort::MemoryInfo> memoryInfo;
    std::vector<std::string> inputNames;
    std::vector<std::string> outputNames;
    std::vector<std::vector<int64_t>> inputShapes;
    std::vector<std::vector<int64_t>> outputShapes;
    bool loaded = false;

    Impl() {
        // Create ONNX Runtime environment
        env = std::make_unique<Ort::Env>(ORT_LOGGING_LEVEL_WARNING, "IrisRecognition");
        memoryInfo = std::make_unique<Ort::MemoryInfo>(
            Ort::MemoryInfo::CreateCpu(OrtArenaAllocator, OrtMemTypeDefault));
    }
};

// ── Constructor / Destructor ──────────────────────────────────

OnnxRuntimeEngine::OnnxRuntimeEngine()
    : m_impl(std::make_unique<Impl>()) {}

OnnxRuntimeEngine::~OnnxRuntimeEngine() = default;

// ── Load model ────────────────────────────────────────────────

bool OnnxRuntimeEngine::loadModel(const std::string& modelPath) {
    try {
        Ort::SessionOptions sessionOptions;
        sessionOptions.SetIntraOpNumThreads(4);
        sessionOptions.SetGraphOptimizationLevel(
            GraphOptimizationLevel::ORT_ENABLE_ALL);

        // Enable CPU providers
        // CUDA/ROCM would be auto-detected if available

        m_impl->session = std::make_unique<Ort::Session>(
            *m_impl->env, modelPath.c_str(), sessionOptions);

        // Extract input/output metadata
        Ort::AllocatorWithDefaultOptions allocator;

        size_t numInputs = m_impl->session->GetInputCount();
        for (size_t i = 0; i < numInputs; ++i) {
            auto name = m_impl->session->GetInputNameAllocated(i, allocator);
            m_impl->inputNames.push_back(name.get());

            auto typeInfo = m_impl->session->GetInputTypeInfo(i);
            auto tensorInfo = typeInfo.GetTensorTypeAndShapeInfo();
            m_impl->inputShapes.push_back(tensorInfo.GetShape());
        }

        size_t numOutputs = m_impl->session->GetOutputCount();
        for (size_t i = 0; i < numOutputs; ++i) {
            auto name = m_impl->session->GetOutputNameAllocated(i, allocator);
            m_impl->outputNames.push_back(name.get());

            auto typeInfo = m_impl->session->GetOutputTypeInfo(i);
            auto tensorInfo = typeInfo.GetTensorTypeAndShapeInfo();
            m_impl->outputShapes.push_back(tensorInfo.GetShape());
        }

        m_impl->loaded = true;

        std::cout << "[OnnxRuntime] Loaded model: " << modelPath << "\n"
                  << "  Inputs:  " << numInputs << "\n"
                  << "  Outputs: " << numOutputs << "\n";

        for (size_t i = 0; i < numInputs; ++i) {
            std::cout << "  Input[" << i << "]: " << m_impl->inputNames[i]
                      << " shape=[";
            for (size_t j = 0; j < m_impl->inputShapes[i].size(); ++j) {
                if (j > 0) std::cout << ",";
                std::cout << m_impl->inputShapes[i][j];
            }
            std::cout << "]\n";
        }

        return true;

    } catch (const Ort::Exception& e) {
        std::cerr << "[OnnxRuntime] Failed to load model: " << modelPath
                  << "\n  Error: " << e.what() << "\n";
        m_impl->loaded = false;
        return false;
    }
}

// ── Inference ─────────────────────────────────────────────────

std::vector<float> OnnxRuntimeEngine::infer(
    const std::vector<float>& input,
    const std::vector<int64_t>& inputShape) {

    if (!m_impl->loaded || !m_impl->session) {
        std::cerr << "[OnnxRuntime] Model not loaded\n";
        return {};
    }

    try {
        // Determine actual shape (use model's dynamic dims if needed)
        std::vector<int64_t> actualShape = inputShape;

        // Replace dynamic dimensions (-1) in model shape with actual values
        if (!m_impl->inputShapes.empty() && !m_impl->inputShapes[0].empty()) {
            const auto& modelShape = m_impl->inputShapes[0];
            if (actualShape.size() == modelShape.size()) {
                for (size_t i = 0; i < modelShape.size(); ++i) {
                    if (modelShape[i] == -1) {
                        // Dynamic dim: use actual value
                        if (i < actualShape.size()) {
                            // Keep actualShape[i] as is
                        }
                    } else if (i < actualShape.size() && actualShape[i] <= 0) {
                        // Use model's static dim
                        actualShape[i] = modelShape[i];
                    }
                }
            }
        }

        // Calculate total elements
        size_t totalElements = 1;
        for (auto dim : actualShape) {
            if (dim > 0) totalElements *= static_cast<size_t>(dim);
        }

        if (input.size() != totalElements) {
            std::cerr << "[OnnxRuntime] Input size mismatch: got "
                      << input.size() << ", expected " << totalElements << "\n";
            return {};
        }

        // Create input tensor
        Ort::Value inputTensor = Ort::Value::CreateTensor<float>(
            *m_impl->memoryInfo,
            const_cast<float*>(input.data()),
            input.size(),
            actualShape.data(),
            actualShape.size());

        // Run inference
        std::vector<const char*> inNames;
        std::vector<const char*> outNames;
        for (const auto& name : m_impl->inputNames)  inNames.push_back(name.c_str());
        for (const auto& name : m_impl->outputNames) outNames.push_back(name.c_str());

        auto outputTensors = m_impl->session->Run(
            Ort::RunOptions{nullptr},
            inNames.data(), &inputTensor, inNames.size(),
            outNames.data(), outNames.size());

        // Extract output data
        std::vector<float> result;
        for (auto& tensor : outputTensors) {
            auto typeInfo = tensor.GetTensorTypeAndShapeInfo();
            size_t numElements = typeInfo.GetElementCount();
            float* data = tensor.GetTensorMutableData<float>();

            result.insert(result.end(), data, data + numElements);
        }

        return result;

    } catch (const Ort::Exception& e) {
        std::cerr << "[OnnxRuntime] Inference error: " << e.what() << "\n";
        return {};
    }
}

// ── Accessors ─────────────────────────────────────────────────

bool OnnxRuntimeEngine::isLoaded() const {
    return m_impl->loaded;
}

size_t OnnxRuntimeEngine::numInputs() const {
    return m_impl->inputNames.size();
}

size_t OnnxRuntimeEngine::numOutputs() const {
    return m_impl->outputNames.size();
}

std::vector<std::string> OnnxRuntimeEngine::inputNames() const {
    return m_impl->inputNames;
}

std::vector<std::string> OnnxRuntimeEngine::outputNames() const {
    return m_impl->outputNames;
}

std::vector<int64_t> OnnxRuntimeEngine::getInputShape(size_t index) const {
    if (index < m_impl->inputShapes.size()) {
        return m_impl->inputShapes[index];
    }
    return {};
}

std::vector<int64_t> OnnxRuntimeEngine::getOutputShape(size_t index) const {
    if (index < m_impl->outputShapes.size()) {
        return m_impl->outputShapes[index];
    }
    return {};
}

} // namespace iris
