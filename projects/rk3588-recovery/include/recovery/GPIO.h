// RK3588 Recovery — GPIO & Power Key Monitor
// Simulates GPIO power key detection for entering recovery mode.
// On real hardware, reads /sys/class/gpio/; on simulator, uses keyboard input.
#pragma once

#include "Types.h"
#include <functional>
#include <thread>
#include <atomic>
#include <chrono>

namespace rk3588 {
namespace recovery {

using PowerKeyCallback = std::function<void(bool long_press)>;

class GPIOMonitor {
public:
    GPIOMonitor();
    ~GPIOMonitor();

    // Start monitoring power key
    bool start(PowerKeyCallback callback);

    // Stop monitoring
    void stop();

    // Check if we should enter recovery mode
    BootMode detect_boot_mode();

    // Simulate long press (for testing)
    void simulate_long_press();

    // Configure GPIO pin and press duration threshold
    void set_press_threshold(std::chrono::milliseconds duration) {
        long_press_threshold_ = duration;
    }

private:
    void monitor_loop();

    std::thread            monitor_thread_;
    std::atomic<bool>      running_{false};
    std::atomic<bool>      long_press_detected_{false};
    PowerKeyCallback       callback_;
    std::chrono::milliseconds long_press_threshold_{3000}; // 3 seconds
};

} // namespace recovery
} // namespace rk3588
