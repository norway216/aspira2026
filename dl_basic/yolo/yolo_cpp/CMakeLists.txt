# =============================================================================
#  YOLOv8 分割模型 ONNX Runtime C++ 推理项目 — CMake 构建配置文件
# =============================================================================
#
#  构建要求：
#    - CMake >= 3.18
#    - C++20 编译器 (GCC 11+ / Clang 14+)
#    - OpenCV 4.x (core, imgproc, imgcodecs)
#    - ONNX Runtime 1.18.0+ (预编译包放在 third_party/onnxruntime/)
#
#  构建命令：
#    mkdir build && cd build
#    cmake .. -DCMAKE_BUILD_TYPE=Release
#    make -j$(nproc)
#
#  运行：
#    ./yolo_seg_inference [图片路径] [模型路径]
# =============================================================================

cmake_minimum_required(VERSION 3.18)
project(YoloCppSegInference LANGUAGES CXX
    DESCRIPTION "YOLOv8 Segmentation ONNX Runtime C++ Inference"
)

# ── C++ 标准 ────────────────────────────────────────────────────────────
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# 输出编译命令行（便于调试）
# set(CMAKE_VERBOSE_MAKEFILE ON)

# ── 查找 OpenCV ────────────────────────────────────────────────────────
# 使用 CONFIG 模式查找 OpenCV 的 CMake 配置文件
find_package(OpenCV REQUIRED COMPONENTS core imgproc imgcodecs dnn)

message(STATUS "OpenCV 版本: ${OpenCV_VERSION}")
message(STATUS "OpenCV 头文件: ${OpenCV_INCLUDE_DIRS}")

# ── 配置 ONNX Runtime ──────────────────────────────────────────────────
# ONNX Runtime 以预编译包形式放在 third_party/onnxruntime/ 目录中
# 目录结构：
#   third_party/onnxruntime/
#   ├── include/  (onnxruntime_cxx_api.h 等头文件)
#   └── lib/      (libonnxruntime.so)

set(ONNXRUNTIME_ROOT "${CMAKE_SOURCE_DIR}/third_party/onnxruntime")

if(NOT EXISTS "${ONNXRUNTIME_ROOT}/include/onnxruntime_cxx_api.h")
    message(FATAL_ERROR
        "ONNX Runtime 头文件未找到！\n"
        "请将 ONNX Runtime 预编译包解压到: ${ONNXRUNTIME_ROOT}\n"
        "下载地址: https://github.com/microsoft/onnxruntime/releases"
    )
endif()

if(NOT EXISTS "${ONNXRUNTIME_ROOT}/lib/libonnxruntime.so")
    message(FATAL_ERROR
        "ONNX Runtime 库文件未找到！\n"
        "预期位置: ${ONNXRUNTIME_ROOT}/lib/libonnxruntime.so"
    )
endif()

# 将 ONNX Runtime 作为 IMPORTED 库引入
add_library(onnxruntime SHARED IMPORTED)
set_target_properties(onnxruntime PROPERTIES
    IMPORTED_LOCATION "${ONNXRUNTIME_ROOT}/lib/libonnxruntime.so"
    INTERFACE_INCLUDE_DIRECTORIES "${ONNXRUNTIME_ROOT}/include"
)

message(STATUS "ONNX Runtime 位置: ${ONNXRUNTIME_ROOT}")

# ── 源文件 ─────────────────────────────────────────────────────────────
set(SOURCES
    src/main.cpp
    src/yolo_seg_inference.cpp
)

set(HEADERS
    src/yolo_seg_inference.h
)

# ── 可执行文件 ─────────────────────────────────────────────────────────
add_executable(yolo_seg_inference ${SOURCES} ${HEADERS})

# ── 头文件搜索路径 ────────────────────────────────────────────────────
target_include_directories(yolo_seg_inference PRIVATE
    ${CMAKE_SOURCE_DIR}/src          # 项目内部头文件（相对路径根目录）
    ${OpenCV_INCLUDE_DIRS}           # OpenCV 头文件
)

# ── 链接库 ─────────────────────────────────────────────────────────────
target_link_libraries(yolo_seg_inference PRIVATE
    ${OpenCV_LIBS}                   # OpenCV 核心库
    onnxruntime                      # ONNX Runtime 推理引擎
)

# ── RPATH 设置 ─────────────────────────────────────────────────────────
# 确保运行时能在构建目录中找到 libonnxruntime.so
# 不设置 RPATH 的话，需要将 libonnxruntime.so 复制到 /usr/lib 或使用 LD_LIBRARY_PATH
set_target_properties(yolo_seg_inference PROPERTIES
    BUILD_RPATH "${ONNXRUNTIME_ROOT}/lib"
    INSTALL_RPATH "${ONNXRUNTIME_ROOT}/lib"
)

# ── 编译选项 ───────────────────────────────────────────────────────────
target_compile_options(yolo_seg_inference PRIVATE
    -Wall -Wextra -Wpedantic           # 严格的编译警告（帮助发现潜在问题）

    # Release 构建：优化性能
    $<$<CONFIG:Release>:-O2 -DNDEBUG>

    # Debug 构建：完整调试信息，便于 GDB 调试
    $<$<CONFIG:Debug>:-g3 -ggdb -O0 -fno-omit-frame-pointer -DDEBUG>

    # RelWithDebInfo：优化 + 调试信息（适合性能分析）
    $<$<CONFIG:RelWithDebInfo>:-g3 -ggdb -O2 -fno-omit-frame-pointer>
)

# ── 编译宏定义 ─────────────────────────────────────────────────────────
# 通过 CMake 变量传递版本信息到 C++ 代码
target_compile_definitions(yolo_seg_inference PRIVATE
    YOLO_CPP_VERSION="${PROJECT_VERSION}"
)

# ── 构建信息汇总 ───────────────────────────────────────────────────────
message(STATUS "")
message(STATUS "═══════════════════════════════════════════════════")
message(STATUS "  YoloCppSegInference 构建配置完成")
message(STATUS "  C++ 标准: C++${CMAKE_CXX_STANDARD}")
message(STATUS "  编译器: ${CMAKE_CXX_COMPILER_ID} ${CMAKE_CXX_COMPILER_VERSION}")
message(STATUS "  构建类型: ${CMAKE_BUILD_TYPE}")
message(STATUS "  OpenCV: ${OpenCV_VERSION}")
message(STATUS "═══════════════════════════════════════════════════")
