<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/api/client'

const { t } = useI18n()
const authStore = useAuthStore()
const stats = ref({ domains: 0, databases: 0 })
const monitor = ref<any>(null)
const loading = ref(true)
let timer: number | null = null

async function fetchAll() {
  try {
    const [dRes, dbRes] = await Promise.allSettled([
      api.get('/api/v1/domains'),
      api.get('/api/v1/databases'),
    ])
    if (dRes.status === 'fulfilled') stats.value.domains = dRes.value.data.total || 0
    if (dbRes.status === 'fulfilled') stats.value.databases = dbRes.value.data.total || 0
  } catch {}
}

async function fetchMonitor() {
  try {
    const r = await api.get('/api/v1/monitor/stats')
    monitor.value = r.data
  } catch {}
}

onMounted(async () => {
  await Promise.all([fetchAll(), fetchMonitor()])
  loading.value = false
  // Live every 2s — lightweight polling (monitor endpoint is cached 500ms server-side)
  timer = window.setInterval(fetchMonitor, 2000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

function pct(v: number | undefined) {
  if (v == null) return 0
  return Math.max(0, Math.min(100, v))
}
function barColor(p: number) {
  if (p > 85) return 'var(--aura-danger)'
  if (p > 65) return 'var(--aura-warning)'
  return 'var(--aura-accent)'
}
</script>

<template>
  <div class="dash">
    <div class="dash-head">
      <div>
        <h1>{{ t('dashboard.welcome', { name: authStore.user?.username || '' }) }}</h1>
        <p>{{ t('dashboard.subtitle') }}</p>
      </div>
      <div class="head-actions">
        <span class="pill">{{ t('dashboard.version') }}</span>
        <span class="live" v-if="monitor"><span class="live-dot"></span> {{ t('dashboard.live') }}</span>
      </div>
    </div>

    <!-- Live resources — 3 distinct cards, not identical -->
    <div class="live-grid">
      <div class="aura-card live-card cpu">
        <div class="live-top">
          <span class="kicker">{{ t('dashboard.cpu') }}</span>
          <span class="live-badge"><span class="dot"></span>{{ t('dashboard.instant') }}</span>
        </div>
        <div class="live-value">
          <span class="num">{{ monitor?.cpu?.usage_percent ?? '—' }}</span>
          <span class="unit" v-if="monitor?.cpu">%</span>
          <span class="cores" v-if="monitor?.cpu">· {{ monitor.cpu.cores }} {{ t('dashboard.cores') }}</span>
        </div>
        <div class="bar-track"><div class="bar-fill" :style="{ width: pct(monitor?.cpu?.usage_percent) + '%', background: barColor(pct(monitor?.cpu?.usage_percent)) }"></div></div>
        <div class="live-foot">
          <span>{{ t('dashboard.load1', { value: monitor?.load?.load1?.toFixed(2) ?? '—' }) }}</span>
          <span class="muted">{{ monitor ? t('dashboard.refresh2s') : '—' }}</span>
        </div>
      </div>

      <div class="aura-card live-card ram">
        <div class="live-top">
          <span class="kicker">{{ t('dashboard.ram') }}</span>
          <span class="ram-meta" v-if="monitor?.memory">{{ monitor.memory.used_gb }} / {{ monitor.memory.total_gb }} GB</span>
        </div>
        <div class="live-value">
          <span class="num">{{ monitor?.memory?.usage_percent ?? '—' }}</span>
          <span class="unit" v-if="monitor?.memory">%</span>
        </div>
        <div class="bar-track"><div class="bar-fill" :style="{ width: pct(monitor?.memory?.usage_percent) + '%', background: barColor(pct(monitor?.memory?.usage_percent)) }"></div></div>
        <div class="live-foot">
          <span>{{ monitor?.memory?.free_gb ?? '—' }} {{ t('common.freeGb') }}</span>
          <span class="muted">{{ loading ? t('common.loading') : t('common.realtime') }}</span>
        </div>
      </div>

      <div class="aura-card live-card disk">
        <div class="live-top">
          <span class="kicker">{{ t('dashboard.disk') }}</span>
          <span class="disk-free" v-if="monitor?.disk">{{ t('dashboard.free', { value: monitor.disk.free_gb }) }}</span>
        </div>
        <div class="live-value">
          <span class="num">{{ monitor?.disk?.usage_percent ?? '—' }}</span>
          <span class="unit" v-if="monitor?.disk">%</span>
        </div>
        <div class="bar-track"><div class="bar-fill" :style="{ width: pct(monitor?.disk?.usage_percent) + '%', background: barColor(pct(monitor?.disk?.usage_percent)) }"></div></div>
        <div class="live-foot">
          <span>{{ t('dashboard.usedTotal', { used: monitor?.disk?.used_gb ?? '—', total: monitor?.disk?.total_gb ?? '—' }) }}</span>
          <span class="muted">/</span>
        </div>
      </div>
    </div>

    <!-- Domain / DB — compact, not bento -->
    <div class="mini-grid">
      <div class="aura-card mini">
        <div class="mini-kicker">{{ t('dashboard.domains') }}</div>
        <div class="mini-value">{{ loading ? '…' : stats.domains }}</div>
        <div class="mini-label">{{ t('dashboard.domainsLabel') }}</div>
      </div>
      <div class="aura-card mini">
        <div class="mini-kicker">{{ t('dashboard.databases') }}</div>
        <div class="mini-value">{{ loading ? '…' : stats.databases }}</div>
        <div class="mini-label">{{ t('dashboard.databasesLabel') }}</div>
      </div>
      <div class="aura-card mini">
        <div class="mini-kicker">{{ t('dashboard.uptimeLabel') }}</div>
        <div class="mini-value" style="font-size: 16px;">{{ monitor?.uptime_seconds ? t('dashboard.uptimeValue', { hours: Math.floor(monitor.uptime_seconds/3600) }) : '—' }}</div>
        <div class="mini-label">{{ t('dashboard.optimization') }}</div>
      </div>
    </div>

    <div class="quick">
      <h2>{{ t('dashboard.quickTitle') }}</h2>
      <div class="quick-grid">
        <button class="aura-card quick-card" @click="$router.push('/domains')">
          <span class="qc-title">{{ t('dashboard.addDomain') }}</span>
          <span class="qc-desc">{{ t('dashboard.addDomainDesc') }}</span>
          <span class="qc-arrow">→</span>
        </button>
        <button class="aura-card quick-card" @click="$router.push('/files')">
          <span class="qc-title">{{ t('dashboard.files') }}</span>
          <span class="qc-desc">{{ t('dashboard.filesDesc') }}</span>
          <span class="qc-arrow">→</span>
        </button>
        <button class="aura-card quick-card" @click="$router.push('/databases')">
          <span class="qc-title">{{ t('dashboard.database') }}</span>
          <span class="qc-desc">{{ t('dashboard.databaseDesc') }}</span>
          <span class="qc-arrow">→</span>
        </button>
        <button class="aura-card quick-card" @click="$router.push('/monitor')">
          <span class="qc-title">{{ t('dashboard.monitoring') }}</span>
          <span class="qc-desc">{{ t('dashboard.monitoringDesc') }}</span>
          <span class="qc-arrow">→</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dash { display: flex; flex-direction: column; gap: 20px; }
.dash-head { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
.dash-head h1 { font-size: 22px; font-weight: 720; letter-spacing: -0.02em; color: var(--aura-text); }
.dash-head p { margin-top: 4px; font-size: 13px; color: var(--aura-text-muted); }
.head-actions { display: flex; align-items: center; gap: 8px; }
.pill { font-size: 11px; font-weight: 600; letter-spacing: 0.04em; padding: 6px 10px; border-radius: 999px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); color: var(--aura-text-faint); }
.live { display: inline-flex; align-items: center; gap: 6px; font-size: 11px; font-weight: 600; color: var(--aura-success); }
.live-dot { width: 7px; height: 7px; border-radius: 999px; background: var(--aura-success); box-shadow: 0 0 0 4px rgba(5,150,105,0.15); animation: pulse 1.8s infinite; }
@keyframes pulse { 0% { box-shadow: 0 0 0 0 rgba(5,150,105,0.25); } 70% { box-shadow: 0 0 0 6px rgba(5,150,105,0); } 100% { box-shadow: 0 0 0 0 rgba(5,150,105,0); } }
@media (prefers-reduced-motion: reduce) { .live-dot { animation: none; } }

/* Live grid — not identical cards: cpu has cores, ram has GB, disk has free */
.live-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 14px; }
@media (max-width: 980px) { .live-grid { grid-template-columns: 1fr; } }
.live-card { padding: 16px 16px 14px; display: flex; flex-direction: column; gap: 10px; }
.live-top { display: flex; align-items: center; justify-content: space-between; }
.kicker { font-size: 10px; font-weight: 700; letter-spacing: 0.05em; text-transform: uppercase; color: var(--aura-text-faint); }
.live-badge { font-size: 10px; font-weight: 600; padding: 4px 8px; border-radius: 999px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); color: var(--aura-text-faint); display: inline-flex; align-items: center; gap: 5px; }
.live-badge .dot { width: 6px; height: 6px; border-radius: 999px; background: var(--aura-success); }
.ram-meta, .disk-free { font-size: 11px; font-weight: 550; color: var(--aura-text-muted); }

.live-value { display: flex; align-items: baseline; gap: 4px; line-height: 1; }
.live-value .num { font-size: 30px; font-weight: 750; letter-spacing: -0.03em; color: var(--aura-text); }
.live-value .unit { font-size: 16px; font-weight: 600; color: var(--aura-text-muted); }
.live-value .cores { font-size: 11px; color: var(--aura-text-faint); margin-left: 6px; }

.bar-track { height: 6px; border-radius: 999px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); overflow: hidden; }
.bar-fill { height: 100%; border-radius: 999px; transition: width 0.6s cubic-bezier(0.2,0.8,0.2,1), background 0.3s; }

