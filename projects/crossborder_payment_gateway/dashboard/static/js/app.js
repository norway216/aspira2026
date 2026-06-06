/* ===== Aspira Pay SPA Router & App Shell ===== */
const App = {
    currentPage: null,
    user: null,
    transitioning: false,

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

        // Handle keyboard shortcuts
        document.addEventListener('keydown', function (e) {
            // Escape to close modals
            if (e.key === 'Escape') {
                const overlay = document.getElementById('modal-overlay');
                if (overlay && !overlay.classList.contains('hidden')) {
                    closeModal();
                }
                // Close sidebar on mobile
                const sidebar = document.getElementById('sidebar');
                if (sidebar && sidebar.classList.contains('open') && window.innerWidth <= 768) {
                    sidebar.classList.remove('open');
                    const overlay = document.getElementById('sidebar-overlay');
                    if (overlay) overlay.classList.remove('active');
                }
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
        if (App.transitioning) return;

        params = params || {};

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
        if (history.state && history.state.page === page) {
            history.replaceState({ page: page, params: params }, '', '#' + page);
        } else {
            history.pushState({ page: page, params: params }, '', '#' + page);
        }

        // Smooth page transition
        App.transitionTo(page, params);
    },

    transitionTo(page, params) {
        App.transitioning = true;

        // Run cleanup for current page
        if (App.currentCleanup && typeof App.currentCleanup === 'function') {
            try { App.currentCleanup(); } catch (e) { }
            App.currentCleanup = null;
        }

        const prevPage = App.currentPage;
        const fromEl = prevPage ? document.getElementById('page-' + prevPage) : null;
        const toEl = document.getElementById('page-' + page);

        // Hide all other pages first
        document.querySelectorAll('.page-content').forEach(function (p) {
            if (p !== toEl) {
                p.style.display = 'none';
                p.style.animation = '';
            }
        });

        // Show target page with animation
        if (toEl) {
            toEl.style.display = 'block';
            toEl.style.animation = 'pageEnter 0.4s cubic-bezier(0.34, 1.56, 0.64, 1) forwards';

            // Re-trigger scroll reveal for new page content
            setTimeout(function () {
                const cards = toEl.querySelectorAll('.card:not([data-revealed]), .stat-card:not([data-revealed]), .account-card:not([data-revealed]), .filter-bar:not([data-revealed])');
                cards.forEach(function (el, i) {
                    el.setAttribute('data-revealed', 'true');
                    el.style.opacity = '0';
                    el.style.transform = 'translateY(16px)';
                    el.style.transition = 'opacity 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) ' + (i * 0.04) + 's, transform 0.5s cubic-bezier(0.34, 1.56, 0.64, 1) ' + (i * 0.04) + 's';
                    requestAnimationFrame(function () {
                        el.style.opacity = '1';
                        el.style.transform = 'translateY(0)';
                    });
                });
            }, 50);
        }

        App.currentPage = page;

        // Call page init function
        setTimeout(function () {
            App.transitioning = false;

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

            if (typeof initFn === 'function') {
                initFn(params);
            }
        }, 100);
    },

    showPage(page, params) {
        // Legacy method - use transitionTo
        params = params || {};

        // Hide all pages
        document.querySelectorAll('.page-content').forEach(function (p) {
            p.style.display = 'none';
            p.style.animation = '';
        });

        // Show target page
        const pageEl = document.getElementById('page-' + page);
        if (pageEl) {
            pageEl.style.display = 'block';
            pageEl.style.animation = 'pageEnter 0.35s cubic-bezier(0.34, 1.56, 0.64, 1) forwards';
            App.currentPage = page;

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
        // Animate logout
        const mainContent = document.getElementById('main-content');
        if (mainContent) {
            mainContent.style.opacity = '0';
            mainContent.style.transform = 'scale(0.98)';
            mainContent.style.transition = 'opacity 0.3s ease, transform 0.3s ease';
        }

        setTimeout(function () {
            API.clearToken();
            WS.disconnect();
            App.user = null;
            App.showLogin();

            if (mainContent) {
                mainContent.style.opacity = '1';
                mainContent.style.transform = 'scale(1)';
            }

            history.pushState({ page: 'login' }, '', '#login');
        }, 250);
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

    // Notification button click
    const notifBtn = document.getElementById('notifBtn');
    if (notifBtn) {
        notifBtn.addEventListener('click', function () {
            const badge = document.getElementById('notifBadge');
            if (badge && !badge.classList.contains('hidden')) {
                Animations.bounceBadge(badge);
                Toast.info('暂无新通知');
            } else {
                Toast.info('暂无新通知');
            }
        });
    }

    // Initialize the app
    App.init();
});
