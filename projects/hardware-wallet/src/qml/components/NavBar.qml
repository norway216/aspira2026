import QtQuick 2.15
import QtQuick.Controls 2.15
import ".."

Rectangle {
    id: navBar
    width: parent ? parent.width : 400
    height: 56
    color: Theme.bgCard

    Row {
        anchors.fill: parent
        Repeater {
            model: [
                { name: "Home", icon: "🏠", page: "qrc:/qml/pages/DashboardPage.qml" },
                { name: "Send", icon: "📤", page: "qrc:/qml/pages/SendPage.qml" },
                { name: "History", icon: "📋", page: "qrc:/qml/pages/HistoryPage.qml" },
                { name: "Settings", icon: "⚙", page: "qrc:/qml/pages/SettingsPage.qml" }
            ]

            Rectangle {
                width: navBar.width / 4
                height: navBar.height
                color: "transparent"

                Column {
                    anchors.centerIn: parent
                    spacing: 2
                    Text {
                        text: modelData.icon
                        font.pixelSize: 20
                        anchors.horizontalCenter: parent.horizontalCenter
                    }
                    Text {
                        text: modelData.name
                        font.pixelSize: 10
                        color: Theme.textSecondary
                        anchors.horizontalCenter: parent.horizontalCenter
                    }
                }

                MouseArea {
                    anchors.fill: parent
                    onClicked: {
                        var s = stackView
                        while (s.depth > 1) s.pop()
                        s.replace(modelData.page)
                    }
                }
            }
        }
    }
}
