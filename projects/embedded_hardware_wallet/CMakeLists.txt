cmake_minimum_required(VERSION 3.16)
project(embedded_hardware_wallet VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)
set(CMAKE_EXPORT_COMPILE_COMMANDS ON)

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
    Network
)

# ---- Find OpenSSL ----
find_package(OpenSSL REQUIRED)

# ---- Source files ----
set(CXX_SOURCES
    src/main.cpp
    # Core
    src/core/SecureMemory.cpp
    src/core/CryptoProvider.cpp
    src/core/WalletCore.cpp
    # Transaction
    src/transaction/TransactionManager.cpp
    # Backup
    src/backup/BackupManager.cpp
    # Network
    src/network/NetworkManager.cpp
    # Security
    src/security/SecurityManager.cpp
    # UI
    src/ui/AppCore.cpp
    src/ui/WalletImageProvider.cpp
)

set(CXX_HEADERS
    include/Common.h
    # Pipeline (header-only)
    src/pipeline/ThreadPool.h
    src/pipeline/RingBuffer.h
    # Core
    src/core/SecureMemory.h
    src/core/CryptoProvider.h
    src/core/WalletCore.h
    # Transaction
    src/transaction/TransactionManager.h
    # Backup
    src/backup/BackupManager.h
    # Network
    src/network/NetworkManager.h
    # Security
    src/security/SecurityManager.h
    # UI
    src/ui/AppCore.h
    src/ui/WalletImageProvider.h
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
    Qt5::Network
    OpenSSL::SSL
    OpenSSL::Crypto
)

# ---- Include paths ----
target_include_directories(${PROJECT_NAME} PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/src
)

# ---- Compiler flags ----
if(MSVC)
    target_compile_options(${PROJECT_NAME} PRIVATE /W4 /utf-8)
else()
    target_compile_options(${PROJECT_NAME} PRIVATE -Wall -Wextra -Wpedantic)
endif()

# Release optimization flags
target_compile_options(${PROJECT_NAME} PRIVATE
    $<$<CONFIG:Release>:-O3 -march=native -DNDEBUG>
    $<$<CONFIG:Debug>:-O2 -g>
)

# ---- Defines ----
target_compile_definitions(${PROJECT_NAME} PRIVATE
    QT_DISABLE_DEPRECATED_BEFORE=0x050F00
)

# ---- Install ----
install(TARGETS ${PROJECT_NAME} DESTINATION bin)

# ---- Build info ----
message(STATUS "Build type: ${CMAKE_BUILD_TYPE}")
message(STATUS "Qt5 version: ${Qt5_VERSION}")
message(STATUS "OpenSSL version: ${OPENSSL_VERSION}")
message(STATUS "C++ Standard: ${CMAKE_CXX_STANDARD}")
