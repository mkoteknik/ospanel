<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref } from 'vue'
import { api } from '@/api/client'
import { useMonitor } from '@/composables/useMonitor'

const { data: liveStats, wsOk } = useMonitor(2000)
const stats = liveStats
const loading = ref(true)
const { t } = useI18n()
const hostname = ref({ hostname: '', fqdn: '', ip: '' })
const showHostnameEdit = ref(false)
const newHostname = ref('')

async function loadHostname() {
  try { const r = await api.get('/api/v1/server/hostname'); hostname.value = r.data } catch { }
}

async function setHostname() {
  try { await api.put('/api/v1/server/hostname', { hostname: newHostname.value }); showHostnameEdit.value = false; loadHostname() } catch { }
}

// Fallback loading state
setTimeout(() => { loading.value = false }, 1200)
loadHostname()
</script>

<template>
  <div class="monitor-page">
    <div class="page-head">
      <div>
        <h2>{{ t('auto.662044') }}</h2>
        <p>{{ t('auto.47d9c7') }}</p>
      </div>
      <span class="kicker live"><span class="dot"></span>{ t('monitor.livePoll') }</span>
    </div>

    <div class="aura-card host-card">
      <div class="host-left">
        <div class="host-icon">◰</div>
        <div class="host-info">
          <span class="kicker">Sunucu hostname</span>
          <div class="host-value">{{ hostname.hostname || '—' }}</div>
          <div class="host-meta">IP: {{ hostname.ip || '—' }} · FQDN: {{ hostname.fqdn || '—' }}</div>
        </div>
      </div>
      <button class="aura-btn aura-btn-ghost sm" @click="newHostname = hostname.hostname; showHostnameEdit = true">{{ t('auto.dcecac') }}</button>
    </div>

    <div v-if="showHostnameEdit" class="overlay" @click.self="showHostnameEdit=false">
      <div class="aura-card modal sm">
        <div class="modal-head">
          <div>
            <span class="kicker">Hostname</span>
            <h3 class="modal-title">{{ t('auto.cd6453') }}</h3>
          </div>
          <button class="icon-btn" @click="showHostnameEdit=false">×</button>
        </div>
        <div class="modal-body">
          <label class="field">
            <span class="kicker">Yeni hostname</span>
            <input v-model="newHostname" placeholder="server.site.com" @keyup.enter="setHostname" />
          </label>
          <p class="hint">{{ t('auto.2977d8') }}</p>
        </div>
        <div class="modal-foot">
          <button class="aura-btn aura-btn-ghost" @click="showHostnameEdit=false">{{ t('common.cancel') }}</button>
          <button class="aura-btn aura-btn-primary" @click="setHostname">{{ t('common.save') }}</button>
        </div>
      </div>
    </div>

    <div class="stats-grid">
      <div class="aura-card stat">
        <span class="kicker">{{ t('auto.d8c117') }}</span>
        <div class="stat-value">{{ stats.cpu?.cores ?? '—' }}</div>
        <span class="kicker subtle">{{ t('dashboard.cpu') }}</span>
      </div>
      <div class="aura-card stat">
        <span class="kicker">Goroutines</span>
        <div class="stat-value">{{ stats.goroutines ?? '—' }}</div>
        <span class="kicker subtle">{{ t('common.active') }}</span>
      </div>
      <div class="aura-card stat">
        <span class="kicker">Go Versiyon</span>
        <div class="stat-value mono">{{ stats.go_version || '—' }}</div>
        <span class="kicker subtle">Runtime</span>
      </div>
      <div class="aura-card stat">
        <span class="kicker">{{ t('auto.61f6e7') }}</span>
        <div class="stat-value mono">{{ stats.os ? stats.os.toUpperCase() : '—' }}</div>
        <span class="kicker subtle">{{ stats.arch || '—' }}</span>
      </div>
    </div>

    <div class="aura-card info-card">
      <div class="info-head">
        <span class="kicker">Sistem bilgisi</span>
        <h3>{{ t('auto.1e0fb4') }}</h3>
      </div>
      <div class="info-rows">
        <div class="info-row"><span class="kicker">{{ t('auto.61f6e7') }}</span><span class="info-value">{{ stats.os ? stats.os + ' / ' + stats.arch : '—' }}</span></div>
        <div class="info-row"><span class="kicker">Go Versiyon</span><span class="info-value mono">{{ stats.go_version || '—' }}</span></div>
        <div class="info-row"><span class="kicker">{{ t('auto.d8c117') }}</span><span class="info-value">{{ stats.cpu?.cores ?? '—' }}</span></div>
        <div class="info-row"><span class="kicker">Aktif Goroutines</span><span class="info-value">{{ stats.goroutines ?? '—' }}</span></div>
      </div>
    </div>

    <p class="note">{{ t('auto.3bd258') }}</p>
  </div>
