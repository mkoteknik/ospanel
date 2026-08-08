<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

const { t } = useI18n()

const stats = ref<any>({ installed: false })
const info = ref<any>({})
const loading = ref(true)
const actionLoading = ref('')
const showInfo = ref(false)
const installLoading = ref(false)

async function loadStatus() {
  try {
    const [statusRes, svcRes] = await Promise.all([
      api.get('/api/v1/cache/status'),
      api.get('/api/v1/services'),
    ])
    stats.value = statusRes.data

    // Servis durumunu da kontrol et
    const services = svcRes.data.services || []
    const redisSvc = services.find((s: any) => s.name === 'redis')
    if (redisSvc) {
      stats.value.installed = redisSvc.installed
      stats.value.active = redisSvc.active
      stats.value.enabled = redisSvc.enabled
    }
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

async function installRedis() {
  installLoading.value = true
  try {
    await api.post('/api/v1/services/action', { service: 'redis', action: 'install' })
    await loadStatus()
  } catch { }
  finally { installLoading.value = false }
}

async function toggleRedis(enable: boolean) {
  actionLoading.value = enable ? 'start' : 'stop'
  try {
    await api.post('/api/v1/services/action', { service: 'redis', action: enable ? 'start' : 'stop' })
    await loadStatus()
  } catch { }
  finally { actionLoading.value = '' }
}

async function doAction(action: string) {
  actionLoading.value = action
  try {
    await api.post('/api/v1/services/action', { service: 'redis', action })
    await loadStatus()
  } catch { }
  finally { actionLoading.value = '' }
}

async function flushCache() {
  if (!confirm(t('cache.confirmClear'))) return
  actionLoading.value = 'flush'
  try {
    await api.post('/api/v1/cache/flush')
    await loadStatus()
  } catch { }
  finally { actionLoading.value = '' }
}

onMounted(loadStatus)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>{{ t('cache.title') }}</h2>
        <p>{{ t('cache.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <button v-if="stats.installed" class="btn-secondary" @click="loadInfo">📋 {{ t('cache.detail') }}</button>
        <button v-if="stats.installed" class="btn-danger" :disabled="actionLoading === 'flush'" @click="flushCache">
          {{ actionLoading === 'flush' ? '⏳' : '🗑️' }} {{ t('cache.clearCache') }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>{{ t('cache.checking') }}</p>
    </div>

    <!-- Kurulu Değil -->
    <div v-else-if="!stats.installed" class="install-prompt">
      <div class="prompt-icon">⚡</div>
      <h3>{{ t('cache.notInstalled') }}</h3>
      <p>{{ t('cache.notInstalledDesc') }}</p>
      <div class="prompt-actions">
        <button class="btn-install" :disabled="installLoading" @click="installRedis">
          <span v-if="installLoading" class="btn-spinner"></span>
          {{ installLoading ? t('cache.installing') : '📥 ' + t('cache.install') }}
        </button>
        <code class="prompt-cmd">{{ t('cache.installCmd') }}</code>
      </div>
    </div>

    <!-- Kurulu -->
    <div v-else>
      <!-- Power Card -->
      <div class="power-card" :class="{ active: stats.active }">
        <div class="power-left">
          <div class="power-icon-wrap" :class="{ pulse: stats.active }">
            <span class="power-icon">⚡</span>
            <span v-if="stats.active" class="power-dot"></span>
          </div>
          <div class="power-info">
            <span class="power-name">{{ t('cache.redisServer') }}</span>
            <span class="power-status">{{ stats.active ? t('cache.running') : t('cache.stopped') }}</span>
          </div>
        </div>
        <div class="power-actions">
          <span v-if="stats.active" class="status-tag running">{{ t('security.active') }}</span>
          <span v-else class="status-tag stopped">{{ t('security.inactive') }}</span>
          <label class="toggle-switch" :class="{ loading: !!actionLoading && actionLoading !== 'flush' && actionLoading !== 'info' }">
            <input
              type="checkbox"
              :checked="stats.active"
              :disabled="!!actionLoading"
              @change="toggleRedis(!stats.active)"
            />
            <span class="toggle-slider"></span>
          </label>
        </div>
      </div>

      <!-- Stats Grid -->
      <div class="stats-grid">
        <div class="stat-card">
          <span class="stat-icon">📦</span>
          <span class="stat-val">{{ stats.version || '-' }}</span>
          <span class="stat-lbl">{{ t('cache.version') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-icon">⏱️</span>
          <span class="stat-val">{{ stats.uptime_days || '0' }} {{ t('cache.uptimeDays').replace('{count}', '') }}</span>
          <span class="stat-lbl">{{ t('cache.uptime') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-icon">💾</span>
          <span class="stat-val">{{ stats.used_memory || '-' }}</span>
          <span class="stat-lbl">{{ t('cache.memory') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-icon">👥</span>
          <span class="stat-val">{{ stats.connected || '0' }}</span>
          <span class="stat-lbl">{{ t('cache.connection') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-icon">🔑</span>
          <span class="stat-val">{{ stats.total_keys || '0' }}</span>
          <span class="stat-lbl">{{ t('cache.keys') }}</span>
        </div>
        <div class="stat-card">
          <span class="stat-icon">⚡</span>
          <span class="stat-val">{{ stats.ops_per_sec || '0' }}/s</span>
          <span class="stat-lbl">{{ t('cache.ops') }}</span>
        </div>
      </div>

      <!-- Boot & Restart -->
      <div class="action-bar">
        <button class="btn-sm" :disabled="!!actionLoading" @click="doAction('restart')">
          🔄 {{ t('cache.restart') }}
        </button>
        <button v-if="stats.enabled" class="btn-sm btn-boot-on" @click="doAction('disable')">
          🟢 {{ t('cache.bootOn') }}
        </button>
        <button v-else class="btn-sm btn-boot-off" @click="doAction('enable')">
          ⏻ {{ t('cache.bootOff') }}
        </button>
      </div>

      <!-- Info Modal -->
      <div v-if="showInfo" class="modal-overlay" @click.self="showInfo = false">
        <div class="modal modal-lg">
          <div class="modal-header">
            <h3>📋 {{ t('cache.detailInfo') }}</h3>
            <button class="modal-close" @click="showInfo = false">✕</button>
          </div>
          <div class="modal-body">
            <div class="info-grid">
              <div v-for="(val, key) in info" :key="key" class="info-row">
                <span class="info-key">{{ key }}</span>
                <span class="info-val">{{ val }}</span>
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
.page-header h2 { margin: 0; font-size: 22px; color: #1a1a2e; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 13px; }
.header-actions { display: flex; gap: 8px; }
.btn-secondary { padding: 8px 16px; background: white; border: 1px solid #e0e0e0; border-radius: 8px; font-size: 13px; cursor: pointer; }
.btn-secondary:hover { background: #f5f5f5; }
.btn-danger { padding: 8px 16px; background: #c0392b; color: white; border: none; border-radius: 8px; font-size: 13px; font-weight: 600; cursor: pointer; }
.btn-danger:hover { background: #a93226; }
.btn-danger:disabled { opacity: 0.6; cursor: not-allowed; }

/* Loading */
.loading-state { text-align: center; padding: 80px 0; }
.spinner { width: 36px; height: 36px; border: 3px solid #e0e0e0; border-top-color: #0f3460; border-radius: 50%; animation: spin 0.8s linear infinite; margin: 0 auto 16px; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Install Prompt */
.install-prompt { text-align: center; padding: 60px 20px; background: white; border-radius: 16px; border: 2px dashed #e0e0e0; }
.prompt-icon { font-size: 56px; margin-bottom: 12px; }
.install-prompt h3 { margin: 0 0 8px; font-size: 20px; color: #1a1a2e; }
.install-prompt p { color: #888; margin: 0 0 20px; }
.prompt-actions { display: flex; flex-direction: column; align-items: center; gap: 12px; }
.btn-install { display: flex; align-items: center; gap: 8px; padding: 14px 32px; background: linear-gradient(135deg, #0f3460, #1a4a7a); color: white; border: none; border-radius: 10px; font-size: 15px; font-weight: 600; cursor: pointer; transition: all 0.2s; }
.btn-install:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 6px 20px rgba(15,52,96,0.3); }
.btn-install:disabled { opacity: 0.7; cursor: not-allowed; }
.btn-spinner { width: 14px; height: 14px; border: 2px solid rgba(255,255,255,0.3); border-top-color: white; border-radius: 50%; animation: spin 0.6s linear infinite; }
.prompt-cmd { padding: 10px 20px; background: #1a1a2e; color: #4ecb71; border-radius: 8px; font-family: 'Consolas', monospace; font-size: 13px; }

/* Power Card */
.power-card { display: flex; align-items: center; justify-content: space-between; padding: 20px 24px; background: white; border-radius: 14px; border: 1px solid #f0f0f0; margin-bottom: 20px; transition: all 0.2s; }
.power-card.active { border-left: 3px solid #27ae60; background: #fafffe; }
.power-left { display: flex; align-items: center; gap: 14px; }
.power-icon-wrap { position: relative; width: 48px; height: 48px; display: flex; align-items: center; justify-content: center; background: #f5f6fa; border-radius: 12px; }
.power-icon { font-size: 26px; }
.power-dot { position: absolute; bottom: -2px; right: -2px; width: 10px; height: 10px; background: #27ae60; border: 2px solid white; border-radius: 50%; }
.pulse .power-dot { animation: pulse-dot 2s infinite; }
@keyframes pulse-dot { 0%,100%{box-shadow:0 0 0 0 rgba(39,174,96,0.4)} 50%{box-shadow:0 0 0 6px rgba(39,174,96,0)} }
.power-info { display: flex; flex-direction: column; }
.power-name { font-weight: 700; font-size: 16px; color: #1a1a2e; }
.power-status { font-size: 12px; color: #888; }
.power-actions { display: flex; align-items: center; gap: 14px; }
.status-tag { padding: 4px 12px; border-radius: 20px; font-size: 11px; font-weight: 600; }
.status-tag.running { background: #d4edda; color: #155724; }
.status-tag.stopped { background: #f8d7da; color: #721c24; }

/* Toggle Switch */
.toggle-switch { position: relative; display: inline-block; width: 48px; height: 26px; cursor: pointer; }
.toggle-switch input { opacity: 0; width: 0; height: 0; }
.toggle-slider { position: absolute; inset: 0; background: #ccc; border-radius: 26px; transition: 0.3s; }
.toggle-slider::before { content: ''; position: absolute; height: 20px; width: 20px; left: 3px; bottom: 3px; background: white; border-radius: 50%; transition: 0.3s; box-shadow: 0 1px 3px rgba(0,0,0,0.2); }
.toggle-switch input:checked + .toggle-slider { background: #27ae60; }
.toggle-switch input:checked + .toggle-slider::before { transform: translateX(22px); }
.toggle-switch.loading { opacity: 0.6; pointer-events: none; }

/* Stats */
.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; margin-bottom: 20px; }
.stat-card { background: white; border-radius: 12px; padding: 18px; display: flex; flex-direction: column; align-items: center; gap: 6px; border: 1px solid #f0f0f0; }
.stat-icon { font-size: 24px; }
.stat-val { font-size: 20px; font-weight: 700; color: #1a1a2e; }
.stat-lbl { font-size: 11px; color: #888; text-transform: uppercase; letter-spacing: 0.5px; }

/* Action Bar */
.action-bar { display: flex; gap: 8px; }
.btn-sm { padding: 8px 16px; background: white; border: 1px solid #e0e0e0; border-radius: 8px; font-size: 13px; cursor: pointer; transition: all 0.15s; }
.btn-sm:hover:not(:disabled) { background: #f5f5f5; }
.btn-sm:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-boot-on { border-color: #c3e6cb; background: #f0fff4; color: #155724; }
.btn-boot-off { background: #fafafa; color: #aaa; }

/* Modal */
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 14px; width: 90%; max-width: 700px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; max-height: 60vh; overflow-y: auto; }
.info-grid { display: flex; flex-direction: column; gap: 2px; }
.info-row { display: flex; padding: 5px 0; border-bottom: 1px solid #f5f5f5; font-size: 13px; }
.info-key { color: #888; width: 200px; flex-shrink: 0; }
.info-val { color: #333; font-family: monospace; word-break: break-all; }
</style>
