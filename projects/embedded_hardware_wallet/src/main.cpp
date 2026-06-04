#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QQmlContext>
#include <QQuickImageProvider>
#include <QIcon>
#include <QFile>
#include <QResource>

#include <QTimer>

#include "Common.h"
#include "core/WalletCore.h"
#include "ui/AppCore.h"
#include "ui/WalletImageProvider.h"

int main(int argc, char* argv[]) {
    // ========================================================================
    // Application Setup
    // ========================================================================
    QGuiApplication app(argc, argv);
    app.setApplicationName("Embedded Hardware Wallet");
    app.setApplicationVersion("1.0.0");
    app.setOrganizationName("EmbeddedHW");

    // ========================================================================
    // Load BIP39 Wordlist
    // ========================================================================
    ehw::log(ehw::LogLevel::Info, "Loading BIP39 wordlist...");

    // Load from Qt resource system
    QFile wordlistFile(":/bip39_english.txt");
    if (wordlistFile.open(QIODevice::ReadOnly | QIODevice::Text)) {
        QByteArray content = wordlistFile.readAll();
        wordlistFile.close();

        if (ehw::WalletCore::loadWordlistFromMemory(content.constData(),
                                                      static_cast<size_t>(content.size()))) {
            ehw::log(ehw::LogLevel::Info, "BIP39 wordlist loaded: {} words",
                     ehw::WalletCore::getWordlist().size());
        } else {
            ehw::log(ehw::LogLevel::Error, "Failed to parse BIP39 wordlist");
            return 1;
        }
    } else {
        ehw::log(ehw::LogLevel::Error, "Cannot open BIP39 wordlist resource");
        return 1;
    }

    // ========================================================================
    // Create Core Components
    // ========================================================================
    auto* appCore = new ehw::AppCore(&app);
    auto* imageProvider = new ehw::WalletImageProvider();

    // ========================================================================
    // Setup QML Engine
    // ========================================================================
    QQmlApplicationEngine engine;

    // Register C++ types with QML
    engine.rootContext()->setContextProperty("AppCore", appCore);

    // Register image provider for QR codes
    engine.addImageProvider("wallet", imageProvider);

    // Load QML UI
    const QUrl url(QStringLiteral("qrc:/main.qml"));
    QObject::connect(&engine, &QQmlApplicationEngine::objectCreated,
        &app, [url](QObject* obj, const QUrl& objUrl) {
            if (!obj && url == objUrl) {
                ehw::log(ehw::LogLevel::Error, "Failed to load QML: {}", url.toString().toStdString());
                QCoreApplication::exit(-1);
            }
        }, Qt::QueuedConnection);

    engine.load(url);

    if (engine.rootObjects().isEmpty()) {
        ehw::log(ehw::LogLevel::Error, "No QML root objects created");
        return -1;
    }

    // ========================================================================
    // Startup
    // ========================================================================
    ehw::log(ehw::LogLevel::Info, "Embedded Hardware Wallet started");
    ehw::log(ehw::LogLevel::Info, "C++ Standard: {}", __cplusplus);
    ehw::log(ehw::LogLevel::Info, "Build: {} {}", __DATE__, __TIME__);

    // Initial network check (non-blocking)
    QTimer::singleShot(1000, [appCore]() {
        appCore->checkNetworkStatus();
    });

    return app.exec();
}
