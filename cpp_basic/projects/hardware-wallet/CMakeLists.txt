cmake_minimum_required(VERSION 3.18)
project(HardwareWallet VERSION 1.0.0 LANGUAGES CXX)

# Force GCC 14 for full C++20 support
if(NOT CMAKE_CXX_COMPILER)
    set(CMAKE_CXX_COMPILER /usr/bin/g++-14 CACHE STRING "C++ compiler" FORCE)
endif()

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

# Qt auto-processing
set(CMAKE_AUTOMOC ON)
set(CMAKE_AUTORCC ON)

# --- Dependencies ---
find_package(Qt5 5.15 REQUIRED COMPONENTS Core Qml Quick QuickControls2 Sql Concurrent)
find_package(OpenSSL REQUIRED)
find_package(spdlog REQUIRED)
find_package(nlohmann_json REQUIRED)

# --- Compiler flags ---
add_compile_options(-Wall -Wextra -Wpedantic -O2)
add_compile_options(-Wno-unknown-pragmas)

# --- Sources ---
file(GLOB_RECURSE DOMAIN_SRC    src/domain/*.cpp)
file(GLOB_RECURSE CRYPTO_SRC    src/crypto/*.cpp)
file(GLOB_RECURSE PERSIST_SRC   src/persistence/*.cpp)
file(GLOB_RECURSE SERVICE_SRC   src/service/*.cpp)
file(GLOB_RECURSE VM_SRC        src/viewmodel/*.cpp)
file(GLOB_RECURSE WORKER_SRC    src/worker/*.cpp)
file(GLOB_RECURSE UTIL_SRC      src/util/*.cpp)
file(GLOB_RECURSE APP_SRC       src/app/*.cpp)

# Qt resources
set(QML_RESOURCES src/resources/resources.qrc)

add_executable(hardware-wallet
    src/main.cpp
    ${DOMAIN_SRC}
    ${CRYPTO_SRC}
    ${PERSIST_SRC}
    ${SERVICE_SRC}
    ${VM_SRC}
    ${WORKER_SRC}
    ${UTIL_SRC}
    ${APP_SRC}
    ${QML_RESOURCES}
)

target_include_directories(hardware-wallet PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/src
)

target_link_libraries(hardware-wallet PRIVATE
    Qt5::Core
    Qt5::Qml
    Qt5::Quick
    Qt5::QuickControls2
    Qt5::Sql
    Qt5::Concurrent
    OpenSSL::Crypto
    spdlog::spdlog
    nlohmann_json::nlohmann_json
)

# --- Install ---
install(TARGETS hardware-wallet RUNTIME DESTINATION bin)

# --- Tests ---
option(BUILD_TESTS "Build unit tests" OFF)
if(BUILD_TESTS)
    enable_testing()
    find_package(GTest REQUIRED)
    add_subdirectory(tests)
endif()
