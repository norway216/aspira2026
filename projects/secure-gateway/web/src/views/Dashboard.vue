<template>
  <div class="dashboard">
    <h2 class="page-title">系统仪表盘</h2>

    <!-- Stats Cards -->
    <el-row :gutter="20" class="stats-row">
      <el-col :xs="12" :sm="6" v-for="card in statCards" :key="card.label">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card-inner">
            <div class="stat-icon" :style="{ background: card.color }">
              <el-icon :size="24"><component :is="card.icon" /></el-icon>
            </div>
            <div class="stat-info">
              <span class="stat-value">{{ card.value }}</span>
              <span class="stat-label">{{ card.label }}</span>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Charts -->
    <el-row :gutter="20" class="charts-row">
      <el-col :span="16">
        <el-card shadow="hover">
          <template #header>
            <span>实时流量趋势 (24h)</span>
          </template>
          <div ref="trafficChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <template #header>
            <span>节点区域分布</span>
          </template>
          <div ref="distributionChartRef" class="chart-container"></div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Top Users -->
    <el-row :gutter="20" class="charts-row">
      <el-col :span="24">
        <el-card shadow="hover">
          <template #header>
            <span>流量排行 Top 10</span>
          </template>
          <el-table :data="topUsers" stripe style="width: 100%" v-loading="loading">
            <el-table-column type="index" label="#" width="60" />
            <el-table-column prop="username" label="用户名" />
            <el-table-column prop="total_bytes" label="总流量" width="200">
              <template #default="{ row }">
                {{ formatBytes(row.total_bytes) }}
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, shallowRef } from 'vue'
import { Monitor, Connection, DataLine, Warning } from '@element-plus/icons-vue'
import { dashboardAPI } from '@/api'
import * as echarts from 'echarts'

// Stats
const stats = ref({
  online_nodes: 0,
  total_connections: 0,
  today_traffic_bytes: 0,
  alert_count: 0
})

const topUsers = ref<any[]>([])
const loading = ref(false)

const statCards = ref([
  { label: '在线节点', value: '0', icon: Monitor, color: '#409eff' },
  { label: '总连接数', value: '0', icon: Connection, color: '#67c23a' },
  { label: '今日流量', value: '0 B', icon: DataLine, color: '#e6a23c' },
  { label: '告警数量', value: '0', icon: Warning, color: '#f56c6c' }
])

// Charts
const trafficChartRef = shallowRef<HTMLDivElement>()
const distributionChartRef = shallowRef<HTMLDivElement>()
let trafficChart: echarts.ECharts | null = null
let distributionChart: echarts.ECharts | null = null
let timer: number | null = null

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[i]
}

const fetchData = async () => {
  loading.value = true
  try {
    const [statsRes, topRes] = await Promise.all([
      dashboardAPI.getStats(),
      dashboardAPI.getTopUsers()
    ])

    if (statsRes.data.code === 200) {
      stats.value = statsRes.data.data
      statCards.value[0].value = String(stats.value.online_nodes)
      statCards.value[1].value = String(stats.value.total_connections)
      statCards.value[2].value = formatBytes(stats.value.today_traffic_bytes)
      statCards.value[3].value = String(stats.value.alert_count)
    }

    if (topRes.data.code === 200) {
      topUsers.value = topRes.data.data
    }
  } catch (err) {
    console.error('Failed to fetch dashboard data', err)
  } finally {
    loading.value = false
  }
}

const initTrafficChart = async () => {
  if (!trafficChartRef.value) return
  trafficChart = echarts.init(trafficChartRef.value)

  try {
    const res = await dashboardAPI.getTrafficHistory()
    if (res.data.code !== 200) return
    const data = res.data.data
    const times = data.map((d: any) => {
      const t = new Date(d.timestamp)
      return `${t.getHours().toString().padStart(2, '0')}:${t.getMinutes().toString().padStart(2, '0')}`
    })
    const values = data.map((d: any) => Math.round(d.bytes / 1024)) // KB

    trafficChart.setOption({
      tooltip: { trigger: 'axis' },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'category',
        data: times,
        axisLabel: { rotate: 45, fontSize: 10 }
      },
      yAxis: { type: 'value', name: 'KB/s' },
      series: [{
        data: values,
        type: 'line',
        smooth: true,
        areaStyle: { opacity: 0.3 },
        lineStyle: { color: '#409eff' },
        itemStyle: { color: '#409eff' }
      }]
    })
  } catch (err) {
    // No data yet - show empty chart
    trafficChart.setOption({
      title: { text: '暂无数据', left: 'center', top: 'center' }
    })
  }
}

const initDistributionChart = async () => {
  if (!distributionChartRef.value) return
  distributionChart = echarts.init(distributionChartRef.value)

  try {
    const res = await dashboardAPI.getDistribution()
    if (res.data.code !== 200) return
    const data = res.data.data

    distributionChart.setOption({
      tooltip: { trigger: 'item' },
      series: [{
        type: 'pie',
        radius: ['40%', '70%'],
        data: data.map((d: any) => ({ name: d.region, value: d.count })),
        label: { show: true, formatter: '{b}: {c}' }
      }]
    })
  } catch (err) {
    distributionChart.setOption({
      title: { text: '暂无数据', left: 'center', top: 'center' }
    })
  }
}

onMounted(() => {
  fetchData()
  initTrafficChart()
  initDistributionChart()

  // Auto-refresh every 30 seconds
  timer = window.setInterval(() => {
    fetchData()
    initTrafficChart()
    initDistributionChart()
  }, 30000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
  trafficChart?.dispose()
  distributionChart?.dispose()
})
</script>

<style scoped>
.dashboard {
  height: 100%;
}

.page-title {
  font-size: 22px;
  color: #303133;
  margin-bottom: 20px;
}

.stats-row {
  margin-bottom: 20px;
}

.stat-card {
  border-radius: 8px;
}

.stat-card-inner {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 52px;
  height: 52px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}

.stat-info {
  display: flex;
  flex-direction: column;
}

.stat-value {
  font-size: 28px;
  font-weight: bold;
  color: #303133;
}

.stat-label {
  font-size: 13px;
  color: #909399;
}

.charts-row {
  margin-bottom: 20px;
}

.chart-container {
  width: 100%;
  height: 350px;
}
</style>