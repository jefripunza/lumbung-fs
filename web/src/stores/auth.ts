import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '@/lib/api'
import axios from 'axios'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>(localStorage.getItem('lumbungfs_token') || '')
  const username = ref<string>(localStorage.getItem('lumbungfs_user') || '')
  const loginError = ref<string>('')
  const isLoading = ref(false)

  const isAuthenticated = computed(() => !!token.value)

  async function login(user: string, password: string) {
    loginError.value = ''
    isLoading.value = true
    try {
      const { data } = await api.post('/auth/login', {
        username: user,
        password: password,
      })
      token.value = data.token
      username.value = user
      localStorage.setItem('lumbungfs_token', data.token)
      localStorage.setItem('lumbungfs_user', user)
      return true
    } catch (err) {
      if (axios.isAxiosError(err)) {
        loginError.value = err.response?.data?.error || 'Login failed'
      } else {
        loginError.value = 'Login failed'
      }
      return false
    } finally {
      isLoading.value = false
    }
  }

  function logout() {
    token.value = ''
    username.value = ''
    localStorage.removeItem('lumbungfs_token')
    localStorage.removeItem('lumbungfs_user')
  }

  return { token, username, loginError, isLoading, isAuthenticated, login, logout }
})
