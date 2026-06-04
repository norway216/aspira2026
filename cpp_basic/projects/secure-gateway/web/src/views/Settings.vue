<template>
  <div class="settings-page">
    <h2 class="page-title">系统设置</h2>

    <el-card shadow="hover" class="settings-card">
      <template #header>
        <span>API 连接信息</span>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="API 地址">
          {{ apiBaseUrl }}
        </el-descriptions-item>
        <el-descriptions-item label="登录用户">
          {{ username || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="用户角色">
          <el-tag :type="role === 'admin' ? 'danger' : 'primary'" size="small">
            {{ role || '-' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="Token 状态">
          {{ hasToken ? '已登录' : '未登录' }}
        </el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card shadow="hover" class="settings-card">
      <template #header>
        <span>关于系统</span>
      </template>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="系统名称">Secure Gateway</el-descriptions-item>
        <el-descriptions-item label="版本">1.0.0</el-descriptions-item>
        <el-descriptions-item label="技术栈">Go + Vue 3 + PostgreSQL + Redis</el-descriptions-item>
        <el-descriptions-item label="架构">控制面 / 数据面 / 管理面</el-descriptions-item>
      </el-descriptions>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const username = ref(localStorage.getItem('username'))
const role = ref(localStorage.getItem('user_role'))
const hasToken = ref(!!localStorage.getItem('access_token'))
const apiBaseUrl = ref(import.meta.env.DEV ? '/api/v1' : '/api/v1')
</script>

<style scoped>
.page-title {
  font-size: 22px;
  color: #303133;
  margin-bottom: 20px;
}

.settings-card {
  margin-bottom: 20px;
}
</style>