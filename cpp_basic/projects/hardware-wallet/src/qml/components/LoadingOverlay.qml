import QtQuick 2.15
import QtQuick.Controls 2.15

Rectangle {
    id: loadingOverlay
    property string message: "Loading..."
    property bool visible: false

    anchors.fill: parent
    color: "#80000000"
    z: 1000
    visible: false

    Column {
        anchors.centerIn: parent
        spacing: 20

        BusyIndicator {
            anchors.horizontalCenter: parent.horizontalCenter
            running: loadingOverlay.visible
        }

        Text {
            text: message
            color: "white"
            font.pixelSize: 16
            anchors.horizontalCenter: parent.horizontalCenter
        }
    }

    MouseArea {
        anchors.fill: parent
        // Block clicks through overlay
    }
}
