// RK3588 Recovery — Terminal UI Implementation
// Pure ANSI escape sequences, no external UI library required.
#include "ui/RecoveryUI.h"
#include <iostream>
#include <sstream>
#include <iomanip>
#include <cstring>
#include <termios.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <dirent.h>
#include <algorithm>

namespace rk3588 {
namespace ui {

// ─── ANSI Escape Codes ─────────────────────────────────
namespace ansi {
    const char* CLEAR      = "\033[2J\033[H";
    const char* HIDE_CURSOR = "\033[?25l";
    const char* SHOW_CURSOR = "\033[?25h";
    const char* BOLD        = "\033[1m";
    const char* DIM         = "\033[2m";
    const char* RESET       = "\033[0m";
    const char* FG_RED      = "\033[31m";
    const char* FG_GREEN    = "\033[32m";
    const char* FG_YELLOW   = "\033[33m";
    const char* FG_BLUE     = "\033[34m";
    const char* FG_CYAN     = "\033[36m";
    const char* FG_WHITE    = "\033[37m";
    const char* BG_BLUE     = "\033[44m";
    const char* BG_GREEN    = "\033[42m";
    const char* BG_RED      = "\033[41m";

    void move_cursor(int x, int y) {
        std::cout << "\033[" << y << ";" << x << "H" << std::flush;
    }
    void progress_bar_chars() {
        // Unicode block characters for smooth progress
        // " ▏▎▍▌▋▊▉█"
    }
}

RecoveryUI::RecoveryUI() = default;
RecoveryUI::~RecoveryUI() { shutdown(); }

bool RecoveryUI::init() {
    get_terminal_size();
    set_raw_mode(true);
    std::cout << ansi::CLEAR << ansi::HIDE_CURSOR << std::flush;
    initialized_ = true;
    return true;
}

void RecoveryUI::shutdown() {
    if (!initialized_) return;
    std::cout << ansi::SHOW_CURSOR << ansi::RESET << std::flush;
    set_raw_mode(false);
    initialized_ = false;
}

void RecoveryUI::clear() {
    std::cout << ansi::CLEAR << std::flush;
}

// ─── Main Menu ─────────────────────────────────────────
int RecoveryUI::show_main_menu() {
    std::vector<MenuItem> items = {
        {1, '1', "Install System",    "Install system image from storage"},
        {2, '2', "Backup System",     "Backup current system partition"},
        {3, '3', "Restore System",    "Restore system from backup"},
        {4, '4', "Backup Apps",       "Backup user applications & data"},
        {5, '5', "Restore Apps",      "Restore user applications & data"},
        {6, '6', "View Logs",         "View operation history"},
        {7, '7', "Verify Image",      "Verify backup image integrity"},
        {8, '8', "Exit / Reboot",     "Exit recovery and reboot"},
    };

    int selected = 0;
    while (true) {
        clear();
        draw_box(1, 1, width_ - 2, height_ - 2, " RK3588 Recovery System v2.0 ");
        draw_menu(items, selected);

        // Status bar
        ansi::move_cursor(1, height_ - 1);
        std::cout << ansi::DIM << "↑↓: Navigate  Enter: Select  Q: Quit"
                  << ansi::RESET << std::flush;

        int ch = get_char();
        switch (ch) {
            case 'A': // UP
                selected = (selected - 1 + static_cast<int>(items.size())) % items.size();
                break;
            case 'B': // DOWN
                selected = (selected + 1) % static_cast<int>(items.size());
                break;
            case '\n': case '\r':
                return items[selected].id;
            case 'q': case 'Q':
                return 8; // Exit
            default:
                // Check direct number keys
                if (ch >= '1' && ch <= '8') {
                    return ch - '0';
                }
                break;
        }
    }
}

// ─── Progress Display ──────────────────────────────────
void RecoveryUI::show_progress(const std::string& title, int percent,
                                const std::string& status_line) {
    std::lock_guard<std::mutex> lock(ui_mutex_);

    clear();
    int center_y = height_ / 2 - 2;
    int bar_width = width_ - 12;
    int bar_x = 6;

    draw_box(1, 1, width_ - 2, height_ - 2, title.c_str());

    // Progress bar
    ansi::move_cursor(bar_x, center_y);
    std::cout << ansi::BOLD << "[";

    int filled = (percent * bar_width) / 100;
    for (int i = 0; i < bar_width; i++) {
        if (i < filled) {
            std::cout << ansi::BG_GREEN << " " << ansi::RESET;
        } else {
            std::cout << ansi::DIM << "·" << ansi::RESET;
        }
    }
    std::cout << ansi::BOLD << "] " << std::setw(3) << percent << "%"
              << ansi::RESET;

    // Status text
    ansi::move_cursor(bar_x, center_y + 2);
    std::cout << ansi::DIM << status_line << ansi::RESET;

    // Bottom hint
    ansi::move_cursor(1, height_ - 1);
    std::cout << ansi::DIM << "Press Q to cancel operation" << ansi::RESET;

    std::cout << std::flush;
}

// ─── Message Dialog ────────────────────────────────────
void RecoveryUI::show_message(const std::string& title,
                               const std::string& message) {
    std::lock_guard<std::mutex> lock(ui_mutex_);

    clear();
    draw_box(1, 1, width_ - 2, height_ - 2, title.c_str());

    int center_y = height_ / 2;
    ansi::move_cursor(4, center_y);
    std::cout << message;

    ansi::move_cursor(4, center_y + 3);
    std::cout << ansi::BOLD << "Press any key to continue..." << ansi::RESET;
    std::cout << std::flush;

    get_char();
}

// ─── Confirmation Dialog ───────────────────────────────
bool RecoveryUI::confirm(const std::string& title, const std::string& message) {
    std::lock_guard<std::mutex> lock(ui_mutex_);

    clear();
    draw_box(1, 1, width_ - 2, height_ - 2, title.c_str());

    int center_y = height_ / 2;
    ansi::move_cursor(4, center_y);
    std::cout << ansi::FG_YELLOW << "⚠ " << message << ansi::RESET;

    ansi::move_cursor(4, center_y + 3);
    std::cout << ansi::BOLD << "[Y] Yes  [N] No" << ansi::RESET;
    std::cout << std::flush;

    while (true) {
        int ch = get_char();
        if (ch == 'y' || ch == 'Y' || ch == '\n') return true;
        if (ch == 'n' || ch == 'N' || ch == 'q' || ch == 'Q') return false;
    }
}

// ─── Log Viewer ────────────────────────────────────────
void RecoveryUI::show_logs(const std::vector<std::string>& log_lines, int page) {
    std::lock_guard<std::mutex> lock(ui_mutex_);

    const int lines_per_page = height_ - 6;
    int total_pages = (static_cast<int>(log_lines.size()) + lines_per_page - 1) / lines_per_page;
    if (total_pages < 1) total_pages = 1;
    if (page >= total_pages) page = total_pages - 1;
    if (page < 0) page = 0;

    clear();
    draw_box(1, 1, width_ - 2, height_ - 2, " Operation Logs ");

    int start = page * lines_per_page;
    for (int i = 0; i < lines_per_page && (start + i) < static_cast<int>(log_lines.size()); i++) {
        ansi::move_cursor(3, 4 + i);
        std::string line = log_lines[start + i];
        if (line.length() > static_cast<size_t>(width_ - 6)) {
            line = line.substr(0, width_ - 9) + "...";
        }
        std::cout << line;
    }

    ansi::move_cursor(1, height_ - 1);
    std::cout << ansi::DIM << "Page " << (page + 1) << "/" << total_pages
              << "  ←→: Navigate  Q: Back" << ansi::RESET << std::flush;
}

// ─── File Selector ─────────────────────────────────────
std::string RecoveryUI::select_file(const std::string& directory,
                                     const std::string& pattern) {
    std::vector<std::string> files;
    DIR* dir = opendir(directory.c_str());
    if (dir) {
        struct dirent* entry;
        while ((entry = readdir(dir)) != nullptr) {
            std::string name = entry->d_name;
            if (name == "." || name == "..") continue;
            // Simple pattern matching
            if (pattern.empty() || name.find(pattern) != std::string::npos) {
                files.push_back(name);
            }
        }
        closedir(dir);
    }

    if (files.empty()) return "";

    int selected = 0;
    while (true) {
        clear();
        draw_box(1, 1, width_ - 2, height_ - 2, " Select File ");
        for (size_t i = 0; i < files.size(); i++) {
            ansi::move_cursor(4, 4 + static_cast<int>(i));
            if (static_cast<int>(i) == selected) {
                std::cout << ansi::BG_BLUE << ansi::FG_WHITE << " > " << files[i]
                          << std::string(width_ - 10 - files[i].length(), ' ')
                          << ansi::RESET;
            } else {
                std::cout << "   " << files[i];
            }
            if (static_cast<int>(i) > height_ - 8) break;
        }
        std::cout << std::flush;

        int ch = get_char();
        if (ch == 'A' && selected > 0) selected--;
        if (ch == 'B' && selected < static_cast<int>(files.size()) - 1) selected++;
        if (ch == '\n' || ch == '\r') return directory + "/" + files[selected];
        if (ch == 'q' || ch == 'Q') return "";
    }
}

// ─── Helpers ───────────────────────────────────────────
void RecoveryUI::draw_box(int x, int y, int w, int h, const char* title) {
    ansi::move_cursor(x, y);
    std::cout << ansi::FG_CYAN << "╔" << std::string(w - 2, '=') << "╗";

    if (title) {
        ansi::move_cursor(x + 2, y);
        std::cout << ansi::BOLD << " " << title << " " << ansi::RESET << ansi::FG_CYAN;
    }

    for (int i = 1; i < h - 1; i++) {
        ansi::move_cursor(x, y + i);
        std::cout << "║";
        ansi::move_cursor(x + w - 1, y + i);
        std::cout << "║";
    }
    ansi::move_cursor(x, y + h - 1);
    std::cout << "╚" << std::string(w - 2, '=') << "╝" << ansi::RESET;
}

void RecoveryUI::draw_menu(const std::vector<MenuItem>& items, int selected) {
    int start_y = 4;
    for (size_t i = 0; i < items.size(); i++) {
        int y = start_y + static_cast<int>(i) * 2;
        ansi::move_cursor(4, y);

        if (static_cast<int>(i) == selected) {
            std::cout << ansi::BG_BLUE << ansi::FG_WHITE << ansi::BOLD
                      << " " << items[i].key << " " << items[i].label;
            std::cout << std::string(width_ - 12 - items[i].label.length(), ' ');
            std::cout << ansi::RESET;
            ansi::move_cursor(6, y + 1);
            std::cout << ansi::DIM << items[i].description << ansi::RESET;
        } else {
            std::cout << ansi::FG_CYAN << " " << items[i].key << " "
                      << ansi::RESET << items[i].label;
            ansi::move_cursor(6, y + 1);
            std::cout << ansi::DIM << items[i].description << ansi::RESET;
        }
    }
}

void RecoveryUI::draw_progress_bar(int x, int y, int width, int percent,
                                    const std::string& label) {
    ansi::move_cursor(x, y);
    std::cout << label << " [";
    int filled = (percent * width) / 100;
    for (int i = 0; i < width; i++) {
        std::cout << (i < filled ? '#' : '-');
    }
    std::cout << "] " << percent << "%" << std::flush;
}

void RecoveryUI::set_status(const std::string& status) {
    status_line_ = status;
    ansi::move_cursor(1, height_);
    std::cout << ansi::DIM << status
              << std::string(width_ - status.length(), ' ') << ansi::RESET;
    std::cout << std::flush;
}

// ─── Terminal I/O ──────────────────────────────────────
void RecoveryUI::set_raw_mode(bool enable) {
    static struct termios old_tio;
    if (enable) {
        tcgetattr(STDIN_FILENO, &old_tio);
        struct termios new_tio = old_tio;
        new_tio.c_lflag &= ~(ICANON | ECHO);
        tcsetattr(STDIN_FILENO, TCSANOW, &new_tio);
    } else {
        tcsetattr(STDIN_FILENO, TCSANOW, &old_tio);
    }
}

int RecoveryUI::get_char() {
    char buf[4] = {0};
    if (read(STDIN_FILENO, buf, sizeof(buf)) <= 0) return -1;

    // Arrow keys: \033[A, \033[B, etc.
    if (buf[0] == '\033' && buf[1] == '[') {
        return buf[2]; // Return 'A', 'B', etc.
    }
    return buf[0];
}

void RecoveryUI::get_terminal_size() {
    struct winsize ws;
    if (ioctl(STDOUT_FILENO, TIOCGWINSZ, &ws) == 0) {
        width_ = ws.ws_col;
        height_ = ws.ws_row;
    }
    if (width_ < 40) width_ = 80;
    if (height_ < 15) height_ = 24;
}

} // namespace ui
} // namespace rk3588
