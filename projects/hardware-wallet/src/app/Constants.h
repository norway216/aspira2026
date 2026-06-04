#pragma once

#include <QString>
#include <chrono>

namespace AppConstants {

// Application
constexpr const char* APP_NAME = "RK3568 Hardware Wallet";
constexpr const char* APP_VERSION = "1.0.0";
constexpr const char* ORG_NAME = "RKWallet";

// Database
constexpr const char* DB_PATH = "wallet.db";
constexpr const char* DB_CONNECTION_NAME = "wallet_main";

// Security
constexpr int MIN_PASSWORD_LENGTH = 8;
constexpr int MAX_FAILED_ATTEMPTS = 3;       // Architecture §4: 3 strikes -> wipe
constexpr int LOCKOUT_DURATION_SECONDS = 30;
constexpr int DEFAULT_AUTO_LOCK_SECONDS = 300;  // 5 minutes
constexpr int AUTO_LOCK_WARNING_SECONDS = 30;

// Self-destruct policy (per architecture §4)
constexpr bool WIPE_WALLET_ON_MAX_FAILED_ATTEMPTS = true;
constexpr bool DELETE_USER_ON_WIPE = false;
constexpr bool ENABLE_SQLITE_SECURE_DELETE = true;
constexpr bool ENABLE_VACUUM_AFTER_WIPE = true;

// KDF (Argon2id)
constexpr int KDF_OPS_LIMIT = 4;        // crypto_pwhash_OPSLIMIT_MODERATE
constexpr int KDF_MEM_LIMIT = 268435456; // crypto_pwhash_MEMLIMIT_MODERATE (256 MiB)

// KDF for tests (lighter)
constexpr int KDF_OPS_LIMIT_INTERACTIVE = 2;
constexpr int KDF_MEM_LIMIT_INTERACTIVE = 67108864; // 64 MiB

// Backup
constexpr const char* BACKUP_FORMAT_ID = "rk3568-wallet-backup";
constexpr int BACKUP_FORMAT_VERSION = 1;
constexpr int MAX_BACKUP_SNAPSHOTS = 3;

// Transactions
constexpr double DEFAULT_FEE = 0.0;
constexpr int TX_HISTORY_PAGE_SIZE = 20;

// Timeouts
constexpr auto SESSION_CHECK_INTERVAL = std::chrono::seconds(10);

} // namespace AppConstants
