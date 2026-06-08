// RK3588 Recovery System — Main Entry Point
// Cross-platform recovery/backup/restore system for RK3588 platform.
// In simulator mode, uses file-based partitions for development/testing.

#include "recovery/Types.h"
#include "recovery/Partition.h"
#include "recovery/BackupEngine.h"
#include "recovery/ChecksumVerifier.h"
#include "recovery/Logger.h"
#include "recovery/GPIO.h"
#include "ui/RecoveryUI.h"

#include <iostream>
#include <csignal>
#include <cstdlib>
#include <thread>
#include <chrono>

using namespace rk3588::recovery;
using namespace rk3588::ui;

// Global pointers for signal handler cleanup
static RecoveryUI*    g_ui     = nullptr;
static Logger*        g_logger = nullptr;
static GPIOMonitor*   g_gpio   = nullptr;
static bool           g_running = true;

void signal_handler(int sig) {
    std::cout << "\n\nSignal " << sig << " received. Shutting down...\n";
    g_running = false;
    if (g_ui) g_ui->shutdown();
    exit(0);
}

// ─── Simulated Partition Setup ─────────────────────────
void setup_simulated_partitions(PartitionManager& pm) {
    std::cout << "[INIT] Creating simulated partitions..." << std::endl;

    // Create test partition files
    std::string base = "/tmp/rk3588_recovery_sim";
    system(("mkdir -p " + base + "/mnt/{system,recovery,userdata}").c_str());

    pm.create_simulated_partition("boot",     base + "/boot.img",     64ULL*1024*1024,    "fat32");
    pm.create_simulated_partition("system",   base + "/system.img",   8ULL*1024*1024*1024, "ext4");
    pm.create_simulated_partition("recovery", base + "/recovery.img", 512ULL*1024*1024,    "ext4");
    pm.create_simulated_partition("userdata", base + "/userdata.img", 16ULL*1024*1024*1024,"ext4");

    // Write some test data to system partition
    std::string cmd = "dd if=/dev/urandom of=" + base + "/system.img bs=1M count=64 2>/dev/null";
    system(cmd.c_str());

    // Create test app data
    system(("mkdir -p " + base + "/userdata/apps/test_app").c_str());
    system(("echo 'test data' > " + base + "/userdata/apps/test_app/config.txt").c_str());

    pm.mount_all();
    std::cout << "[INIT] Partitions ready" << std::endl;
}

