import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Rectangle {
    id: txRow
    property string type: "sent"  // "sent" or "received"
    property double amount: 0.0
    property string address: ""
    property string timestamp: ""
    property string status: "confirmed"

    width: parent ? parent.width : 300
    height: 64
    color: Theme.bgCard
    radius: Theme.radiusSM

    Row {
        anchors.fill: parent
        anchors.margins: 12
        spacing: 12

        Rectangle {
            width: 40; height: 40
            radius: 20
            color: type === "received" ? Theme.txReceived : Theme.txSent
            anchors.verticalCenter: parent.verticalCenter
            Text {
                anchors.centerIn: parent
                text: type === "received" ? "↓" : "↑"
                color: "white"
                font.pixelSize: 20
                font.bold: true
            }
        }

        Column {
            anchors.verticalCenter: parent.verticalCenter
            width: parent.width - 64
            spacing: 2

            Text {
                text: type === "received" ? "Received" : "Sent"
                color: Theme.textPrimary
                font.pixelSize: Theme.fontSizeMD
                font.bold: true
            }

            Text {
                text: address
                color: Theme.textMuted
                font.pixelSize: Theme.fontSizeXS
                font.family: "monospace"
                elide: Text.ElideMiddle
                width: parent.width
            }

            Text {
                text: timestamp
                color: Theme.textMuted
                font.pixelSize: Theme.fontSizeXS
            }
        }
    }
}
