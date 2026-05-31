import QtQuick 2.15
import QtQuick.Controls 2.15
import QtQuick.Window 2.15
import "pages"
import "components"

ApplicationWindow {
    id: appWindow
    visible: true
    width: 480
    height: 854
    title: "RK3568 Hardware Wallet"
    color: "#121212"

    property string currentPage: "login"

    StackView {
        id: stackView
        anchors.fill: parent
        initialItem: loginPageComp
    }

    Component {
        id: loginPageComp
        LoginPage {}
    }

    // Navigate to dashboard on login success
    Connections {
        target: LoginViewModel
        function onLoginSucceeded() {
            console.log("Login succeeded, navigating to dashboard")
            DashboardViewModel.refresh()
            stackView.replace(dashboardPageComp)
        }
    }

    // Navigate to login after registration success
    Connections {
        target: RegisterViewModel
        function onRegisterSucceeded() {
            console.log("Registration succeeded, going to login")
            stackView.replace(loginPageComp)
        }
    }

    // Navigate to login on logout
    Connections {
        target: SettingsViewModel
        function onLogoutCompleted() {
            console.log("Logout completed")
            stackView.replace(loginPageComp)
        }
    }

    // Navigate to login on session expiry
    Connections {
        target: AuthService
        function onSessionExpired() {
            console.log("Session expired - showing lock screen")
            // Push lock screen, clear all pages beneath
            stackView.replace(lockPageComp)
        }
        function onAutoLockWarning(secondsLeft) {
            console.log("Auto-lock warning:", secondsLeft, "seconds remaining")
        }
    }

    // Lock screen component
    Component {
        id: lockPageComp
        LockScreenPage {}
    }

    // Send success -> go back to dashboard
    Connections {
        target: SendViewModel
        function onSendSucceeded(txId) {
            console.log("Send succeeded:", txId)
            DashboardViewModel.refresh()
            stackView.replace(dashboardPageComp)
        }
    }

    // Backup success
    Connections {
        target: BackupViewModel
        function onExportSucceeded() {
            console.log("Backup succeeded")
        }
    }

    // Restore success -> go to login
    Connections {
        target: RestoreViewModel
        function onImportSucceeded(usersRestored) {
            console.log("Restore succeeded:", usersRestored, "users")
            stackView.replace(loginPageComp)
        }
    }

    Component {
        id: dashboardPageComp
        DashboardPage {}
    }

    Component.onCompleted: {
        console.log("Hardware Wallet launched")
    }
}
