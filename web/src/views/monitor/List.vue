<script setup lang="ts">
import { ref } from 'vue'
import { api } from '@/api/client'

const stats = ref<any>({})
const loading = ref(true)

async function loadStats() {
  try {
    const res = await api.get('/api/v1/monitor/stats')
    stats.value = res.data
  } catch { }
  finally { loading.value = false }
}

loadStats()
setInterval(loadStats, 10000) // 10 saniyede bir yenile
</script>

<template>
  <div class="monitor-page">
    <h2>📈 Sistem Monitoring</h2>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon">🖥️</div>
        <div class="stat-body">
          <div class="stat-value">{{ stats.cpu?.cores || '-' }}</div>
          <div class="stat-label">CPU Çekirdek</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">🧠</div>
        <div class="stat-body">
          <div class="stat-value">{{ stats.goroutines || '-' }}</div>
          <div class="stat-label">Goroutines</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">⚙️</div>
        <div class="stat-body">
          <div class="stat-value">{{ stats.go_version || '-' }}</div>
          <div class="stat-label">Go Versiyon</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon">💻</div>
        <div class="stat-body">
          <div class="stat-value">{{ stats.os?.toUpperCase() || '-' }}</div>
          <div class="stat-label">İşletim Sistemi</div>
        </div>
      </div>
    </div>

    <div class="info-card">
      <h3>Sistem Bilgisi</h3>
      <table class="info-table">
        <tr><td>İşletim Sistemi</td><td>{{ stats.os }} / {{ stats.arch }}</td></tr>
        <tr><td>Go Versiyon</td><td>{{ stats.go_version }}</td></tr>
        <tr><td>CPU Çekirdek</td><td>{{ stats.cpu?.cores }}</td></tr>
        <tr><td>Aktif Goroutines</td><td>{{ stats.goroutines }}</td></tr>
      </table>
    </div>

    <p class="note">⚡ Linux sunucuda CPU/RAM/Disk gerçek zamanlı metrikleri aktif olacak.</p>
  </div>
</template>

<style scoped>
.monitor-page { max-width: 1200px; }
h2 { margin: 0 0 24px; }

.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px; }
.stat-card { background: white; padding: 20px; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); display: flex; align-items: center; gap: 16px; }
.stat-icon { font-size: 32px; }
.stat-value { font-size: 24px; font-weight: 700; color: #1a1a2e; }
.stat-label { font-size: 13px; color: #888; }

.info-card { background: white; padding: 24px; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); margin-bottom: 16px; }
.info-card h3 { margin: 0 0 16px; }
.info-table { width: 100%; }
.info-table td { padding: 8px 0; border-bottom: 1px solid #f0f0f0; font-size: 14px; }
.info-table td:first-child { color: #888; width: 150px; }

.note { color: #888; font-size: 13px; text-align: center; margin-top: 20px; }
</style>
