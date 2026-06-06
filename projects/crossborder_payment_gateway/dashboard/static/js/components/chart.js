/* ===== ECharts Wrapper ===== */
const Chart = {
    instances: {},

    /**
     * Create a chart in the given container
     * @param {string} containerId - DOM element ID
     * @param {object} opts - ECharts option overrides
     * @returns {object} - { chart, update(data), dispose(), resize() }
     */
    create(containerId, opts) {
        const el = document.getElementById(containerId);
        if (!el) {
            console.warn('[Chart] Container not found:', containerId);
            return null;
        }

        // Dispose existing if any
        const existingKey = containerId + '_echarts';
        if (this.instances[existingKey]) {
            this.instances[existingKey].dispose();
            delete this.instances[existingKey];
        }

        const chart = echarts.init(el, 'dark', { renderer: 'canvas' });
        this.instances[existingKey] = chart;

        // Default dark theme options for this dashboard
        const baseOptions = {
            backgroundColor: 'transparent',
            tooltip: {
                trigger: 'axis',
                backgroundColor: 'rgba(28,28,30,0.95)',
                borderColor: 'rgba(255,255,255,0.08)',
                borderWidth: 1,
                textStyle: {
                    color: '#F5F5F7',
                    fontSize: 12
                },
                axisPointer: {
                    type: 'shadow',
                    shadowStyle: { color: 'rgba(255,255,255,0.03)' }
                }
            },
            legend: {
                textStyle: { color: '#98989D', fontSize: 12 },
                icon: 'roundRect',
                itemWidth: 12,
                itemHeight: 4
            },
            grid: {
                top: 20,
                right: 20,
                bottom: 30,
                left: 50,
                containLabel: true
            },
            xAxis: {
                type: 'category',
                axisLine: { lineStyle: { color: 'rgba(255,255,255,0.06)' } },
                axisTick: { show: false },
                axisLabel: { color: '#6E6E73', fontSize: 11 },
                splitLine: { show: false }
            },
            yAxis: {
                type: 'value',
                axisLine: { show: false },
                axisTick: { show: false },
                axisLabel: { color: '#6E6E73', fontSize: 11 },
                splitLine: {
                    lineStyle: {
                        color: 'rgba(255,255,255,0.04)',
                        type: 'dashed'
                    }
                }
            },
            // Merge user options
            ...opts
        };

        // Apply palette if not specified
        if (!baseOptions.color) {
            baseOptions.color = ['#FF2D55', '#AF52DE', '#007AFF', '#30D158', '#FF9F0A', '#5AC8FA'];
        }

        chart.setOption(baseOptions);

        // Auto resize
        const resizeHandler = () => { chart.resize(); };
        window.addEventListener('resize', resizeHandler);

        // Return control interface
        const api = {
            chart: chart,
            update: function (newOpts) {
                chart.setOption(newOpts, { notMerge: false });
                return api;
            },
            setOption: function (opt, notMerge) {
                chart.setOption(opt, notMerge !== false);
                return api;
            },
            resize: function () {
                chart.resize();
                return api;
            },
            dispose: function () {
                window.removeEventListener('resize', resizeHandler);
                chart.dispose();
                delete Chart.instances[existingKey];
            },
            showLoading: function () {
                chart.showLoading('default', {
                    text: '',
                    color: '#AF52DE',
                    maskColor: 'rgba(0,0,0,0)',
                    lineWidth: 2
                });
                return api;
            },
            hideLoading: function () {
                chart.hideLoading();
                return api;
            },
            clear: function () {
                chart.clear();
                return api;
            }
        };

        return api;
    },

    dispose(chart) {
        if (chart && typeof chart.dispose === 'function') {
            chart.dispose();
        }
    }
};

// Backward compatibility
function createChart(containerId, options) {
    return Chart.create(containerId, options);
}

function disposeChart(chart) {
    Chart.dispose(chart);
}
