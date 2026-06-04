#pragma once

#include <QQuickImageProvider>
#include <QImage>
#include <QMutex>
#include <atomic>

/// Provides ultrasound frames to QML via the "image://ultrasound/frame" URL scheme.
class UltrasoundImageProvider : public QQuickImageProvider
{
public:
    explicit UltrasoundImageProvider();

    /// Called by the QML engine to get an image.
    QImage requestImage(const QString &id, QSize *size,
                        const QSize &requestedSize) override;

    /// Called by the frame generator to push a new frame.
    void updateFrame(const QImage &frame);

    /// Get the current frame for external inspection.
    QImage currentFrame() const;

private:
    QImage m_frame;
    mutable QMutex m_mutex;
};
