<script setup lang="ts">
import { ref } from 'vue'
import { api } from '@/api/client'

const stats = ref<any>({})
const loading = ref(true)
const hostname = ref({ hostname: '', fqdn: '', ip: '' })
const showHostnameEdit = ref(false)
const newHostname = ref('')

async function loadHostname() {
  try { const r = await api.get('/api/v1/server/hostname'); hostname.value = r.data } catch { }
}

async function setHostname() {
  try { await api.put('/api/v1/server/hostname', { hostname: newHostname.value }); showHostnameEdit.value = false; loadHostname() } catch { }
}

async function loadStats() {
  try {
    const res = await api.get('/api/v1/monitor/stats')
    stats.value = res.data
  } catch { }
  finally { loading.value = false }
}

loadStats(); loadHostname()
setInterval(loadStats, 10000)
</script>

<template>
  <div class="monitor-page">
    <h2>📈 Sistem Monitoring</h2>

    <!-- Hostname -->
    <div class="hostname-card">
      <div class="hn-left">
        <span class="hn-icon">🖥️</span>
        <div>
          <div class="hn-label">Sunucu Hostname</div>
          <div class="hn-value">{{ hostname.hostname || '...' }}</div>
          <div class="hn-meta">IP: {{ hostname.ip }} | FQDN: {{ hostname.fqdn || '-' }}</div>
        </div>
      </div>
      <button class="btn-sm" @click="newHostname = hostname.hostname; showHostnameEdit = true">✏️ Değiştir</button>
    </div>

    <!-- Hostname Edit Modal -->
    <div v-if="showHostnameEdit" class="modal-overlay" @click.self="showHostnameEdit=false">
      <div class="modal modal-sm">
        <div class="modal-header"><h3>🖥️ Hostname Değiştir</h3><button class="modal-close" @click="showHostnameEdit=false">✕</button></div>
        <div class="modal-body">
          <div class="form-group"><label>Yeni Hostname</label><input v-model="newHostname" placeholder="server.site.com" @keyup.enter="setHostname" /></div>
          <small>PTR kaydını hosting panelinizden bu hostname'e yönlendirin.</small>
        </div>
        <div class="modal-footer"><button class="btn-cancel" @click="showHostnameEdit=false">İptal</button><button class="btn-primary" @click="setHostname">Kaydet</button></div>
      </div>
    </div>

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
.monitor-page { width: 100%; }
h2 { margin: 0 0 24px; }

.hostname-card { display: flex; justify-content: space-between; align-items: center; background: white; padding: 20px; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); margin-bottom: 20px; }
.hn-left { display: flex; align-items: center; gap: 14px; }
.hn-icon { font-size: 28px; }
.hn-label { font-size: 11px; color: #888; text-transform: uppercase; margin-bottom: 2px; }
.hn-value { font-size: 18px; font-weight: 700; color: #1a1a2e; font-family: monospace; }
.hn-meta { font-size: 12px; color: #888; margin-top: 2px; }
.btn-sm { padding: 8px 14px; background: white; border: 1px solid #ddd; border-radius: 6px; font-size: 13px; cursor: pointer; }
.btn-sm:hover { background: #f5f5f5; }
.btn-primary { padding: 10px 20px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-primary:hover { background: #1a4a7a; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-sm { max-width: 420px; }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; cursor: pointer; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 14px; font-weight: 600; margin-bottom: 6px; }
.form-group input { width: 100%; padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; font-family: monospace; }
.form-group input:focus { outline: none; border-color: #0f3460; }

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
