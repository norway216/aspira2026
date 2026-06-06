/* ===== Stat Card Updater ===== */
const StatCard = {
    /**
     * Update a stat card's value with animation
     * @param {string} id - element ID (e.g., 'stat-tps')
     * @param {object} opts - { value, label?, trend?, trendUp?, format? }
     *   format: 'number', 'currency', 'percent', 'tps', 'custom'
     */
    update(id, opts) {
        const card = document.getElementById(id);
        if (!card) return;

        opts = opts || {};

        // Update value with animation
        if (opts.value !== undefined) {
            const valueEl = card.querySelector('.stat-value');
            if (valueEl) {
                var formatted = formatStatValue(opts.value, opts.format || 'number');
                // Animate change
                if (valueEl.textContent !== formatted) {
                    valueEl.textContent = formatted;
                    valueEl.classList.remove('pulse');
                    // Force reflow
                    void valueEl.offsetWidth;
                    valueEl.classList.add('pulse');
                }
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
                // Create trend element if it doesn't exist
                trendEl = document.createElement('div');
                trendEl.className = 'stat-trend';
                card.appendChild(trendEl);
            }
            if (trendEl) {
                if (opts.trend) {
                    const trendUp = opts.trendUp !== false;
                    trendEl.className = 'stat-trend ' + (trendUp ? 'trend-up' : 'trend-down');
                    trendEl.innerHTML = (trendUp ? '&#9650;' : '&#9660;') + ' ' + escapeHtml(String(opts.trend));
                    trendEl.style.display = '';
                } else {
                    trendEl.style.display = 'none';
                }
            }
        }
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
