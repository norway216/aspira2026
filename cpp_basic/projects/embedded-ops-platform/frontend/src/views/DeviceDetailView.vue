<template>
  <n-layout has-sider>
    <Sidebar />
    <n-layout>
      <n-layout-header class="header">
        <n-space align="center">
          <n-button @click="router.push('/devices')" text>
            ← Back
          </n-button>
          <h2 v-if="device">{{ device.hostname || device.device_id }}</h2>
        </n-space>
      </n-layout-header>
      <n-layout-content class="content">
        <n-spin :show="loading">
          <n-grid :cols="2" :x-gap="16" v-if="device">
            <!-- Device Info -->
            <n-grid-item>
              <n-card title="Device Information">
                <n-descriptions label-placement="left" :column="1">
                  <n-descriptions-item label="Device ID">{{ device.device_id }}</n-descriptions-item>
                  <n-descriptions-item label="Hostname">{{ device.hostname }}</n-descriptions-item>
                  <n-descriptions-item label="IP Address">{{ device.ip_address }}</n-descriptions-item>
                  <n-descriptions-item label="MAC Address">{{ device.mac_address }}</n-descriptions-item>
                  <n-descriptions-item label="Board Type">{{ device.board_type }}</n-descriptions-item>
                  <n-descriptions-item label="Architecture">{{ device.arch }}</n-descriptions-item>
                  <n-descriptions-item label="OS">{{ device.os_name }} {{ device.os_version }}</n-descriptions-item>
                  <n-descriptions-item label="Kernel">{{ device.kernel_version }}</n-descriptions-item>
                  <n-descriptions-item label="Agent Version">{{ device.agent_version }}</n-descriptions-item>
                  <n-descriptions-item label="Status">
                    <n-tag :type="device.status === 'online' ? 'success' : 'warning'">
                      {{ device.status }}
                    </n-tag>
                  </n-descriptions-item>
                  <n-descriptions-item label="Last Seen">{{ device.last_seen }}</n-descriptions-item>
                </n-descriptions>
              </n-card>
            </n-grid-item>

            <!-- Live Metrics -->
            <n-grid-item>
              <n-card title="Current Metrics">
                <n-descriptions v-if="metrics" label-placement="left" :column="1">
                  <n-descriptions-item label="CPU Usage">
                    <n-progress type="line" :percentage="Math.round(metrics.cpu_usage)" :height="16" />
                  </n-descriptions-item>
                  <n-descriptions-item label="Memory Usage">
                    <n-progress type="line" :percentage="Math.round(metrics.memory_usage)" :height="16" status="info" />
                  </n-descriptions-item>
                  <n-descriptions-item label="Disk Usage">
                    <n-progress type="line" :percentage="Math.round(metrics.disk_usage)" :height="16" status="warning" />
                  </n-descriptions-item>
                  <n-descriptions-item label="Temperature">{{ metrics.temperature }}°C</n-descriptions-item>
                  <n-descriptions-item label="Load Average (1m)">{{ metrics.load_avg_1m }}</n-descriptions-item>
                </n-descriptions>
                <n-empty v-else description="No metrics data" />
              </n-card>
            </n-grid-item>

            <!-- Remote Commands -->
            <n-grid-item :span="2" style="margin-top: 16px;">
              <n-card title="Remote Commands">
                <n-space>
                  <n-button v-for="action in actions" :key="action.name" :type="action.risk_level === 'high' ? 'error' : 'primary'" secondary @click="executeCommand(action.name, {})">
                    {{ action.description || action.name }}
                  </n-button>
                </n-space>
              </n-card>
            </n-grid-item>

            <!-- Command History -->
            <n-grid-item :span="2" style="margin-top: 16px;">
              <n-card title="Command History">
                <n-data-table :columns="cmdColumns" :data="commands" :max-height="300" />
              </n-card>
            </n-grid-item>
          </n-spin>
        </n-loading>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage, useDialog } from 'naive-ui'
import { deviceApi, type Device, type DeviceMetric, type CommandTask } from '../api/devices'
import Sidebar from '../components/Sidebar.vue'

const route = useRoute()
const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const deviceId = route.params.deviceId as string
const loading = ref(false)
const device = ref<Device | null>(null)
const metrics = ref<DeviceMetric | null>(null)
const commands = ref<CommandTask[]>([])
const actions = ref<any[]>([])

const cmdColumns = [
  { title: 'Task ID', key: 'task_id', width: 200 },
  { title: 'Action', key: 'action', width: 150 },
  { title: 'Status', key: 'status', width: 100 },
  { title: 'Exit Code', key: 'exit_code', width: 80 },
  { title: 'Created', key: 'created_at', width: 180 },
  { title: 'Stdout', key: 'stdout', ellipsis: true },
]

async function fetchDevice() {
  loading.value = true
  try {
    const [devRes, cmdRes, actionRes] = await Promise.all([
      deviceApi.getDevice(deviceId),
      deviceApi.getCommands(deviceId, 20),
      deviceApi.getCommandActions(),
    ])
    device.value = devRes.data.device
    metrics.value = devRes.data.metrics
    commands.value = cmdRes.data.commands
    actions.value = actionRes.data.actions
  } catch (err: any) {
    message.error('Failed to load device')
  } finally {
    loading.value = false
  }
}

async function executeCommand(action: string, params: Record<string, any>) {
  dialog.warning({
    title: 'Execute Command',
    content: `Send "${action}" to device ${deviceId}?`,
    positiveText: 'Execute',
    negativeText: 'Cancel',
    onPositiveClick: async () => {
      try {
        await deviceApi.executeCommand(deviceId, action, params)
        message.success(`Command "${action}" sent`)
        // Refresh command list
        const cmdRes = await deviceApi.getCommands(deviceId, 20)
        commands.value = cmdRes.data.commands
      } catch (err: any) {
        message.error(err.response?.data?.error || 'Command failed')
      }
    },
  })
}

onMounted(fetchDevice)
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