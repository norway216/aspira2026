#pragma once

#include <QString>

enum class ErrorCode {
    // General
    Success = 0,
    Unknown = 1,

    // Auth
    UserNotFound = 100,
    InvalidPassword = 101,
    UserAlreadyExists = 102,
    UserLocked = 103,
    SessionExpired = 104,
    NotLoggedIn = 105,
    PasswordTooShort = 106,
    UsernameTooShort = 107,
    PasswordMismatch = 108,

    // Crypto
    CryptoInitFailed = 200,
    KDFFailed = 201,
    EncryptionFailed = 202,
    DecryptionFailed = 203,
    SignatureFailed = 204,
    VerificationFailed = 205,
    KeyGenerationFailed = 206,
    RandomGenerationFailed = 207,

    // Database
    DatabaseOpenFailed = 300,
    DatabaseMigrationFailed = 301,
    DatabaseQueryFailed = 302,
    DatabaseInsertFailed = 303,
    DatabaseUpdateFailed = 304,
    DatabaseDeleteFailed = 305,
    DatabaseTransactionFailed = 306,
    DatabaseIntegrityFailed = 307,

    // Transaction
    InsufficientBalance = 400,
    InvalidAmount = 401,
    InvalidRecipient = 402,
    DuplicateNonce = 403,
    TransactionVerifyFailed = 404,
    RecipientNotFound = 405,

    // Wallet
    WalletNotFound = 500,
    WalletAlreadyExists = 501,
    InvalidPublicKey = 502,

    // Backup
    BackupFileNotFound = 600,
    BackupFileCorrupted = 601,
    BackupVersionMismatch = 602,
    BackupDecryptionFailed = 603,
    BackupIntegrityFailed = 604,
    BackupExportFailed = 605,
    BackupRestoreFailed = 606,

    // System
    FileIOError = 700,
    DiskFull = 701,

    // Security (800-899) — per architecture §15
    WalletWiped = 800,
    WalletWipeFailed = 801,
    MaxPasswordAttemptsExceeded = 802,
    SecurityPolicyViolation = 803,
    SecureCleanupFailed = 804,
};

struct Error {
    ErrorCode code = ErrorCode::Success;
    QString message;

    Error() = default;
    Error(ErrorCode c, QString msg) : code(c), message(std::move(msg)) {}

    bool isOk() const { return code == ErrorCode::Success; }
    bool isFail() const { return code != ErrorCode::Success; }

    static Error ok() { return Error{ErrorCode::Success, {}}; }
    static Error fail(ErrorCode c, QString msg) { return Error{c, std::move(msg)}; }
};
