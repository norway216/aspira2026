<template>
  <div class="traffic-page">
    <h2 class="page-title">流量统计</h2>

    <!-- Summary -->
    <el-row :gutter="20" class="summary-row">
      <el-col :span="8">
        <el-card shadow="hover">
          <div class="summary-item">
            <span class="summary-label">总接收流量</span>
            <span class="summary-value">{{ formatBytes(summary.total_rx_bytes) }}</span>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <div class="summary-item">
            <span class="summary-label">总发送流量</span>
            <span class="summary-value">{{ formatBytes(summary.total_tx_bytes) }}</span>
          </div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="hover">
          <div class="summary-item">
            <span class="summary-label">总时长</span>
            <span class="summary-value">{{ formatDuration(summary.total_duration_seconds) }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Traffic Records -->
    <el-card shadow="hover" class="records-card">
      <template #header>
        <span>流量记录</span>
      </template>
      <el-table :data="records" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="user_id" label="用户 ID" width="80" />
        <el-table-column prop="node_id" label="节点" min-width="120" />
        <el-table-column prop="rx_bytes" label="接收" width="120">
          <template #default="{ row }">{{ formatBytes(row.rx_bytes) }}</template>
        </el-table-column>
        <el-table-column prop="tx_bytes" label="发送" width="120">
          <template #default="{ row }">{{ formatBytes(row.tx_bytes) }}</template>
        </el-table-column>
        <el-table-column prop="duration_seconds" label="时长" width="100">
          <template #default="{ row }">{{ formatDuration(row.duration_seconds) }}</template>
        </el-table-column>
        <el-table-column prop="created_at" label="时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { trafficAPI } from '@/api'

interface TrafficRecord {
  id: number
  user_id: number
  node_id: string
  rx_bytes: number
  tx_bytes: number
  duration_seconds: number
  created_at: string
}

const records = ref<TrafficRecord[]>([])
const loading = ref(false)
const summary = reactive({
  total_rx_bytes: 0,
  total_tx_bytes: 0,
  total_duration_seconds: 0
})

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[i]
}

const formatDuration = (seconds: number): string => {
  if (!seconds) return '0s'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

const formatTime = (t: string): string => {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

const fetchData = async () => {
  loading.value = true
  try {
    const [recordsRes, summaryRes] = await Promise.all([
      trafficAPI.list({ limit: 100 }),
      trafficAPI.getSummary()
    ])
    if (recordsRes.data.code === 200) {
      records.value = recordsRes.data.data
    }
    if (summaryRes.data.code === 200) {
      Object.assign(summary, summaryRes.data.data)
    }
  } catch (err) {
    console.error('Failed to fetch traffic data', err)
  } finally {
    loading.value = false
  }
}

onMounted(fetchData)
</script>

<style scoped>
.page-title {
  font-size: 22px;
  color: #303133;
  margin-bottom: 20px;
}

.summary-row {
  margin-bottom: 20px;
}

.summary-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 10px 0;
}

.summary-label {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.summary-value {
  font-size: 24px;
  font-weight: bold;
  color: #303133;
}
</style>