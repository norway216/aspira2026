import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Page {
    id: registerPage
    anchors.fill: parent
    background: Rectangle { color: Theme.bgDark }

    Column {
        anchors.centerIn: parent
        spacing: Theme.spacingLG
        width: parent.width - 60

        Text {
            text: "Create Wallet"
            font.pixelSize: Theme.fontSizeXXL
            color: Theme.textPrimary
            anchors.horizontalCenter: parent.horizontalCenter
            font.bold: true
        }

        Text {
            text: "Register a new account"
            font.pixelSize: Theme.fontSizeMD
            color: Theme.textSecondary
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Item { height: Theme.spacingSM; width: 1 }

        TextField {
            id: usernameField
            placeholderText: "Username (min 3 characters)"
            width: parent.width
            height: Theme.inputHeight
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeMD
            background: Rectangle { color: Theme.bgInput; radius: Theme.radiusMD }
            onTextChanged: RegisterViewModel.username = text
        }

        TextField {
            id: passwordField
            placeholderText: "Password (min 8 characters)"
            width: parent.width
            height: Theme.inputHeight
            echoMode: TextInput.Password
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeMD
            background: Rectangle { color: Theme.bgInput; radius: Theme.radiusMD }
            onTextChanged: RegisterViewModel.password = text
        }

        TextField {
            id: confirmField
            placeholderText: "Confirm Password"
            width: parent.width
            height: Theme.inputHeight
            echoMode: TextInput.Password
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeMD
            background: Rectangle { color: Theme.bgInput; radius: Theme.radiusMD }
            onTextChanged: RegisterViewModel.confirmPassword = text
        }

        Text {
            id: errorText
            text: RegisterViewModel.errorMessage
            color: Theme.accentDanger
            font.pixelSize: Theme.fontSizeSM
            anchors.horizontalCenter: parent.horizontalCenter
            visible: text !== ""
            wrapMode: Text.Wrap
            width: parent.width
        }

        Button {
            text: "Register"
            width: parent.width
            height: Theme.buttonHeight
            enabled: usernameField.text.length >= 3
                     && passwordField.text.length >= 8
                     && passwordField.text === confirmField.text
                     && !RegisterViewModel.loading
            background: Rectangle {
                color: parent.enabled ? Theme.primary : Theme.bgSurface
                radius: Theme.radiusMD
            }
            contentItem: Item {
                BusyIndicator {
                    anchors.centerIn: parent
                    running: RegisterViewModel.loading
                    visible: RegisterViewModel.loading
                    width: 24; height: 24
                }
                Text {
                    anchors.centerIn: parent
                    text: RegisterViewModel.loading ? "" : "Register"
                    color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeLG
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                    visible: !RegisterViewModel.loading
                }
            }
            onClicked: RegisterViewModel.registerUser()
        }

        Text {
            text: "Back to Login"
            font.pixelSize: Theme.fontSizeSM
            color: Theme.primary
            anchors.horizontalCenter: parent.horizontalCenter
            MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                onClicked: stackView.pop()
            }
        }
    }
}
