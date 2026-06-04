<template>
  <div class="audit-page">
    <h2 class="page-title">审计日志</h2>

    <el-card shadow="hover">
      <el-table :data="logs" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="user_id" label="用户 ID" width="80" />
        <el-table-column prop="action" label="操作" width="120">
          <template #default="{ row }">
            <el-tag size="small">{{ row.action }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="resource" label="资源" min-width="180" />
        <el-table-column prop="ip_addr" label="IP 地址" width="140" />
        <el-table-column prop="detail" label="详情" min-width="160" show-overflow-tooltip />
        <el-table-column prop="created_at" label="时间" width="170">
          <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { auditAPI } from '@/api'

interface AuditLog {
  id: number
  user_id: number | null
  action: string
  resource: string
  ip_addr: string
  detail: string
  created_at: string
}

const logs = ref<AuditLog[]>([])
const loading = ref(false)

const formatTime = (t: string): string => {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

const fetchLogs = async () => {
  loading.value = true
  try {
    const res = await auditAPI.list()
    if (res.data.code === 200) {
      logs.value = res.data.data
    }
  } catch (err) {
    console.error('Failed to fetch audit logs', err)
  } finally {
    loading.value = false
  }
}

onMounted(fetchLogs)
</script>

<style scoped>
.page-title {
  font-size: 22px;
  color: #303133;
  margin-bottom: 20px;
}
</style>