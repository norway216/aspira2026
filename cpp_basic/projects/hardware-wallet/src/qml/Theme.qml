pragma Singleton
import QtQuick 2.15

QtObject {
    // Primary palette
    readonly property color primary: "#1A73E8"
    readonly property color primaryDark: "#1557B0"
    readonly property color primaryLight: "#4A90D9"

    // Accent
    readonly property color accent: "#00C853"
    readonly property color accentWarning: "#FF9800"
    readonly property color accentDanger: "#F44336"

    // Background
    readonly property color bgDark: "#121212"
    readonly property color bgCard: "#1E1E2E"
    readonly property color bgSurface: "#252538"
    readonly property color bgInput: "#2A2A3C"

    // Text
    readonly property color textPrimary: "#FFFFFF"
    readonly property color textSecondary: "#B0B0C0"
    readonly property color textMuted: "#707088"

    // Transaction colors
    readonly property color txSent: "#F44336"
    readonly property color txReceived: "#00C853"

    // Spacing
    readonly property int spacingXS: 4
    readonly property int spacingSM: 8
    readonly property int spacingMD: 16
    readonly property int spacingLG: 24
    readonly property int spacingXL: 32

    // Font sizes
    readonly property int fontSizeXS: 10
    readonly property int fontSizeSM: 12
    readonly property int fontSizeMD: 14
    readonly property int fontSizeLG: 18
    readonly property int fontSizeXL: 24
    readonly property int fontSizeXXL: 32

    // Radii
    readonly property int radiusSM: 4
    readonly property int radiusMD: 8
    readonly property int radiusLG: 12
    readonly property int radiusXL: 20

    // Sizes
    readonly property int buttonHeight: 48
    readonly property int inputHeight: 52
    readonly property int iconSize: 24
}
