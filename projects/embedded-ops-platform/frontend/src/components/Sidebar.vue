<template>
  <n-layout-sider
    bordered
    :width="220"
    :native-scrollbar="false"
    class="sidebar"
  >
    <div class="logo">
      <h3>Embedded Ops</h3>
      <n-text depth="3" style="font-size: 12px;">Device Management</n-text>
    </div>
    <n-menu
      :value="activeKey"
      :options="menuOptions"
      @update:value="handleMenuSelect"
    />
  </n-layout-sider>
</template>

<script setup lang="ts">
import { computed, h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { NIcon } from 'naive-ui'
import { DashboardOutlined, DesktopOutlined, AlertOutlined, SettingOutlined } from '@vicons/antd'

const router = useRouter()
const route = useRoute()

const activeKey = computed(() => route.path)

function renderIcon(icon: any) {
  return () => h(NIcon, null, { default: () => h(icon) })
}

const menuOptions = [
  { label: 'Dashboard', key: '/dashboard', icon: renderIcon(DashboardOutlined) },
  { label: 'Devices', key: '/devices', icon: renderIcon(DesktopOutlined) },
  { label: 'Alerts', key: '/alerts', icon: renderIcon(AlertOutlined) },
  { label: 'Settings', key: '/settings', icon: renderIcon(SettingOutlined), disabled: true },
]

function handleMenuSelect(key: string) {
  router.push(key)
}
</script>

<style scoped>
.sidebar {
  min-height: 100vh;
}

.logo {
  padding: 16px;
  border-bottom: 1px solid var(--n-border-color, #eee);
  margin-bottom: 8px;
}
</style>