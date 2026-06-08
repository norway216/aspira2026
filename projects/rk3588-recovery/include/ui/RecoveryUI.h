// RK3588 Recovery — Main TUI Interface
// Provides a terminal-based UI with menus, progress bars, and status display.
// Uses raw terminal escape sequences (no ncurses dependency required).
#pragma once

#include <string>
#include <functional>
#include <atomic>
#include <mutex>

namespace rk3588 {
namespace ui {

struct MenuItem {
    int  id;
    char key;            // shortcut key
    std::string label;
    std::string description;
};

class RecoveryUI {
public:
    RecoveryUI();
    ~RecoveryUI();

    // Initialize terminal for TUI mode
    bool init();

    // Restore terminal to normal
    void shutdown();

    // Display main menu and return selection
    int show_main_menu();

    // Show progress bar during operations
    void show_progress(const std::string& title, int percent,
                       const std::string& status_line);

    // Show a message dialog
    void show_message(const std::string& title, const std::string& message);

    // Show confirmation dialog (returns true if confirmed)
    bool confirm(const std::string& title, const std::string& message);

    // Show log viewer
    void show_logs(const std::vector<std::string>& log_lines, int page);

    // Show file selection dialog
    std::string select_file(const std::string& directory,
                            const std::string& pattern);

    // Update a status bar at the bottom
    void set_status(const std::string& status);

    // Clear screen
    void clear();

    // Get terminal dimensions
    int width()  const { return width_; }
    int height() const { return height_; }

private:
    void draw_box(int x, int y, int w, int h, const char* title = nullptr);
    void draw_menu(const std::vector<MenuItem>& items, int selected);
    void draw_progress_bar(int x, int y, int width, int percent,
                           const std::string& label);
    int  get_char();
    void set_raw_mode(bool enable);
    void get_terminal_size();

    int width_  = 80;
    int height_ = 24;
    bool initialized_ = false;
    std::string status_line_;
    mutable std::mutex ui_mutex_;
};

} // namespace ui
} // namespace rk3588
