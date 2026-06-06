/* ===== Aspira Pay SPA Router & App Shell ===== */
const App = {
    currentPage: null,
    user: null,

    init() {
        // Handle sidebar nav clicks
        document.querySelectorAll('#sidebar .nav-item').forEach(function (item) {
            item.addEventListener('click', function (e) {
                e.preventDefault();
                const page = this.dataset.page;
                if (page) App.navigate(page);
            });
        });

        // Check for saved auth token
        const token = API.getToken();
        if (token) {
            App.validateToken();
        } else {
            App.showLogin();
        }

        // Handle browser back/forward
        window.addEventListener('popstate', function (e) {
            if (e.state && e.state.page) {
                App.showPage(e.state.page, e.state.params || {});
            }
        });
    },

    async validateToken() {
        try {
            const data = await API.get('/api/v1/auth/profile');
            App.user = data.user || data;
            App.showApp();
            WS.connect();
        } catch (e) {
            API.clearToken();
            App.user = null;
            App.showLogin();
        }
    },

    showLogin() {
        const sidebar = document.getElementById('sidebar');
        const header = document.getElementById('header');
        if (sidebar) sidebar.style.display = 'none';
        if (header) header.style.display = 'none';
        App.showPage('login');
    },

    showApp() {
        const sidebar = document.getElementById('sidebar');
        const header = document.getElementById('header');
        if (sidebar) sidebar.style.display = 'flex';
        if (header) header.style.display = 'flex';
        App.updateUserInfo();
        App.navigate('dashboard');
    },

    navigate(page, params) {
        params = params || {};
        App.showPage(page, params);

        // Update active nav item
        document.querySelectorAll('#sidebar .nav-item').forEach(function (item) {
            item.classList.toggle('active', item.dataset.page === page);
        });

        // Update page title in header
        const titles = {
            dashboard: '仪表盘',
            transactions: '交易管理',
            'transaction-detail': '交易详情',
            accounts: '账户管理',
            merchants: '商户管理',
            audit: '审计日志',
            settings: '系统设置'
        };
        const titleEl = document.getElementById('page-title');
        if (titleEl) titleEl.textContent = titles[page] || page;

        // Push state for back/forward
        history.pushState({ page: page, params: params }, '', '#' + page);
    },

    showPage(page, params) {
        params = params || {};

        // Hide all pages
        document.querySelectorAll('.page-content').forEach(function (p) {
            p.style.display = 'none';
        });

        // Show target page
        const pageEl = document.getElementById('page-' + page);
        if (pageEl) {
            pageEl.style.display = 'block';
            App.currentPage = page;

            // Call page init function if exists
            var initFn = null;
            switch (page) {
                case 'login':
                    initFn = init_login;
                    break;
                case 'dashboard':
                    initFn = init_dashboard;
                    break;
                case 'transactions':
                    initFn = init_transactions;
                    break;
                case 'transaction-detail':
                    initFn = init_transaction_detail;
                    break;
                case 'accounts':
                    initFn = init_accounts;
                    break;
                case 'merchants':
                    initFn = init_merchants;
                    break;
                case 'audit':
                    initFn = init_audit;
                    break;
                case 'settings':
                    initFn = init_settings;
                    break;
            }

            // Cleanup previous page if needed
            if (App.currentCleanup && typeof App.currentCleanup === 'function') {
                try { App.currentCleanup(); } catch (e) { }
                App.currentCleanup = null;
            }

            if (typeof initFn === 'function') {
                initFn(params);
            }
        }
    },

    updateUserInfo() {
        if (App.user) {
            const nameEl = document.getElementById('user-name');
            const roleEl = document.getElementById('user-role');
            const initialsEl = document.getElementById('userInitials');
            if (nameEl) nameEl.textContent = App.user.username || App.user.name || '用户';
            if (roleEl) roleEl.textContent = App.user.role || '管理员';
            if (initialsEl) {
                const name = App.user.username || App.user.name || 'A';
                initialsEl.textContent = name.charAt(0).toUpperCase();
            }
        }
    },

    logout() {
        API.clearToken();
        WS.disconnect();
        App.user = null;
        App.showLogin();
        history.pushState({ page: 'login' }, '', '#login');
    },

    // Register cleanup function for current page
    setCleanup(fn) {
        App.currentCleanup = fn;
    }
};

// Logout button in header
document.addEventListener('DOMContentLoaded', function () {
    const logoutBtn = document.querySelector('.logout-btn');
    if (logoutBtn) {
        logoutBtn.addEventListener('click', function (e) {
            e.stopPropagation();
            App.logout();
        });
    }

    // Also initialize the app
    App.init();
});
