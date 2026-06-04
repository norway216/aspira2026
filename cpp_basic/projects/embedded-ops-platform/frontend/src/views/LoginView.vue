<template>
  <div class="login-page">
    <n-card class="login-card" title="Embedded Ops Platform" size="large">
      <n-form ref="formRef" :model="form" :rules="rules">
        <n-form-item label="Username" path="username">
          <n-input v-model:value="form.username" placeholder="admin" />
        </n-form-item>
        <n-form-item label="Password" path="password">
          <n-input v-model:value="form.password" type="password" show-password-on="click" placeholder="admin123" />
        </n-form-item>
        <n-button type="primary" block :loading="loading" @click="handleLogin">
          Login
        </n-button>
      </n-form>
      <template #footer>
        <n-text depth="3" style="font-size: 12px">
          Default: admin / admin123
        </n-text>
      </template>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { authApi } from '../api/devices'
import { useAppStore } from '../stores/app'

const router = useRouter()
const message = useMessage()
const appStore = useAppStore()

const formRef = ref<any>(null)
const loading = ref(false)
const form = reactive({
  username: 'admin',
  password: 'admin123',
})

const rules = {
  username: { required: true, message: 'Please enter username', trigger: 'blur' },
  password: { required: true, message: 'Please enter password', trigger: 'blur' },
}

async function handleLogin() {
  loading.value = true
  try {
    const res = await authApi.login(form.username, form.password)
    appStore.setToken(res.data.access_token)
    message.success('Login successful')
    router.push('/dashboard')
  } catch (err: any) {
    message.error(err.response?.data?.error || 'Login failed')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100vh;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
  width: 400px;
}
</style>