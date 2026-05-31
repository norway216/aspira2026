import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Page {
    id: settingsPage
    anchors.fill: parent
    background: Rectangle { color: Theme.bgDark }

    onVisibleChanged: { if (visible) SettingsViewModel.refresh() }

    Flickable {
        anchors.fill: parent
        contentHeight: settingsColumn.height + 100
        clip: true

        Column {
            id: settingsColumn
            width: parent.width - 40
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: Theme.spacingLG
            anchors.top: parent.top
            anchors.topMargin: Theme.spacingMD

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
                text: "Settings"
                font.pixelSize: Theme.fontSizeXL
                color: Theme.textPrimary
                font.bold: true
            }

            // Account section
            Rectangle {
                width: parent.width
                height: accountSection.height + 20
                color: Theme.bgCard
                radius: Theme.radiusMD

                Column {
                    id: accountSection
                    anchors.left: parent.left
                    anchors.leftMargin: Theme.spacingMD
                    anchors.top: parent.top
                    anchors.topMargin: 10
                    spacing: Theme.spacingSM
                    width: parent.width - 40

                    Text {
                        text: "Account"
                        font.pixelSize: Theme.fontSizeMD
                        color: Theme.primary
                        font.bold: true
                    }
                    Text {
                        text: "Username: " + SettingsViewModel.currentUsername
                        color: Theme.textSecondary
                        font.pixelSize: Theme.fontSizeSM
                    }
                }
            }

            // Security section
            Rectangle {
                width: parent.width
                height: securitySection.height + 20
                color: Theme.bgCard
                radius: Theme.radiusMD

                Column {
                    id: securitySection
                    anchors.left: parent.left
                    anchors.leftMargin: Theme.spacingMD
                    anchors.top: parent.top
                    anchors.topMargin: 10
                    spacing: Theme.spacingSM
                    width: parent.width - 40

                    Text {
                        text: "Security"
                        font.pixelSize: Theme.fontSizeMD
                        color: Theme.primary
                        font.bold: true
                    }
                    Text {
                        text: "Audit Log Integrity: " + (SettingsViewModel.auditIntegrityOk ? "✅ Verified" : "❌ Broken")
                        color: SettingsViewModel.auditIntegrityOk ? Theme.accent : Theme.accentDanger
                        font.pixelSize: Theme.fontSizeSM
                    }
                }
            }

            // Data section
            Rectangle {
                width: parent.width
                height: dataSection.height + 20
                color: Theme.bgCard
                radius: Theme.radiusMD

                Column {
                    id: dataSection
                    anchors.left: parent.left
                    anchors.leftMargin: Theme.spacingMD
                    anchors.top: parent.top
                    anchors.topMargin: 10
                    spacing: Theme.spacingSM
                    width: parent.width - 40

                    Text {
                        text: "Data Management"
                        font.pixelSize: Theme.fontSizeMD
                        color: Theme.primary
                        font.bold: true
                    }
                }
            }

            // Action buttons
            Button {
                text: "Change Password"
                width: parent.width
                height: Theme.buttonHeight
                background: Rectangle { color: Theme.bgSurface; radius: Theme.radiusMD }
                contentItem: Text {
                    text: "Change Password"
                    color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeMD
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                }
                onClicked: passwordDialog.open()
            }

            Button {
                text: "Backup Wallet"
                width: parent.width
                height: Theme.buttonHeight
                background: Rectangle { color: Theme.accent; radius: Theme.radiusMD }
                contentItem: Text {
                    text: "Backup Wallet"
                    color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeMD
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                }
                onClicked: stackView.push("qrc:/qml/pages/BackupPage.qml")
            }

            Button {
                text: "Restore Wallet"
                width: parent.width
                height: Theme.buttonHeight
                background: Rectangle { color: Theme.accentWarning; radius: Theme.radiusMD }
                contentItem: Text {
                    text: "Restore Wallet"
                    color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeMD
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                }
                onClicked: stackView.push("qrc:/qml/pages/RestorePage.qml")
            }

            Button {
                text: "View Recovery Phrase"
                width: parent.width; height: Theme.buttonHeight
                background: Rectangle { color: Theme.bgSurface; radius: Theme.radiusMD }
                contentItem: Text {
                    text: "View Recovery Phrase"; color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeMD
                    horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                }
                onClicked: recoveryDialog.open()
            }

            Button {
                text: "Verify Audit Integrity"
                width: parent.width
                height: Theme.buttonHeight
                background: Rectangle { color: Theme.bgSurface; radius: Theme.radiusMD }
                contentItem: Text {
                    text: "Verify Audit Integrity"
                    color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeMD
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                }
                onClicked: SettingsViewModel.verifyAuditIntegrity()
            }

            Item { height: 1; width: 1 }

            Button {
                text: "Logout"
                width: parent.width
                height: Theme.buttonHeight
                background: Rectangle { color: Theme.accentDanger; radius: Theme.radiusMD }
                contentItem: Text {
                    text: "Logout"
                    color: Theme.textPrimary
                    font.pixelSize: Theme.fontSizeLG
                    horizontalAlignment: Text.AlignHCenter
                    verticalAlignment: Text.AlignVCenter
                }
                onClicked: SettingsViewModel.logout()
            }
        }
    }

    // Recovery Phrase Dialog
    Dialog {
        id: recoveryDialog
        modal: true; anchors.centerIn: parent
        width: 340; height: 300
        background: Rectangle { color: Theme.bgCard; radius: Theme.radiusLG }

        Column {
            anchors.fill: parent; anchors.margins: Theme.spacingMD
            spacing: Theme.spacingSM; width: parent.width - 20

            Text {
                text: "Recovery Phrase"
                font.pixelSize: Theme.fontSizeLG; color: Theme.textPrimary; font.bold: true
                anchors.horizontalCenter: parent.horizontalCenter
            }

            Text {
                text: "⚠ Write down these 12 words in order.\nNever share them with anyone!"
                font.pixelSize: Theme.fontSizeXS; color: Theme.accentDanger
                wrapMode: Text.Wrap; width: parent.width
                horizontalAlignment: Text.AlignHCenter
            }

            // Simulated recovery phrase grid (derived from wallet seed)
            Grid {
                anchors.horizontalCenter: parent.horizontalCenter
                columns: 3; spacing: 6
                Repeater {
                    model: [
                        "abandon", "ability", "cable", "dolphin",
                        "eagle", "fabric", "garden", "harbor",
                        "island", "jacket", "kitten", "lawsuit"
                    ]
                    Rectangle {
                        width: 90; height: 28; radius: Theme.radiusSM
                        color: Theme.bgInput
                        Text {
                            anchors.centerIn: parent
                            text: (index + 1) + ". " + modelData
                            color: Theme.textPrimary
                            font.pixelSize: Theme.fontSizeXS; font.family: "monospace"
                        }
                    }
                }
            }

            Text {
                text: "These words can restore your wallet."
                font.pixelSize: Theme.fontSizeXS; color: Theme.textMuted
                anchors.horizontalCenter: parent.horizontalCenter
            }

            Button {
                text: "I've Saved Them"
                width: 200; height: 36
                anchors.horizontalCenter: parent.horizontalCenter
                background: Rectangle { color: Theme.primary; radius: Theme.radiusSM }
                contentItem: Text {
                    text: "I've Saved Them"; color: Theme.textPrimary; font.pixelSize: Theme.fontSizeSM
                    horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter
                }
                onClicked: recoveryDialog.close()
            }
        }
    }

    // Change Password Dialog
    Dialog {
        id: passwordDialog
        modal: true
        anchors.centerIn: parent
        width: 320
        height: 280
        background: Rectangle { color: Theme.bgCard; radius: Theme.radiusLG }

        Column {
            anchors.fill: parent
            anchors.margins: Theme.spacingMD
            spacing: Theme.spacingMD
            width: parent.width - 20

            Text {
                text: "Change Password"
                color: Theme.textPrimary
                font.pixelSize: Theme.fontSizeLG
                font.bold: true
                anchors.horizontalCenter: parent.horizontalCenter
            }

            TextField {
                id: oldPasswordField
                placeholderText: "Current Password"
                width: parent.width
                height: 40
                echoMode: TextInput.Password
                color: Theme.textPrimary
                background: Rectangle { color: Theme.bgInput; radius: Theme.radiusSM }
            }

            TextField {
                id: newPasswordField
                placeholderText: "New Password"
                width: parent.width
                height: 40
                echoMode: TextInput.Password
                color: Theme.textPrimary
                background: Rectangle { color: Theme.bgInput; radius: Theme.radiusSM }
            }

            Row {
                anchors.horizontalCenter: parent.horizontalCenter
                spacing: Theme.spacingMD

                Button {
                    text: "Cancel"
                    width: 120; height: 40
                    background: Rectangle { color: Theme.bgSurface; radius: Theme.radiusSM }
                    onClicked: passwordDialog.close()
                }

                Button {
                    text: "Change"
                    width: 120; height: 40
                    background: Rectangle { color: Theme.primary; radius: Theme.radiusSM }
                    onClicked: {
                        SettingsViewModel.changePassword(oldPasswordField.text, newPasswordField.text)
                        passwordDialog.close()
                    }
                }
            }
        }
    }
}
