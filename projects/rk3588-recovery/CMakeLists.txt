cmake_minimum_required(VERSION 3.16)
project(rk3588_recovery VERSION 2.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)

# Performance flags
set(CMAKE_CXX_FLAGS_RELEASE "-O3 -march=native -DNDEBUG -flto")
set(CMAKE_CXX_FLAGS_DEBUG   "-g -O0 -fsanitize=address,undefined")

# Find required packages
find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

# ─── Sources ──────────────────────────────────────────
set(SOURCES
    src/main.cpp
    src/Partition.cpp
    src/BackupEngine.cpp
    src/ChecksumVerifier.cpp
    src/Logger.cpp
    src/GPIO.cpp
    src/ui/RecoveryUI.cpp
)

# ─── Main executable ───────────────────────────────────
add_executable(rk3588_recovery ${SOURCES})

target_include_directories(rk3588_recovery PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(rk3588_recovery PRIVATE
    OpenSSL::SSL
    OpenSSL::Crypto
    Threads::Threads
)

# ─── Tests ────────────────────────────────────────────
add_executable(test_engine
    tests/test_engine.cpp
    src/Partition.cpp
    src/BackupEngine.cpp
    src/ChecksumVerifier.cpp
    src/Logger.cpp
)

target_include_directories(test_engine PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(test_engine PRIVATE
    OpenSSL::SSL
    OpenSSL::Crypto
    Threads::Threads
)

# ─── Install ──────────────────────────────────────────
install(TARGETS rk3588_recovery RUNTIME DESTINATION /usr/local/bin)
install(DIRECTORY config/ DESTINATION /etc/rk3588-recovery/)
