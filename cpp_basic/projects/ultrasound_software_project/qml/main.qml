import QtQuick 2.15
import QtQuick.Controls 2.15
import QtQuick.Layouts 1.15

ApplicationWindow {
    id: mainWindow
    visible: true
    width: 1200
    height: 720
    title: "Ultrasound Scanner"
    color: "#1a1a2e"

    // ---- Menu bar ----
    menuBar: MenuBar {
        Menu {
            title: "&File"
            MenuItem {
                text: "&Load Config..."
                onTriggered: configManager.loadConfig(configManager.defaultConfigPath())
            }
            MenuItem {
                text: "&Save Config"
                onTriggered: configManager.saveConfig(configManager.defaultConfigPath())
            }
            MenuSeparator {}
            MenuItem {
                text: "&Quit (Ctrl+Q)"
                onTriggered: Qt.quit()
            }
        }
        Menu {
            title: "&View"
            MenuItem {
                text: "&Reset Parameters"
                onTriggered: paramManager.resetDefaults()
            }
        }
    }

    // ---- Toolbar ----
    header: ToolBar {
        id: toolbar
        background: Rectangle { color: "#16213e" }
        RowLayout {
            anchors.fill: parent
            spacing: 8

            // Freeze button
            ToolButton {
                text: paramManager.freeze ? "▶ Run" : "⏸ Freeze"
                font.pixelSize: 13
                onClicked: appCore.toggleFreeze()
                background: Rectangle {
                    color: paramManager.freeze ? "#e94560" : "#0f3460"
                    radius: 4
                }
                contentItem: Text {
                    text: parent.text
                    color: "white"
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                }
            }

            // Mode selector
            ComboBox {
                id: modeCombo
                model: ["B", "M", "Doppler"]
                currentIndex: {
                    if (paramManager.mode === "M") return 1;
                    if (paramManager.mode === "Doppler") return 2;
                    return 0;
                }
                onCurrentTextChanged: paramManager.mode = currentText
                background: Rectangle {
                    color: "#0f3460"
                    radius: 4
                    border.color: "#533483"
                }
                contentItem: Text {
                    text: modeCombo.currentText
                    color: "white"
                    verticalAlignment: Text.AlignVCenter
                }
            }

            // Status label
            Text {
                text: appCore.statusText
                color: "#a0a0b0"
                font.pixelSize: 13
                Layout.fillWidth: true
            }

            // Frame counter
            Text {
                text: "Frames: " + appCore.frameCount
                color: "#a0a0b0"
                font.pixelSize: 13
            }

            // Frequency display
            Text {
                text: "Freq: " + paramManager.frequency + " MHz"
                color: "#a0a0b0"
                font.pixelSize: 13
            }
        }
    }

    // ---- Main body: left panel | image | right panel ----
    RowLayout {
        anchors.fill: parent
        anchors.topMargin: 4
        spacing: 0

        // ===== LEFT PANEL =====
        Rectangle {
            Layout.preferredWidth: 200
            Layout.fillHeight: true
            color: "#16213e"

            ColumnLayout {
                anchors.fill: parent
                anchors.margins: 8
                spacing: 6

                Text {
                    text: "TGC Controls"
                    color: "#e0e0e0"
                    font.bold: true
                    font.pixelSize: 14
                }

                // 8 TGC sliders
                Column {
                    Layout.fillWidth: true
                    spacing: 2
                    Repeater {
                        model: 8
                        RowLayout {
                            spacing: 4
                            Text {
                                text: "TGC" + index
                                color: "#9090a0"
                                font.pixelSize: 11
                                Layout.preferredWidth: 32
                            }
                            Slider {
                                id: tgcSlider
                                Layout.fillWidth: true
                                from: 0; to: 100
                                value: {
                                    switch (index) {
                                        case 0: return paramManager.tgc0
                                        case 1: return paramManager.tgc1
                                        case 2: return paramManager.tgc2
                                        case 3: return paramManager.tgc3
                                        case 4: return paramManager.tgc4
                                        case 5: return paramManager.tgc5
                                        case 6: return paramManager.tgc6
                                        case 7: return paramManager.tgc7
                                    }
                                    return 50
                                }
                                onValueChanged: {
                                    switch (index) {
                                        case 0: paramManager.tgc0 = value; break
                                        case 1: paramManager.tgc1 = value; break
                                        case 2: paramManager.tgc2 = value; break
                                        case 3: paramManager.tgc3 = value; break
                                        case 4: paramManager.tgc4 = value; break
                                        case 5: paramManager.tgc5 = value; break
                                        case 6: paramManager.tgc6 = value; break
                                        case 7: paramManager.tgc7 = value; break
                                    }
                                }
                                background: Rectangle {
                                    x: tgcSlider.leftPadding
                                    y: tgcSlider.topPadding + tgcSlider.availableHeight / 2 - height / 2
                                    width: tgcSlider.availableWidth
                                    height: 4
                                    radius: 2
                                    color: "#1a1a3e"
                                    Rectangle {
                                        width: tgcSlider.visualPosition * parent.width
                                        height: parent.height
                                        color: "#533483"
                                        radius: 2
                                    }
                                }
                                handle: Rectangle {
                                    x: tgcSlider.leftPadding + tgcSlider.visualPosition * (tgcSlider.availableWidth - width)
                                    y: tgcSlider.topPadding + tgcSlider.availableHeight / 2 - height / 2
                                    width: 14; height: 14
                                    radius: 7
                                    color: "#e94560"
                                }
                            }
                        }
                    }
                }

                // Depth
                Text { text: "Depth: " + paramManager.depth + " mm"; color: "#c0c0d0"; font.pixelSize: 12 }
                Slider {
                    Layout.fillWidth: true
                    from: 20; to: 300
                    value: paramManager.depth
                    onValueChanged: paramManager.depth = value
                }

                // Focus depth
                Text { text: "Focus: " + paramManager.focusDepth + " mm"; color: "#c0c0d0"; font.pixelSize: 12 }
                Slider {
                    Layout.fillWidth: true
                    from: 10; to: 200
                    value: paramManager.focusDepth
                    onValueChanged: paramManager.focusDepth = value
                }

                Item { Layout.fillHeight: true }
            }
        }

        // ===== CENTER: IMAGE DISPLAY =====
        Rectangle {
            Layout.fillWidth: true
            Layout.fillHeight: true
            color: "#000000"
            border.color: "#333355"

            // The ultrasound image
            Image {
                id: ultrasoundImage
                anchors.centerIn: parent
                width: Math.min(parent.width, parent.height) * 0.95
                height: width
                fillMode: Image.PreserveAspectFit
                cache: false
                // Use the C++ image provider
                source: "image://ultrasound/frame"

                // Refresh the image on each frame
                property int refreshCounter: 0
                Timer {
                    interval: 33  // ~30 fps refresh
                    running: true
                    repeat: true
                    onTriggered: {
                        ultrasoundImage.refreshCounter++
                        // Force reload by alternating source
                        ultrasoundImage.source = ""
                        ultrasoundImage.source = "image://ultrasound/frame"
                    }
                }
            }

            // Depth scale overlay
            Column {
                anchors.right: parent.right
                anchors.rightMargin: 25
                anchors.verticalCenter: parent.verticalCenter
                spacing: 0
                Repeater {
                    model: [
                        { label: "0 cm", pos: 0.05 },
                        { label: "5 cm", pos: 0.25 },
                        { label: "10 cm", pos: 0.45 },
                        { label: "15 cm", pos: 0.65 },
                        { label: "20 cm", pos: 0.85 }
                    ]
                    Text {
                        text: modelData.label
                        color: "#888888"
                        font.pixelSize: 10
                        y: ultrasoundImage.height * modelData.pos - 6
                    }
                }
            }

            // Sector outline (decorative)
            Canvas {
                anchors.fill: parent
                onPaint: {
                    var ctx = getContext("2d")
                    ctx.strokeStyle = "#334466"
                    ctx.lineWidth = 1
                    ctx.beginPath()
                    var cx = width / 2
                    var topY = height * 0.08
                    var r = height * 0.75
                    var angle = Math.PI / 6
                    ctx.arc(cx, topY, r, -Math.PI/2 - angle, -Math.PI/2 + angle)
                    ctx.stroke()
                }
            }
        }

        // ===== RIGHT PANEL =====
        Rectangle {
            Layout.preferredWidth: 200
            Layout.fillHeight: true
            color: "#16213e"

            ColumnLayout {
                anchors.fill: parent
                anchors.margins: 8
                spacing: 8

                Text {
                    text: "Image Controls"
                    color: "#e0e0e0"
                    font.bold: true
                    font.pixelSize: 14
                }

                // Gain
                Text { text: "Gain: " + paramManager.gain; color: "#c0c0d0"; font.pixelSize: 12 }
                Slider {
                    Layout.fillWidth: true
                    from: 0; to: 100
                    value: paramManager.gain
                    onValueChanged: paramManager.gain = value
                    background: Rectangle {
                        x: parent.leftPadding; y: parent.topPadding + parent.availableHeight / 2 - 2
                        width: parent.availableWidth; height: 4; radius: 2; color: "#1a1a3e"
                        Rectangle {
                            width: parent.visualPosition * parent.width
                            height: parent.height; color: "#0f3460"; radius: 2
                        }
                    }
                }

                // Dynamic Range
                Text { text: "DR: " + paramManager.dynamicRange + " dB"; color: "#c0c0d0"; font.pixelSize: 12 }
                Slider {
                    Layout.fillWidth: true
                    from: 30; to: 100
                    value: paramManager.dynamicRange
                    onValueChanged: paramManager.dynamicRange = value
                }

                // Frequency
                Text { text: "Freq: " + paramManager.frequency + " MHz"; color: "#c0c0d0"; font.pixelSize: 12 }
                Slider {
                    Layout.fillWidth: true
                    from: 1; to: 15
                    stepSize: 1
                    value: paramManager.frequency
                    onValueChanged: paramManager.frequency = value
                }

                // Frame Rate
                Text { text: "FPS: " + paramManager.frameRate; color: "#c0c0d0"; font.pixelSize: 12 }
                Slider {
                    Layout.fillWidth: true
                    from: 1; to: 60
                    stepSize: 1
                    value: paramManager.frameRate
                    onValueChanged: paramManager.frameRate = value
                }

                Rectangle { Layout.fillHeight: true; color: "transparent" }

                // Preset buttons
                Button {
                    text: "Save Preset"
                    Layout.fillWidth: true
                    onClicked: appCore.savePreset("default")
                    background: Rectangle { color: "#0f3460"; radius: 4 }
                    contentItem: Text {
                        text: parent.text; color: "white"
                        horizontalAlignment: Text.AlignHCenter
                        verticalAlignment: Text.AlignVCenter
                    }
                }

                Button {
                    text: "Load Preset"
                    Layout.fillWidth: true
                    onClicked: appCore.loadPreset("default")
                    background: Rectangle { color: "#0f3460"; radius: 4 }
                    contentItem: Text {
                        text: parent.text; color: "white"
                        horizontalAlignment: Text.AlignHCenter
                        verticalAlignment: Text.AlignVCenter
                    }
                }

                Button {
                    text: "Reset All"
                    Layout.fillWidth: true
                    onClicked: paramManager.resetDefaults()
                    background: Rectangle { color: "#533483"; radius: 4 }
                    contentItem: Text {
                        text: parent.text; color: "white"
                        horizontalAlignment: Text.AlignHCenter
                        verticalAlignment: Text.AlignVCenter
                    }
                }
            }
        }
    }

    // Footer status bar
    footer: Rectangle {
        height: 24
        color: "#0a0a1a"
        RowLayout {
            anchors.fill: parent
            anchors.leftMargin: 8
            Text { text: "Threads: " + threadPool.activeThreadCount; color: "#606070"; font.pixelSize: 11 }
            Text { text: "  |  Mode: " + paramManager.mode; color: "#606070"; font.pixelSize: 11 }
            Text { text: "  |  " + appCore.statusText; color: "#606070"; font.pixelSize: 11 }
        }
    }
}