.live-foot { display: flex; align-items: center; justify-content: space-between; font-size: 11px; color: var(--aura-text-muted); }
.live-foot .muted { color: var(--aura-text-faint); }

.mini-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
@media (max-width: 700px) { .mini-grid { grid-template-columns: 1fr; } }
.mini { padding: 14px 16px; }
.mini-kicker { font-size: 10px; font-weight: 700; letter-spacing: 0.05em; text-transform: uppercase; color: var(--aura-text-faint); }
.mini-value { margin-top: 6px; font-size: 22px; font-weight: 720; letter-spacing: -0.02em; color: var(--aura-text); }
.mini-label { margin-top: 2px; font-size: 12px; color: var(--aura-text-muted); }

.quick h2 { font-size: 13px; font-weight: 700; letter-spacing: -0.01em; color: var(--aura-text); margin-bottom: 10px; }
.quick-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
@media (max-width: 1100px) { .quick-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 640px) { .quick-grid { grid-template-columns: 1fr; } }
.quick-card { text-align: left; padding: 14px 16px; display: flex; flex-direction: column; gap: 4px; cursor: pointer; position: relative; }
.quick-card:hover { border-color: color-mix(in srgb, var(--aura-accent) 22%, var(--aura-border)); }
.qc-title { font-size: 13px; font-weight: 650; color: var(--aura-text); }
.qc-desc { font-size: 12px; color: var(--aura-text-muted); line-height: 1.4; }
.qc-arrow { position: absolute; top: 14px; right: 12px; color: var(--aura-text-faint); font-size: 14px; transition: transform 0.15s; }
.quick-card:hover .qc-arrow { transform: translateX(2px); color: var(--aura-accent); }
</style>
