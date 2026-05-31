import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Page {
    id: restorePage
    anchors.fill: parent
    background: Rectangle { color: Theme.bgDark }

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
            text: "Restore Wallet"
            font.pixelSize: Theme.fontSizeXL
            color: Theme.textPrimary
            font.bold: true
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Text {
            text: "⚠ WARNING: This will replace ALL existing data with data from the backup file."
            color: Theme.accentDanger
            font.pixelSize: Theme.fontSizeMD
            wrapMode: Text.Wrap
            width: parent.width
            font.bold: true
        }

        TextField {
            id: filePathField
            placeholderText: "Backup file path"
            width: parent.width
            height: Theme.inputHeight
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeMD
            text: RestoreViewModel.filePath
            background: Rectangle { color: Theme.bgInput; radius: Theme.radiusMD }
            onTextChanged: RestoreViewModel.filePath = text
        }

        TextField {
            id: restorePasswordField
            placeholderText: "Backup Password"
            width: parent.width
            height: Theme.inputHeight
            echoMode: TextInput.Password
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeMD
            background: Rectangle { color: Theme.bgInput; radius: Theme.radiusMD }
            onTextChanged: RestoreViewModel.password = text
        }

        Text {
            id: statusText
            text: RestoreViewModel.status
            color: Theme.accent
            font.pixelSize: Theme.fontSizeSM
            anchors.horizontalCenter: parent.horizontalCenter
            visible: text !== ""
        }

        Text {
            id: errorText
            text: RestoreViewModel.errorMessage
            color: Theme.accentDanger
            font.pixelSize: Theme.fontSizeSM
            anchors.horizontalCenter: parent.horizontalCenter
            visible: text !== ""
            wrapMode: Text.Wrap
            width: parent.width
        }

        Button {
            text: "Restore"
            width: parent.width
            height: Theme.buttonHeight
            enabled: restorePasswordField.text.length > 0
                     && filePathField.text.length > 0
                     && !RestoreViewModel.loading
            background: Rectangle {
                color: parent.enabled ? Theme.accentWarning : Theme.bgSurface
                radius: Theme.radiusMD
            }
            contentItem: Item {
                BusyIndicator {
                    anchors.centerIn: parent
                    running: RestoreViewModel.loading
                    visible: RestoreViewModel.loading
                    width: 24; height: 24
                }
                Text {
                    anchors.centerIn: parent
                    text: RestoreViewModel.loading ? "" : "Restore"
                    color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeLG
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                    visible: !RestoreViewModel.loading
                }
            }
            onClicked: RestoreViewModel.importBackup()
        }
    }
}
