#pragma once

#include <QObject>
#include <QImage>
#include <QMutex>
#include <QTimer>
#include <atomic>
#include <memory>
#include <vector>

class ParamManager;

/// Generates simulated B-mode ultrasound frames in a background thread.
/// Produces a sector-shaped fan scan with speckle, cysts, and tissue boundaries.
class FrameGenerator : public QObject
{
    Q_OBJECT
public:
    explicit FrameGenerator(ParamManager *params, QObject *parent = nullptr);
    ~FrameGenerator() override;

    /// Start/stop frame generation loop.
    void start();
    void stop();

    /// Return the latest rendered frame (thread-safe).
    QImage latestFrame() const;

    /// Frame dimensions.
    static constexpr int kWidth  = 512;
    static constexpr int kHeight = 512;

signals:
    /// Emitted when a new frame is ready.
    void frameReady();

private slots:
    void generateFrame();

private:
    /// Render one B-mode frame into m_buffer.
    void renderFrame();
    /// Apply gain to a raw pixel value.
    uint8_t applyGainTgc(uint8_t raw, int y) const;
    /// Generate scanline data for one angular position.
    std::vector<uint8_t> generateScanline(float angleRad, int numSamples) const;

    ParamManager *m_params;
    QTimer       *m_timer;
    QImage        m_buffer;
    mutable QMutex m_mutex;
    std::atomic<bool> m_running{false};

    // Pre-generated anatomical phantom
    std::vector<float> m_phantom;  // grayscale phantom image
    std::vector<float> m_speckle;  // speckle noise field
    int m_frameCounter = 0;
};
