#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QThreadPool>
#include <QDir>
#include <QStandardPaths>

#include "app/Constants.h"
#include "util/Logger.h"
#include "util/Error.h"
#include "crypto/CryptoProvider.h"
#include "persistence/Database.h"
#include "persistence/UserRepository.h"
#include "persistence/WalletRepository.h"
#include "persistence/TransactionRepository.h"
#include "persistence/AuditLogRepository.h"
#include "service/AuthService.h"
#include "service/SecurityService.h"
#include "service/WalletService.h"
#include "service/TransactionService.h"
#include "service/BackupService.h"
#include "service/AuditService.h"
#include "viewmodel/LoginViewModel.h"
#include "viewmodel/RegisterViewModel.h"
#include "viewmodel/DashboardViewModel.h"
#include "viewmodel/SendViewModel.h"
#include "viewmodel/ReceiveViewModel.h"
#include "viewmodel/HistoryViewModel.h"
#include "viewmodel/BackupViewModel.h"
#include "viewmodel/RestoreViewModel.h"
#include "viewmodel/SettingsViewModel.h"

int main(int argc, char *argv[])
{
    // Initialize logging
    Logger::init("wallet_app.log");
    LOG_INFO("{} v{} starting...", AppConstants::APP_NAME, AppConstants::APP_VERSION);

    // Initialize libsodium
    if (!CryptoProvider::instance().initialize()) {
        LOG_CRITICAL("Failed to initialize libsodium");
        return -1;
    }
    LOG_INFO("Crypto provider initialized (ARM crypto: {})",
             CryptoProvider::instance().hasArmCryptoExtensions() ? "yes" : "no");

    // Qt setup
    QGuiApplication app(argc, argv);
    app.setApplicationName(AppConstants::APP_NAME);
    app.setApplicationVersion(AppConstants::APP_VERSION);
    app.setOrganizationName(AppConstants::ORG_NAME);

    // Database path
    QString dataDir = QStandardPaths::writableLocation(QStandardPaths::AppDataLocation);
    QDir().mkpath(dataDir);
    QString dbPath = dataDir + QStringLiteral("/wallet.db");
    LOG_INFO("Database path: {}", dbPath.toStdString());

    // Initialize database
    auto* database = new Database(dbPath);
    auto openResult = database->open();
    if (openResult.isFail()) {
        LOG_CRITICAL("Failed to open database: {}", openResult.error().message.toStdString());
        return -1;
    }
    LOG_INFO("Database opened successfully");

    // Crypto thread pool (2 threads for KDF operations)
    auto* cryptoPool = new QThreadPool(&app);
    cryptoPool->setMaxThreadCount(2);

    // Repositories (shared across services)
    auto* userRepo = new UserRepository(database);
    auto* walletRepo = new WalletRepository(database);
    auto* txRepo = new TransactionRepository(database);
    auto* auditRepo = new AuditLogRepository(database);

    // Services — SecurityService must be created before AuthService
    // (per architecture §14: SecurityService is injected into AuthService)
    auto* securityService = new SecurityService(database, userRepo, walletRepo,
                                                 txRepo, auditRepo, &app);
    auto* authService = new AuthService(database, securityService, cryptoPool, &app);
    auto* walletService = new WalletService(database, authService, cryptoPool, &app);
    auto* txService = new TransactionService(database, authService, cryptoPool, &app);
    auto* backupService = new BackupService(database, authService, &app);
    auto* auditService = new AuditService(database, &app);

    // ViewModels
    auto* loginVM = new LoginViewModel(authService, &app);
    auto* registerVM = new RegisterViewModel(authService, &app);
    auto* dashboardVM = new DashboardViewModel(authService, walletService, txService, &app);
    auto* sendVM = new SendViewModel(authService, walletService, txService, &app);
    auto* receiveVM = new ReceiveViewModel(walletService, &app);
    auto* historyVM = new HistoryViewModel(walletService, txService, &app);
    auto* backupVM = new BackupViewModel(backupService, &app);
    auto* restoreVM = new RestoreViewModel(backupService, &app);
    auto* settingsVM = new SettingsViewModel(authService, auditService, &app);

    // QML engine
    QQmlApplicationEngine engine;
    engine.addImportPath("qrc:/");

    // Register ViewModels with QML context
    QQmlContext* context = engine.rootContext();
    context->setContextProperty("LoginViewModel", loginVM);
    context->setContextProperty("RegisterViewModel", registerVM);
    context->setContextProperty("DashboardViewModel", dashboardVM);
    context->setContextProperty("SendViewModel", sendVM);
    context->setContextProperty("ReceiveViewModel", receiveVM);
    context->setContextProperty("HistoryViewModel", historyVM);
    context->setContextProperty("BackupViewModel", backupVM);
    context->setContextProperty("RestoreViewModel", restoreVM);
    context->setContextProperty("SettingsViewModel", settingsVM);
    context->setContextProperty("AuthService", authService);

    // Connect session expiry to QML
    QObject::connect(authService, &AuthService::sessionExpired, &app, []() {
        LOG_INFO("Session expired - returning to login");
    });

    // Load main QML
    const QUrl url(QStringLiteral("qrc:/qml/main.qml"));

    QObject::connect(&engine, &QQmlApplicationEngine::objectCreated,
        &app, [url](QObject *obj, const QUrl &objUrl) {
            if (!obj && url == objUrl) {
                LOG_ERROR("Failed to load QML: {}", url.toString().toStdString());
                QCoreApplication::exit(-1);
            }
        }, Qt::QueuedConnection);

    engine.load(url);

    if (engine.rootObjects().isEmpty()) {
        LOG_ERROR("No root objects created");
        return -1;
    }

    LOG_INFO("Application started successfully");
    int result = app.exec();

    // Cleanup
    cryptoPool->waitForDone();
    database->close();

    Logger::shutdown();
    return result;
}
