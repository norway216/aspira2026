cmake_minimum_required(VERSION 3.16)
project(embedded_face_recognition VERSION 1.0.0 LANGUAGES CXX)

# ---- C++ 标准 ----
set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# ---- 编译选项 ----
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

if(CMAKE_BUILD_TYPE STREQUAL "Release")
    add_compile_options(-O3 -DNDEBUG -march=native)
else()
    add_compile_options(-O2 -g -Wall -Wextra)
endif()

# ---- 查找依赖 ----
find_package(OpenCV REQUIRED COMPONENTS
    core
    imgproc
    videoio
    highgui
    objdetect
    imgcodecs
)

find_package(dlib REQUIRED)

# Boost 不需要（dlib 在独立模式下工作）
# find_package(Boost COMPONENTS system thread QUIET)

# ---- 源文件 ----
set(SOURCES
    main.cpp
    src/camera/CameraManager.cpp
    src/detection/FaceDetector.cpp
    src/recognition/FaceRecognizer.cpp
    src/ui/DisplayManager.cpp
)

# ---- 头文件（用于 IDE 工程） ----
set(HEADERS
    include/Common.h
    src/pipeline/RingBuffer.h
    src/pipeline/ThreadPool.h
    src/camera/CameraManager.h
    src/detection/FaceDetector.h
    src/recognition/FaceRecognizer.h
    src/ui/DisplayManager.h
)

# ---- 可执行文件 ----
add_executable(${PROJECT_NAME} ${SOURCES} ${HEADERS})

# ---- 包含路径 ----
target_include_directories(${PROJECT_NAME} PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/src
    ${OpenCV_INCLUDE_DIRS}
    ${dlib_INCLUDE_DIRS}
)

# ---- 链接库 ----
target_link_libraries(${PROJECT_NAME} PRIVATE
    ${OpenCV_LIBS}
    dlib::dlib
)

# ---- 编译定义 ----
target_compile_definitions(${PROJECT_NAME} PRIVATE
    DLIB_JPEG_SUPPORT
    DLIB_PNG_SUPPORT
)

# ---- 安装规则 ----
install(TARGETS ${PROJECT_NAME} DESTINATION bin)

# ---- 打印配置摘要 ----
message(STATUS "========================================")
message(STATUS "  嵌入式人脸检测与识别系统 - Build Config")
message(STATUS "========================================")
message(STATUS "C++ Standard:     ${CMAKE_CXX_STANDARD}")
message(STATUS "Build Type:       ${CMAKE_BUILD_TYPE}")
message(STATUS "OpenCV version:   ${OpenCV_VERSION}")
message(STATUS "dlib version:     ${dlib_VERSION}")
message(STATUS "dlib include:     ${dlib_INCLUDE_DIRS}")
message(STATUS "========================================")
