#pragma once

#include <QQuickImageProvider>
#include <QImage>
#include <QPixmap>
#include <QMutex>

namespace ehw {

/**
 * QQuickImageProvider for wallet-related images (QR codes, icons).
 */
class WalletImageProvider : public QQuickImageProvider {
public:
    WalletImageProvider()
        : QQuickImageProvider(QQuickImageProvider::Image) {}

    QImage requestImage(const QString& id, QSize* size,
                        const QSize& requestedSize) override;

    // Update the QR code data (e.g., for receiving address)
    void setQRCodeData(const QString& data);
    void setQRCodePixmap(const QPixmap& pixmap);

private:
    QImage generateQRPlaceholder(const QString& data, const QSize& size);
    QMutex m_mutex;
    QPixmap m_qrPixmap;
    QString m_qrData;
};

} // namespace ehw
