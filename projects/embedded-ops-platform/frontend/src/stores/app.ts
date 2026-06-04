import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { darkTheme } from 'naive-ui'

export const useAppStore = defineStore('app', () => {
  const isDark = ref(false)
  const token = ref(localStorage.getItem('access_token') || '')
  const wsConnected = ref(false)

  const theme = computed(() => isDark.value ? darkTheme : null)

  function setToken(newToken: string) {
    token.value = newToken
    localStorage.setItem('access_token', newToken)
  }

  function logout() {
    token.value = ''
    localStorage.removeItem('access_token')
    window.location.href = '/login'
  }

  function toggleTheme() {
    isDark.value = !isDark.value
  }

  return {
    isDark,
    token,
    wsConnected,
    theme,
    setToken,
    logout,
    toggleTheme,
  }
})