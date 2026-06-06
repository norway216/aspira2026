/* ===== Stat Card Updater with Animated Counters ===== */
const StatCard = {
    // Track current values for smooth transitions
    _currentValues: {},

    /**
     * Update a stat card's value with animation
     * @param {string} id - element ID (e.g., 'stat-tps')
     * @param {object} opts - { value, label?, trend?, trendUp?, format?, animated? }
     *   format: 'number', 'currency', 'percent', 'tps', 'custom'
     */
    update(id, opts) {
        const card = document.getElementById(id);
        if (!card) return;

        opts = opts || {};
        const animated = opts.animated !== false;
        const duration = opts.duration || 600;

        // Update value with smooth counter animation
        if (opts.value !== undefined) {
            const valueEl = card.querySelector('.stat-value');
            if (valueEl) {
                const newVal = parseFloat(opts.value) || 0;
                const oldVal = StatCard._currentValues[id] !== undefined
                    ? StatCard._currentValues[id]
                    : newVal;
                const format = opts.format || 'number';

                if (animated && oldVal !== newVal && typeof Animations !== 'undefined') {
                    // Build formatter based on format type
                    var formatter;
                    switch (format) {
                        case 'currency':
                            formatter = function (v) {
                                if (v >= 100000000) return '¥' + (v / 100000000).toFixed(1) + '亿';
                                if (v >= 10000) return '¥' + (v / 10000).toFixed(1) + '万';
                                return '¥' + v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
                            };
                            break;
                        case 'usd':
                            formatter = function (v) {
                                if (v >= 1000000) return '$' + (v / 1000000).toFixed(2) + 'M';
                                if (v >= 1000) return '$' + (v / 1000).toFixed(1) + 'K';
                                return '$' + v.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
                            };
                            break;
                        case 'percent':
                            formatter = function (v) { return v.toFixed(1) + '%'; };
                            break;
                        case 'tps':
                            formatter = function (v) { return v.toFixed(1); };
                            break;
                        default:
                            formatter = function (v) {
                                if (v >= 1000000) return (v / 1000000).toFixed(1) + 'M';
                                if (v >= 1000) return (v / 1000).toFixed(1) + 'K';
                                return Math.round(v).toLocaleString();
                            };
                    }

                    Animations.animateValue(valueEl, oldVal, newVal, duration, formatter);
                } else {
                    // Fallback: instant update with pulse
                    var formatted = formatStatValue(opts.value, format);
                    if (valueEl.textContent !== formatted) {
                        valueEl.textContent = formatted;
                        valueEl.classList.remove('pulse');
                        void valueEl.offsetWidth;
                        valueEl.classList.add('pulse');
                    }
                }

                StatCard._currentValues[id] = newVal;
            }
        }

        // Update label
        if (opts.label !== undefined) {
            const labelEl = card.querySelector('.stat-label');
            if (labelEl) {
                labelEl.textContent = opts.label;
            }
        }

        // Update trend
        if (opts.trend !== undefined) {
            let trendEl = card.querySelector('.stat-trend');
            if (!trendEl && opts.trend) {
                trendEl = document.createElement('div');
                trendEl.className = 'stat-trend';
                card.appendChild(trendEl);
            }
            if (trendEl) {
                if (opts.trend) {
                    const trendUp = opts.trendUp !== false;
                    trendEl.className = 'stat-trend ' + (trendUp ? 'trend-up' : 'trend-down');
                    trendEl.innerHTML = (trendUp ? '▲' : '▼') + ' ' + escapeHtml(String(opts.trend));
                    trendEl.style.display = '';
                    // Animate trend appearance
                    trendEl.style.animation = 'none';
                    void trendEl.offsetWidth;
                    trendEl.style.animation = 'slideUpFade 0.3s cubic-bezier(0.34, 1.56, 0.64, 1)';
                } else {
                    trendEl.style.display = 'none';
                }
            }
        }
    },

    // Reset tracked values (called on page cleanup)
    reset() {
        StatCard._currentValues = {};
    }
};

function formatStatValue(value, format) {
    if (value === undefined || value === null) return '--';
    var num = parseFloat(value);

    switch (format) {
        case 'currency':
            if (num >= 100000000) {
                return '¥' + (num / 100000000).toFixed(1) + '亿';
            } else if (num >= 10000) {
                return '¥' + (num / 10000).toFixed(1) + '万';
            }
            return '¥' + num.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
        case 'usd':
            if (num >= 1000000) {
                return '$' + (num / 1000000).toFixed(2) + 'M';
            } else if (num >= 1000) {
                return '$' + (num / 1000).toFixed(1) + 'K';
            }
            return '$' + num.toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 });
        case 'percent':
            return num.toFixed(1) + '%';
        case 'tps':
            return num.toFixed(1);
        default:
            if (num >= 1000000) {
                return (num / 1000000).toFixed(1) + 'M';
            } else if (num >= 1000) {
                return (num / 1000).toFixed(1) + 'K';
            }
            return num.toLocaleString();
    }
}

// Backward compatibility
function updateStatCard(id, opts) {
    StatCard.update(id, opts);
}
