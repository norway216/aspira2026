/* ===== API Client ===== */
const API = {
    baseURL: '',
    token: null,

    setToken(token) {
        this.token = token;
        localStorage.setItem('auth_token', token);
    },

    getToken() {
        return this.token || localStorage.getItem('auth_token');
    },

    clearToken() {
        this.token = null;
        localStorage.removeItem('auth_token');
    },

    async request(method, path, body) {
        const headers = { 'Content-Type': 'application/json' };
        const token = this.getToken();
        if (token) headers['Authorization'] = `Bearer ${token}`;

        const opts = { method, headers };
        if (body !== undefined && body !== null) opts.body = JSON.stringify(body);

        const resp = await fetch(this.baseURL + path, opts);
        const data = await resp.json();

        if (!resp.ok) {
            const err = new Error(data.error || data.message || `Request failed (${resp.status})`);
            err.status = resp.status;
            err.data = data;
            throw err;
        }
        return data;
    },

    get(path) {
        return this.request('GET', path);
    },

    post(path, body) {
        return this.request('POST', path, body);
    },

    put(path, body) {
        return this.request('PUT', path, body);
    },

    del(path) {
        return this.request('DELETE', path);
    }
};
