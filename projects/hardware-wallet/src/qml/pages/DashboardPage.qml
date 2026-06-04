import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Page {
    id: dashboardPage
    anchors.fill: parent
    background: Rectangle { color: Theme.bgDark }
    onVisibleChanged: { if (visible) DashboardViewModel.refresh() }

    Flickable {
        anchors.fill: parent
        contentHeight: contentColumn.height + 20
        clip: true

        Column {
            id: contentColumn
            width: parent.width - 32
            anchors.horizontalCenter: parent.horizontalCenter
            anchors.top: parent.top
            anchors.topMargin: Theme.spacingMD
            spacing: Theme.spacingMD

            Row {
                width: parent.width
                Text {
                    text: "Dashboard"
                    font.pixelSize: Theme.fontSizeXL
                    color: Theme.textPrimary; font.bold: true
                    width: parent.width - 50
                }
                Text {
                    text: "⚙"; font.pixelSize: 24
                    color: Theme.textSecondary
                    anchors.verticalCenter: parent.verticalCenter
                    MouseArea {
                        anchors.fill: parent
                        cursorShape: Qt.PointingHandCursor
                        onClicked: stackView.push("qrc:/qml/pages/SettingsPage.qml")
                    }
                }
            }

            Text {
                text: "Welcome, " + DashboardViewModel.username
                font.pixelSize: Theme.fontSizeSM
                color: Theme.textSecondary
            }

            // Balance card
            Rectangle {
                width: parent.width; height: 130
                color: Theme.bgCard; radius: Theme.radiusLG
                Column {
                    anchors.centerIn: parent; spacing: 8
                    Text {
                        text: DashboardViewModel.hasWallet ? "Balance" : "No Wallet"
                        color: Theme.textSecondary
                        font.pixelSize: Theme.fontSizeSM
                        anchors.horizontalCenter: parent.horizontalCenter
                    }
                    Text {
                        text: DashboardViewModel.hasWallet
                              ? DashboardViewModel.balance.toFixed(2) + " SIM"
                              : "Create a wallet to start"
                        color: Theme.textPrimary
                        font.pixelSize: DashboardViewModel.hasWallet ? Theme.fontSizeXXL : Theme.fontSizeMD
                        font.bold: DashboardViewModel.hasWallet
                        anchors.horizontalCenter: parent.horizontalCenter
                    }
                    Text {
                        text: DashboardViewModel.address
                        color: Theme.textMuted; font.pixelSize: Theme.fontSizeXS
                        font.family: "monospace"
                        visible: DashboardViewModel.hasWallet
                        anchors.horizontalCenter: parent.horizontalCenter
                    }
                }
            }

            // Action row
            Row {
                width: parent.width; spacing: Theme.spacingMD
                visible: DashboardViewModel.hasWallet
                Button {
                    width: (parent.width - Theme.spacingMD * 3) / 4
                    height: Theme.buttonHeight
                    background: Rectangle { color: Theme.primary; radius: Theme.radiusMD }
                    contentItem: Text {
                        text: "Send"; color: Theme.textPrimary
                        font.pixelSize: Theme.fontSizeMD
                        horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                    }
                    onClicked: stackView.push("qrc:/qml/pages/SendPage.qml")
                }
                Button {
                    width: (parent.width - Theme.spacingMD * 3) / 4
                    height: Theme.buttonHeight
                    background: Rectangle { color: Theme.accent; radius: Theme.radiusMD }
                    contentItem: Text {
                        text: "Receive"; color: Theme.textPrimary
                        font.pixelSize: Theme.fontSizeMD
                        horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                    }
                    onClicked: stackView.push("qrc:/qml/pages/ReceivePage.qml")
                }
                Button {
                    width: (parent.width - Theme.spacingMD * 3) / 4
                    height: Theme.buttonHeight
                    background: Rectangle { color: Theme.bgSurface; radius: Theme.radiusMD }
                    contentItem: Text {
                        text: "Refresh"; color: Theme.textPrimary
                        font.pixelSize: Theme.fontSizeMD
                        horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                    }
                    onClicked: DashboardViewModel.refresh()
                }
                Button {
                    width: (parent.width - Theme.spacingMD * 3) / 4
                    height: Theme.buttonHeight
                    background: Rectangle { color: Theme.accentDanger; radius: Theme.radiusMD }
                    contentItem: Text {
                        text: "Delete"; color: Theme.textPrimary
                        font.pixelSize: Theme.fontSizeMD
                        horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                    }
                    onClicked: deleteConfirmDialog.open()
                }
            }

            Button {
                width: parent.width; height: Theme.buttonHeight
                visible: !DashboardViewModel.hasWallet && !DashboardViewModel.loading
                background: Rectangle { color: Theme.accent; radius: Theme.radiusMD }
                contentItem: Text {
                    text: "Create Wallet"; color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeLG
                    horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                }
                onClicked: DashboardViewModel.createWallet()
            }

            BusyIndicator {
                anchors.horizontalCenter: parent.horizontalCenter
                running: DashboardViewModel.loading; visible: DashboardViewModel.loading
            }

            Text {
                text: DashboardViewModel.errorMessage
                color: Theme.accentDanger; font.pixelSize: Theme.fontSizeSM
                visible: text !== ""; wrapMode: Text.Wrap; width: parent.width
            }

            // Transactions header
            Row {
                width: parent.width; visible: DashboardViewModel.hasWallet
                Text {
                    text: "Recent Transactions"
                    font.pixelSize: Theme.fontSizeLG
                    color: Theme.textPrimary; font.bold: true
                    width: parent.width - 80
                }
                Text {
                    text: "View All →"
                    font.pixelSize: Theme.fontSizeSM; color: Theme.primary
                    anchors.verticalCenter: parent.verticalCenter
                    MouseArea {
                        anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                        onClicked: stackView.push("qrc:/qml/pages/HistoryPage.qml")
                    }
                }
            }

            // Transaction list
            Repeater {
                model: DashboardViewModel.transactions
                delegate: Rectangle {
                    width: contentColumn.width; height: 56
                    color: Theme.bgCard; radius: Theme.radiusSM
                    Row {
                        anchors.fill: parent; anchors.margins: 8; spacing: 12
                        Rectangle {
                            width: 36; height: 36; radius: 18
                            color: model.type === "received" ? Theme.txReceived : Theme.txSent
                            anchors.verticalCenter: parent.verticalCenter
                            Text {
                                anchors.centerIn: parent
                                text: model.type === "received" ? "↓" : "↑"
                                color: "white"; font.pixelSize: 18; font.bold: true
                            }
                        }
                        Column {
                            anchors.verticalCenter: parent.verticalCenter
                            width: parent.width - 120
                            Text {
                                text: model.type === "received" ? "Received" : "Sent"
                                color: Theme.textPrimary
                                font.pixelSize: Theme.fontSizeMD; font.bold: true
                            }
                            Text {
                                text: model.timestamp
                                color: Theme.textMuted; font.pixelSize: Theme.fontSizeXS
                            }
                        }
                        Text {
                            text: (model.type === "received" ? "+" : "-") + model.amount.toFixed(2) + " SIM"
                            color: model.type === "received" ? Theme.txReceived : Theme.txSent
                            font.pixelSize: Theme.fontSizeMD; font.bold: true
                            anchors.verticalCenter: parent.verticalCenter
                        }
                    }
                }
            }
        }
    }

    Dialog {
        id: deleteConfirmDialog
        modal: true; anchors.centerIn: parent; width: 300; height: 180
        background: Rectangle { color: Theme.bgCard; radius: Theme.radiusLG }
        Column {
            anchors.centerIn: parent; spacing: Theme.spacingMD; width: parent.width - 20
            Text {
                text: "Delete Wallet?"
                color: Theme.accentDanger; font.pixelSize: Theme.fontSizeLG; font.bold: true
                anchors.horizontalCenter: parent.horizontalCenter
            }
            Text {
                text: "This will delete your wallet, keys, and all transactions."
                color: Theme.textSecondary; font.pixelSize: Theme.fontSizeSM
                wrapMode: Text.Wrap; width: parent.width
                horizontalAlignment: Text.AlignHCenter
            }
            Row {
                anchors.horizontalCenter: parent.horizontalCenter; spacing: Theme.spacingMD
                Button {
                    text: "Cancel"; width: 100; height: 36
                    background: Rectangle { color: Theme.bgSurface; radius: Theme.radiusSM }
                    onClicked: deleteConfirmDialog.close()
                }
                Button {
                    text: "Delete"; width: 100; height: 36
                    background: Rectangle { color: Theme.accentDanger; radius: Theme.radiusSM }
                    onClicked: { deleteConfirmDialog.close(); DashboardViewModel.deleteWallet() }
                }
            }
        }
    }
}
