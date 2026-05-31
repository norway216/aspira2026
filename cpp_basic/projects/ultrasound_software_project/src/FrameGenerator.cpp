#include "FrameGenerator.h"
#include "ParamManager.h"

#include <QTimer>
#include <QMutexLocker>
#include <QtMath>
#include <algorithm>
#include <cmath>
#include <random>
#include <numbers>

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------
static std::mt19937 &rng()
{
    static std::mt19937 r(std::random_device{}());
    return r;
}

/// Smooth a circular dark cyst at (cx, cy) with radius r.
static void addCyst(std::vector<float> &img, int w, int h,
                    float cx, float cy, float r, float darkness)
{
    for (int y = 0; y < h; ++y) {
        for (int x = 0; x < w; ++x) {
            float dx = (float(x) - cx) / r;
            float dy = (float(y) - cy) / r;
            float dist = std::sqrt(dx * dx + dy * dy);
            if (dist < 1.0f) {
                float fade = 0.5f * (1.0f + std::cos(float(std::numbers::pi) * dist));
                img[y * w + x] *= (1.0f - darkness * fade);
            }
        }
    }
}

/// Add a bright horizontal layer.
static void addLayer(std::vector<float> &img, int w, int h,
                     float yCenter, float thickness, float brightness)
{
    for (int y = 0; y < h; ++y) {
        float dy = (y - yCenter) / thickness;
        float alpha = std::exp(-dy * dy * 2.0f);
        for (int x = 0; x < w; ++x) {
            img[y * w + x] += brightness * alpha;
        }
    }
}

/// Smooth a hyperechoic (bright) oval region.
static void addHyperechoic(std::vector<float> &img, int w, int h,
                           float cx, float cy, float rx, float ry, float brightness)
{
    for (int y = 0; y < h; ++y) {
        for (int x = 0; x < w; ++x) {
            float dx = (float(x) - cx) / rx;
            float dy = (float(y) - cy) / ry;
            float dist = std::sqrt(dx * dx + dy * dy);
            if (dist < 1.0f) {
                float fade = 0.5f * (1.0f + std::cos(float(std::numbers::pi) * dist));
                img[y * w + x] += brightness * fade;
            }
        }
    }
}

// ---------------------------------------------------------------------------
FrameGenerator::FrameGenerator(ParamManager *params, QObject *parent)
    : QObject(parent)
    , m_params(params)
    , m_timer(new QTimer(this))
    , m_buffer(kWidth, kHeight, QImage::Format_Grayscale8)
{
    m_buffer.fill(0);

    // ---- Build anatomical phantom ----
    m_phantom.assign(kWidth * kHeight, 0.15f);  // baseline grey

    // Tissue layers at different depths
    addLayer(m_phantom, kWidth, kHeight, 80,  6, 0.25f);
    addLayer(m_phantom, kWidth, kHeight, 160, 8, 0.30f);
    addLayer(m_phantom, kWidth, kHeight, 260, 5, 0.22f);
    addLayer(m_phantom, kWidth, kHeight, 360, 7, 0.28f);

    // Cysts (anechoic / hypoechoic regions)
    addCyst(m_phantom, kWidth, kHeight, 200, 130, 22, 0.70f);
    addCyst(m_phantom, kWidth, kHeight, 310, 220, 16, 0.55f);
    addCyst(m_phantom, kWidth, kHeight, 180, 310, 12, 0.50f);

    // Hyperechoic regions (calcification-like)
    addHyperechoic(m_phantom, kWidth, kHeight, 260, 280, 14, 10, 0.40f);
    addHyperechoic(m_phantom, kWidth, kHeight, 320, 120, 20, 8,  0.35f);

    // Clamp
    for (auto &v : m_phantom) v = std::clamp(v, 0.0f, 1.0f);

    // ---- Pre-generate speckle noise field ----
    std::normal_distribution<float> noiseDist(0.0f, 0.04f);
    m_speckle.resize(kWidth * kHeight);
    for (auto &v : m_speckle)
        v = noiseDist(rng());

    // ---- Timer-driven generation ----
    connect(m_timer, &QTimer::timeout, this, &FrameGenerator::generateFrame);
}

FrameGenerator::~FrameGenerator()
{
    stop();
}

void FrameGenerator::start()
{
    if (m_running.exchange(true)) return;
    int interval = std::max(16, 1000 / std::max(1, m_params->frameRate()));
    m_timer->start(interval);
}

void FrameGenerator::stop()
{
    m_running = false;
    m_timer->stop();
}

QImage FrameGenerator::latestFrame() const
{
    QMutexLocker lock(&m_mutex);
    return m_buffer.copy();
}

void FrameGenerator::generateFrame()
{
    if (!m_running) return;
    if (m_params->freeze()) return;

    // Update timer interval in case frame rate changed
    int interval = std::max(16, 1000 / std::max(1, m_params->frameRate()));
    if (m_timer->interval() != interval)
        m_timer->start(interval);

    renderFrame();
    emit frameReady();
}

