import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Column {
    id: pinInput
    property string pin: ""
    property int maxLength: 6
    width: parent.width
    spacing: 16

    // PIN dots display
    Row {
        anchors.horizontalCenter: parent.horizontalCenter
        spacing: 16
        Repeater {
            model: maxLength
            Rectangle {
                width: 16; height: 16
                radius: 8
                color: index < pin.length ? Theme.primary : Theme.bgSurface
                border.color: Theme.textMuted
            }
        }
    }

    // Number pad
    Grid {
        anchors.horizontalCenter: parent.horizontalCenter
        columns: 3
        spacing: 8

        Repeater {
            model: ["1","2","3","4","5","6","7","8","9","","0","⌫"]
            Rectangle {
                width: 72; height: 56
                visible: modelData !== ""
                radius: 8
                color: pinPadMouse.pressed ? Theme.primaryDark : Theme.bgSurface
                Text {
                    anchors.centerIn: parent
                    text: modelData
                    color: Theme.textPrimary
                    font.pixelSize: 24
                }
                MouseArea {
                    id: pinPadMouse
                    anchors.fill: parent
                    onClicked: {
                        if (modelData === "⌫") {
                            pin = pin.substring(0, pin.length - 1)
                        } else if (pin.length < maxLength) {
                            pin += modelData
                        }
                    }
                }
            }
        }
    }
}
