import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Page {
    id: loginPage
    anchors.fill: parent
    background: Rectangle { color: Theme.bgDark }

    Column {
        anchors.centerIn: parent
        spacing: Theme.spacingLG
        width: parent.width - 60

        Text {
            text: "Hardware Wallet"
            font.pixelSize: Theme.fontSizeXXL
            color: Theme.textPrimary
            anchors.horizontalCenter: parent.horizontalCenter
            font.bold: true
        }

        Text {
            text: "Sign in to your wallet"
            font.pixelSize: Theme.fontSizeMD
            color: Theme.textSecondary
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Item { height: Theme.spacingSM; width: 1 }

        TextField {
            id: usernameField
            placeholderText: "Username"
            width: parent.width
            height: Theme.inputHeight
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeMD
            background: Rectangle {
                color: Theme.bgInput
                radius: Theme.radiusMD
                border.color: usernameField.focus ? Theme.primary : Theme.bgSurface
            }
            onTextChanged: LoginViewModel.username = text
        }

        TextField {
            id: passwordField
            placeholderText: "Password"
            width: parent.width
            height: Theme.inputHeight
            echoMode: TextInput.Password
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeMD
            background: Rectangle {
                color: Theme.bgInput
                radius: Theme.radiusMD
                border.color: passwordField.focus ? Theme.primary : Theme.bgSurface
            }
            onTextChanged: LoginViewModel.password = text
        }

        Text {
            id: errorText
            text: LoginViewModel.errorMessage
            color: Theme.accentDanger
            font.pixelSize: Theme.fontSizeSM
            anchors.horizontalCenter: parent.horizontalCenter
            visible: text !== ""
            wrapMode: Text.WordWrap
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
        }

        // Security warning - shown when only 1 attempt remains (per architecture §11.3)
        Text {
            id: securityWarning
            text: LoginViewModel.securityWarning
            color: "#F44336"
            font.pixelSize: Theme.fontSizeSM
            font.bold: true
            anchors.horizontalCenter: parent.horizontalCenter
            visible: LoginViewModel.remainingAttempts === 1 && text !== ""
            wrapMode: Text.WordWrap
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
        }

        // Attempts counter
        Text {
            id: attemptsText
            text: "Attempts remaining: " + LoginViewModel.remainingAttempts + " of " + 3
            color: LoginViewModel.remainingAttempts <= 1 ? Theme.accentDanger : Theme.textSecondary
            font.pixelSize: Theme.fontSizeXS
            anchors.horizontalCenter: parent.horizontalCenter
            visible: LoginViewModel.remainingAttempts < 3
        }

        Button {
            text: "Login"
            width: parent.width
            height: Theme.buttonHeight
            enabled: usernameField.text.length > 0 && passwordField.text.length >= 6 && !LoginViewModel.loading
            background: Rectangle {
                color: parent.enabled ? Theme.primary : Theme.bgSurface
                radius: Theme.radiusMD
            }
            contentItem: Item {
                BusyIndicator {
                    anchors.centerIn: parent
                    running: LoginViewModel.loading
                    visible: LoginViewModel.loading
                    width: 24; height: 24
                }
                Text {
                    anchors.centerIn: parent
                    text: LoginViewModel.loading ? "" : "Login"
                    color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeLG
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                    visible: !LoginViewModel.loading
                }
            }
            onClicked: LoginViewModel.login()
        }

        Text {
            text: "Create new wallet"
            font.pixelSize: Theme.fontSizeSM
            color: Theme.primary
            anchors.horizontalCenter: parent.horizontalCenter
            MouseArea {
                anchors.fill: parent
                cursorShape: Qt.PointingHandCursor
                onClicked: stackView.push("qrc:/qml/pages/RegisterPage.qml")
            }
        }
    }
}
