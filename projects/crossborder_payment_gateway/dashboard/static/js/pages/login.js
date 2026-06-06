/* ===== Login Page ===== */
function init_login() {
    const form = document.getElementById('loginForm');
    const errorEl = document.getElementById('loginError');
    const usernameInput = document.getElementById('loginUsername');
    const passwordInput = document.getElementById('loginPassword');
    const loginBtn = document.getElementById('loginBtn');
    const spinner = document.getElementById('loginSpinner');

    if (!form) return;

    // Remove old listener by cloning
    const newForm = form.cloneNode(true);
    form.parentNode.replaceChild(newForm, form);

    // Focus username
    setTimeout(function () {
        const u = document.getElementById('loginUsername');
        if (u) u.focus();
    }, 100);

    newForm.addEventListener('submit', async function (e) {
        e.preventDefault();

        const username = document.getElementById('loginUsername').value.trim();
        const password = document.getElementById('loginPassword').value;
        const btn = document.getElementById('loginBtn');
        const spn = document.getElementById('loginSpinner');

        if (errorEl) errorEl.textContent = '';

        if (!username || !password) {
            if (errorEl) errorEl.textContent = '请输入用户名和密码';
            return;
        }

        // Show loading
        if (btn) btn.disabled = true;
        if (spn) spn.classList.remove('hidden');

        try {
            const data = await API.post('/api/v1/auth/login', { username: username, password: password });
            API.setToken(data.access_token || data.token);
            App.user = data.user || { username: username, role: data.role || 'admin' };
            localStorage.setItem('user', JSON.stringify(App.user));
            App.showApp();
            WS.connect();
        } catch (e) {
            if (errorEl) errorEl.textContent = e.message || '登录失败，请检查用户名和密码';
            Toast.error(e.message || '登录失败');
        } finally {
            if (btn) btn.disabled = false;
            if (spn) spn.classList.add('hidden');
        }
    });
}
