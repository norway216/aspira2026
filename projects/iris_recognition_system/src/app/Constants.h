#pragma once

#include <cstddef>
#include <cstdint>
#include <string>

namespace iris {

// ── Version ────────────────────────────────────────────────────
inline constexpr const char* APP_VERSION   = "1.0.0";
inline constexpr const char* APP_NAME      = "Iris Recognition System";

// ── Pipeline dimensions ────────────────────────────────────────
inline constexpr int NORMALIZED_WIDTH      = 512;
inline constexpr int NORMALIZED_HEIGHT     = 64;
inline constexpr int IRIS_CODE_BYTES       = 256;   // 2048 bits
inline constexpr int EMBEDDING_DIM         = 256;
inline constexpr int DEFAULT_CAMERA_WIDTH  = 640;
inline constexpr int DEFAULT_CAMERA_HEIGHT = 480;
inline constexpr int DEFAULT_CAMERA_FPS    = 30;

// ── Eye ROI ────────────────────────────────────────────────────
inline constexpr int EYE_ROI_SIZE          = 256;
inline constexpr int EYE_ROI_MARGIN        = 40;

// ── Gabor filter defaults ──────────────────────────────────────
inline constexpr int GABOR_SCALES          = 4;
inline constexpr int GABOR_ORIENTATIONS    = 8;

// ── Thresholds ─────────────────────────────────────────────────
inline constexpr float DEFAULT_QUALITY_THRESHOLD   = 0.35f;
inline constexpr float DEFAULT_LIVENESS_THRESHOLD  = 0.55f;
inline constexpr float DEFAULT_MATCHING_THRESHOLD  = 0.35f;  // Hamming distance (lower = better)

// ── Enrollment ─────────────────────────────────────────────────
inline constexpr int ENROLLMENT_FRAMES_REQUIRED = 3;

// ── Database ───────────────────────────────────────────────────
inline constexpr const char* DEFAULT_DB_PATH = "iris_data.db";

} // namespace iris
