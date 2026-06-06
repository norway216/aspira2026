cmake_minimum_required(VERSION 3.16)
project(payment_engine VERSION 1.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_FLAGS_RELEASE "-O3 -march=native -DNDEBUG")
set(CMAKE_CXX_FLAGS_DEBUG "-g -O0 -fsanitize=address,undefined")

find_package(OpenSSL REQUIRED)
find_package(Threads REQUIRED)

set(SOURCES
    src/main.cpp
    src/config/EngineConfig.cpp
    src/network/TCPServer.cpp
    src/network/Session.cpp
    src/processor/TransactionProcessor.cpp
    src/processor/ValidationEngine.cpp
    src/processor/ExchangeEngine.cpp
    src/storage/AccountStore.cpp
    src/storage/TransactionStore.cpp
    src/crypto/HashChain.cpp
    src/crypto/AESCipher.cpp
)

add_executable(payment_engine ${SOURCES})

target_include_directories(payment_engine PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/src
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

target_link_libraries(payment_engine PRIVATE
    OpenSSL::SSL
    OpenSSL::Crypto
    Threads::Threads
)
