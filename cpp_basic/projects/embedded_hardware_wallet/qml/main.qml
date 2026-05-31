import QtQuick 2.15
import QtQuick.Controls 2.15
import QtQuick.Layouts 1.15

ApplicationWindow {
    id: appWindow
    visible: true
    width: 800
    height: 600
    minimumWidth: 640
    minimumHeight: 480
    title: "Embedded Hardware Wallet"

    // Crypto dark theme
    color: "#0a0e27"

    // Access to C++ backend
    property var backend: AppCore

    // Fonts
    font.family: "monospace"

    // Status bar
    footer: Rectangle {
        id: statusBar
        width: parent.width
        height: 32
        color: "#141832"
        border.color: "#1e2348"
        border.width: 1

        RowLayout {
            anchors.fill: parent
            anchors.margins: 4
            spacing: 16

            Rectangle {
                width: 10; height: 10; radius: 5
                color: backend.networkConnected ? "#00ff88" : "#ff4444"
            }
            Text {
                text: backend.networkConnected ? "Connected" : "Offline"
                color: backend.networkConnected ? "#00ff88" : "#ff4444"
                font.pixelSize: 11
            }

            Rectangle { width: 1; height: 20; color: "#2a2f5a" }

            Text {
                text: "Security: " + backend.securityLevel
                color: "#8890c0"
                font.pixelSize: 11
            }

            Rectangle { width: 1; height: 20; color: "#2a2f5a" }

            Text {
                text: "Threads: " + backend.threadCount
                color: "#8890c0"
                font.pixelSize: 11
            }

            Item { Layout.fillWidth: true }

            Text {
                text: backend.lastError
                color: "#ff6666"
                font.pixelSize: 11
                elide: Text.ElideRight
                Layout.maximumWidth: 200
                visible: backend.lastError !== ""
            }
        }
    }

    // Main content
    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 20
        spacing: 12

        // Header
        Rectangle {
            Layout.fillWidth: true
            height: 56
            color: "#0d1137"
            radius: 8
            border.color: "#1e2360"
            border.width: 1

            RowLayout {
                anchors.fill: parent
                anchors.margins: 12

                Text {
                    text: "🔐"
                    font.pixelSize: 24
                }
                Text {
                    text: "Hardware Wallet"
                    color: "#e0e4ff"
                    font.pixelSize: 20
                    font.bold: true
                }
                Item { Layout.fillWidth: true }

                // Lock indicator
                Rectangle {
                    width: 80; height: 28; radius: 14
                    color: backend.walletLocked ? "#3a2040" : "#1a3a2a"
                    border.color: backend.walletLocked ? "#8b44aa" : "#22aa55"

                    Text {
                        anchors.centerIn: parent
                        text: backend.walletLocked ? "🔒 Locked" : "🔓 Unlocked"
                        color: backend.walletLocked ? "#cc99ff" : "#44ff88"
                        font.pixelSize: 11
                    }
                }
            }
        }

        // Tab bar
        TabBar {
            id: tabBar
            Layout.fillWidth: true
            background: Rectangle { color: "transparent" }

            TabButton {
                text: "🏦 Wallet"
                contentItem: Text {
                    text: parent.text
                    color: parent.checked ? "#00ddff" : "#6670a0"
                    font.pixelSize: 13
                    horizontalAlignment: Text.AlignHCenter
                }
                background: Rectangle {
                    color: parent.checked ? "#141d50" : "transparent"
                    radius: 6
                    border.color: parent.checked ? "#0066aa" : "transparent"
                }
            }
            TabButton {
                text: "📋 Transactions"
                contentItem: Text {
                    text: parent.text
                    color: parent.checked ? "#00ddff" : "#6670a0"
                    font.pixelSize: 13
                    horizontalAlignment: Text.AlignHCenter
                }
                background: Rectangle {
                    color: parent.checked ? "#141d50" : "transparent"
                    radius: 6
                    border.color: parent.checked ? "#0066aa" : "transparent"
                }
            }
            TabButton {
                text: "💾 Backup"
                contentItem: Text {
                    text: parent.text
                    color: parent.checked ? "#00ddff" : "#6670a0"
                    font.pixelSize: 13
                    horizontalAlignment: Text.AlignHCenter
                }
                background: Rectangle {
                    color: parent.checked ? "#141d50" : "transparent"
                    radius: 6
                    border.color: parent.checked ? "#0066aa" : "transparent"
                }
            }
            TabButton {
                text: "⚙ Settings"
                contentItem: Text {
                    text: parent.text
                    color: parent.checked ? "#00ddff" : "#6670a0"
                    font.pixelSize: 13
                    horizontalAlignment: Text.AlignHCenter
                }
                background: Rectangle {
                    color: parent.checked ? "#141d50" : "transparent"
                    radius: 6
                    border.color: parent.checked ? "#0066aa" : "transparent"
                }
            }
        }

        // Tab contents
        StackLayout {
            id: tabStack
            Layout.fillWidth: true
            Layout.fillHeight: true
            currentIndex: tabBar.currentIndex

            // ---- Wallet Tab ----
            Rectangle {
                color: "#0d1137"
                radius: 8
                border.color: "#1e2360"
                border.width: 1

                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 20
                    spacing: 16

                    // Wallet not created state
                    Rectangle {
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        color: "transparent"
                        visible: !backend.walletInitialized

                        ColumnLayout {
                            anchors.centerIn: parent
                            spacing: 16

                            Text {
                                text: "🔐 No Wallet Found"
                                color: "#8890c0"
                                font.pixelSize: 20
                                font.bold: true
                                Layout.alignment: Qt.AlignHCenter
                            }
                            Text {
                                text: "Create a new wallet or recover an existing one to get started."
                                color: "#6670a0"
                                font.pixelSize: 13
                                Layout.alignment: Qt.AlignHCenter
                            }

                            RowLayout {
                                Layout.alignment: Qt.AlignHCenter
                                spacing: 12

                                Button {
                                    text: "🌟 Create New Wallet"
                                    onClicked: createWalletDialog.open()
                                    contentItem: Text { text: parent.text; color: "#ffffff"; font.pixelSize: 13 }
                                    background: Rectangle {
                                        color: "#0055cc"
                                        radius: 6
                                    }
                                }
                                Button {
                                    text: "🔑 Recover Wallet"
                                    onClicked: recoverWalletDialog.open()
                                    contentItem: Text { text: parent.text; color: "#ffffff"; font.pixelSize: 13 }
                                    background: Rectangle {
                                        color: "#3a2550"
                                        radius: 6
                                        border.color: "#6b44aa"
                                    }
                                }
                            }
                        }
                    }

                    // Wallet active state
                    ColumnLayout {
                        visible: backend.walletInitialized
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        spacing: 12

                        // Balance card
                        Rectangle {
                            Layout.fillWidth: true
                            height: 100
                            color: "#0f1745"
                            radius: 10
                            border.color: "#1a2b6a"
                            border.width: 1

                            ColumnLayout {
                                anchors.centerIn: parent
                                spacing: 4

                                Text {
                                    text: "Total Balance"
                                    color: "#6670a0"
                                    font.pixelSize: 12
                                    Layout.alignment: Qt.AlignHCenter
                                }
                                Text {
                                    id: balanceText
                                    text: backend.balance + " BTC"
                                    color: "#00ff88"
                                    font.pixelSize: 28
                                    font.bold: true
                                    Layout.alignment: Qt.AlignHCenter
                                }
                                Text {
                                    text: "Address: " + backend.address
                                    color: "#5560a0"
                                    font.pixelSize: 10
                                    Layout.alignment: Qt.AlignHCenter
                                }
                            }
                        }

                        // QR Code placeholder
                        RowLayout {
                            Layout.fillWidth: true
                            spacing: 16

                            Rectangle {
                                width: 120; height: 120
                                color: "#ffffff"
                                radius: 4
                                Image {
                                    anchors.fill: parent
                                    anchors.margins: 4
                                    source: "image://wallet/qr_receive"
                                    fillMode: Image.PreserveAspectFit
                                }
                            }

                            ColumnLayout {
                                spacing: 8
                                Text {
                                    text: "Receive Address"
                                    color: "#8890c0"
                                    font.pixelSize: 14
                                    font.bold: true
                                }
                                Text {
                                    text: backend.address
                                    color: "#5560a0"
                                    font.pixelSize: 10
                                    wrapMode: Text.WrapAnywhere
                                    Layout.fillWidth: true
                                }
                                Button {
                                    text: "🔄 Refresh Balance"
                                    onClicked: backend.refreshBalance()
                                    contentItem: Text { text: parent.text; color: "#ffffff"; font.pixelSize: 11 }
                                    background: Rectangle {
                                        color: "#1a2b6a"
                                        radius: 4
                                    }
                                }
                            }
                        }

                        // Send form
                        GroupBox {
                            title: "Send Transaction"
                            Layout.fillWidth: true
                            font.pixelSize: 13

                            background: Rectangle {
                                color: "#0f1745"
                                radius: 6
                                border.color: "#1a2b6a"
                            }
                            label: Text {
                                text: "📤 Send Transaction"
                                color: "#8890c0"
                                font.pixelSize: 13
                            }

                            GridLayout {
                                columns: 2
                                rowSpacing: 8
                                columnSpacing: 12

                                Text { text: "To Address:"; color: "#8890c0"; font.pixelSize: 12 }
                                TextField {
                                    id: toAddressField
                                    Layout.fillWidth: true
                                    placeholderText: "Enter recipient address..."
                                    color: "#e0e4ff"
                                    background: Rectangle {
                                        color: "#0a0e27"
                                        radius: 4
                                        border.color: "#1e2360"
                                    }
                                }
                                Text { text: "Amount (satoshis):"; color: "#8890c0"; font.pixelSize: 12 }
                                TextField {
                                    id: amountField
                                    Layout.fillWidth: true
                                    placeholderText: "Enter amount in satoshis"
                                    color: "#e0e4ff"
                                    validator: IntValidator { bottom: 1 }
                                    background: Rectangle {
                                        color: "#0a0e27"
                                        radius: 4
                                        border.color: "#1e2360"
                                    }
                                }
                                Item {}
                                Button {
                                    text: "✍ Sign Transaction"
                                    onClicked: backend.signTransaction(toAddressField.text, amountField.text)
                                    enabled: backend.walletInitialized && !backend.walletLocked
                                    contentItem: Text { text: parent.text; color: "#ffffff"; font.pixelSize: 12 }
                                    background: Rectangle {
                                        color: parent.enabled ? "#006633" : "#333"
                                        radius: 4
                                    }
                                }
                            }
                        }
                    }
                }
            }

            // ---- Transactions Tab ----
            Rectangle {
                color: "#0d1137"
                radius: 8
                border.color: "#1e2360"
                border.width: 1

                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 20
                    spacing: 12

                    Text {
                        text: "📋 Transaction History (" + backend.transactionCount + ")"
                        color: "#8890c0"
                        font.pixelSize: 16
                        font.bold: true
                    }

                    ListView {
                        id: txList
                        Layout.fillWidth: true
                        Layout.fillHeight: true
                        model: backend.getTransactionHistory()
                        clip: true
                        spacing: 6

                        delegate: Rectangle {
                            width: txList.width
                            height: 56
                            color: "#0f1745"
                            radius: 6
                            border.color: "#1a2b6a"

                            RowLayout {
                                anchors.fill: parent
                                anchors.margins: 10
                                spacing: 12

                                Rectangle {
                                    width: 32; height: 32; radius: 16
                                    color: modelData.signed ? "#1a3a2a" : "#3a2020"
                                    Text {
                                        anchors.centerIn: parent
                                        text: modelData.signed ? "✓" : "⏳"
                                        color: modelData.signed ? "#44ff88" : "#ff6644"
                                        font.pixelSize: 14
                                    }
                                }

                                ColumnLayout {
                                    spacing: 2
                                    Text {
                                        text: modelData.txid.substring(0, 16) + "..."
                                        color: "#e0e4ff"
                                        font.pixelSize: 12
                                        font.bold: true
                                    }
                                    Text {
                                        text: modelData.inputs + " in / " + modelData.outputs + " out"
                                        color: "#6670a0"
                                        font.pixelSize: 10
                                    }
                                }

                                Item { Layout.fillWidth: true }

                                Text {
                                    text: modelData.amount ? (modelData.amount / 100000000).toFixed(8) + " BTC" : ""
                                    color: "#00ff88"
                                    font.pixelSize: 12
                                }
                            }
                        }

                        Text {
                            anchors.centerIn: parent
                            text: "No transactions yet"
                            color: "#5560a0"
                            font.pixelSize: 14
                            visible: txList.count === 0
                        }
                    }

                    Button {
                        text: "📡 Broadcast Last Signed TX"
                        onClicked: backend.broadcastSampleTransaction()
                        contentItem: Text { text: parent.text; color: "#ffffff"; font.pixelSize: 12 }
                        background: Rectangle { color: "#0055cc"; radius: 4 }
                    }
                }
            }

            // ---- Backup Tab ----
            Rectangle {
                color: "#0d1137"
                radius: 8
                border.color: "#1e2360"
                border.width: 1

                Flickable {
                    anchors.fill: parent
                    anchors.margins: 20
                    contentHeight: backupColumn.height
                    clip: true

                    ColumnLayout {
                        id: backupColumn
                        width: parent.width
                        spacing: 16

                        Text {
                            text: "💾 Distributed Backup (Shamir's Secret Sharing)"
                            color: "#8890c0"
                            font.pixelSize: 16
                            font.bold: true
                        }
                        Text {
                            text: "Split your wallet seed into multiple encrypted shares. Require K of N shares to recover."
                            color: "#6670a0"
                            font.pixelSize: 12
                            wrapMode: Text.WordWrap
                            Layout.fillWidth: true
                        }

                        // Create backup
                        GroupBox {
                            title: "Create Backup"
                            Layout.fillWidth: true
                            background: Rectangle { color: "#0f1745"; radius: 6; border.color: "#1a2b6a" }
                            label: Text { text: "📦 Create New Backup"; color: "#8890c0"; font.pixelSize: 13 }

                            GridLayout {
                                columns: 2
                                rowSpacing: 8
                                columnSpacing: 12

                                Text { text: "Total Shares (N):"; color: "#8890c0"; font.pixelSize: 12 }
                                SpinBox {
                                    id: totalSharesSpin
                                    from: 2; to: 10; value: 5
                                    contentItem: Text { text: parent.value; color: "#e0e4ff" }
                                }
                                Text { text: "Threshold (K):"; color: "#8890c0"; font.pixelSize: 12 }
                                SpinBox {
                                    id: thresholdSpin
                                    from: 2; to: 10; value: 3
                                    contentItem: Text { text: parent.value; color: "#e0e4ff" }
                                }
                                Text { text: "Passphrase:"; color: "#8890c0"; font.pixelSize: 12 }
                                TextField {
                                    id: backupPassphraseField
                                    echoMode: TextInput.Password
                                    placeholderText: "Backup encryption passphrase"
                                    color: "#e0e4ff"
                                    background: Rectangle { color: "#0a0e27"; radius: 4; border.color: "#1e2360" }
                                }
                                Item {}
                                Button {
                                    text: "🔐 Create Backup"
                                    onClicked: backend.createBackup(totalSharesSpin.value, thresholdSpin.value, backupPassphraseField.text)
                                    contentItem: Text { text: parent.text; color: "#ffffff"; font.pixelSize: 12 }
                                    background: Rectangle { color: "#553399"; radius: 4 }
                                }
                            }
                        }

                        // Recover backup
                        GroupBox {
                            title: "Recover Wallet"
                            Layout.fillWidth: true
                            background: Rectangle { color: "#0f1745"; radius: 6; border.color: "#1a2b6a" }
                            label: Text { text: "🔓 Recover from Shares"; color: "#8890c0"; font.pixelSize: 13 }

                            ColumnLayout {
                                spacing: 8

                                TextArea {
                                    id: sharesTextArea
                                    Layout.fillWidth: true
                                    height: 80
                                    placeholderText: "Enter backup shares (one per line)..."
                                    color: "#e0e4ff"
                                    background: Rectangle { color: "#0a0e27"; radius: 4; border.color: "#1e2360" }
                                    wrapMode: TextEdit.Wrap
                                }
                                RowLayout {
                                    spacing: 12
                                    Text { text: "Threshold:"; color: "#8890c0"; font.pixelSize: 12 }
                                    SpinBox {
                                        id: recoverThresholdSpin
                                        from: 2; to: 10; value: 3
                                    }
                                    Text { text: "Passphrase:"; color: "#8890c0"; font.pixelSize: 12 }
                                    TextField {
                                        id: recoverPassphraseField
                                        echoMode: TextInput.Password
                                        placeholderText: "Passphrase"
                                        color: "#e0e4ff"
                                        background: Rectangle { color: "#0a0e27"; radius: 4; border.color: "#1e2360" }
                                    }
                                }
                                Button {
                                    text: "🔓 Recover"
                                    onClicked: {
                                        var lines = sharesTextArea.text.split('\n').filter(function(l) { return l.trim() !== '' })
                                        backend.recoverBackup(lines, recoverThresholdSpin.value, recoverPassphraseField.text)
                                    }
                                    contentItem: Text { text: parent.text; color: "#ffffff"; font.pixelSize: 12 }
                                    background: Rectangle { color: "#006633"; radius: 4 }
                                }
                            }
                        }

                        // Backup history
                        Text {
                            text: "Backup History"
                            color: "#8890c0"
                            font.pixelSize: 14
                            font.bold: true
                        }
                        Text {
                            text: backend.getBackupHistoryJson() || "No backups created yet"
                            color: "#5560a0"
                            font.pixelSize: 11
                            font.family: "monospace"
                            wrapMode: Text.WordWrap
                        }
                    }
                }
            }

            // ---- Settings Tab ----
            Rectangle {
                color: "#0d1137"
                radius: 8
                border.color: "#1e2360"
                border.width: 1

                ColumnLayout {
                    anchors.fill: parent
                    anchors.margins: 20
                    spacing: 16

                    Text {
                        text: "⚙ Settings"
                        color: "#8890c0"
                        font.pixelSize: 16
                        font.bold: true
                    }

                    GroupBox {
                        title: "Network"
                        Layout.fillWidth: true
                        background: Rectangle { color: "#0f1745"; radius: 6; border.color: "#1a2b6a" }
                        label: Text { text: "🌐 Network Configuration"; color: "#8890c0"; font.pixelSize: 13 }

                        GridLayout {
                            columns: 2
                            Text { text: "Status:"; color: "#8890c0"; font.pixelSize: 12 }
                            Text {
                                text: backend.networkConnected ? "✓ Connected" : "✗ Disconnected"
                                color: backend.networkConnected ? "#44ff88" : "#ff6644"
                                font.pixelSize: 12
                            }
                            Item {}
                            Button {
                                text: "🔄 Check Connection"
                                onClicked: backend.checkNetworkStatus()
                                contentItem: Text { text: parent.text; color: "#ffffff"; font.pixelSize: 11 }
                                background: Rectangle { color: "#1a4a7a"; radius: 4 }
                            }
                        }
                    }

                    GroupBox {
                        title: "Security"
                        Layout.fillWidth: true
                        background: Rectangle { color: "#0f1745"; radius: 6; border.color: "#1a2b6a" }
                        label: Text { text: "🛡 Security Status"; color: "#8890c0"; font.pixelSize: 13 }

                        GridLayout {
                            columns: 2
                            rowSpacing: 8
                            columnSpacing: 16
                            Text { text: "Level:"; color: "#8890c0"; font.pixelSize: 12 }
                            Text { text: backend.securityLevel; color: "#44ff88"; font.pixelSize: 12 }
                            Text { text: "Events:"; color: "#8890c0"; font.pixelSize: 12 }
                            Text { text: backend.securityEvents ? "⚠ " + backend.securityEvents : "✓ None"; color: backend.securityEvents ? "#ff6644" : "#44ff88"; font.pixelSize: 12 }
                            Text { text: "Wallet:"; color: "#8890c0"; font.pixelSize: 12 }
                            Text { text: backend.walletLocked ? "🔒 Locked" : "🔓 Unlocked"; color: backend.walletLocked ? "#cc99ff" : "#44ff88"; font.pixelSize: 12 }
                        }
                    }

                    GroupBox {
                        title: "Mnemonics"
                        Layout.fillWidth: true
                        visible: backend.walletInitialized
                        background: Rectangle { color: "#0f1745"; radius: 6; border.color: "#1a2b6a" }
                        label: Text { text: "🔑 Recovery Phrase"; color: "#8890c0"; font.pixelSize: 13 }

                        ColumnLayout {
                            spacing: 8
                            Text {
                                text: "⚠ Keep this phrase secret and offline. Anyone with these words can access your funds."
                                color: "#ff6644"
                                font.pixelSize: 11
                                wrapMode: Text.WordWrap
                                Layout.fillWidth: true
                            }
                            Rectangle {
                                Layout.fillWidth: true
                                height: 60
                                color: "#0a0e27"
                                radius: 4
                                border.color: "#3a2040"
                                TextArea {
                                    id: mnemonicText
                                    anchors.fill: parent
                                    anchors.margins: 8
                                    text: backend.mnemonic
                                    color: "#cc99ff"
                                    font.pixelSize: 11
                                    font.family: "monospace"
                                    readOnly: true
                                    wrapMode: TextEdit.Wrap
                                    background: Rectangle { color: "transparent" }
                                }
                            }
                            Button {
                                text: "📋 Copy Mnemonic"
                                onClicked: mnemonicText.selectAll()
                                contentItem: Text { text: parent.text; color: "#ffffff"; font.pixelSize: 11 }
                                background: Rectangle { color: "#3a2550"; radius: 4 }
                            }
                        }
                    }

                    GroupBox {
                        title: "About"
                        Layout.fillWidth: true
                        background: Rectangle { color: "#0f1745"; radius: 6; border.color: "#1a2b6a" }
                        label: Text { text: "ℹ About"; color: "#8890c0"; font.pixelSize: 13 }

                        ColumnLayout {
                            spacing: 4
                            Text { text: "Embedded Hardware Wallet v1.0.0"; color: "#e0e4ff"; font.pixelSize: 13; font.bold: true }
                            Text { text: "C++20 / Qt5 QML / OpenSSL 3.0"; color: "#6670a0"; font.pixelSize: 11 }
                            Text { text: "ECC: secp256k1 | AES-256-GCM | BIP39/32"; color: "#6670a0"; font.pixelSize: 11 }
                            Text { text: "Shamir's Secret Sharing | HD Wallet"; color: "#6670a0"; font.pixelSize: 11 }
                        }
                    }

                    Item { Layout.fillHeight: true }
                }
            }
        }
    }

    // ---- Dialogs ----
    Dialog {
        id: createWalletDialog
        title: "Create New Wallet"
        modal: true
        standardButtons: Dialog.Ok | Dialog.Cancel

        background: Rectangle { color: "#0d1137"; radius: 8; border.color: "#1e2360" }

        ColumnLayout {
            spacing: 12
            width: 400

            Text {
                text: "Enter a passphrase to encrypt your wallet keys:"
                color: "#8890c0"
                font.pixelSize: 13
                wrapMode: Text.WordWrap
                Layout.fillWidth: true
            }
            TextField {
                id: createPassphraseField
                Layout.fillWidth: true
                echoMode: TextInput.Password
                placeholderText: "Wallet passphrase (required)"
                color: "#e0e4ff"
                background: Rectangle { color: "#0a0e27"; radius: 4; border.color: "#1e2360" }
            }
        }

        onAccepted: {
            backend.createWallet(createPassphraseField.text)
        }
    }

    Dialog {
        id: recoverWalletDialog
        title: "Recover Wallet from Mnemonic"
        modal: true
        standardButtons: Dialog.Ok | Dialog.Cancel

        background: Rectangle { color: "#0d1137"; radius: 8; border.color: "#1e2360" }

        ColumnLayout {
            spacing: 12
            width: 400

            Text {
                text: "Enter your BIP39 recovery phrase (24 words):"
                color: "#8890c0"
                font.pixelSize: 13
                wrapMode: Text.WordWrap
                Layout.fillWidth: true
            }
            TextArea {
                id: mnemonicRecoveryField
                Layout.fillWidth: true
                height: 100
                placeholderText: "word1 word2 word3 ... word24"
                color: "#e0e4ff"
                wrapMode: TextEdit.Wrap
                background: Rectangle { color: "#0a0e27"; radius: 4; border.color: "#1e2360" }
            }
            TextField {
                id: recoverPassphraseField2
                Layout.fillWidth: true
                echoMode: TextInput.Password
                placeholderText: "Passphrase (if used during creation)"
                color: "#e0e4ff"
                background: Rectangle { color: "#0a0e27"; radius: 4; border.color: "#1e2360" }
            }
        }

        onAccepted: {
            backend.recoverWalletFromMnemonic(mnemonicRecoveryField.text, recoverPassphraseField2.text)
        }
    }

    // ---- Connections to backend signals ----
    Connections {
        target: AppCore
        function onOperationStarted(op) { statusBar.children[0].children[0].text = op }
        function onOperationCompleted(op) { /* handled */ }
        function onErrorOccurred(err) { /* shown in status bar */ }
        function onBackupCreated(shares) {
            backupResultDialog.shares = shares
            backupResultDialog.open()
        }
    }

    Dialog {
        id: backupResultDialog
        title: "Backup Created Successfully"
        modal: true
        property var shares: []

        background: Rectangle { color: "#0d1137"; radius: 8; border.color: "#1e2360" }

        ColumnLayout {
            spacing: 12
            width: 450

            Text {
                text: "Store these shares securely. Each share is encrypted."
                color: "#44ff88"
                font.pixelSize: 13
                wrapMode: Text.WordWrap
                Layout.fillWidth: true
            }
            TextArea {
                id: backupResultText
                Layout.fillWidth: true
                height: 150
                readOnly: true
                color: "#cc99ff"
                font.pixelSize: 10
                font.family: "monospace"
                wrapMode: TextEdit.Wrap
                background: Rectangle { color: "#0a0e27"; radius: 4; border.color: "#1e2360" }

                Component.onCompleted: {
                    var text = ""
                    for (var i = 0; i < backupResultDialog.shares.length; i++) {
                        text += "Share " + (i+1) + ": " + backupResultDialog.shares[i].substring(0, 40) + "...\n\n"
                    }
                    backupResultText.text = text
                }
            }
        }
    }
}
