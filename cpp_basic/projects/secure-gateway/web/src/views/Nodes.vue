<template>
  <div class="nodes-page">
    <h2 class="page-title">节点管理</h2>

    <el-card shadow="hover">
      <el-table :data="nodes" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="node_id" label="Node ID" min-width="140" />
        <el-table-column prop="name" label="名称" min-width="120" />
        <el-table-column prop="region" label="区域" width="120" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="cpu_usage" label="CPU" width="100">
          <template #default="{ row }">
            <span>{{ row.cpu_usage?.toFixed(1) ?? '-' }}%</span>
          </template>
        </el-table-column>
        <el-table-column prop="mem_usage" label="内存" width="100">
          <template #default="{ row }">
            <span>{{ row.mem_usage?.toFixed(1) ?? '-' }}%</span>
          </template>
        </el-table-column>
        <el-table-column prop="active_connections" label="连接数" width="100" />
        <el-table-column label="RX/TX" min-width="150">
          <template #default="{ row }">
            <span v-if="row.rx_bytes_per_sec != null">
              ↓ {{ formatSpeed(row.rx_bytes_per_sec) }} / ↑ {{ formatSpeed(row.tx_bytes_per_sec) }}
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="max_connections" label="上限" width="80" />
        <el-table-column prop="public_addr" label="地址" min-width="140" />
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { nodeAPI } from '@/api'

interface NodeItem {
  id: number
  node_id: string
  name: string
  region: string
  status: string
  cpu_usage?: number
  mem_usage?: number
  active_connections?: number
  rx_bytes_per_sec?: number
  tx_bytes_per_sec?: number
  max_connections: number
  public_addr: string
}

const nodes = ref<NodeItem[]>([])
const loading = ref(false)

const statusType = (status: string): string => {
  switch (status) {
    case 'online': return 'success'
    case 'offline': return 'danger'
    case 'draining': return 'warning'
    default: return 'info'
  }
}

const formatSpeed = (bytes: number): string => {
  if (!bytes) return '0 B/s'
  const units = ['B/s', 'KB/s', 'MB/s', 'GB/s']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

const fetchNodes = async () => {
  loading.value = true
  try {
    const res = await nodeAPI.list()
    if (res.data.code === 200) {
      nodes.value = res.data.data
    }
  } catch (err) {
    console.error('Failed to fetch nodes', err)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchNodes()
  // Auto-refresh
  setInterval(fetchNodes, 15000)
})
</script>

<style scoped>
.page-title {
  font-size: 22px;
  color: #303133;
  margin-bottom: 20px;
}
</style>