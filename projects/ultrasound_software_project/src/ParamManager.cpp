#include "ParamManager.h"
#include <QJsonObject>
#include <QMutexLocker>

ParamManager::ParamManager(QObject *parent)
    : QObject(parent)
{
}

// ---------------------------------------------------------------------------
// Getters
// ---------------------------------------------------------------------------
#define DEF_GETTER(Type, Name, Member) \
    Type ParamManager::Name() const { QMutexLocker lock(&m_mutex); return m_##Member; }

DEF_GETTER(int,    gain,         gain)
DEF_GETTER(int,    depth,        depth)
DEF_GETTER(int,    frequency,    frequency)
DEF_GETTER(int,    dynamicRange, dynamicRange)
DEF_GETTER(int,    focusDepth,   focusDepth)
DEF_GETTER(int,    frameRate,    frameRate)
DEF_GETTER(bool,   freeze,       freeze)
DEF_GETTER(QString, mode,        mode)

#define DEF_TGC_GETTER(i) \
    int ParamManager::tgc##i() const { QMutexLocker lock(&m_mutex); return m_tgc[i]; }

DEF_TGC_GETTER(0) DEF_TGC_GETTER(1) DEF_TGC_GETTER(2) DEF_TGC_GETTER(3)
DEF_TGC_GETTER(4) DEF_TGC_GETTER(5) DEF_TGC_GETTER(6) DEF_TGC_GETTER(7)

#undef DEF_GETTER
#undef DEF_TGC_GETTER

// ---------------------------------------------------------------------------
// Setters
// ---------------------------------------------------------------------------
#define DEF_SETTER(Type, Name, Member, Signal)                                  \
    void ParamManager::set##Name(Type v) {                                      \
        bool changed = false;                                                   \
        { QMutexLocker lock(&m_mutex); changed = setIfChanged(m_##Member, v); } \
        if (changed) { emit Signal(); emit paramsChanged(); }                   \
    }

DEF_SETTER(int,    Gain,         gain,         gainChanged)
DEF_SETTER(int,    Depth,        depth,        depthChanged)
DEF_SETTER(int,    Frequency,    frequency,    frequencyChanged)
DEF_SETTER(int,    DynamicRange, dynamicRange, dynamicRangeChanged)
DEF_SETTER(int,    FocusDepth,   focusDepth,   focusDepthChanged)
DEF_SETTER(int,    FrameRate,    frameRate,    frameRateChanged)
DEF_SETTER(bool,   Freeze,       freeze,       freezeChanged)

// QString setter needs const-ref to match header signature
void ParamManager::setMode(const QString &v)
{
    bool changed = false;
    { QMutexLocker lock(&m_mutex); changed = setIfChanged(m_mode, v); }
    if (changed) { emit modeChanged(); emit paramsChanged(); }
}

#undef DEF_SETTER

#define DEF_TGC_SETTER(i)                                                       \
    void ParamManager::setTgc##i(int v) {                                       \
        bool changed = false;                                                   \
        { QMutexLocker lock(&m_mutex); changed = setIfChanged(m_tgc[i], v); }   \
        if (changed) { emit tgcChanged(); emit paramsChanged(); }               \
    }

DEF_TGC_SETTER(0) DEF_TGC_SETTER(1) DEF_TGC_SETTER(2) DEF_TGC_SETTER(3)
DEF_TGC_SETTER(4) DEF_TGC_SETTER(5) DEF_TGC_SETTER(6) DEF_TGC_SETTER(7)

#undef DEF_TGC_SETTER

// ---------------------------------------------------------------------------
std::array<float, 8> ParamManager::tgcCurve() const
{
    QMutexLocker lock(&m_mutex);
    std::array<float, 8> curve{};
    for (int i = 0; i < 8; ++i)
        curve[i] = m_tgc[i] / 100.0f * 60.0f;  // map 0-100 → 0-60 dB
    return curve;
}

void ParamManager::resetDefaults()
{
    setGain(50);
    setDepth(100);
    setFrequency(5);
    setDynamicRange(60);
    setFocusDepth(50);
    setFrameRate(30);
    setFreeze(false);
    setMode(QStringLiteral("B"));
    for (int i = 0; i < 8; ++i) {
        // Need to call each setter; use a small helper
        switch (i) {
            case 0: setTgc0(50); break; case 1: setTgc1(50); break;
            case 2: setTgc2(50); break; case 3: setTgc3(50); break;
            case 4: setTgc4(50); break; case 5: setTgc5(50); break;
            case 6: setTgc6(50); break; case 7: setTgc7(50); break;
        }
    }
}

QJsonObject ParamManager::toJson() const
{
    QMutexLocker lock(&m_mutex);
    QJsonObject obj;
    obj["gain"]         = m_gain;
    obj["depth"]        = m_depth;
    obj["frequency"]    = m_frequency;
    obj["dynamicRange"] = m_dynamicRange;
    obj["focusDepth"]   = m_focusDepth;
    obj["frameRate"]    = m_frameRate;
    obj["freeze"]       = m_freeze;
    obj["mode"]         = m_mode;

    QJsonObject tgcObj;
    for (int i = 0; i < 8; ++i)
        tgcObj[QString::number(i)] = m_tgc[i];
    obj["tgc"] = tgcObj;
    return obj;
}

void ParamManager::fromJson(const QJsonObject &obj)
{
    setGain(obj.value("gain").toInt(50));
    setDepth(obj.value("depth").toInt(100));
    setFrequency(obj.value("frequency").toInt(5));
    setDynamicRange(obj.value("dynamicRange").toInt(60));
    setFocusDepth(obj.value("focusDepth").toInt(50));
    setFrameRate(obj.value("frameRate").toInt(30));
    setFreeze(obj.value("freeze").toBool(false));
    setMode(obj.value("mode").toString(QStringLiteral("B")));

    QJsonObject tgcObj = obj.value("tgc").toObject();
    for (int i = 0; i < 8; ++i) {
        int v = tgcObj.value(QString::number(i)).toInt(50);
        switch (i) {
            case 0: setTgc0(v); break; case 1: setTgc1(v); break;
            case 2: setTgc2(v); break; case 3: setTgc3(v); break;
            case 4: setTgc4(v); break; case 5: setTgc5(v); break;
            case 6: setTgc6(v); break; case 7: setTgc7(v); break;
        }
    }
}
