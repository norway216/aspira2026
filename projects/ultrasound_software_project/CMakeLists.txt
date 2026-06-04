cmake_minimum_required(VERSION 3.16)
project(UltrasoundSystem VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Qt integration
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)

# ---- Find Qt5 (from conda) ----
if(DEFINED ENV{CONDA_PREFIX})
    list(APPEND CMAKE_PREFIX_PATH "$ENV{CONDA_PREFIX}")
endif()

find_package(Qt5 5.15 REQUIRED COMPONENTS
    Core
    Gui
    Quick
    Qml
    QuickControls2
)

# ---- Source files ----
set(CXX_SOURCES
    src/main.cpp
    src/AppCore.cpp
    src/FrameGenerator.cpp
    src/ParamManager.cpp
    src/ThreadPool.cpp
    src/ConfigManager.cpp
    src/UltrasoundImageProvider.cpp
)

set(CXX_HEADERS
    src/AppCore.h
    src/FrameGenerator.h
    src/ParamManager.h
    src/ThreadPool.h
    src/ConfigManager.h
    src/UltrasoundImageProvider.h
)

# ---- Executable ----
add_executable(${PROJECT_NAME}
    ${CXX_SOURCES}
    ${CXX_HEADERS}
    resources/qml.qrc
)

# ---- Link libraries ----
target_link_libraries(${PROJECT_NAME} PRIVATE
    Qt5::Core
    Qt5::Gui
    Qt5::Quick
    Qt5::Qml
    Qt5::QuickControls2
)

# ---- Include paths ----
target_include_directories(${PROJECT_NAME} PRIVATE ${CMAKE_CURRENT_SOURCE_DIR}/src)

# ---- Compiler flags ----
if(MSVC)
    target_compile_options(${PROJECT_NAME} PRIVATE /W4 /utf-8)
else()
    target_compile_options(${PROJECT_NAME} PRIVATE -Wall -Wextra -Wpedantic)
endif()

# ---- Defines ----
target_compile_definitions(${PROJECT_NAME} PRIVATE
    QT_DISABLE_DEPRECATED_BEFORE=0x050F00
)
