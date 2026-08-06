<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/api/client'

const authStore = useAuthStore()
const stats = ref({ domains: 0, databases: 0 })
const loading = ref(true)

onMounted(async () => {
  try {
    const [dRes, dbRes] = await Promise.all([
      api.get('/api/v1/domains'),
      api.get('/api/v1/databases'),
    ])
    stats.value.domains = dRes.data.total || 0
    stats.value.databases = dbRes.data.total || 0
  } catch {
    // API henüz hazır değil
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="dashboard">
    <h2>👋 Hoş Geldin, {{ authStore.user?.username }}</h2>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon">🌐</div>
        <div class="stat-info">
          <div class="stat-value">{{ loading ? '...' : stats.domains }}</div>
          <div class="stat-label">Domainler</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">🗄️</div>
        <div class="stat-info">
          <div class="stat-value">{{ loading ? '...' : stats.databases }}</div>
          <div class="stat-label">Veritabanları</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">📧</div>
        <div class="stat-info">
          <div class="stat-value">-</div>
          <div class="stat-label">E-Posta</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">💿</div>
        <div class="stat-info">
          <div class="stat-value">-</div>
          <div class="stat-label">Disk</div>
        </div>
      </div>
    </div>

    <div class="info-card">
      <h3>Sistem Bilgisi</h3>
      <p>OpenSpeed Panel v0.1.0 — Go + Vue 3 + OpenLiteSpeed</p>
      <p>Tüm sistemler aktif ve çalışıyor.</p>
    </div>
  </div>
</template>

<style scoped>
.dashboard h2 { margin: 0 0 24px; }

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: white;
  padding: 20px;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon { font-size: 32px; }

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #1a1a2e;
}

.stat-label {
  font-size: 13px;
  color: #888;
}

.info-card {
  background: white;
  padding: 24px;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}

.info-card h3 { margin: 0 0 12px; }
.info-card p { margin: 4px 0; color: #666; font-size: 14px; }
</style>
