<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

const { t } = useI18n()

interface SSLDomain { domain: string; domain_id: number; ssl_enabled: boolean; cert: { days_left: number; issuer: string; expires_at: string } | null }

const sslDomains = ref<SSLDomain[]>([])
const loading = ref(false)

async function loadSSL() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/ssl')
    sslDomains.value = res.data.ssl_domains || []
  } catch { sslDomains.value = [] }
  finally { loading.value = false }
}

async function renewCert(domainId: number) {
  try {
    await api.post('/api/v1/ssl/' + domainId + '/renew')
    alert('SSL yenilendi!')
    await loadSSL()
  } catch { }
}

async function deleteCert(domainId: number) {
  if (!confirm(t('ssl.confirmDelete'))) return
  try { await api.delete('/api/v1/ssl/' + domainId); await loadSSL() }
  catch { }
}

function daysClass(days: number) {
  if (days > 60) return 'tone-ok'
  if (days > 30) return 'tone-warn'
  return 'tone-critical'
}

onMounted(loadSSL)
</script>

<template>
  <div class="ssl-page">
    <div class="page-head">
      <div>
        <h2>{{ t('auto.62c50d') }}</h2>
        <p>{{ t('auto.114051') }}</p>
      </div>
      <button class="aura-btn aura-btn-ghost" @click="loadSSL">{{ t('common.refresh') }}</button>
    </div>

    <div v-if="loading" class="state muted">{{ t('common.loading') }}</div>

    <div v-else-if="sslDomains.length === 0" class="aura-card empty">
      <div class="empty-icon">◈</div>
      <div class="empty-value">{{ t('auto.928812') }}</div>
      <p class="empty-desc">{{ t('auto.286c62') }}</p>
    </div>

    <div v-else class="ssl-grid">
      <div v-for="d in sslDomains" :key="d.domain_id" class="aura-card ssl-card" :class="{ active: d.ssl_enabled && d.cert }">
        <div class="ssl-top">
          <span class="kicker">{{ d.ssl_enabled && d.cert ? t('common.active') : t('common.inactive') }}</span>
          <span v-if="d.ssl_enabled && d.cert" :class="'pill ' + daysClass(d.cert.days_left)">{{ d.cert.days_left }} {{ t('common.days') }}</span>
          <span v-else class="pill tone-none">SSL Yok</span>
        </div>
        <div class="ssl-domain">{{ d.domain }}</div>
        <div v-if="d.cert" class="ssl-meta">
          <div class="meta-row">
            <span class="kicker">Sertifika</span>
            <span class="meta-value plain">{{ d.cert.issuer }}</span>
          </div>
          <div class="meta-row">
            <span class="kicker">{{ t('auto.7cd21b') }}</span>
            <span class="meta-value plain">{{ d.cert.expires_at?.split('T')[0] || '-' }}</span>
          </div>
        </div>
        <div v-else class="ssl-meta muted">
          <span class="kicker">{{ t('auto.d11b15') }}</span>
        </div>
        <div class="ssl-actions">
          <template v-if="d.cert">
            <button class="aura-btn aura-btn-ghost sm" @click="renewCert(d.domain_id)">{{ t('common.refresh') }}</button>
            <button class="aura-btn aura-btn-ghost sm danger" @click="deleteCert(d.domain_id)">{{ t('common.delete') }}</button>
          </template>
          <router-link v-else :to="'/domains/' + d.domain_id" class="aura-btn aura-btn-primary sm">SSL Kur</router-link>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ssl-page { display: flex; flex-direction: column; gap: 16px; }
.page-head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
.page-head h2 { font-size: 20px; font-weight: 700; letter-spacing: -0.02em; color: var(--aura-text); }
.page-head p { margin-top: 4px; font-size: 13px; color: var(--aura-text-muted); max-width: 560px; }
.kicker { font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; color: var(--aura-text-faint); }
.state { text-align: center; padding: 32px; font-size: 13px; }
.state.muted { color: var(--aura-text-muted); }
.empty { text-align: center; padding: 32px 20px; display: flex; flex-direction: column; align-items: center; gap: 8px; }
.empty-icon { width: 44px; height: 44px; border-radius: 12px; display: grid; place-items: center; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); color: var(--aura-text-muted); font-size: 18px; }
.empty-value { font-size: 15px; font-weight: 650; color: var(--aura-text); }
.empty-desc { font-size: 13px; color: var(--aura-text-muted); }

.ssl-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 14px; }
.ssl-card { padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.ssl-card.active { border-color: color-mix(in srgb, var(--aura-success) 18%, var(--aura-border)); }
.ssl-top { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.ssl-domain { font-size: 16px; font-weight: 700; letter-spacing: -0.015em; color: var(--aura-text); word-break: break-all; }
.ssl-meta { display: flex; flex-direction: column; gap: 8px; padding: 10px 12px; border-radius: 10px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); }
.ssl-meta.muted { align-items: center; justify-content: center; min-height: 52px; }
.meta-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.meta-value.plain { font-size: 12px; color: var(--aura-text-muted); }
.pill { display: inline-flex; align-items: center; padding: 4px 10px; border-radius: 999px; font-size: 11px; font-weight: 700; border: 1px solid transparent; }
.pill.tone-ok { background: color-mix(in srgb, var(--aura-success) 12%, var(--aura-surface)); color: var(--aura-success); border-color: color-mix(in srgb, var(--aura-success) 18%, transparent); }
.pill.tone-warn { background: color-mix(in srgb, var(--aura-warning) 12%, var(--aura-surface)); color: var(--aura-warning); border-color: color-mix(in srgb, var(--aura-warning) 18%, transparent); }
.pill.tone-critical { background: color-mix(in srgb, var(--aura-danger) 10%, var(--aura-surface)); color: var(--aura-danger); border-color: color-mix(in srgb, var(--aura-danger) 18%, transparent); }
.pill.tone-none { background: var(--aura-bg-subtle); color: var(--aura-text-faint); border-color: var(--aura-border); }
.ssl-actions { display: flex; gap: 8px; }
.ssl-actions .sm { flex: 1; padding: 8px 10px; font-size: 13px; text-align: center; justify-content: center; }
.sm.danger { color: var(--aura-danger); }
.sm.danger:hover { background: color-mix(in srgb, var(--aura-danger) 8%, var(--aura-surface)); border-color: color-mix(in srgb, var(--aura-danger) 18%, var(--aura-border)); }
</style>
