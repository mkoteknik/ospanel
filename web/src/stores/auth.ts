import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api/client'
import router from '@/router'

interface User {
  id: number
  username: string
  email: string
  role: string
  totp_enabled: boolean
  status: string
  created_at: string
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const accessToken = ref<string | null>(localStorage.getItem('access_token'))
  const refreshToken = ref<string | null>(localStorage.getItem('refresh_token'))

  const isAuthenticated = computed(() => !!accessToken.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isReseller = computed(() => user.value?.role === 'reseller')
  const hasRole = (roles: string[]) => !!user.value && roles.includes(user.value.role)

  async function login(username: string, password: string, totpCode?: string) {
    const res = await api.post('/api/v1/auth/login', {
      username,
      password,
      totp_code: totpCode,
    })

    accessToken.value = res.data.access_token
    refreshToken.value = res.data.refresh_token
    user.value = res.data.user

    localStorage.setItem('access_token', res.data.access_token)
    localStorage.setItem('refresh_token', res.data.refresh_token)

    return res.data
  }

  async function fetchMe() {
    try {
      const res = await api.get('/api/v1/auth/me')
      user.value = res.data
    } catch {
      logout()
    }
  }

  function logout() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
    router.push('/login')
  }

  return {
    user,
    accessToken,
    refreshToken,
    isAuthenticated,
    isAdmin,
    isReseller,
    hasRole,
    login,
    fetchMe,
    logout,
  }
})
