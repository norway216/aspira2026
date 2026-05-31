import QtQuick 2.15

import ".."
Rectangle {
    id: balanceCard
    property double balance: 0.0
    property string address: ""

    width: parent.width
    height: 120
    color: Theme.bgCard
    radius: Theme.radiusLG

    Column {
        anchors.centerIn: parent
        spacing: 8

        Text {
            text: "Balance"
            color: Theme.textSecondary
            font.pixelSize: Theme.fontSizeSM
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Text {
            text: balance.toFixed(8) + " SIM"
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeXXL
            font.bold: true
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Text {
            text: address.length > 40 ? address.substring(0, 20) + "..." + address.substring(address.length - 10) : address
            color: Theme.textMuted
            font.pixelSize: Theme.fontSizeXS
            anchors.horizontalCenter: parent.horizontalCenter
            font.family: "monospace"
        }
    }
}