// ---------------------------------------------------------------------------
// Frame rendering
// ---------------------------------------------------------------------------
std::vector<uint8_t> FrameGenerator::generateScanline(float angleRad, int numSamples) const
{
    std::vector<uint8_t> samples(numSamples, 0);

    // Scanline origin: top center of the image
    const float originX = kWidth  * 0.5f;
    const float originY = 0.0f;

    const float maxRadius = kHeight;  // pixels

    for (int s = 0; s < numSamples; ++s) {
        float t = float(s) / (numSamples - 1);   // 0 (near) → 1 (far)
        float r = 10.0f + t * (maxRadius - 10.0f);  // skip first few pixels
        float px = originX + r * std::sin(angleRad);
        float py = originY + r * std::cos(angleRad);

        int ix = std::clamp(static_cast<int>(px), 0, kWidth  - 1);
        int iy = std::clamp(static_cast<int>(py), 0, kHeight - 1);

        float phantomVal = m_phantom[iy * kWidth + ix];

        // Depth-dependent attenuation
        float attenuation = std::exp(-t * 1.8f);

        // Focus effect: boost around focus depth
        float focusDepthNorm = m_params->focusDepth() / 300.0f;
        float focusBoost = 1.0f + 0.4f * std::exp(-std::pow((t - focusDepthNorm) / 0.08f, 2.0f));

        float speckle = m_speckle[iy * kWidth + ix];
        // Temporal variation to make it look "live"
        float temporal = 0.015f * std::sin(t * 20.0f + m_frameCounter * 0.3f + angleRad * 8.0f);

        float raw = phantomVal * attenuation * focusBoost + speckle + temporal;
        raw = std::clamp(raw, 0.0f, 1.0f);

        // Log compression (simulating ultrasound dynamic range compression)
        float drNorm = m_params->dynamicRange() / 100.0f; // 0.3 – 1.0
        float compressed = std::log10(1.0f + raw * 9.0f * drNorm) / std::log10(1.0f + 9.0f * drNorm);

        samples[s] = static_cast<uint8_t>(compressed * 255.0f);
    }
    return samples;
}

uint8_t FrameGenerator::applyGainTgc(uint8_t raw, int y) const
{
    float tgcCurve[8];
    {
        auto curve = m_params->tgcCurve();
        std::copy(curve.begin(), curve.end(), tgcCurve);
    }
    float globalGain = m_params->gain() / 50.0f; // 0 – 2.0

    // Interpolate TGC for this row
    float zone = std::clamp(float(y) / kHeight * 8.0f, 0.0f, 7.999f);
    int idx0 = static_cast<int>(zone);
    int idx1 = std::min(idx0 + 1, 7);
    float frac = zone - idx0;
    float tgcDb = tgcCurve[idx0] * (1.0f - frac) + tgcCurve[idx1] * frac;

    float linear = raw / 255.0f;
    float gained = linear * globalGain * std::pow(10.0f, tgcDb / 20.0f);
    gained = std::clamp(gained, 0.0f, 1.0f);
    return static_cast<uint8_t>(gained * 255.0f);
}

void FrameGenerator::renderFrame()
{
    ++m_frameCounter;

    const int numScanlines = 128;     // angular resolution
    const int samplesPerLine = 512;   // radial samples
    const float sectorAngle = float(std::numbers::pi) / 6.0f;  // ±30°

    // Temporary accumulation buffer (float, to average overlapping scanlines)
    std::vector<float> accum(kWidth * kHeight, 0.0f);
    std::vector<int>   count(kWidth * kHeight, 0);

    for (int sl = 0; sl < numScanlines; ++sl) {
        float t = sl / float(numScanlines - 1);            // 0 → 1
        float angle = (t - 0.5f) * sectorAngle;            // -30° → +30°

        auto samples = generateScanline(angle, samplesPerLine);

        const float originX = kWidth * 0.5f;
        const float originY = 0.0f;
        const float maxRadius = kHeight;

        for (int s = 0; s < samplesPerLine; ++s) {
            float frac = s / float(samplesPerLine - 1);
            float r = 10.0f + frac * (maxRadius - 10.0f);
            float px = originX + r * std::sin(angle);
            float py = originY + r * std::cos(angle);

            int ix = std::clamp(static_cast<int>(px), 0, kWidth  - 1);
            int iy = std::clamp(static_cast<int>(py), 0, kHeight - 1);

            accum[iy * kWidth + ix] += samples[s];
            count[iy * kWidth + ix]++;
        }
    }

    // Normalize and apply gain/TGC
    {
        QMutexLocker lock(&m_mutex);
        for (int y = 0; y < kHeight; ++y) {
            auto *scanline = m_buffer.scanLine(y);
            for (int x = 0; x < kWidth; ++x) {
                int idx = y * kWidth + x;
                if (count[idx] > 0) {
                    uint8_t raw = static_cast<uint8_t>(
                        std::clamp(accum[idx] / count[idx], 0.0f, 255.0f));
                    scanline[x] = applyGainTgc(raw, y);
                } else {
                    scanline[x] = 0;
                }
            }
        }

        // Draw sector overlay (optional depth markers)
        // Depth scale markers on the right side
        for (int d = 50; d <= 300; d += 50) {
            int y = d * kHeight / 300;
            if (y >= 0 && y < kHeight) {
                for (int x = kWidth - 30; x < kWidth - 20; ++x)
                    m_buffer.scanLine(y)[x] = 200;
            }
        }
    }
}
