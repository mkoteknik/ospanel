import axios from 'axios'
import { useAuthStore } from '@/stores/auth'

export const api = axios.create({
  baseURL: '',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Request interceptor - JWT token ekle
api.interceptors.request.use((config) => {
  const authStore = useAuthStore()
  if (authStore.accessToken) {
    config.headers.Authorization = `Bearer ${authStore.accessToken}`
  }
  return config
})

// Response interceptor - 401 yakala
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      const authStore = useAuthStore()

      // Refresh token dene
      if (authStore.refreshToken) {
        try {
          const res = await axios.post('/api/v1/auth/refresh', {
            refresh_token: authStore.refreshToken,
          })

          authStore.accessToken = res.data.access_token
          authStore.refreshToken = res.data.refresh_token

          localStorage.setItem('access_token', res.data.access_token)
          localStorage.setItem('refresh_token', res.data.refresh_token)

          // Orijinal isteği yeni token ile tekrar dene
          error.config.headers.Authorization = `Bearer ${res.data.access_token}`
          return api(error.config)
        } catch {
          authStore.logout()
        }
      } else {
        authStore.logout()
      }
    }
    return Promise.reject(error)
  }
)
