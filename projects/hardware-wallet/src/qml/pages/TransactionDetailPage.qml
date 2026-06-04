import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."
import "../components"

Page {
    id: txDetailPage
    anchors.fill: parent
    background: Rectangle { color: Theme.bgDark }

    property string txType: "sent"
    property double txAmount: 0.0
    property string txAddress: ""
    property string txTimestamp: ""
    property string txStatus: ""
    property string txNonce: ""
    property string txId: ""

    Flickable {
        anchors.fill: parent
        contentHeight: detailColumn.height + 40
        clip: true

        Column {
            id: detailColumn
            width: parent.width - 32
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.top: parent.top
            anchors.topMargin: Theme.spacingMD
            spacing: Theme.spacingMD

            Text {
                text: "← Back"
                font.pixelSize: Theme.fontSizeMD; color: Theme.primary
                MouseArea {
                    anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                    onClicked: stackView.pop()
                }
            }

            Text {
                text: "Transaction Details"
                font.pixelSize: Theme.fontSizeXL; color: Theme.textPrimary; font.bold: true
            }

            Rectangle {
                width: 120; height: 32; radius: 16
                color: txStatus === "Confirmed" ? Theme.accent : Theme.accentWarning
                anchors.horizontalCenter: parent.horizontalCenter
                Text {
                    anchors.centerIn: parent; text: txStatus
                    color: "white"; font.pixelSize: Theme.fontSizeSM; font.bold: true
                }
            }

            Rectangle {
                width: parent.width; height: 80
                color: Theme.bgCard; radius: Theme.radiusMD
                Column {
                    anchors.centerIn: parent; spacing: 4
                    Text {
                        text: "Amount"
                        color: Theme.textSecondary; font.pixelSize: Theme.fontSizeXS
                        anchors.horizontalCenter: parent.horizontalCenter
                    }
                    Text {
                        text: (txType === "received" ? "+" : "-") + txAmount.toFixed(8) + " SIM"
                        color: txType === "received" ? Theme.txReceived : Theme.txSent
                        font.pixelSize: Theme.fontSizeXL; font.bold: true
                        anchors.horizontalCenter: parent.horizontalCenter
                    }
                }
            }

            DetailRow { label: "Type"; value: txType === "received" ? "Incoming" : "Outgoing" }
            DetailRow { label: "Address"; value: txAddress; mono: true }
            DetailRow { label: "Time"; value: txTimestamp }
            DetailRow { label: "Nonce"; value: txNonce; mono: true }
            DetailRow { label: "Tx ID"; value: txId; mono: true }

            Rectangle {
                width: parent.width; height: 44
                color: Theme.bgCard; radius: Theme.radiusSM
                Row {
                    anchors.centerIn: parent; spacing: 8
                    Text { text: "🔏"; font.pixelSize: 18 }
                    Text {
                        text: "Ed25519 Signed & Verified"
                        color: Theme.accent; font.pixelSize: Theme.fontSizeSM
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }
            }
        }
    }
}
