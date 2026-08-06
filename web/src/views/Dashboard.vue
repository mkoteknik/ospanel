<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/api/client'

const authStore = useAuthStore()
const stats = ref({ domains: 0, databases: 0, emails: 0 })
const loading = ref(true)

onMounted(async () => {
  try {
    const [dRes, dbRes] = await Promise.all([
      api.get('/api/v1/domains'),
      api.get('/api/v1/databases'),
    ])
    stats.value.domains = dRes.data.total || 0
    stats.value.databases = dbRes.data.total || 0
  } catch { }
  finally { loading.value = false }
})

function getStat(key: string): number | string {
  if (loading.value) return '...'
  return (stats.value as any)[key] ?? '-'
}
</script>

<template>
  <div class="dashboard">
    <div class="welcome">
      <div>
        <h2>👋 Hoş Geldin, {{ authStore.user?.username }}</h2>
        <p class="welcome-sub">OpenSpeed Panel v0.1.1 — Go + Vue 3 + OpenLiteSpeed</p>
      </div>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon" style="background: #0f3460">🌐</div>
        <div class="stat-body">
          <div class="stat-value">{{ getStat('domains') }}</div>
          <div class="stat-label">Domainler</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: #1a4a7a">🗄️</div>
        <div class="stat-body">
          <div class="stat-value">{{ getStat('databases') }}</div>
          <div class="stat-label">Veritabanları</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: #2d6a9f">📧</div>
        <div class="stat-body">
          <div class="stat-value">{{ getStat('emails') }}</div>
          <div class="stat-label">E-Posta</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background: #3a7bbf">💿</div>
        <div class="stat-body">
          <div class="stat-value">-</div>
          <div class="stat-label">Disk</div>
        </div>
      </div>
    </div>

    <div class="quick-grid">
      <div class="quick-card" @click="$router.push('/domains')">
        <span class="qc-icon">🌐</span>
        <div>
          <strong>Domain Yönetimi</strong>
          <p>Web sitelerinizi ekleyin ve yönetin</p>
        </div>
        <span class="qc-arrow">→</span>
      </div>
      <div class="quick-card" @click="$router.push('/files')">
        <span class="qc-icon">📁</span>
        <div>
          <strong>Dosya Yöneticisi</strong>
          <p>Dosyalarınızı görüntüleyin ve düzenleyin</p>
        </div>
        <span class="qc-arrow">→</span>
      </div>
      <div class="quick-card" @click="$router.push('/databases')">
        <span class="qc-icon">🗄️</span>
        <div>
          <strong>Veritabanları</strong>
          <p>MySQL/MariaDB veritabanı oluşturun</p>
        </div>
        <span class="qc-arrow">→</span>
      </div>
      <div class="quick-card" @click="$router.push('/monitor')">
        <span class="qc-icon">📈</span>
        <div>
          <strong>Sistem Durumu</strong>
          <p>Sunucu kaynaklarını izleyin</p>
        </div>
        <span class="qc-arrow">→</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard { width: 100%; }

.welcome { margin-bottom: 24px; }
.welcome h2 { margin: 0; font-size: 22px; }
.welcome-sub { color: #888; margin: 4px 0 0; font-size: 14px; }

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
  border: 1px solid #f0f0f0;
}

.stat-icon {
  width: 48px; height: 48px;
  border-radius: 12px;
  display: flex; align-items: center; justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}

.stat-value { font-size: 26px; font-weight: 700; color: #1a1a2e; }
.stat-label { font-size: 13px; color: #888; margin-top: 2px; }

.quick-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 12px;
}

.quick-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 14px;
  cursor: pointer;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
  border: 1px solid #f0f0f0;
  transition: all 0.2s;
}

.quick-card:hover {
  box-shadow: 0 4px 16px rgba(0,0,0,0.08);
  border-color: #0f3460;
}

.qc-icon { font-size: 28px; flex-shrink: 0; }
.quick-card strong { display: block; font-size: 14px; color: #1a1a2e; margin-bottom: 2px; }
.quick-card p { margin: 0; font-size: 12px; color: #888; }
.qc-arrow { margin-left: auto; color: #ccc; font-size: 18px; }
</style>
