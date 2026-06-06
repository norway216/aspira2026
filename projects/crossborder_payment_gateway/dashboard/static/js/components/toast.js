/* ===== Toast Notification System ===== */
const Toast = {
    maxVisible: 5,
    activeToasts: [],

    show(message, type, duration) {
        if (!message) return;
        type = type || 'info';
        duration = duration || 4000;

        // Remove oldest if at max
        if (this.activeToasts.length >= this.maxVisible) {
            const oldest = this.activeToasts.shift();
            if (oldest && oldest.parentNode) {
                oldest.remove();
            }
        }

        const container = document.getElementById('toast-container');
        if (!container) return;

        const icons = {
            success: '<svg class="toast-icon" viewBox="0 0 20 20" fill="none"><circle cx="10" cy="10" r="9" stroke="#30D158" stroke-width="1.5"/><path d="M6 10L9 13L14 7" stroke="#30D158" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/></svg>',
            error: '<svg class="toast-icon" viewBox="0 0 20 20" fill="none"><circle cx="10" cy="10" r="9" stroke="#FF453A" stroke-width="1.5"/><path d="M7 7L13 13M13 7L7 13" stroke="#FF453A" stroke-width="1.5" stroke-linecap="round"/></svg>',
            warning: '<svg class="toast-icon" viewBox="0 0 20 20" fill="none"><circle cx="10" cy="10" r="9" stroke="#FF9F0A" stroke-width="1.5"/><path d="M10 6V11M10 14V14.01" stroke="#FF9F0A" stroke-width="1.5" stroke-linecap="round"/></svg>',
            info: '<svg class="toast-icon" viewBox="0 0 20 20" fill="none"><circle cx="10" cy="10" r="9" stroke="#007AFF" stroke-width="1.5"/><path d="M10 6V11M10 14V14.01" stroke="#007AFF" stroke-width="1.5" stroke-linecap="round"/></svg>'
        };

        const toast = document.createElement('div');
        toast.className = 'toast toast-' + type;
        toast.innerHTML = '' +
            (icons[type] || icons.info) +
            '<span class="toast-message">' + escapeHtml(message) + '</span>' +
            '<button class="toast-close" onclick="Toast.dismiss(this.parentElement)">' +
            '<svg width="14" height="14" viewBox="0 0 14 14" fill="none"><path d="M3 3L11 11M11 3L3 11" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>' +
            '</button>';

        container.appendChild(toast);
        this.activeToasts.push(toast);

        // Auto dismiss
        if (duration > 0) {
            setTimeout(() => {
                this.dismiss(toast);
            }, duration);
        }

        return toast;
    },

    dismiss(toast) {
        if (!toast || !toast.parentNode) return;
        toast.style.transition = 'all 0.3s ease';
        toast.style.opacity = '0';
        toast.style.transform = 'translateX(100%)';
        setTimeout(() => {
            if (toast.parentNode) {
                toast.remove();
                const idx = this.activeToasts.indexOf(toast);
                if (idx !== -1) this.activeToasts.splice(idx, 1);
            }
        }, 300);
    },

    success(msg, duration) {
        return this.show(msg, 'success', duration);
    },

    error(msg, duration) {
        return this.show(msg, 'error', duration || 5000);
    },

    warning(msg, duration) {
        return this.show(msg, 'warning', duration);
    },

    info(msg, duration) {
        return this.show(msg, 'info', duration);
    }
};

// Backward compatibility
function showToast(message, type, duration) {
    return Toast.show(message, type, duration);
}
