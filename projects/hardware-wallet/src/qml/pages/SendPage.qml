import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Page {
    id: sendPage
    anchors.fill: parent
    background: Rectangle { color: Theme.bgDark }
    onVisibleChanged: { if (visible) SendViewModel.refresh() }

    Column {
        anchors.fill: parent
        anchors.margins: Theme.spacingMD
        spacing: Theme.spacingLG

        Row {
            width: parent.width
            Text {
                text: "← Back to Dashboard"
                font.pixelSize: Theme.fontSizeMD; color: Theme.primary
                MouseArea {
                    anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                    onClicked: stackView.pop()
                }
            }
        }

        Text {
            text: "Send"
            font.pixelSize: Theme.fontSizeXL; color: Theme.textPrimary; font.bold: true
        }

        Text {
            text: "Balance: " + SendViewModel.balance.toFixed(2) + " SIM"
            font.pixelSize: Theme.fontSizeMD; color: Theme.textSecondary
        }

        // Recipient picker
        Text {
            text: "Recipient"
            font.pixelSize: Theme.fontSizeSM; color: Theme.textSecondary
        }

        // Show selected recipient
        Rectangle {
            width: parent.width; height: Theme.inputHeight
            color: Theme.bgInput; radius: Theme.radiusMD
            Row {
                anchors.fill: parent; anchors.margins: 12; spacing: 8
                Text {
                    text: SendViewModel.recipientAddress
                        ? "Selected: " + SendViewModel.recipientAddress.substring(0, 24) + "..."
                        : "Tap to select recipient"
                    color: SendViewModel.recipientAddress ? Theme.textPrimary : Theme.textMuted
                    font.pixelSize: Theme.fontSizeSM; font.family: "monospace"
                    anchors.verticalCenter: parent.verticalCenter
                    width: parent.width - 50
                    elide: Text.ElideMiddle
                }
                Text {
                    text: "▼"; color: Theme.primary; font.pixelSize: 18
                    anchors.verticalCenter: parent.verticalCenter
                }
            }
            MouseArea {
                anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                onClicked: {
                    SendViewModel.loadWallets()
                    walletPicker.visible = !walletPicker.visible
                }
            }
        }

        // Wallet picker list
        Rectangle {
            id: walletPicker
            width: parent.width
            height: Math.min(200, walletListView.contentHeight + 10)
            color: Theme.bgCard; radius: Theme.radiusMD
            visible: false; clip: true

            ListView {
                id: walletListView
                anchors.fill: parent; anchors.margins: 5
                model: SendViewModel.walletList
                delegate: Rectangle {
                    width: walletListView.width; height: 40
                    color: "transparent"
                    Rectangle {
                        anchors.fill: parent; anchors.margins: 2
                        color: mouseArea.containsMouse ? Theme.bgSurface : "transparent"
                        radius: Theme.radiusSM
                    }
                    Row {
                        anchors.fill: parent; anchors.margins: 8; spacing: 8
                        Text {
                            text: "User: " + model.userId.substring(0, 8) + "..."
                            color: Theme.textPrimary; font.pixelSize: Theme.fontSizeSM
                            anchors.verticalCenter: parent.verticalCenter
                            width: parent.width - 100
                        }
                        Text {
                            text: model.balance.toFixed(2) + " SIM"
                            color: Theme.textSecondary; font.pixelSize: Theme.fontSizeXS
                            anchors.verticalCenter: parent.verticalCenter
                        }
                    }
                    MouseArea {
                        id: mouseArea
                        anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                        hoverEnabled: true
                        onClicked: {
                            SendViewModel.selectWallet(index)
                            walletPicker.visible = false
                        }
                    }
                }
            }
        }

        // Amount
        TextField {
            id: amountField
            placeholderText: "Amount"
            width: parent.width; height: Theme.inputHeight
            color: Theme.textPrimary; font.pixelSize: Theme.fontSizeMD
            validator: DoubleValidator { bottom: 0.000001; decimals: 8 }
            background: Rectangle {
                color: Theme.bgInput; radius: Theme.radiusMD
                border.color: amountField.focus ? Theme.primary : Theme.bgSurface
            }
            onTextChanged: { var v = parseFloat(text); if (!isNaN(v)) SendViewModel.amount = v }
        }

        // Memo
        TextField {
            id: memoField
            placeholderText: "Memo (optional)"
            width: parent.width; height: Theme.inputHeight
            color: Theme.textPrimary; font.pixelSize: Theme.fontSizeMD
            background: Rectangle { color: Theme.bgInput; radius: Theme.radiusMD }
            onTextChanged: SendViewModel.memo = text
        }

        Text {
            text: SendViewModel.errorMessage
            color: Theme.accentDanger; font.pixelSize: Theme.fontSizeSM
            visible: text !== ""; wrapMode: Text.Wrap; width: parent.width
        }

        Button {
            text: "Review Transaction"
            width: parent.width; height: Theme.buttonHeight
            enabled: SendViewModel.recipientAddress.length > 0
                     && parseFloat(amountField.text) > 0
                     && !SendViewModel.loading
            background: Rectangle {
                color: parent.enabled ? Theme.primary : Theme.bgSurface
                radius: Theme.radiusMD
            }
            contentItem: Text {
                text: "Review Transaction"
                color: Theme.textPrimary; font.pixelSize: Theme.fontSizeLG
                horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
            }
            onClicked: confirmDialog.open()
        }
    }

    // Transaction Confirmation Dialog (What You See Is What You Sign)
    Dialog {
        id: confirmDialog
        modal: true; anchors.centerIn: parent
        width: 340; height: 380
        background: Rectangle { color: Theme.bgCard; radius: Theme.radiusLG }

        Column {
            anchors.fill: parent; anchors.margins: Theme.spacingMD
            spacing: Theme.spacingSM; width: parent.width - 20

            Text {
                text: "Confirm Transaction"
                font.pixelSize: Theme.fontSizeLG; color: Theme.textPrimary; font.bold: true
                anchors.horizontalCenter: parent.horizontalCenter
            }

            Text {
                text: "⚠ Please verify all details"
                font.pixelSize: Theme.fontSizeXS; color: Theme.accentWarning
                anchors.horizontalCenter: parent.horizontalCenter
            }

            Item { height: 4; width: 1 }

            // Detail rows
            ConfirmRow { label: "Send Amount"; value: parseFloat(amountField.text).toFixed(8) + " SIM"; valueColor: Theme.txSent }
            ConfirmRow { label: "Fee"; value: "0.00 SIM"; valueColor: Theme.textMuted }
            ConfirmRow { label: "Recipient"; value: SendViewModel.recipientAddress.substring(0, 16) + "..."; valueColor: Theme.textSecondary }
            ConfirmRow { label: "Total Cost"; value: parseFloat(amountField.text).toFixed(8) + " SIM"; valueColor: Theme.accentDanger }
            ConfirmRow { label: "Memo"; value: memoField.text || "(none)"; valueColor: Theme.textMuted }

            Item { height: 8; width: 1 }

            Row {
                anchors.horizontalCenter: parent.horizontalCenter; spacing: Theme.spacingMD
                Button {
                    text: "Cancel"; width: 130; height: 44
                    background: Rectangle { color: Theme.bgSurface; radius: Theme.radiusSM }
                    contentItem: Text {
                        text: "Cancel"; color: Theme.textPrimary
                        horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                    }
                    onClicked: confirmDialog.close()
                }
                Button {
                    text: SendViewModel.loading ? "..." : "Confirm & Send"
                    width: 160; height: 44
                    enabled: !SendViewModel.loading
                    background: Rectangle {
                        color: parent.enabled ? Theme.primary : Theme.bgSurface
                        radius: Theme.radiusSM
                    }
                    contentItem: Text {
                        text: SendViewModel.loading ? "Sending..." : "Confirm & Send"
                        color: Theme.textPrimary; font.pixelSize: Theme.fontSizeSM
                        horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                    }
                    onClicked: { confirmDialog.close(); SendViewModel.send() }
                }
            }
        }
    }

    // Row item for confirmation details
    component ConfirmRow: Rectangle {
        property string label: ""
        property string value: ""
        property color valueColor: Theme.textPrimary

        width: parent.width; height: 28; color: "transparent"
        Row {
            anchors.fill: parent
            Text {
                text: label; color: Theme.textMuted; font.pixelSize: Theme.fontSizeXS
                width: 100; anchors.verticalCenter: parent.verticalCenter
            }
            Text {
                text: value; color: valueColor; font.pixelSize: Theme.fontSizeXS; font.bold: true
                anchors.verticalCenter: parent.verticalCenter
            }
        }
    }
}
