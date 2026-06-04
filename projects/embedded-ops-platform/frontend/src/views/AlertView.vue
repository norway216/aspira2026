<template>
  <n-layout has-sider>
    <Sidebar />
    <n-layout>
      <n-layout-header class="header">
        <h2>Alert Center</h2>
      </n-layout-header>
      <n-layout-content class="content">
        <n-card>
          <n-data-table
            :columns="columns"
            :data="alerts"
            :loading="loading"
            :pagination="pagination"
          />
        </n-card>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted } from 'vue'
import { useMessage } from 'naive-ui'
import { deviceApi } from '../api/devices'
import Sidebar from '../components/Sidebar.vue'

const message = useMessage()
const loading = ref(false)
const alerts = ref<any[]>([])

const columns = [
  { title: 'Alert ID', key: 'id', width: 80 },
  { title: 'Device ID', key: 'device_id', width: 180 },
  { title: 'Type', key: 'alert_type', width: 150 },
  {
    title: 'Severity', key: 'severity', width: 90,
    render: (row: any) => {
      const type = row.severity === 'critical' ? 'error' : row.severity === 'warning' ? 'warning' : 'info'
      return h('n-tag', { type, size: 'small' }, row.severity)
    }
  },
  { title: 'Message', key: 'message', ellipsis: true },
  { title: 'Status', key: 'status', width: 90 },
  { title: 'Created', key: 'created_at', width: 180 },
  {
    title: 'Actions', key: 'actions', width: 100,
    render: (row: any) => row.status === 'open'
      ? h('n-button', { size: 'small', type: 'primary', onClick: () => handleResolve(row.id) }, 'Resolve')
      : null,
  },
]

const pagination = reactive({
  page: 1,
  pageSize: 20,
})

async function fetchAlerts() {
  loading.value = true
  try {
    const res = await deviceApi.getAlerts(pagination.page, pagination.pageSize)
    alerts.value = res.data.alerts
  } catch (err: any) {
    message.error('Failed to load alerts')
  } finally {
    loading.value = false
  }
}

async function handleResolve(alertId: number) {
  try {
    await deviceApi.resolveAlert(alertId)
    message.success('Alert resolved')
    fetchAlerts()
  } catch (err: any) {
    message.error('Failed to resolve alert')
  }
}

onMounted(fetchAlerts)
</script>

<style scoped>
.header {
  padding: 16px 24px;
  border-bottom: 1px solid var(--n-border-color, #eee);
}

.content {
  padding: 24px;
}
</style>