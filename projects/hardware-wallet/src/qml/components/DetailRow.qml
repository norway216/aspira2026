import QtQuick 2.15
import ".."

Rectangle {
    property string label: ""
    property string value: ""
    property bool mono: false

    width: parent ? parent.width : 300; height: 44
    color: Theme.bgCard; radius: Theme.radiusSM
    Row {
        anchors.fill: parent; anchors.margins: 12
        Text {
            text: label; color: Theme.textSecondary
            font.pixelSize: Theme.fontSizeSM; width: 80
            anchors.verticalCenter: parent.verticalCenter
        }
        Text {
            text: value; color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeSM
            font.family: mono ? "monospace" : ""
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width - 92
            elide: Text.ElideMiddle; wrapMode: Text.WrapAnywhere
        }
    }
}
