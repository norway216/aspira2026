#pragma once

#include <QObject>
#include <QAtomicInt>
#include <QMutex>
#include <array>
#include <memory>

/// Central parameter store shared between C++ backend and QML UI.
/// All setters emit changed() so QML property bindings react automatically.
class ParamManager : public QObject
{
    Q_OBJECT

    // ---- B-mode imaging parameters ----
    Q_PROPERTY(int    gain         READ gain         WRITE setGain         NOTIFY gainChanged)
    Q_PROPERTY(int    depth        READ depth        WRITE setDepth        NOTIFY depthChanged)
    Q_PROPERTY(int    frequency    READ frequency    WRITE setFrequency    NOTIFY frequencyChanged)
    Q_PROPERTY(int    dynamicRange READ dynamicRange WRITE setDynamicRange NOTIFY dynamicRangeChanged)
    Q_PROPERTY(int    focusDepth   READ focusDepth   WRITE setFocusDepth   NOTIFY focusDepthChanged)
    Q_PROPERTY(int    frameRate    READ frameRate    WRITE setFrameRate    NOTIFY frameRateChanged)
    Q_PROPERTY(bool   freeze       READ freeze       WRITE setFreeze       NOTIFY freezeChanged)
    Q_PROPERTY(QString mode       READ mode         WRITE setMode         NOTIFY modeChanged)

    // ---- TGC sliders (8 points) ----
    Q_PROPERTY(int tgc0 READ tgc0 WRITE setTgc0 NOTIFY tgcChanged)
    Q_PROPERTY(int tgc1 READ tgc1 WRITE setTgc1 NOTIFY tgcChanged)
    Q_PROPERTY(int tgc2 READ tgc2 WRITE setTgc2 NOTIFY tgcChanged)
    Q_PROPERTY(int tgc3 READ tgc3 WRITE setTgc3 NOTIFY tgcChanged)
    Q_PROPERTY(int tgc4 READ tgc4 WRITE setTgc4 NOTIFY tgcChanged)
    Q_PROPERTY(int tgc5 READ tgc5 WRITE setTgc5 NOTIFY tgcChanged)
    Q_PROPERTY(int tgc6 READ tgc6 WRITE setTgc6 NOTIFY tgcChanged)
    Q_PROPERTY(int tgc7 READ tgc7 WRITE setTgc7 NOTIFY tgcChanged)

public:
    explicit ParamManager(QObject *parent = nullptr);

    // Getters
    int    gain()         const;
    int    depth()        const;
    int    frequency()    const;
    int    dynamicRange() const;
    int    focusDepth()   const;
    int    frameRate()    const;
    bool   freeze()       const;
    QString mode()        const;

    int tgc0() const; int tgc1() const; int tgc2() const; int tgc3() const;
    int tgc4() const; int tgc5() const; int tgc6() const; int tgc7() const;

    /// Return TGC curve as an 8-element array (dB gain per depth zone).
    std::array<float, 8> tgcCurve() const;

    /// Convenience: write all parameters to a JSON object.
    QJsonObject toJson() const;
    /// Convenience: load all parameters from a JSON object.
    void fromJson(const QJsonObject &obj);

public slots:
    // Setters
    void setGain(int v);
    void setDepth(int v);
    void setFrequency(int v);
    void setDynamicRange(int v);
    void setFocusDepth(int v);
    void setFrameRate(int v);
    void setFreeze(bool v);
    void setMode(const QString &v);

    void setTgc0(int v); void setTgc1(int v); void setTgc2(int v); void setTgc3(int v);
    void setTgc4(int v); void setTgc5(int v); void setTgc6(int v); void setTgc7(int v);

    /// Reset all parameters to defaults.
    Q_INVOKABLE void resetDefaults();

signals:
    void gainChanged();
    void depthChanged();
    void frequencyChanged();
    void dynamicRangeChanged();
    void focusDepthChanged();
    void frameRateChanged();
    void freezeChanged();
    void modeChanged();
    void tgcChanged();
    /// Emitted whenever any parameter changes (for generic listeners).
    void paramsChanged();

private:
    template <typename T>
    bool setIfChanged(T &member, T value) {
        if (member == value) return false;
        member = value;
        return true;
    }

    mutable QMutex m_mutex;

    int    m_gain         = 50;   // 0-100
    int    m_depth        = 100;  // mm, 20-300
    int    m_frequency    = 5;    // MHz, 1-15
    int    m_dynamicRange = 60;   // dB, 30-100
    int    m_focusDepth   = 50;   // mm, 10-200
    int    m_frameRate    = 30;   // fps, 1-60
    bool   m_freeze       = false;
    QString m_mode        = QStringLiteral("B");  // B, M, Doppler

    std::array<int, 8> m_tgc = {50, 50, 50, 50, 50, 50, 50, 50}; // 0-100
};
