<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  if (!username.value || !password.value) {
    error.value = 'Kullanıcı adı ve şifre gerekli'
    return
  }

  loading.value = true
  error.value = ''
  try {
    await authStore.login(username.value, password.value)
    router.push('/')
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Giriş başarısız'
  } finally {
    loading.value = false
  }
}

// Hızlı giriş - geliştirme aşaması için
async function quickLogin() {
  username.value = 'admin'
  password.value = '123456'
  loading.value = true
  error.value = ''
  try {
    await authStore.login('admin', '123456')
    router.push('/')
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Giriş başarısız'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-card">
      <h1>⚡ OpenSpeed Panel</h1>
      <p class="subtitle">Modern Hosting Kontrol Paneli</p>

      <form @submit.prevent="handleLogin" class="login-form">
        <div class="form-group">
          <label>Kullanıcı Adı</label>
          <input
            v-model="username"
            type="text"
            placeholder="admin"
            :disabled="loading"
          />
        </div>
        <div class="form-group">
          <label>Şifre</label>
          <input
            v-model="password"
            type="password"
            placeholder="••••••••"
            :disabled="loading"
          />
        </div>

        <div v-if="error" class="error">{{ error }}</div>

        <button type="submit" :disabled="loading" class="btn-primary">
          {{ loading ? 'Giriş yapılıyor...' : 'Giriş Yap' }}
        </button>

        <button type="button" class="btn-quick" @click="quickLogin" :disabled="loading">
          ⚡ Hızlı Giriş (admin)
        </button>
      </form>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%);
}

.login-card {
  background: white;
  border-radius: 16px;
  padding: 40px;
  width: 100%;
  max-width: 400px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

h1 {
  margin: 0;
  text-align: center;
  font-size: 28px;
  color: #1a1a2e;
}

.subtitle {
  text-align: center;
  color: #666;
  margin: 8px 0 30px;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 14px;
  font-weight: 600;
  color: #333;
}

.form-group input {
  padding: 12px 16px;
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  font-size: 16px;
  transition: border-color 0.2s;
}

.form-group input:focus {
  outline: none;
  border-color: #0f3460;
}

.error {
  background: #ffe0e0;
  color: #c0392b;
  padding: 10px 16px;
  border-radius: 8px;
  font-size: 14px;
}

.btn-primary {
  padding: 14px;
  background: #0f3460;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: #1a4a7a;
}

.btn-primary:disabled {
  background: #999;
  cursor: not-allowed;
}

.btn-quick {
  padding: 10px;
  background: transparent;
  color: #0f3460;
  border: 2px dashed #0f3460;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-quick:hover {
  background: rgba(15, 52, 96, 0.05);
  border-style: solid;
}

.btn-quick:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
