import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Page {
    id: receivePage
    anchors.fill: parent
    background: Rectangle { color: Theme.bgDark }

    onVisibleChanged: { if (visible) ReceiveViewModel.refresh() }

    Column {
        anchors.centerIn: parent
        spacing: Theme.spacingLG
        width: parent.width - 60

        Row {
            width: parent.width
            Text {
                text: "← Back"
                font.pixelSize: Theme.fontSizeMD
                color: Theme.primary
                MouseArea {
                    anchors.fill: parent
                    cursorShape: Qt.PointingHandCursor
                    onClicked: stackView.pop()
                }
            }
        }

        Text {
            text: "Receive"
            font.pixelSize: Theme.fontSizeXL
            color: Theme.textPrimary
            font.bold: true
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Rectangle {
            width: 200; height: 200
            color: Theme.bgSurface
            radius: Theme.radiusLG
            anchors.horizontalCenter: parent.horizontalCenter
            Text {
                anchors.centerIn: parent
                text: "QR Code"
                color: Theme.textMuted
                font.pixelSize: Theme.fontSizeMD
            }
        }

        Text {
            text: "Your Public Key:"
            color: Theme.textSecondary
            font.pixelSize: Theme.fontSizeSM
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Rectangle {
            width: parent.width
            height: 80
            color: Theme.bgCard
            radius: Theme.radiusMD
            anchors.horizontalCenter: parent.horizontalCenter

            Text {
                anchors.centerIn: parent
                text: ReceiveViewModel.publicKey || "Loading..."
                color: Theme.textPrimary
                font.pixelSize: Theme.fontSizeXS
                font.family: "monospace"
                wrapMode: Text.WrapAnywhere
                width: parent.width - 20
                horizontalAlignment: Text.AlignHCenter
            }
        }

        Button {
            text: "Copy Public Key"
            width: parent.width
            height: Theme.buttonHeight
            background: Rectangle { color: Theme.primary; radius: Theme.radiusMD }
            contentItem: Text {
                text: "Copy Public Key"
                color: Theme.textPrimary
                font.pixelSize: Theme.fontSizeLG
                horizontalAlignment: Text.AlignHCenter
                verticalAlignment: Text.AlignVCenter
            }
            onClicked: {
                // Clipboard would be used in real implementation
                console.log("Copy:", ReceiveViewModel.publicKey)
            }
        }
    }
}
