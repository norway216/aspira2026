import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."
import "../components"

Page {
    id: lockScreenPage
    anchors.fill: parent
    background: Rectangle { color: Theme.bgDark }

    property string unlockPassword: ""
    property int attemptsLeft: 5

    Column {
        anchors.centerIn: parent
        spacing: Theme.spacingXL
        width: parent.width - 60

        Text {
            text: "🔒"
            font.pixelSize: 64
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Text {
            text: "Wallet Locked"
            font.pixelSize: Theme.fontSizeXL
            color: Theme.textPrimary
            font.bold: true
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Text {
            text: "Enter your password to unlock"
            font.pixelSize: Theme.fontSizeSM
            color: Theme.textSecondary
            anchors.horizontalCenter: parent.horizontalCenter
        }

        TextField {
            id: unlockField
            placeholderText: "Password"
            width: parent.width; height: Theme.inputHeight
            echoMode: TextInput.Password
            color: Theme.textPrimary; font.pixelSize: Theme.fontSizeMD
            background: Rectangle { color: Theme.bgInput; radius: Theme.radiusMD }
        }

        Text {
            id: errorMsg
            text: ""; color: Theme.accentDanger
            font.pixelSize: Theme.fontSizeSM
            anchors.horizontalCenter: parent.horizontalCenter
            visible: text !== ""
        }

        Button {
            text: "Unlock"
            width: parent.width; height: Theme.buttonHeight
            enabled: unlockField.text.length >= 6
            background: Rectangle {
                color: parent.enabled ? Theme.primary : Theme.bgSurface
                radius: Theme.radiusMD
            }
            contentItem: Text {
                text: "Unlock"; color: Theme.textPrimary
                font.pixelSize: Theme.fontSizeLG
                horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
            }
            onClicked: {
                // Verify password and navigate back
                AuthService.login(AuthService.currentUsername, unlockField.text)
            }
        }

        Text {
            text: "Attempts remaining: " + attemptsLeft
            font.pixelSize: Theme.fontSizeXS
            color: Theme.textMuted
            anchors.horizontalCenter: parent.horizontalCenter
        }
    }

    // Listen for login result
    Connections {
        target: LoginViewModel
        function onLoginSucceeded() {
            stackView.pop()
            DashboardViewModel.refresh()
        }
        function onLoginFailed(error) {
            errorMsg.text = error
            attemptsLeft--
        }
    }
}
