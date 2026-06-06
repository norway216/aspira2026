/* ===== Login Page — Overlay Mode ===== */
var _loginInitialized = false;

function init_login() {
    if (_loginInitialized) return;

    const form = document.getElementById('loginForm');
    const errorEl = document.getElementById('loginError');
    const usernameInput = document.getElementById('loginUsername');
    const loginBtn = document.getElementById('loginBtn');
    const spinner = document.getElementById('loginSpinner');

    if (!form) return;

    _loginInitialized = true;

    // Focus username
    setTimeout(function () {
        const u = document.getElementById('loginUsername');
        if (u) u.focus();
    }, 200);

    form.addEventListener('submit', async function (e) {
        e.preventDefault();

        const username = document.getElementById('loginUsername').value.trim();
        const password = document.getElementById('loginPassword').value;
        const btn = document.getElementById('loginBtn');
        const spn = document.getElementById('loginSpinner');
        const errEl = document.getElementById('loginError');

        if (errEl) errEl.textContent = '';

        if (!username || !password) {
            if (errEl) errEl.textContent = '请输入用户名和密码';
            return;
        }

        // Show loading
        if (btn) { btn.disabled = true; btn.classList.add('loading'); }
        if (spn) spn.classList.remove('hidden');

        try {
            const data = await API.post('/api/v1/auth/login', { username: username, password: password });
            API.setToken(data.access_token || data.token);
            App.user = data.user || { username: username, role: data.role || 'admin' };
            localStorage.setItem('user', JSON.stringify(App.user));
            App.showApp();
            WS.connect();
            _loginInitialized = false; // Reset for next login
        } catch (e) {
            if (errEl) {
                errEl.textContent = e.message || '登录失败，请检查用户名和密码';
                if (typeof Animations !== 'undefined') Animations.shake(errEl);
            }
        } finally {
            if (btn) { btn.disabled = false; btn.classList.remove('loading'); }
            if (spn) spn.classList.add('hidden');
        }
    });

    // Allow Enter key on password field
    const pwdInput = document.getElementById('loginPassword');
    if (pwdInput) {
        pwdInput.addEventListener('keydown', function (e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                form.dispatchEvent(new Event('submit', { cancelable: true }));
            }
        });
    }
}
