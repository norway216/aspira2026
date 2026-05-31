import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Page {
    id: backupPage
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
            text: "Backup Wallet"
            font.pixelSize: Theme.fontSizeXL
            color: Theme.textPrimary
            font.bold: true
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Text {
            text: "Create an encrypted backup of your wallet data. You will need the backup password to restore later."
            color: Theme.textSecondary
            font.pixelSize: Theme.fontSizeMD
            wrapMode: Text.Wrap
            width: parent.width
        }

        TextField {
            id: filePathField
            placeholderText: "File path (e.g., wallet_backup.dat)"
            width: parent.width
            height: Theme.inputHeight
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeMD
            text: BackupViewModel.filePath
            background: Rectangle { color: Theme.bgInput; radius: Theme.radiusMD }
            onTextChanged: BackupViewModel.filePath = text
        }

        TextField {
            id: backupPasswordField
            placeholderText: "Backup Password (min 8 chars)"
            width: parent.width
            height: Theme.inputHeight
            echoMode: TextInput.Password
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeMD
            background: Rectangle { color: Theme.bgInput; radius: Theme.radiusMD }
            onTextChanged: BackupViewModel.password = text
        }

        TextField {
            id: confirmBackupPasswordField
            placeholderText: "Confirm Backup Password"
            width: parent.width
            height: Theme.inputHeight
            echoMode: TextInput.Password
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeMD
            background: Rectangle { color: Theme.bgInput; radius: Theme.radiusMD }
            onTextChanged: BackupViewModel.confirmPassword = text
        }

        Text {
            id: statusText
            text: BackupViewModel.status
            color: Theme.accent
            font.pixelSize: Theme.fontSizeSM
            anchors.horizontalCenter: parent.horizontalCenter
            visible: text !== ""
        }

        Text {
            id: errorText
            text: BackupViewModel.errorMessage
            color: Theme.accentDanger
            font.pixelSize: Theme.fontSizeSM
            anchors.horizontalCenter: parent.horizontalCenter
            visible: text !== ""
            wrapMode: Text.Wrap
            width: parent.width
        }

        Button {
            text: "Create Backup"
            width: parent.width
            height: Theme.buttonHeight
            enabled: backupPasswordField.text.length >= 8
                     && backupPasswordField.text === confirmBackupPasswordField.text
                     && filePathField.text.length > 0
                     && !BackupViewModel.loading
            background: Rectangle {
                color: parent.enabled ? Theme.accent : Theme.bgSurface
                radius: Theme.radiusMD
            }
            contentItem: Item {
                BusyIndicator {
                    anchors.centerIn: parent
                    running: BackupViewModel.loading
                    visible: BackupViewModel.loading
                    width: 24; height: 24
                }
                Text {
                    anchors.centerIn: parent
                    text: BackupViewModel.loading ? "" : "Create Backup"
                    color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeLG
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                    visible: !BackupViewModel.loading
                }
            }
            onClicked: BackupViewModel.exportBackup()
        }
    }
}
