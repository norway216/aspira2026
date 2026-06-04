cmake_minimum_required(VERSION 3.18)
project(IrisRecognitionSystem LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# ── OpenCV (required) ──────────────────────────────────────────
find_package(OpenCV REQUIRED COMPONENTS core imgproc highgui videoio objdetect)

# ── Optional dependencies ──────────────────────────────────────
find_package(OpenSSL QUIET)
if(OpenSSL_FOUND)
    add_definitions(-DHAS_OPENSSL)
    message(STATUS "OpenSSL found: template encryption enabled")
else()
    message(STATUS "OpenSSL not found: using simplified encryption")
endif()

find_package(SQLite3 QUIET)
if(SQLite3_FOUND)
    add_definitions(-DHAS_SQLITE3)
    message(STATUS "SQLite3 found: database persistence enabled")
else()
    message(STATUS "SQLite3 not found: using file-based storage")
endif()

# ── ONNX Runtime (optional, local bundle) ──────────────────────
set(ONNXRUNTIME_ROOT "${CMAKE_SOURCE_DIR}/third_party/onnxruntime")
if(EXISTS "${ONNXRUNTIME_ROOT}/include/onnxruntime_cxx_api.h")
    add_definitions(-DHAS_ONNXRUNTIME)
    set(ONNXRUNTIME_INCLUDE_DIR "${ONNXRUNTIME_ROOT}/include")
    set(ONNXRUNTIME_LIB_DIR "${ONNXRUNTIME_ROOT}/lib")
    message(STATUS "ONNX Runtime found: NN inference enabled")
else()
    message(STATUS "ONNX Runtime not found: classical CV only")
endif()

# ── Source files ───────────────────────────────────────────────
set(SOURCES
    src/main.cpp
    src/capture/CameraDevice.cpp
    src/image/ImagePreprocessor.cpp
    src/image/QualityChecker.cpp
    src/image/IrisNormalizer.cpp
    src/image/GeometryUtils.cpp
    src/model/EyeDetector.cpp
    src/model/IrisSegmenter.cpp
    src/model/IrisFeatureExtractor.cpp
    src/model/IrisLivenessDetector.cpp
    src/matching/IrisMatcher.cpp
    src/matching/DecisionEngine.cpp
    src/persistence/Database.cpp
    src/security/CryptoProvider.cpp
    src/security/TemplateEncryptor.cpp
    src/service/EnrollmentService.cpp
    src/service/RecognitionService.cpp
    src/service/LivenessService.cpp
    src/service/AuditService.cpp
    src/evaluation/Benchmark.cpp
    src/evaluation/FpsBenchmark.cpp
)

if(EXISTS "${ONNXRUNTIME_ROOT}/include/onnxruntime_cxx_api.h")
    list(APPEND SOURCES src/nn/OnnxRuntimeEngine.cpp)
    list(APPEND SOURCES src/model/YoloDetector.cpp)
endif()

# ── Executable ─────────────────────────────────────────────────
add_executable(iris_app ${SOURCES})

target_include_directories(iris_app PRIVATE
    ${CMAKE_SOURCE_DIR}/src
    ${OpenCV_INCLUDE_DIRS}
)

target_link_libraries(iris_app PRIVATE
    ${OpenCV_LIBS}
)

if(SQLite3_FOUND)
    target_link_libraries(iris_app PRIVATE SQLite::SQLite3)
endif()

if(OpenSSL_FOUND)
    target_link_libraries(iris_app PRIVATE OpenSSL::SSL OpenSSL::Crypto)
endif()

if(EXISTS "${ONNXRUNTIME_ROOT}/include/onnxruntime_cxx_api.h")
    target_include_directories(iris_app PRIVATE ${ONNXRUNTIME_INCLUDE_DIR})
    target_link_directories(iris_app PRIVATE ${ONNXRUNTIME_LIB_DIR})
    target_link_libraries(iris_app PRIVATE onnxruntime)

    # Set rpath so the binary finds libonnxruntime.so at runtime
    set_target_properties(iris_app PROPERTIES
        BUILD_RPATH "${ONNXRUNTIME_LIB_DIR}"
        INSTALL_RPATH "${ONNXRUNTIME_LIB_DIR}"
    )
endif()

# ── Compiler flags ─────────────────────────────────────────────
target_compile_options(iris_app PRIVATE
    -Wall -Wextra -Wpedantic
    $<$<CONFIG:Release>:-O2 -DNDEBUG>
    $<$<CONFIG:Debug>:-g3 -ggdb -O0 -fno-omit-frame-pointer -DDEBUG>
    $<$<CONFIG:RelWithDebInfo>:-g3 -ggdb -O2 -fno-omit-frame-pointer>
)

# Debug build: add address sanitizer if available (GCC/Clang)
option(ENABLE_ASAN "Enable AddressSanitizer in Debug builds" OFF)
if(ENABLE_ASAN AND CMAKE_BUILD_TYPE MATCHES "Debug")
    target_compile_options(iris_app PRIVATE -fsanitize=address -fno-omit-frame-pointer)
    target_link_options(iris_app PRIVATE -fsanitize=address)
    message(STATUS "AddressSanitizer enabled for Debug build")
endif()

message(STATUS "Iris Recognition System configured successfully")
