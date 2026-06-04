<template>
  <n-layout has-sider>
    <Sidebar />
    <n-layout>
      <n-layout-header class="header">
        <h2>Dashboard</h2>
        <n-space>
          <n-tag v-if="wsConnected" type="success">WebSocket Connected</n-tag>
          <n-tag v-else type="warning">WebSocket Disconnected</n-tag>
          <n-button @click="appStore.toggleTheme">
            {{ appStore.isDark ? 'Light' : 'Dark' }}
          </n-button>
          <n-button @click="appStore.logout">Logout</n-button>
        </n-space>
      </n-layout-header>
      <n-layout-content class="content">
        <n-grid :cols="4" :x-gap="16">
          <n-grid-item>
            <n-card title="Total Devices" hoverable>
              <n-h2 style="margin: 0;">{{ stats.total_count }}</n-h2>
            </n-card>
          </n-grid-item>
          <n-grid-item>
            <n-card title="Online" hoverable>
              <n-h2 style="margin: 0; color: #18a058;">{{ stats.online_count }}</n-h2>
            </n-card>
          </n-grid-item>
          <n-grid-item>
            <n-card title="Offline" hoverable>
              <n-h2 style="margin: 0; color: #d03050;">{{ stats.offline_count }}</n-h2>
            </n-card>
          </n-grid-item>
          <n-grid-item>
            <n-card title="Alerts" hoverable>
              <n-h2 style="margin: 0; color: #f0a020;">{{ alertCount }}</n-h2>
            </n-card>
          </n-grid-item>
        </n-grid>

        <n-card title="Recent Devices" style="margin-top: 16px;">
          <n-data-table
            :columns="columns"
            :data="devices"
            :loading="loading"
            :pagination="pagination"
            @update:page="handlePageChange"
          />
        </n-card>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { deviceApi, type Device, type DashboardStats } from '../api/devices'
import { useAppStore } from '../stores/app'
import Sidebar from '../components/Sidebar.vue'

const router = useRouter()
const message = useMessage()
const appStore = useAppStore()

const loading = ref(false)
const devices = ref<Device[]>([])
const alertCount = ref(0)
const wsConnected = ref(false)
let ws: WebSocket | null = null

const stats = reactive<DashboardStats>({
  online_count: 0,
  offline_count: 0,
  total_count: 0,
})

const columns = [
  { title: 'Name', key: 'hostname', ellipsis: true },
  { title: 'Device ID', key: 'device_id', ellipsis: true },
  { title: 'IP Address', key: 'ip_address' },
  { title: 'Board Type', key: 'board_type' },
  { title: 'Status', key: 'status', render: (row: Device) => row.status === 'online' ? h('span', { style: 'color:#18a058' }, 'Online') : h('span', { style: 'color:#d03050' }, 'Offline') },
  { title: 'Kernel', key: 'kernel_version', ellipsis: true },
  {
    title: 'Actions',
    key: 'actions',
    render: (row: Device) => h('a', {
      style: 'cursor:pointer;color:#2080f0',
      onClick: () => router.push(`/devices/${row.device_id}`)
    }, 'View'),
  },
]

const pagination = reactive({
  page: 1,
  pageSize: 20,
  showSizePicker: false,
})

async function fetchDevices() {
  loading.value = true
  try {
    const res = await deviceApi.getDevices(pagination.page, pagination.pageSize)
    devices.value = res.data.devices
    stats.online_count = res.data.online_count
    stats.total_count = res.data.total_count
    stats.offline_count = res.data.total_count - res.data.online_count
    pagination.page = res.data.page
  } catch (err: any) {
    message.error('Failed to load devices')
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  fetchDevices()
}

function connectWebSocket() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/ws`
  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    wsConnected.value = true
  }

  ws.onclose = () => {
    wsConnected.value = false
    // Auto-reconnect after 5 seconds
    setTimeout(connectWebSocket, 5000)
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'device_online' || msg.type === 'device_offline') {
        fetchDevices()
      }
    } catch (e) {
      // Ignore parse errors
    }
  }
}

onMounted(() => {
  fetchDevices()
  connectWebSocket()
})

onUnmounted(() => {
  if (ws) {
    ws.close()
  }
})
</script>

<style scoped>
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-bottom: 1px solid var(--n-border-color, #eee);
}

.content {
  padding: 24px;
}
</style>