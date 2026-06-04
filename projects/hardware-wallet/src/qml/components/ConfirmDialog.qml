import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Dialog {
    id: confirmDialog
    property string confirmTitle: "Confirm"
    property string confirmMessage: "Are you sure?"
    property string confirmText: "Confirm"
    property string cancelText: "Cancel"
    signal accepted
    signal rejected

    modal: true
    anchors.centerIn: parent
    width: 320
    height: 200

    background: Rectangle {
        color: Theme.bgCard
        radius: Theme.radiusLG
    }

    Column {
        anchors.fill: parent
        anchors.margins: Theme.spacingMD
        spacing: Theme.spacingMD

        Text {
            text: confirmTitle
            color: Theme.textPrimary
            font.pixelSize: Theme.fontSizeLG
            font.bold: true
            anchors.horizontalCenter: parent.horizontalCenter
        }

        Text {
            text: confirmMessage
            color: Theme.textSecondary
            font.pixelSize: Theme.fontSizeMD
            wrapMode: Text.Wrap
            width: parent.width
            horizontalAlignment: Text.AlignHCenter
        }

        Row {
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: Theme.spacingMD

            Button {
                text: cancelText
                width: 120; height: 40
                background: Rectangle { color: Theme.bgSurface; radius: Theme.radiusSM }
                contentItem: Text { text: cancelText; color: Theme.textPrimary; horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter }
                onClicked: { confirmDialog.rejected(); confirmDialog.close() }
            }

            Button {
                text: confirmText
                width: 120; height: 40
                background: Rectangle { color: Theme.primary; radius: Theme.radiusSM }
                contentItem: Text { text: confirmText; color: Theme.textPrimary; horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter }
                onClicked: { confirmDialog.accepted(); confirmDialog.close() }
            }
        }
    }
}
