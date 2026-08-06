<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

const stats = ref<any>({ installed: false })
const info = ref<any>({})
const loading = ref(true)
const flushing = ref(false)
const showInfo = ref(false)

async function loadStatus() {
  try {
    const res = await api.get('/api/v1/cache/status')
    stats.value = res.data
  } catch { }
  finally { loading.value = false }
}

async function loadInfo() {
  try {
    const res = await api.get('/api/v1/cache/info')
    info.value = res.data
    showInfo.value = true
  } catch { }
}

async function flushCache() {
  if (!confirm('Tüm Redis cache temizlenecek. Emin misiniz?')) return
  flushing.value = true
  try {
    await api.post('/api/v1/cache/flush')
    await loadStatus()
  } catch { }
  finally { flushing.value = false }
}

onMounted(loadStatus)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>⚡ Redis Cache</h2>
        <p>Yüksek performanslı bellek içi cache yönetimi.</p>
      </div>
      <div class="header-actions">
        <button class="btn-action" @click="loadInfo">📋 Detaylı Bilgi</button>
        <button class="btn-danger" :disabled="flushing" @click="flushCache">
          {{ flushing ? 'Temizleniyor...' : '🗑️ Cache Temizle' }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading">Redis durumu kontrol ediliyor...</div>

    <div v-else-if="!stats.installed" class="not-installed">
      <div class="ni-icon">⚡</div>
      <h3>Redis kurulu değil</h3>
      <p>Redis cache, web sitelerinizi 10 kata kadar hızlandırabilir.</p>
      <div class="install-cmd">
        <code>sudo apt install redis-server -y</code>
      </div>
      <p class="ni-note">Linux sunucuda kurulum scripti Redis'i otomatik kuracaktır.</p>
    </div>

    <div v-else>
      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon">📦</div>
          <div class="stat-body">
            <div class="stat-value">{{ stats.version || '-' }}</div>
            <div class="stat-label">Versiyon</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon">⏱️</div>
          <div class="stat-body">
            <div class="stat-value">{{ stats.uptime_days || '0' }} gün</div>
            <div class="stat-label">Uptime</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon">💾</div>
          <div class="stat-body">
            <div class="stat-value">{{ stats.used_memory || '-' }}</div>
            <div class="stat-label">Bellek</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon">👥</div>
          <div class="stat-body">
            <div class="stat-value">{{ stats.connected || '0' }}</div>
            <div class="stat-label">Bağlantı</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon">🔑</div>
          <div class="stat-body">
            <div class="stat-value">{{ stats.total_keys || '0' }}</div>
            <div class="stat-label">Anahtar</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon">⚡</div>
          <div class="stat-body">
            <div class="stat-value">{{ stats.ops_per_sec || '0' }}</div>
            <div class="stat-label">İşlem/sn</div>
          </div>
        </div>
      </div>

      <!-- Detaylı Bilgi Modal -->
      <div v-if="showInfo" class="modal-overlay" @click.self="showInfo = false">
        <div class="modal modal-lg">
          <div class="modal-header">
            <h3>📋 Redis Detaylı Bilgi</h3>
            <button class="modal-close" @click="showInfo = false">✕</button>
          </div>
          <div class="modal-body">
            <div class="info-grid">
              <div v-for="entry in Object.entries(info).filter(([k,v]) => typeof v === 'string')" :key="entry[0]" class="info-row">
                <span class="info-key">{{ entry[0] }}</span>
                <span class="info-val">{{ entry[1] }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 14px; }

.header-actions { display: flex; gap: 8px; }
.btn-action {
  padding: 10px 20px; background: white; border: 1px solid #ddd;
  border-radius: 8px; font-size: 14px; cursor: pointer;
}
.btn-action:hover { background: #f5f5f5; }
.btn-danger {
  padding: 10px 20px; background: #c0392b; color: white;
  border: none; border-radius: 8px; font-size: 14px; cursor: pointer; font-weight: 600;
}
.btn-danger:hover { background: #a93226; }
.btn-danger:disabled { background: #999; cursor: not-allowed; }

.loading { text-align: center; padding: 60px; color: #888; }

.not-installed {
  text-align: center; padding: 60px 20px; background: white;
  border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04);
}
.ni-icon { font-size: 56px; margin-bottom: 16px; }
.not-installed h3 { margin: 0 0 8px; color: #1a1a2e; }
.not-installed p { color: #888; margin: 0 0 16px; }
.install-cmd { margin-bottom: 12px; }
.install-cmd code {
  padding: 10px 16px; background: #1a1a2e; color: #e0e0e0;
  border-radius: 8px; font-size: 14px; font-family: 'Consolas', monospace;
}
.ni-note { font-size: 13px; color: #aaa; }

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
}

.stat-card {
  background: white; border-radius: 12px; padding: 20px;
  display: flex; align-items: center; gap: 16px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04); border: 1px solid #f0f0f0;
}
.stat-icon { font-size: 28px; }
.stat-value { font-size: 22px; font-weight: 700; color: #1a1a2e; }
.stat-label { font-size: 13px; color: #888; margin-top: 2px; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; max-width: 700px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; max-height: 60vh; overflow-y: auto; }
.info-grid { display: grid; gap: 4px; }
.info-row { display: flex; padding: 6px 0; border-bottom: 1px solid #f5f5f5; font-size: 13px; }
.info-key { color: #888; width: 200px; flex-shrink: 0; }
.info-val { color: #333; font-family: monospace; word-break: break-all; }
</style>
