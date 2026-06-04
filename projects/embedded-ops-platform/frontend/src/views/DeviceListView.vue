<template>
  <n-layout has-sider>
    <Sidebar />
    <n-layout>
      <n-layout-header class="header">
        <h2>Device Management</h2>
      </n-layout-header>
      <n-layout-content class="content">
        <n-card>
          <n-data-table
            :columns="columns"
            :data="devices"
            :loading="loading"
            :pagination="pagination"
            @update:page="handlePageChange"
            @update:page-size="handlePageSizeChange"
          />
        </n-card>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, reactive, h, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { deviceApi, type Device } from '../api/devices'
import Sidebar from '../components/Sidebar.vue'

const router = useRouter()
const message = useMessage()

const loading = ref(false)
const devices = ref<Device[]>([])

const columns = [
  { title: 'Device ID', key: 'device_id', width: 180 },
  { title: 'Hostname', key: 'hostname', width: 160 },
  { title: 'IP', key: 'ip_address', width: 140 },
  { title: 'Board', key: 'board_type', width: 100 },
  { title: 'Arch', key: 'arch', width: 80 },
  { title: 'OS', key: 'os_name', ellipsis: true },
  { title: 'Kernel', key: 'kernel_version', ellipsis: true },
  { title: 'Agent', key: 'agent_version', width: 80 },
  {
    title: 'Status', key: 'status', width: 80,
    render: (row: Device) => row.status === 'online'
      ? h('n-tag', { type: 'success', size: 'small' }, 'Online')
      : h('n-tag', { type: 'warning', size: 'small' }, 'Offline')
  },
  { title: 'Last Seen', key: 'last_seen', width: 180 },
  {
    title: 'Actions', key: 'actions', width: 80,
    render: (row: Device) => h('n-button', {
      size: 'small',
      onClick: () => router.push(`/devices/${row.device_id}`)
    }, { default: () => 'Details' }),
  },
]

const pagination = reactive({
  page: 1,
  pageSize: 20,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
})

async function fetchDevices() {
  loading.value = true
  try {
    const res = await deviceApi.getDevices(pagination.page, pagination.pageSize)
    devices.value = res.data.devices
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

function handlePageSizeChange(size: number) {
  pagination.pageSize = size
  fetchDevices()
}

onMounted(fetchDevices)
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