</template>

<style scoped>
.monitor-page { display: flex; flex-direction: column; gap: 16px; }
.page-head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
.page-head h2 { font-size: 20px; font-weight: 700; letter-spacing: -0.02em; color: var(--aura-text); }
.page-head p { margin-top: 4px; font-size: 13px; color: var(--aura-text-muted); max-width: 560px; }
.kicker { font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; color: var(--aura-text-faint); }
.kicker.subtle { color: var(--aura-text-faint); font-weight: 600; }
.kicker.live { display: inline-flex; align-items: center; gap: 6px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); padding: 6px 10px; border-radius: 999px; color: var(--aura-text-muted); }
.dot { width: 6px; height: 6px; border-radius: 999px; background: var(--aura-success); box-shadow: 0 0 0 4px color-mix(in srgb, var(--aura-success) 18%, transparent); }

.host-card { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 16px 18px; }
.host-left { display: flex; align-items: center; gap: 14px; min-width: 0; }
.host-icon { width: 40px; height: 40px; border-radius: 10px; display: grid; place-items: center; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); color: var(--aura-text-muted); font-size: 18px; }
.host-info { min-width: 0; }
.host-value { font-size: 16px; font-weight: 700; letter-spacing: -0.015em; color: var(--aura-text); font-family: ui-monospace, monospace; }
.host-meta { font-size: 12px; color: var(--aura-text-faint); margin-top: 2px; }
.sm { padding: 8px 12px; font-size: 12px; }

.overlay { position: fixed; inset: 0; background: rgba(15,23,42,0.45); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 16px; }
.modal { width: 100%; overflow: hidden; }
.modal.sm { max-width: 420px; }
.modal-head { display: flex; justify-content: space-between; align-items: flex-start; padding: 18px 20px; border-bottom: 1px solid var(--aura-border); }
.modal-title { font-size: 15px; font-weight: 700; color: var(--aura-text); margin-top: 2px; }
.modal-body { padding: 18px 20px; display: flex; flex-direction: column; gap: 14px; }
.modal-foot { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid var(--aura-border); }
.field { display: flex; flex-direction: column; gap: 6px; }
.field input { width: 100%; padding: 10px 12px; border: 1px solid var(--aura-border); border-radius: 10px; font-size: 14px; background: var(--aura-surface); color: var(--aura-text); font-family: ui-monospace, monospace; }
.field input:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
.hint { font-size: 12px; color: var(--aura-text-faint); }
.icon-btn { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 8px; border: 1px solid var(--aura-border); background: var(--aura-surface); color: var(--aura-text-muted); cursor: pointer; }
.icon-btn:hover { background: var(--aura-surface-hover); color: var(--aura-text); }

.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 14px; }
.stat { padding: 16px; display: flex; flex-direction: column; gap: 6px; }
.stat-value { font-size: 22px; font-weight: 750; letter-spacing: -0.02em; color: var(--aura-text); line-height: 1.1; }
.stat-value.mono { font-size: 14px; font-family: ui-monospace, monospace; font-weight: 600; word-break: break-all; }

.info-card { padding: 18px; display: flex; flex-direction: column; gap: 14px; }
.info-head h3 { font-size: 14px; font-weight: 650; color: var(--aura-text); margin-top: 2px; }
.info-rows { display: flex; flex-direction: column; }
.info-row { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 10px 0; border-bottom: 1px solid var(--aura-border); }
.info-row:last-child { border-bottom: none; }
.info-value { font-size: 13px; font-weight: 550; color: var(--aura-text); }
.info-value.mono { font-family: ui-monospace, monospace; font-size: 12px; }

.note { color: var(--aura-text-faint); font-size: 12px; text-align: center; }
</style>
