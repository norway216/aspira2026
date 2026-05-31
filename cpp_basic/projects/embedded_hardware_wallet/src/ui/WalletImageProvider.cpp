#include "WalletImageProvider.h"
#include <QPainter>
#include <QFontMetrics>
#include <QtMath>
#include <QCryptographicHash>

namespace ehw {

QImage WalletImageProvider::requestImage(const QString& id, QSize* size,
                                          const QSize& requestedSize)
{
    QMutexLocker lock(&m_mutex);

    QSize imgSize = requestedSize.isValid() ? requestedSize : QSize(256, 256);
    if (size) *size = imgSize;

    if (id == "qr_receive" && !m_qrData.isEmpty()) {
        return generateQRPlaceholder(m_qrData, imgSize);
    }
    if (id == "qr_backup_share" && !m_qrData.isEmpty()) {
        return generateQRPlaceholder(m_qrData, imgSize);
    }

    // Return placeholder image
    QImage img(imgSize, QImage::Format_ARGB32);
    img.fill(QColor(10, 14, 39));
    QPainter painter(&img);
    painter.setPen(QColor(83, 52, 131));
    painter.setFont(QFont("monospace", 12));
    painter.drawText(img.rect(), Qt::AlignCenter, "QR Code\nUnavailable");
    painter.end();
    return img;
}

void WalletImageProvider::setQRCodeData(const QString& data) {
    QMutexLocker lock(&m_mutex);
    m_qrData = data;
}

void WalletImageProvider::setQRCodePixmap(const QPixmap& pixmap) {
    QMutexLocker lock(&m_mutex);
    m_qrPixmap = pixmap;
}

QImage WalletImageProvider::generateQRPlaceholder(const QString& data, const QSize& size) {
    // Generate a QR-code-like placeholder pattern
    // In a production embedded system, you'd use libqrencode
    QImage img(size, QImage::Format_ARGB32);
    img.fill(QColor(255, 255, 255));

    QPainter painter(&img);
    painter.setRenderHint(QPainter::Antialiasing, false);

    // Draw a simplified position detection pattern (finder patterns)
    int s = qMin(size.width(), size.height());
    int moduleSize = s / 25; // QR version 1 = 21x21 modules + quiet zone

    auto drawFinder = [&](int x, int y) {
        // Outer square (7x7)
        painter.fillRect(x, y, 7 * moduleSize, 7 * moduleSize, QColor(0, 0, 0));
        // Inner white square (5x5)
        painter.fillRect(x + moduleSize, y + moduleSize,
                        5 * moduleSize, 5 * moduleSize, QColor(255, 255, 255));
        // Inner black square (3x3)
        painter.fillRect(x + 2 * moduleSize, y + 2 * moduleSize,
                        3 * moduleSize, 3 * moduleSize, QColor(0, 0, 0));
    };

    // Top-left, top-right, bottom-left finder patterns
    int qz = 2 * moduleSize; // quiet zone
    drawFinder(qz, qz);                          // Top-left
    drawFinder(s - qz - 7 * moduleSize, qz);    // Top-right
    drawFinder(qz, s - qz - 7 * moduleSize);    // Bottom-left

    // Draw data modules as a simple pattern derived from address data
    // This is a simplified visualization — not a real QR code
    QByteArray hash = QCryptographicHash::hash(data.toUtf8(), QCryptographicHash::Sha256);
    int dataIdx = 0;
    for (int row = 0; row < 21; ++row) {
        for (int col = 0; col < 21; ++col) {
            // Skip finder pattern areas
            bool inFinder = (row < 8 && col < 8) ||   // Top-left
                           (row < 8 && col > 12) ||    // Top-right
                           (row > 12 && col < 8);      // Bottom-left
            if (inFinder) continue;

            int x = qz + col * moduleSize;
            int y = qz + row * moduleSize;
            bool dark = (hash[dataIdx % hash.size()] >> (dataIdx % 8)) & 1;
            painter.fillRect(x, y, moduleSize, moduleSize,
                            dark ? QColor(0, 0, 0) : QColor(255, 255, 255));
            ++dataIdx;
        }
    }

    // Draw address label below QR code area
    painter.setPen(QColor(10, 14, 39));
    painter.setFont(QFont("monospace", qMax(8, s / 30)));
    QString shortAddr = data.left(12) + "..." + data.right(8);
    QRect textRect(0, s - qz / 2, s, qz / 2);
    painter.drawText(textRect, Qt::AlignCenter, shortAddr);

    painter.end();
    return img;
}

} // namespace ehw
