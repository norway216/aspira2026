/* ===== WebSocket Client ===== */
const WS = {
    conn: null,
    reconnectTimer: null,
    messageHandlers: {},
    connected: false,

    connect() {
        if (this.conn && (this.conn.readyState === WebSocket.OPEN || this.conn.readyState === WebSocket.CONNECTING)) {
            return;
        }

        const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
        const url = `${protocol}//${location.host}/ws`;
        const token = API.getToken();
        const wsUrl = token ? `${url}?token=${encodeURIComponent(token)}` : url;

        try {
            this.conn = new WebSocket(wsUrl);
        } catch (e) {
            console.error('WebSocket creation failed:', e);
            this.scheduleReconnect();
            return;
        }

        this.conn.onopen = () => {
            console.log('[WS] Connected');
            this.connected = true;
            this.updateStatusUI('connected');
            if (this.reconnectTimer) {
                clearTimeout(this.reconnectTimer);
                this.reconnectTimer = null;
            }
            // Subscribe to channels
            this.send({ type: 'subscribe', channel: 'transactions' });
            this.send({ type: 'subscribe', channel: 'engine' });
            this.send({ type: 'subscribe', channel: 'dashboard' });
        };

        this.conn.onmessage = (event) => {
            try {
                const msg = JSON.parse(event.data);
                const handlers = this.messageHandlers[msg.type] || [];
                handlers.forEach(fn => {
                    try { fn(msg.data, msg); } catch (e) { console.error('[WS] Handler error:', e); }
                });
            } catch (e) {
                console.error('[WS] Parse error:', e);
            }
        };

        this.conn.onclose = (event) => {
            console.log('[WS] Disconnected (code: ' + event.code + ')');
            this.connected = false;
            this.updateStatusUI('disconnected');
            this.scheduleReconnect();
        };

        this.conn.onerror = (err) => {
            console.error('[WS] Error:', err);
            this.conn.close();
        };
    },

    disconnect() {
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        if (this.conn) {
            this.conn.onclose = null;
            this.conn.close();
            this.conn = null;
        }
        this.connected = false;
        this.updateStatusUI('disconnected');
    },

    scheduleReconnect() {
        if (this.reconnectTimer) return;
        this.updateStatusUI('connecting');
        this.reconnectTimer = setTimeout(() => {
            this.reconnectTimer = null;
            this.connect();
        }, 3000);
    },

    on(type, fn) {
        if (!this.messageHandlers[type]) {
            this.messageHandlers[type] = [];
        }
        this.messageHandlers[type].push(fn);
        // Return unsubscribe function
        return () => {
            const idx = (this.messageHandlers[type] || []).indexOf(fn);
            if (idx !== -1) this.messageHandlers[type].splice(idx, 1);
        };
    },

    off(type, fn) {
        const handlers = this.messageHandlers[type];
        if (handlers) {
            const idx = handlers.indexOf(fn);
            if (idx !== -1) handlers.splice(idx, 1);
        }
    },

    send(data) {
        if (this.conn && this.conn.readyState === WebSocket.OPEN) {
            this.conn.send(JSON.stringify(data));
        }
    },

    updateStatusUI(status) {
        const el = document.getElementById('wsStatus');
        if (!el) return;
        const dot = el.querySelector('.ws-dot');
        const label = el.querySelector('span:last-child');
        if (!dot || !label) return;

        dot.className = 'ws-dot';
        if (status === 'connected') {
            dot.classList.add('ws-connected');
            label.textContent = '已连接';
        } else if (status === 'connecting') {
            dot.classList.add('ws-connecting');
            label.textContent = '重连中...';
        } else {
            dot.classList.add('ws-disconnected');
            label.textContent = '未连接';
        }
    }
};
