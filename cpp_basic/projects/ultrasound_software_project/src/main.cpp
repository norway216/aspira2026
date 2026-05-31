#include <QGuiApplication>
#include <QQmlApplicationEngine>
#include <QSurfaceFormat>

#include "AppCore.h"

int main(int argc, char *argv[])
{
    // Set OpenGL surface format
    QSurfaceFormat fmt;
    fmt.setSwapInterval(1);
    QSurfaceFormat::setDefaultFormat(fmt);

    QGuiApplication app(argc, argv);
    app.setApplicationName("Ultrasound Scanner");
    app.setOrganizationName("UltrasoundProject");
    app.setApplicationVersion("1.0.0");

    QQmlApplicationEngine engine;

    // Create and initialize AppCore
    AppCore appCore;
    if (!appCore.initialize(engine)) {
        qCritical("Failed to initialize AppCore");
        return -1;
    }

    // Clean up on exit
    QObject::connect(&app, &QGuiApplication::aboutToQuit,
                     &appCore, &AppCore::saveSettingsOnExit);

    // Load main QML
    engine.load(QUrl(QStringLiteral("qrc:/qml/main.qml")));
    if (engine.rootObjects().isEmpty()) {
        qCritical("Failed to load QML");
        return -1;
    }

    return app.exec();
}