// ─── Recovery Menu Loop ────────────────────────────────
void run_recovery_loop(RecoveryUI& ui, BackupEngine& engine,
                       ChecksumVerifier& verifier, Logger& logger,
                       PartitionManager& pm) {
    std::string backup_dir = "/tmp/rk3588_recovery_sim/backups";
    system(("mkdir -p " + backup_dir).c_str());

    while (g_running) {
        int choice = ui.show_main_menu();

        if (choice == 8) { // Exit / Reboot
            ui.show_message("Reboot", "System will reboot now...");
            break;
        }

        std::string title;
        Operation op = Operation::NONE;
        bool result = false;

        switch (choice) {
            case 1: { // Install System
                title = "Install System";
                auto file = ui.select_file(backup_dir, ".img");
                if (file.empty()) continue;
                if (!ui.confirm(title, "This will OVERWRITE the system partition. Continue?"))
                    continue;

                logger.log(Operation::INSTALL_SYSTEM, Status::RUNNING,
                           "Installing from %s", file.c_str());

                auto callback = [&ui, &title](const Progress& p) {
                    ui.show_progress(title, p.percent, p.message);
                };
                result = engine.install_system(file, "system", callback);
                op = Operation::INSTALL_SYSTEM;
                break;
            }
            case 2: { // Backup System
                title = "Backup System";
                if (!ui.confirm(title, "Backup system partition? This may take a few minutes."))
                    continue;

                logger.log(Operation::BACKUP_SYSTEM, Status::RUNNING, "Backing up system");

                auto timestamp = std::time(nullptr);
                char out_path[512];
                struct tm tm_info;
                localtime_r(&timestamp, &tm_info);
                strftime(out_path, sizeof(out_path),
                         "/tmp/rk3588_recovery_sim/backups/system_%Y%m%d_%H%M%S.img.xz",
                         &tm_info);

                auto callback = [&ui, &title](const Progress& p) {
                    ui.show_progress(title, p.percent, p.message);
                };
                result = engine.backup_system("system", out_path, callback);
                if (result) {
                    auto checksum = verifier.compute(out_path, ChecksumType::SHA256);
                    ui.show_message(title,
                        std::string("Backup complete!\nFile: ") + out_path +
                        "\nSHA256: " + checksum.hash);
                }
                op = Operation::BACKUP_SYSTEM;
                break;
            }
            case 3: { // Restore System
                title = "Restore System";
                auto file = ui.select_file(backup_dir, ".xz");
                if (file.empty()) continue;
                if (!ui.confirm(title, "This will RESTORE from backup. Current system will be overwritten."))
                    continue;

                logger.log(Operation::RESTORE_SYSTEM, Status::RUNNING,
                           "Restoring from %s", file.c_str());

                auto callback = [&ui, &title](const Progress& p) {
                    ui.show_progress(title, p.percent, p.message);
                };
                result = engine.restore_system(file, "system", callback);
                op = Operation::RESTORE_SYSTEM;
                break;
            }
            case 4: { // Backup Apps
                title = "Backup Apps";
                if (!ui.confirm(title, "Backup user applications and data?"))
                    continue;

                auto timestamp = std::time(nullptr);
                char out_path[512];
                struct tm tm_info;
                localtime_r(&timestamp, &tm_info);
                strftime(out_path, sizeof(out_path),
                         "/tmp/rk3588_recovery_sim/backups/apps_%Y%m%d_%H%M%S.tar.xz",
                         &tm_info);

                std::vector<std::string> paths = {
                    "/tmp/rk3588_recovery_sim/userdata/apps"
                };

                auto callback = [&ui, &title](const Progress& p) {
                    ui.show_progress(title, p.percent, p.message);
                };
                result = engine.backup_apps(paths, out_path, callback);
                if (result) {
                    ui.show_message(title,
                        std::string("App backup complete!\nFile: ") + out_path);
                }
                op = Operation::BACKUP_APPS;
                break;
            }
            case 5: { // Restore Apps
                title = "Restore Apps";
                auto file = ui.select_file(backup_dir, ".tar");
                if (file.empty()) continue;
                if (!ui.confirm(title, "Restore applications from backup?"))
                    continue;

                auto callback = [&ui, &title](const Progress& p) {
                    ui.show_progress(title, p.percent, p.message);
                };
                result = engine.restore_apps(file, "/", callback);
                op = Operation::RESTORE_APPS;
                break;
            }
            case 6: { // View Logs
                auto entries = logger.get_entries(200);
                std::vector<std::string> lines;
                for (const auto& e : entries) {
                    char buf[512];
                    struct tm tm_info;
                    localtime_r(&e.timestamp, &tm_info);
                    char time_buf[32];
                    strftime(time_buf, sizeof(time_buf), "%m-%d %H:%M:%S", &tm_info);
                    snprintf(buf, sizeof(buf), "[%s] %-18s %s",
                             time_buf, operation_name(e.operation), e.message);
                    lines.push_back(buf);
                }
                int page = 0;
                ui.show_logs(lines, page);
                getchar(); // wait for key
                break;
            }
            case 7: { // Verify Image
                title = "Verify Image";
                auto file = ui.select_file(backup_dir, ".xz");
                if (file.empty()) {
                    file = ui.select_file(backup_dir, ".img");
                }
                if (file.empty()) continue;

                ui.show_progress(title, 0, "Computing SHA256 checksum...");
                auto checksum = verifier.compute(file, ChecksumType::SHA256);
                ui.show_message(title,
                    std::string("SHA256: ") + checksum.hash +
                    "\nStatus: " + (checksum.verified ? "VERIFIED" : "FAILED"));
                break;
            }
        }

        // Log result
        if (op != Operation::NONE) {
            logger.log(op, result ? Status::COMPLETED : Status::FAILED,
                       result ? "Operation completed successfully" : "Operation failed");
        }

        if (!result && op != Operation::NONE) {
            ui.show_message("Error", "Operation failed. Check logs for details.");
        }
    }
}

// ─── Main ──────────────────────────────────────────────
int main(int argc, char* argv[]) {
    signal(SIGINT, signal_handler);
    signal(SIGTERM, signal_handler);

    std::cout << "╔══════════════════════════════════════════════════════╗\n";
    std::cout << "║   RK3588 Recovery System v2.0                        ║\n";
    std::cout << "║   System Install / Backup / Restore                  ║\n";
    std::cout << "╚══════════════════════════════════════════════════════╝\n\n";

    bool sim_mode = true;
    if (argc > 1 && std::string(argv[1]) == "--real") {
        sim_mode = false;
        std::cout << "[WARN] Real hardware mode — use with caution!\n";
    }

    // Initialize components
    RecoveryUI ui;
    if (!ui.init()) {
        std::cerr << "Failed to initialize UI\n";
        return 1;
    }
    g_ui = &ui;

    Logger logger("/tmp/rk3588_recovery.log");
    g_logger = &logger;

    PartitionManager pm;
    ChecksumVerifier verifier;
    BackupEngine engine(&pm, &verifier);

    if (sim_mode) {
        std::cout << "[INIT] Running in SIMULATOR mode\n";
        setup_simulated_partitions(pm);
    } else {
        if (!pm.detect_partitions("config/partitions.conf")) {
            std::cerr << "Failed to detect partitions\n";
            return 1;
        }
    }

    // GPIO monitor (simulated keyboard input)
    GPIOMonitor gpio;
    g_gpio = &gpio;

    std::cout << "\n[INFO] Recovery system ready.\n";
    std::cout << "[INFO] Press Enter to enter Recovery Menu...\n";
    getchar();

    // Run recovery menu loop
    run_recovery_loop(ui, engine, verifier, logger, pm);

    // Cleanup
    ui.shutdown();
    std::cout << "\n[INFO] Recovery system shutdown complete.\n";
    std::cout << "[INFO] Simulating reboot...\n";
    std::this_thread::sleep_for(std::chrono::seconds(1));
    std::cout << "Reboot successful.\n";

    return 0;
}
