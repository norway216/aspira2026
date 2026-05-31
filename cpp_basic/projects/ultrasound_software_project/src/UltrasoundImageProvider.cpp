#include "UltrasoundImageProvider.h"
#include <QMutexLocker>

UltrasoundImageProvider::UltrasoundImageProvider()
    : QQuickImageProvider(QQuickImageProvider::Image)
    , m_frame(512, 512, QImage::Format_Grayscale8)
{
    m_frame.fill(0);
}

QImage UltrasoundImageProvider::requestImage(const QString & /*id*/,
                                              QSize *size,
                                              const QSize & /*requestedSize*/)
{
    QMutexLocker lock(&m_mutex);
    if (size)
        *size = m_frame.size();
    return m_frame.copy();
}

void UltrasoundImageProvider::updateFrame(const QImage &frame)
{
    QMutexLocker lock(&m_mutex);
    m_frame = frame.copy();
}

QImage UltrasoundImageProvider::currentFrame() const
{
    QMutexLocker lock(&m_mutex);
    return m_frame.copy();
}
