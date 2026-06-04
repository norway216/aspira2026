<template>
  <div class="users-page">
    <div class="page-header">
      <h2 class="page-title">用户管理</h2>
      <el-button type="primary" @click="showCreateDialog">
        <el-icon><Plus /></el-icon> 创建用户
      </el-button>
    </div>

    <el-card shadow="hover">
      <el-table :data="users" v-loading="loading" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="username" label="用户名" min-width="120" />
        <el-table-column prop="role" label="角色" width="100">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'" size="small">
              {{ row.role }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="traffic_quota_bytes" label="流量配额" width="140">
          <template #default="{ row }">
            {{ formatBytes(row.traffic_quota_bytes) }}
          </template>
        </el-table-column>
        <el-table-column prop="traffic_used_bytes" label="已使用" width="140">
          <template #default="{ row }">
            {{ formatBytes(row.traffic_used_bytes) }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="创建时间" width="170">
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="showEditDialog(row)">编辑</el-button>
            <el-button
              size="small"
              type="danger"
              :disabled="row.role === 'admin'"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Create/Edit Dialog -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEditing ? '编辑用户' : '创建用户'"
      width="500px"
    >
      <el-form :model="userForm" :rules="formRules" ref="formRef" label-width="100px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="userForm.username" :disabled="isEditing" />
        </el-form-item>
        <el-form-item label="密码" :prop="isEditing ? undefined : 'password'">
          <el-input
            v-model="userForm.password"
            type="password"
            show-password
            :placeholder="isEditing ? '留空则不修改' : '请输入密码'"
          />
        </el-form-item>
        <el-form-item label="角色" prop="role">
          <el-select v-model="userForm.role">
            <el-option label="管理员" value="admin" />
            <el-option label="普通用户" value="user" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态" prop="status" v-if="isEditing">
          <el-select v-model="userForm.status">
            <el-option label="启用" value="active" />
            <el-option label="禁用" value="disabled" />
          </el-select>
        </el-form-item>
        <el-form-item label="流量配额" prop="quota_bytes">
          <el-input-number
            v-model="userForm.quota_bytes"
            :min="0"
            :step="1073741824"
            style="width: 100%"
          />
          <span style="margin-left: 8px; color: #909399;">
            ({{ formatBytes(userForm.quota_bytes) }})
          </span>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ isEditing ? '保存' : '创建' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance, FormRules } from 'element-plus'
import { userAPI } from '@/api'

interface UserItem {
  id: number
  username: string
  role: string
  status: string
  traffic_quota_bytes: number
  traffic_used_bytes: number
  created_at: string
}

const users = ref<UserItem[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const isEditing = ref(false)
const submitting = ref(false)
const editingId = ref<number | null>(null)
const formRef = ref<FormInstance>()

const defaultForm = {
  username: '',
  password: '',
  role: 'user',
  status: 'active',
  quota_bytes: 0
}

const userForm = reactive({ ...defaultForm })

const formRules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 64, message: '用户名长度在 3-64 个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur', validator: (_: any, v: string) => !isEditing.value && !v }
  ]
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(2) + ' ' + units[i]
}

const formatTime = (t: string): string => {
  if (!t) return '-'
  return new Date(t).toLocaleString('zh-CN')
}

const fetchUsers = async () => {
  loading.value = true
  try {
    const res = await userAPI.list()
    if (res.data.code === 200) {
      users.value = res.data.data
    }
  } catch (err) {
    console.error('Failed to fetch users', err)
  } finally {
    loading.value = false
  }
}

const showCreateDialog = () => {
  isEditing.value = false
  editingId.value = null
  Object.assign(userForm, defaultForm)
  dialogVisible.value = true
}

const showEditDialog = (user: any) => {
  isEditing.value = true
  editingId.value = user.id
  userForm.username = user.username
  userForm.password = ''
  userForm.role = user.role
  userForm.status = user.status
  userForm.quota_bytes = user.traffic_quota_bytes
  dialogVisible.value = true
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEditing.value && editingId.value) {
      const data: Record<string, any> = {}
      if (userForm.password) data.password = userForm.password
      data.role = userForm.role
      data.status = userForm.status
      data.quota_bytes = userForm.quota_bytes
      await userAPI.update(editingId.value, data)
      ElMessage.success('用户已更新')
    } else {
      await userAPI.create({
        username: userForm.username,
        password: userForm.password,
        role: userForm.role,
        quota_bytes: userForm.quota_bytes
      })
      ElMessage.success('用户已创建')
    }
    dialogVisible.value = false
    fetchUsers()
  } catch (err) {
    console.error('Submit failed', err)
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (row: any) => {
  try {
    await ElMessageBox.confirm(`确定要删除用户 "${row.username}" 吗？`, '警告', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await userAPI.delete(row.id)
    ElMessage.success('用户已删除')
    fetchUsers()
  } catch (err) {
    // cancelled
  }
}

onMounted(fetchUsers)
</script>

<style scoped>
.users-page {
  height: 100%;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-title {
  font-size: 22px;
  color: #303133;
}
</style>