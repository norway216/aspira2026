// RK3588 Recovery — GPIO Monitor Implementation (Simulator)
#include "recovery/GPIO.h"
#include <iostream>
#include <thread>

namespace rk3588 {
namespace recovery {

GPIOMonitor::GPIOMonitor() = default;

GPIOMonitor::~GPIOMonitor() {
    stop();
}

bool GPIOMonitor::start(PowerKeyCallback callback) {
    if (running_) return false;
    callback_ = callback;
    running_ = true;
    monitor_thread_ = std::thread(&GPIOMonitor::monitor_loop, this);
    return true;
}

void GPIOMonitor::stop() {
    running_ = false;
    if (monitor_thread_.joinable()) {
        monitor_thread_.join();
    }
}

BootMode GPIOMonitor::detect_boot_mode() {
    return long_press_detected_ ? BootMode::RECOVERY : BootMode::NORMAL;
}

void GPIOMonitor::simulate_long_press() {
    long_press_detected_ = true;
    if (callback_) callback_(true);
}

void GPIOMonitor::monitor_loop() {
    std::cout << "[GPIO] Power key monitor started (press 'p' for 3s to simulate long press)"
              << std::endl;

    while (running_) {
        // In simulator mode, read keyboard input to simulate power key
        // 'p' held for 3+ seconds simulates long press
        std::cout << "[GPIO] Enter 'p' and hold 3s to enter recovery, 'q' to quit monitor: ";
        std::cout.flush();

        char ch;
        if (std::cin.get(ch)) {
            if (ch == 'p' || ch == 'P') {
                std::cout << "\n[GPIO] Power key pressed..." << std::endl;

                // Wait to see if it's a long press (3 seconds)
                auto press_start = std::chrono::steady_clock::now();
                bool released = false;

                while (running_) {
                    auto elapsed = std::chrono::duration_cast<std::chrono::milliseconds>(
                        std::chrono::steady_clock::now() - press_start);

                    if (elapsed >= long_press_threshold_) {
                        long_press_detected_ = true;
                        std::cout << "[GPIO] LONG PRESS detected (>= 3s)!" << std::endl;
                        if (callback_) callback_(true);
                        break;
                    }

                    // Check for release
                    std::this_thread::sleep_for(std::chrono::milliseconds(200));
                    if (!released && elapsed > std::chrono::milliseconds(500)) {
                        // Simulated release — short press
                        std::cout << "[GPIO] Short press detected (normal boot)" << std::endl;
                        if (callback_) callback_(false);
                        released = true;
                        break;
                    }
                }
            } else if (ch == 'q' || ch == 'Q') {
                break;
            }
        }
    }
    std::cout << "[GPIO] Monitor stopped" << std::endl;
}

} // namespace recovery
} // namespace rk3588
