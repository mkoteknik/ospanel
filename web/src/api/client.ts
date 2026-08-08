import axios from 'axios'
import { useAuthStore } from '@/stores/auth'

export const api = axios.create({
  baseURL: '',
  timeout: 30000,
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

let isRefreshing = false
let failedQueue: Array<{ resolve: (v: unknown) => void; reject: (e: unknown) => void; config: any }> = []

function processQueue(error: unknown, token: string | null = null) {
  failedQueue.forEach((prom) => {
    if (error) {
      prom.reject(error)
    } else {
      prom.config.headers.Authorization = `Bearer ${token}`
      prom.resolve(api(prom.config))
    }
  })
  failedQueue = []
}

// Request interceptor - JWT token ekle (memory + localStorage fallback + cookie)
api.interceptors.request.use((config) => {
  const authStore = useAuthStore()
  const token = authStore.accessToken || localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  // CSRF
  const csrf = document.cookie.match(/(?:^| )csrf_token=([^;]*)/)?.[1]
  if (csrf) {
    config.headers['X-CSRF-Token'] = decodeURIComponent(csrf)
  }
  return config
})

// Response interceptor - 401 yakala + refresh mutex
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config
    if (error.response?.status === 401 && !originalRequest._retry) {
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject, config: originalRequest })
        })
      }

      originalRequest._retry = true
      isRefreshing = true

      const authStore = useAuthStore()
      const rt = authStore.refreshToken || localStorage.getItem('refresh_token')

      if (rt) {
        try {
          const res = await axios.post('/api/v1/auth/refresh', {
            refresh_token: rt,
          }, { withCredentials: true })

          const newAccess = res.data.access_token
          const newRefresh = res.data.refresh_token

          authStore.accessToken = newAccess
          authStore.refreshToken = newRefresh

          localStorage.setItem('access_token', newAccess)
          if (newRefresh) localStorage.setItem('refresh_token', newRefresh)

          originalRequest.headers.Authorization = `Bearer ${newAccess}`
          processQueue(null, newAccess)
          return api(originalRequest)
        } catch (e) {
          processQueue(e, null)
          authStore.logout()
          return Promise.reject(e)
        } finally {
          isRefreshing = false
        }
      } else {
        isRefreshing = false
        authStore.logout()
      }
    }
    return Promise.reject(error)
  }
)
