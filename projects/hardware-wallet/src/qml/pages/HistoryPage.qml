import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Page {
    id: historyPage
    anchors.fill: parent
    background: Rectangle { color: Theme.bgDark }

    onVisibleChanged: { if (visible) HistoryViewModel.refresh() }

    Column {
        anchors.fill: parent
        anchors.margins: Theme.spacingMD
        spacing: Theme.spacingMD

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
            text: "Transaction History"
            font.pixelSize: Theme.fontSizeXL
            color: Theme.textPrimary
            font.bold: true
        }

        // Filter bar
        Row {
            width: parent.width; spacing: Theme.spacingSM
            property string currentFilter: "all"

            Repeater {
                model: ["All", "Sent", "Received"]
                Rectangle {
                    width: 70; height: 30; radius: Theme.radiusSM
                    color: filterRow.currentFilter === modelData.toLowerCase() ? Theme.primary : Theme.bgSurface
                    Text {
                        anchors.centerIn: parent
                        text: modelData; font.pixelSize: Theme.fontSizeXS
                        color: filterRow.currentFilter === modelData.toLowerCase() ? "white" : Theme.textSecondary
                    }
                    MouseArea {
                        anchors.fill: parent; cursorShape: Qt.PointingHandCursor
                        onClicked: {
                            filterRow.currentFilter = modelData.toLowerCase()
                            HistoryViewModel.refresh()
                        }
                    }
                }
            }
        }

        Text {
            text: "Total: " + HistoryViewModel.totalCount
            font.pixelSize: Theme.fontSizeSM
            color: Theme.textSecondary
        }

        ListView {
            id: historyList
            width: parent.width
            height: parent.height - 150
            clip: true
            model: HistoryViewModel.transactions
            spacing: 4
            delegate: Rectangle {
                width: historyList.width
                height: 56
                color: Theme.bgCard
                radius: Theme.radiusSM

                MouseArea {
                    anchors.fill: parent
                    onClicked: {
                        var props = {
                            txType: model.type, txAmount: model.amount,
                            txAddress: model.address, txTimestamp: model.timestamp,
                            txStatus: model.status, txNonce: model.nonce,
                            txId: model.nonce.substring(0, 16)
                        }
                        stackView.push("qrc:/qml/pages/TransactionDetailPage.qml", props)
                    }
                }

                Row {
                    anchors.fill: parent
                    anchors.margins: 8
                    spacing: 12

                    Rectangle {
                        width: 36; height: 36
                        radius: 18
                        color: model.type === "received" ? Theme.txReceived : Theme.txSent
                        anchors.verticalCenter: parent.verticalCenter
                        Text {
                            anchors.centerIn: parent
                            text: model.type === "received" ? "↓" : "↑"
                            color: "white"
                            font.pixelSize: 18
                            font.bold: true
                        }
                    }

                    Column {
                        anchors.verticalCenter: parent.verticalCenter
                        width: parent.width - 150

                        Text {
                            text: model.type === "received" ? "Received" : "Sent"
                            color: Theme.textPrimary
                            font.pixelSize: Theme.fontSizeMD
                            font.bold: true
                        }

                        Text {
                            text: "To: " + model.address.substring(0, 16) + "..."
                            color: Theme.textMuted
                            font.pixelSize: Theme.fontSizeXS
                            elide: Text.ElideMiddle
                            width: parent.width
                        }

                        Text {
                            text: model.status + " | " + model.timestamp
                            color: Theme.textMuted
                            font.pixelSize: Theme.fontSizeXS
                        }
                    }

                    Text {
                        text: model.amount.toFixed(2) + " SIM"
                        color: model.type === "received" ? Theme.txReceived : Theme.txSent
                        font.pixelSize: Theme.fontSizeMD
                        font.bold: true
                        anchors.verticalCenter: parent.verticalCenter
                    }
                }
            }

            footer: Row {
                width: parent.width
                Button {
                    text: "Load More"
                    width: parent.width
                    height: Theme.buttonHeight
                    visible: HistoryViewModel.canLoadMore
                    enabled: !HistoryViewModel.loading
                    background: Rectangle { color: Theme.bgSurface; radius: Theme.radiusMD }
                    contentItem: Text {
                        text: "Load More"
                        color: Theme.textPrimary
                        horizontalAlignment: Text.AlignHCenter
                        verticalAlignment: Text.AlignVCenter
                    }
                    onClicked: HistoryViewModel.loadMore()
                }
            }
        }
    }
}
