<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const domainId = ref(Number(route.params.id))

interface DomainInfo {
  id: number; domain: string; document_root: string; php_version: string
  ssl_enabled: boolean; force_https: boolean; status: string; created_at: string
  disk_usage_mb?: number; home_usage_mb?: number; quota_limit_mb?: number
}

const domain = ref<DomainInfo | null>(null)
const loading = ref(true)
const activeTab = ref('overview')
const error = ref('')

// Database
const databases = ref<any[]>([])
const showCreateDB = ref(false)
const newDB = ref({ name: '', username: '', password: '', charset: 'utf8mb4' })

// Subdomain
const subdomains = ref<any[]>([])
const showSubModal = ref(false)
const newSub = ref({ subdomain: '', php_version: '8.3' })
const subCreating = ref(false)
const subSteps = ref<any[]>([])
const subResult = ref<any>(null)

// Email
const emails = ref<any[]>([])
const showCreateEmail = ref(false)
const newEmail = ref({ email: '', password: '', quota: 1024 })

// Actions
const changingPHP = ref(false)
const phpExtensions = ref<any[]>([])
const installingSSL = ref(false)
const sslResult = ref('')

async function loadPHPExtensions(version?: string) {
  try {
    const res = await api.get('/api/v1/ols/php-extensions', { params: { version: version || domain.value?.php_version || '8.3' } })
    phpExtensions.value = res.data.extensions || []
  } catch { phpExtensions.value = [] }
}

async function loadDomain() {
  loading.value = true
  try {
    const res = await api.get(`/api/v1/domains/${domainId.value}`)
    domain.value = res.data.domain || res.data
    await Promise.all([loadDatabases(), loadEmails(), loadSubdomains(), loadAliases(), loadPHPExtensions()])
  } catch { error.value = t('domainDetail.loadFailed'); router.push('/domains') }
  finally { loading.value = false }
}

async function loadDatabases() {
  try {
    const res = await api.get('/api/v1/databases')
    databases.value = (res.data.databases || []).filter((db: any) =>
      db.name.includes(domain.value!.domain.replace('.', '_'))
    )
  } catch { databases.value = [] }
}

async function loadSubdomains() {
  try {
    const res = await api.get(`/api/v1/domains/${domainId.value}/subdomains`)
    subdomains.value = res.data.subdomains || []
  } catch { subdomains.value = [] }
}

function openSubdomainModal() {
  newSub.value = { subdomain: '', php_version: domain.value?.php_version || '8.3' }
  subSteps.value = []
  subResult.value = null
  showSubModal.value = true
}

async function createSubdomain() {
  if (!newSub.value.subdomain) return
  subCreating.value = true
  subSteps.value = []
  subResult.value = null

  // Başlangıç adımları (pending)
  const stepNames = [
    'Dosya dizini oluşturuluyor...',
    'OLS Sanal Host hazırlanıyor...',
    'DNS kaydı ekleniyor...',
    'SSL kontrolü yapılıyor...',
    'Alt domain tamamlanıyor...',
  ]
  subSteps.value = stepNames.map((n, i) => ({ step: i+1, name: n, status: 'pending', detail: '' }))

  try {
    const res = await api.post(`/api/v1/domains/${domainId.value}/subdomains`, newSub.value)
    // Backend'den gelen gerçek adımları göster
    subSteps.value = res.data.steps || []
    subResult.value = res.data
    await loadSubdomains()
    newSub.value = { subdomain: '', php_version: domain.value?.php_version || '8.3' }
  } catch (err: any) {
    subSteps.value.push({ step: subSteps.value.length+1, name: 'Hata', status: 'failed', detail: err.response?.data?.error || 'Oluşturulamadı' })
  }
  finally { subCreating.value = false }
}

// Aliases (Parked Domain)
const aliases = ref<any[]>([])
const showCreateAlias = ref(false)
const newAlias = ref({ alias: '', type: 'park', target: '' })

async function loadAliases() {
  try {
    const res = await api.get(`/api/v1/domains/${domainId.value}/aliases`)
    aliases.value = res.data.aliases || []
  } catch { aliases.value = [] }
}

async function createAlias() {
  if (!newAlias.value.alias) return
  try {
    await api.post(`/api/v1/domains/${domainId.value}/aliases`, newAlias.value)
    showCreateAlias.value = false
    newAlias.value = { alias: '', type: 'park', target: '' }
    await loadAliases()
  } catch (err: any) { error.value = err.response?.data?.error || 'Alias eklenemedi' }
}

async function deleteAlias(id: number) {
  if (!confirm('Alias silinecek!')) return
  try { await api.delete(`/api/v1/domains/${domainId.value}/aliases/${id}`); await loadAliases() } catch { }
}

async function loadEmails() {
  try {
    const res = await api.get('/api/v1/emails', { params: { domain: domain.value?.domain || '' } })
    emails.value = res.data.emails || []
  } catch { emails.value = [] }
}

function genPass() { return Array(16).fill(0).map(() => 'abcdefghijklmnopqrstuvwxyz0123456789'[Math.floor(Math.random()*36)]).join('') }

async function createDB() {
  try {
    await api.post('/api/v1/databases', newDB.value)
    showCreateDB.value = false
    newDB.value = { name: '', username: '', password: '', charset: 'utf8mb4' }
    await loadDatabases()
  } catch (err: any) { error.value = err.response?.data?.error || t('domainDetail.databaseCreateFailed') }
}

async function deleteDB(id: number) {
  if (!confirm(t('domainDetail.confirmDeleteDB'))) return
  try { await api.delete(`/api/v1/databases/${id}`); await loadDatabases() } catch { }
}

async function changePHP(version: string) {
  changingPHP.value = true
  try {
    await api.put(`/api/v1/domains/${domainId.value}`, { php_version: version })
    domain.value!.php_version = version
    await loadPHPExtensions(version)
  } catch { }
  finally { changingPHP.value = false }
}

async function secureSite() {
  try {
    await api.post(`/api/v1/domains/${domainId.value}/secure`, {})
    sslResult.value = '.htaccess güvenlik dosyası yenilendi ✓'
  } catch (err: any) {
    sslResult.value = err.response?.data?.error || 'Hata oluştu'
  }
}

async function installSSL() {
  installingSSL.value = true
  sslResult.value = ''
  try {
    const res = await api.post(`/api/v1/domains/${domainId.value}/ssl`, { type: 'lets_encrypt' })
    sslResult.value = res.data.message
  } catch (err: any) {
    sslResult.value = err.response?.data?.error || t('domainDetail.sslFailed')
  }
  finally { installingSSL.value = false }
}

// CMS Installer
const showCMS = ref(false)
const installingCMS = ref(false)
const cmsType = ref('wordpress')
const cmsResult = ref<any>(null)
const cmsError = ref('')

async function installCMS() {
  installingCMS.value = true
  cmsError.value = ''
  cmsResult.value = null
  try {
    const res = await api.post(`/api/v1/domains/${domainId.value}/install-cms`, {
      cms: cmsType.value,
      site_title: domain.value?.domain || '',
      admin_email: 'admin@' + (domain.value?.domain || 'localhost'),
    })
    cmsResult.value = res.data
  } catch (err: any) {
    cmsError.value = err.response?.data?.error || 'CMS kurulumu başarısız oldu'
  }
  finally { installingCMS.value = false }
}

async function createEmail() {
  try {
    const emailFull = newEmail.value.email.includes('@')
      ? newEmail.value.email
      : newEmail.value.email + '@' + domain.value!.domain
    await api.post('/api/v1/emails', {
      email: emailFull,
      password: newEmail.value.password,
      quota: newEmail.value.quota,
      domain: domain.value!.domain,
    })
    showCreateEmail.value = false
    newEmail.value = { email: '', password: '', quota: 1024 }
    await loadEmails()
  } catch (err: any) { error.value = err.response?.data?.error || t('domainDetail.emailCreateFailed') }
}

async function deleteEmail(id: number) {
  if (!confirm('Email silinecek!')) return
  try { await api.delete(`/api/v1/emails/${id}`); await loadEmails() } catch { }
}

onMounted(() => loadDomain())
watch(() => route.params.id, () => { domainId.value = Number(route.params.id); loadDomain() })
</script>

<template>
  <div class="domain-detail">
    <div v-if="loading" class="aura-card loading-card">
      <div class="kicker">{{ t('auto.7347ec') }}</div>
      <p class="muted">Domain bilgileri getiriliyor…</p>
    </div>

    <template v-else-if="domain">
      <!-- Aura Header -->
      <div class="detail-head">
        <div class="head-left">
          <button class="aura-btn aura-btn-ghost back-btn" @click="router.push('/domains')">← Domainler</button>
          <div class="head-main">
            <div class="head-title">
              <span class="brand-icon">
                <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="6"/><path d="M2 8h12M8 2a10 10 0 0 1 0 12M8 2a10 10 0 0 0 0 12"/></svg>
              </span>
              <h1>{{ domain.domain }}</h1>
              <span :class="'pill ' + (domain.status === 'active' ? 'pill-on' : 'pill-off')">
                <span class="pill-dot"></span>{{ domain.status === 'active' ? t('common.active') : t('common.inactive') }}
              </span>
            </div>
            <div class="head-meta">
              <span class="meta-item"><span class="kicker">{{ t('auto.061160') }}</span><span class="meta-val mono">{{ domain.document_root }}</span></span>
              <span class="meta-item"><span class="kicker">PHP</span><span class="meta-val">{{ domain.php_version }}</span></span>
              <span class="meta-item"><span class="kicker">SSL</span><span :class="'meta-val ' + (domain.ssl_enabled ? 'ok' : 'muted')">{{ domain.ssl_enabled ? t('common.active') : 'Yok' }}</span></span>
              <span class="meta-item"><span class="kicker">{{ t('common.status') }}</span><span class="meta-val">{{ domain.force_https ? 'HTTPS Zorunlu' : 'HTTP' }}</span></span>
            </div>
          </div>
        </div>
        <div class="head-actions">
          <a :href="'http://' + domain.domain" target="_blank" class="aura-btn aura-btn-ghost">{{ t('auto.7967e8') }}</a>
          <button class="aura-btn aura-btn-primary" @click="installSSL" :disabled="installingSSL">
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4" style="width:14px;height:14px;"><rect x="3.5" y="6.5" width="9" height="7" rx="1.2"/><path d="M6 6.5V4.8a2 2 0 0 1 4 0v1.7"/><circle cx="8" cy="10" r="1"/></svg>
            {{ installingSSL ? 'Kuruluyor…' : 'SSL Kur' }}
          </button>
          <button class="aura-btn aura-btn-ghost" @click="showCMS = true" :disabled="installingCMS">
            📦 {{ installingCMS ? 'Kuruluyor…' : 'CMS Kur' }}
          </button>
        </div>
      </div>

      <div v-if="sslResult" class="aura-card ssl-result">
        <span class="kicker">Bildirim</span>
        <p>{{ sslResult }}</p>
      </div>
      <div v-if="error" class="aura-card error-card">
        <span class="kicker" style="color:var(--aura-danger)">Hata</span>
        <p>{{ error }}</p>
      </div>

      <!-- Aura Tabs -->
      <div class="tabs-wrap aura-card">
        <button
          v-for="tab in [
            {k:'overview', l:'Genel', i:'grid'},
            {k:'subdomain', l:'Alt Domain', i:'globe'},
            {k:'alias', l:'Alias', i:'link'},
            {k:'database', l:t('backup.typeDatabase'), i:'database'},
            {k:'email', l:'E-Posta', i:'mail'},
            {k:'dns', l:'DNS', i:'dns'},
            {k:'files', l:'Dosyalar', i:'folder'},
            {k:'logs', l:'Loglar', i:'file'},
          ]" :key="tab.k"
          :class="'tab ' + (activeTab === tab.k ? 'active' : '')"
          @click="activeTab = tab.k"
        >
          <span class="tab-icon">
            <svg v-if="tab.i==='grid'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="1" y="1" width="6" height="6" rx="1"/><rect x="9" y="1" width="6" height="6" rx="1"/><rect x="1" y="9" width="6" height="6" rx="1"/><rect x="9" y="9" width="6" height="6" rx="1"/></svg>
            <svg v-else-if="tab.i==='globe'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="6"/><path d="M2 8h12M8 2a10 10 0 0 1 0 12M8 2a10 10 0 0 0 0 12"/></svg>
            <svg v-else-if="tab.i==='link'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M9 3h4v4"/><path d="M7 9l6-6"/><path d="M5 3H3v10h10v-2"/></svg>
            <svg v-else-if="tab.i==='database'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><ellipse cx="8" cy="4" rx="5" ry="2.5"/><path d="M3 4v8c0 1.4 2.2 2.5 5 2.5s5-1.1 5-2.5V4"/><path d="M3 8c0 1.4 2.2 2.5 5 2.5s5-1.1 5-2.5"/></svg>
            <svg v-else-if="tab.i==='mail'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="2" y="3" width="12" height="10" rx="1.5"/><path d="M2.5 4.5 8 8.5 13.5 4.5"/></svg>
            <svg v-else-if="tab.i==='dns'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><circle cx="8" cy="8" r="2.5"/><path d="M8 1v1.5M8 13.5V15M1 8h1.5M13.5 8H15M3.2 3.2l1 1M11.8 11.8l1 1M12.8 3.2l-1 1M4.2 11.8l-1 1"/></svg>
            <svg v-else-if="tab.i==='folder'" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M2 5.5A1.5 1.5 0 0 1 3.5 4H6l1.2 1.5H12.5A1.5 1.5 0 0 1 14 7v5.5A1.5 1.5 0 0 1 12.5 14H3.5A1.5 1.5 0 0 1 2 12.5V5.5Z"/></svg>
            <svg v-else viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M5 2.5A1.5 1.5 0 0 1 6.5 1H9l3 3v8.5A1.5 1.5 0 0 1 10.5 14H6.5A1.5 1.5 0 0 1 5 12.5V2.5Z"/><path d="M9 1v3.5H12"/></svg>
          </span>
          {{ tab.l }}
        </button>
      </div>

      <!-- OVERVIEW -->
      <div v-if="activeTab === 'overview'" class="tab-panel">
        <div class="quick-grid">
          <a :href="'http://localhost/adminer/?server=localhost&db=' + (databases[0]?.name || '')" target="_blank" class="aura-card quick-card">
            <span class="qc-icon"><svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><ellipse cx="8" cy="4" rx="5" ry="2.5"/><path d="M3 4v8c0 1.4 2.2 2.5 5 2.5s5-1.1 5-2.5V4"/><path d="M3 8c0 1.4 2.2 2.5 5 2.5s5-1.1 5-2.5"/></svg></span>
            <span class="qc-title">Adminer</span>
            <span class="qc-desc">{{ t('auto.eb33e4') }}</span>
            <span class="qc-arrow">→</span>
          </a>
          <button class="aura-card quick-card" @click="activeTab = 'files'">
            <span class="qc-icon"><svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><path d="M2 5.5A1.5 1.5 0 0 1 3.5 4H6l1.2 1.5H12.5A1.5 1.5 0 0 1 14 7v5.5A1.5 1.5 0 0 1 12.5 14H3.5A1.5 1.5 0 0 1 2 12.5V5.5Z"/></svg></span>
            <span class="qc-title">Dosyalar</span>
            <span class="qc-desc">{{ t('auto.c49df5') }}</span>
            <span class="qc-arrow">→</span>
          </button>
          <button class="aura-card quick-card" @click="activeTab = 'email'">
            <span class="qc-icon"><svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><rect x="2" y="3" width="12" height="10" rx="1.5"/><path d="M2.5 4.5 8 8.5 13.5 4.5"/></svg></span>
            <span class="qc-title">E-Posta</span>
            <span class="qc-desc">{{ t('auto.ed4822') }}</span>
            <span class="qc-arrow">→</span>
          </button>
          <button class="aura-card quick-card" @click="activeTab = 'database'">
            <span class="qc-icon"><svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.4"><ellipse cx="8" cy="4" rx="5" ry="2.5"/><path d="M3 4v8c0 1.4 2.2 2.5 5 2.5s5-1.1 5-2.5V4"/></svg></span>
            <span class="qc-title">{{ t('backup.typeDatabase') }}</span>
            <span class="qc-desc">Yeni DB ekle</span>
            <span class="qc-arrow">→</span>
          </button>
        </div>

        <!-- Disk Kullanım Barı -->
        <div v-if="(domain.home_usage_mb ?? 0) >= 0 && (domain.quota_limit_mb ?? 0) > 0" class="aura-card usage-card">
          <div class="usage-head">
            <span class="kicker">Disk Kullanımı</span>
            <span class="usage-text">{{ ((domain.home_usage_mb ?? 0) / 1024).toFixed(1) }} GB / {{ ((domain.quota_limit_mb ?? 0) / 1024).toFixed(0) }} GB</span>
          </div>
          <div class="usage-bar-track">
            <div class="usage-bar-fill" :style="{ width: Math.min(100, ((domain.home_usage_mb ?? 0) / (domain.quota_limit_mb ?? 1)) * 100) + '%' }" :class="((domain.home_usage_mb ?? 0) / (domain.quota_limit_mb ?? 1)) > 0.9 ? 'danger' : ((domain.home_usage_mb ?? 0) / (domain.quota_limit_mb ?? 1)) > 0.7 ? 'warn' : ''"></div>
          </div>
        </div>

        <div class="detail-grid">
          <div class="aura-card section">
            <div class="section-head">
              <span class="kicker">SFTP Erişimi</span>
              <h3>Dosya Transferi</h3>
            </div>
            <div class="info-grid">
              <div class="info-item"><span class="kicker">Host</span><strong class="mono">{{ domain.domain }}</strong></div>
              <div class="info-item"><span class="kicker">Port</span><strong class="mono">22</strong></div>
              <div class="info-item"><span class="kicker">Protokol</span><strong>SFTP (SSH)</strong></div>
              <div class="info-item"><span class="kicker">Kullanıcı</span><strong class="mono">Kullanıcı adınız</strong></div>
              <div class="info-item" style="grid-column:1/-1">
                <span class="kicker">Komut</span>
                <code class="mono" style="font-size:11px;padding:4px 8px;background:var(--aura-bg-subtle);border-radius:6px">sftp kullanici@{{ domain.domain }}</code>
              </div>
            </div>
          </div>

          <div class="aura-card section">
            <div class="section-head">
              <span class="kicker">{{ t('dashboard.uptimeLabel') }}</span>
              <h3>{{ t('auto.196ab3') }}</h3>
            </div>
            <div class="php-select">
              <select :value="domain.php_version" @change="changePHP(($event.target as HTMLSelectElement).value)" :disabled="changingPHP" class="aura-input">
                <option value="8.4">{{ t('auto.50baf0') }}</option>
                <option value="8.3">{{ t('auto.8b9499') }}</option>
                <option value="8.2">PHP 8.2</option>
              </select>
              <span class="hint muted">{{ changingPHP ? t('common.changing') : t('domainDetail.vhostRewrite') }}</span>
            </div>
            <div v-if="phpExtensions.length > 0" class="ext-list">
              <span class="kicker" style="margin-bottom:6px;display:block">PHP Eklentileri ({{ phpExtensions.length }})</span>
              <div class="ext-tags">
                <span v-for="ext in phpExtensions.slice(0, 30)" :key="ext.name" class="ext-tag">{{ ext.name }}</span>
                <span v-if="phpExtensions.length > 30" class="ext-tag more">+{{ phpExtensions.length - 30 }} daha</span>
              </div>
            </div>
          </div>

          <div class="aura-card section">
            <div class="section-head">
              <span class="kicker">Bilgi</span>
              <h3>Domain Bilgisi</h3>
            </div>
            <div class="info-grid">
              <div class="info-item"><span class="kicker">Domain</span><strong class="mono">{{ domain.domain }}</strong></div>
              <div class="info-item"><span class="kicker">Document Root</span><strong class="mono">{{ domain.document_root }}</strong></div>
              <div class="info-item"><span class="kicker">PHP</span><strong>{{ domain.php_version }}</strong></div>
              <div class="info-item"><span class="kicker">HTTPS</span><strong :class="domain.force_https ? 'ok' : 'muted'">{{ domain.force_https ? t('common.requiredValue') : t('common.optional') }}</strong></div>
              <div class="info-item"><span class="kicker">SSL</span><strong :class="domain.ssl_enabled ? 'ok' : 'muted'">{{ domain.ssl_enabled ? t('common.active') : 'Yok' }}</strong></div>
              <div class="info-item"><span class="kicker">{{ t('auto.69ff4b') }}</span><strong>{{ domain.created_at }}</strong></div>
            </div>
          </div>
        </div>
      </div>

      <!-- SUBDOMAIN -->
      <div v-if="activeTab === 'subdomain'" class="aura-card section">
        <div class="section-head">
          <div>
            <span class="kicker">{{ t('auto.e9b5fc') }}</span>
            <h3>Alt Domainler</h3>
            <p class="muted">blog.{{ domain.domain }}, shop.{{ domain.domain }} gibi</p>
          </div>
          <button class="aura-btn aura-btn-primary" @click="openSubdomainModal">+ Alt Domain</button>
        </div>
        <div v-if="subdomains.length === 0" class="empty">
          <span class="kicker">{{ t('auto.485510') }}</span>
          <p class="muted">{{ t('auto.b55b3c') }}</p>
        </div>
        <div v-else class="data-list">
          <div v-for="sub in subdomains" :key="sub.id" class="data-row">
            <div class="row-main">
              <strong>{{ sub.domain }}</strong>
              <span class="mono muted">{{ sub.document_root }}</span>
            </div>
            <div class="row-actions">
              <span :class="'pill ' + (sub.ssl_enabled ? 'pill-on' : 'pill-off')">{{ sub.ssl_enabled ? 'SSL' : '—' }}</span>
              <span class="kicker">PHP {{ sub.php_version }}</span>
              <a :href="'http://' + sub.domain" target="_blank" class="aura-btn aura-btn-ghost sm">{{ t('auto.476ca0') }}</a>
            </div>
          </div>
        </div>
      </div>

      <!-- ALIAS (Parked Domain) -->
      <div v-if="activeTab === 'alias'" class="aura-card section">
        <div class="section-head">
          <div>
            <span class="kicker">Alias</span>
            <h3>Park Edilmiş Domainler</h3>
            <p class="muted">Aynı siteyi gösteren ek domainler</p>
          </div>
          <button class="aura-btn aura-btn-primary" @click="showCreateAlias = true; newAlias = { alias: '', type: 'park', target: '' }">+ Alias Ekle</button>
        </div>
        <div v-if="aliases.length === 0 && !showCreateAlias" class="empty">
          <span class="kicker">Alias yok</span>
          <p class="muted">Örn: site.net → site.com ile aynı içeriği gösterir</p>
        </div>
        <div v-else class="data-list">
          <div v-for="a in aliases" :key="a.id" class="data-row">
            <div class="row-main">
              <strong>{{ a.alias }}</strong>
              <span class="mono muted">{{ a.type === 'park' ? 'Parked' : 'Redirect → ' + a.target }}</span>
            </div>
            <div class="row-actions">
              <a :href="'http://' + a.alias" target="_blank" class="aura-btn aura-btn-ghost sm">Aç →</a>
              <button class="aura-btn aura-btn-ghost sm danger" @click="deleteAlias(a.id)">Sil</button>
            </div>
          </div>
        </div>
      </div>

      <!-- DATABASE -->
      <div v-if="activeTab === 'database'" class="aura-card section">
        <div class="section-head">
          <div><span class="kicker">Veri</span><h3>{{ t('auto.217018') }}</h3></div>
          <button class="aura-btn aura-btn-primary" @click="showCreateDB = true; newDB = { name: '', username: '', password: '', charset: 'utf8mb4' }">{{ t('auto.a4aa65') }}</button>
        </div>
        <div v-if="databases.length === 0" class="empty"><span class="kicker">{{ t('auto.485510') }}</span><p class="muted">{{ t('auto.e372c0') }}</p></div>
        <div v-else class="data-list">
          <div v-for="db in databases" :key="db.id" class="data-row">
            <div class="row-main">
              <strong>{{ db.name }}</strong>
              <span class="mono muted">{{ db.username }} · {{ db.charset }}</span>
            </div>
            <div class="row-actions">
              <a :href="'http://localhost/adminer/?server=localhost&username=' + db.username + '&db=' + db.name" target="_blank" class="aura-btn aura-btn-ghost sm">Adminer →</a>
              <button class="aura-btn aura-btn-ghost sm danger" @click="deleteDB(db.id)">{{ t('common.delete') }}</button>
            </div>
          </div>
        </div>
      </div>

      <!-- EMAIL -->
      <div v-if="activeTab === 'email'" class="aura-card section">
        <div class="section-head">
          <div><span class="kicker">Posta</span><h3>{{ t('auto.532a29') }}</h3></div>
          <button class="aura-btn aura-btn-primary" @click="showCreateEmail = true; newEmail = { email: '', password: '', quota: 1024 }">+ Email</button>
        </div>
        <div v-if="emails.length === 0" class="empty"><span class="kicker">{{ t('auto.485510') }}</span><p class="muted">{{ t('auto.3c5dce') }}</p></div>
        <div v-else class="data-list">
          <div v-for="e in emails" :key="e.id" class="data-row">
            <div class="row-main">
              <strong>{{ e.email }}</strong>
              <span class="muted">Quota {{ e.quota }}MB</span>
            </div>
            <div class="row-actions">
              <a :href="'https://' + domain.domain + '/webmail'" target="_blank" class="aura-btn aura-btn-ghost sm">Webmail →</a>
              <button class="aura-btn aura-btn-ghost sm danger" @click="deleteEmail(e.id)">{{ t('common.delete') }}</button>
            </div>
          </div>
        </div>
      </div>

      <!-- DNS -->
      <div v-if="activeTab === 'dns'" class="aura-card section">
        <div class="section-head"><div><span class="kicker">{{ t('auto.e9b5fc') }}</span><h3>{{ t('auto.3cd482') }}</h3></div></div>
        <div class="dns-table">
          <div class="dns-head"><span class="kicker">{{ t('common.type') }}</span><span class="kicker">{{ t('common.name') }}</span><span class="kicker">{{ t('common.value') }}</span><span class="kicker">{{ t('common.ttl') }}</span></div>
          <div class="dns-row"><span class="mono">A</span><span class="mono">@</span><span class="mono">SUNUCU_IP</span><span class="muted">3600</span></div>
          <div class="dns-row"><span class="mono">CNAME</span><span class="mono">www</span><span class="mono">{{ domain.domain }}</span><span class="muted">3600</span></div>
          <div class="dns-row"><span class="mono">MX</span><span class="mono">@</span><span class="mono">mail.{{ domain.domain }}</span><span class="muted">3600</span></div>
        </div>
        <p class="hint muted">{{ t('auto.6b8b57') }}</p>
      </div>

      <!-- FILES -->
      <div v-if="activeTab === 'files'" class="aura-card section">
        <div class="section-head"><div><span class="kicker">Dosyalar</span><h3>{{ t('files.title') }}</h3></div></div>
        <div class="file-cta">
          <span class="mono muted">{{ domain.document_root }}</span>
          <button class="aura-btn aura-btn-primary" @click="router.push('/files?path=' + domain.document_root)">{{ t('auto.af55e8') }}</button>
        </div>
      </div>

      <!-- LOGS -->
      <div v-if="activeTab === 'logs'" class="aura-card section">
        <div class="section-head"><div><span class="kicker">{{ t('auto.b624e9') }}</span><h3>Loglar</h3></div></div>
        <div class="log-grid">
          <div class="aura-card log-item">
            <span class="kicker">Access</span>
            <code class="mono">$VH_ROOT/logs/access.log</code>
          </div>
          <div class="aura-card log-item">
            <span class="kicker">Error</span>
            <code class="mono">/usr/local/lsws/logs/error.log</code>
          </div>
        </div>
        <p class="hint muted">{{ t('auto.28c845') }}</p>
      </div>
    </template>
  </div>

  <!-- CMS Kurulum Modal -->
  <div v-if="showCMS" class="overlay" @click.self="showCMS = false">
    <div class="aura-card modal modal-sm">
      <div class="modal-head">
        <div><span class="kicker">CMS Kurulumu</span><h3 class="modal-title">CMS Yükle</h3></div>
        <button class="icon-btn" @click="showCMS = false">×</button>
      </div>
      <div class="modal-body">
        <div v-if="!cmsResult" class="cms-form">
          <p class="hint">Domain için bir CMS seçin. Dosyalar document root dizinine yüklenecek, veritabanı otomatik oluşturulacak.</p>
          <div class="cms-options">
            <label class="cms-opt" :class="{ active: cmsType === 'wordpress' }">
              <input type="radio" v-model="cmsType" value="wordpress" hidden />
              <span class="cms-icon">📝</span>
              <strong>WordPress</strong>
              <small>Dünyanın en popüler CMS'i</small>
            </label>
            <label class="cms-opt" :class="{ active: cmsType === 'joomla' }">
              <input type="radio" v-model="cmsType" value="joomla" hidden />
              <span class="cms-icon">🔷</span>
              <strong>Joomla 5</strong>
              <small>Güçlü ve esnek CMS</small>
            </label>
          </div>
          <div v-if="cmsError" class="alert" style="margin-top:12px">{{ cmsError }}</div>
        </div>
        <div v-else class="cms-success">
          <div class="success-icon">✅</div>
          <h4>{{ cmsResult.message }}</h4>
          <div class="cms-info-grid">
            <div><span>CMS</span><strong>{{ cmsResult.cms?.name }}</strong></div>
            <div><span>Kullanıcı</span><code>{{ cmsResult.cms?.admin_user }}</code></div>
            <div><span>Şifre</span><code>{{ cmsResult.cms?.admin_pass }}</code></div>
            <div><span>Veritabanı</span><code>{{ cmsResult.cms?.database }}</code></div>
            <div><span>DB Kullanıcı</span><code>{{ cmsResult.cms?.db_user }}</code></div>
            <div><span>DB Şifre</span><code>{{ cmsResult.cms?.db_pass }}</code></div>
          </div>
          <a :href="cmsResult.next_step?.admin_url" target="_blank" class="aura-btn aura-btn-primary" style="margin-top:12px;width:100%;justify-content:center;display:flex">Admin Paneline Git →</a>
          <p class="hint muted" style="margin-top:8px">{{ cmsResult.next_step?.note }}</p>
        </div>
      </div>
      <div class="modal-foot" v-if="!cmsResult">
        <button class="aura-btn aura-btn-ghost" @click="showCMS = false">İptal</button>
        <button class="aura-btn aura-btn-primary" @click="installCMS" :disabled="installingCMS">
          {{ installingCMS ? 'Kuruluyor…' : 'Kurulumu Başlat' }}
        </button>
      </div>
    </div>
  </div>

  <!-- Subdomain Ekleme Modal -->
  <div v-if="showSubModal" class="overlay" @click.self="!subCreating && (showSubModal = false)">
    <div class="aura-card modal modal-sm">
      <div class="modal-head">
        <div><span class="kicker">Alt Domain Ekle</span><h3 class="modal-title">{{ newSub.subdomain || 'blog' }}.{{ domain?.domain }}</h3></div>
        <button class="icon-btn" @click="showSubModal = false" :disabled="subCreating">×</button>
      </div>
      <div class="modal-body">
        <!-- Giriş formu -->
        <div v-if="!subCreating && !subResult" class="sub-form">
          <div class="sub-domain-input">
            <input v-model="newSub.subdomain" placeholder="blog" class="aura-input" autofocus @keyup.enter="createSubdomain" />
            <span class="domain-suffix">.{{ domain?.domain }}</span>
          </div>
          <select v-model="newSub.php_version" class="aura-input" style="margin-top:12px">
            <option v-for="v in ['8.4','8.3','8.2']" :key="v" :value="v">PHP {{ v }}</option>
          </select>
        </div>

        <!-- Progress bar -->
        <div v-if="subCreating" class="progress-container">
          <div class="progress-steps">
            <div v-for="(s, i) in subSteps" :key="i" class="progress-step" :class="s.status">
              <div class="step-indicator">
                <span v-if="s.status === 'done'" class="step-icon">✓</span>
                <span v-else-if="s.status === 'failed'" class="step-icon">✗</span>
                <span v-else-if="s.status === 'pending'" class="step-spinner"></span>
                <span v-else class="step-icon">·</span>
              </div>
              <div class="step-info">
                <strong>{{ s.name }}</strong>
                <small v-if="s.detail" class="step-detail">{{ s.detail }}</small>
              </div>
            </div>
          </div>
        </div>

        <!-- Başarılı sonuç -->
        <div v-if="subResult && !subCreating" class="sub-success">
          <div class="success-icon">✅</div>
          <h4>{{ subResult.message }}</h4>
          <div class="sub-result-grid">
            <div><span>Domain</span><strong>{{ subResult.domain?.domain }}</strong></div>
            <div><span>PHP</span><code>{{ subResult.domain?.php_version }}</code></div>
            <div><span>Dizin</span><code class="mono">{{ subResult.domain?.document_root }}</code></div>
          </div>
          <a :href="subResult.url" target="_blank" class="aura-btn aura-btn-primary" style="margin-top:12px;width:100%;justify-content:center;display:flex">Siteyi Aç →</a>
          <button class="aura-btn aura-btn-ghost" style="margin-top:8px;width:100%;justify-content:center;display:flex" @click="showSubModal = false; subResult = null">Kapat</button>
        </div>
      </div>
      <div class="modal-foot" v-if="!subCreating && !subResult">
        <button class="aura-btn aura-btn-ghost" @click="showSubModal = false">İptal</button>
        <button class="aura-btn aura-btn-primary" @click="createSubdomain" :disabled="!newSub.subdomain">Oluştur</button>
      </div>
    </div>
  </div>

  <!-- Email Ekleme Modal -->
  <div v-if="showCreateEmail" class="overlay" @click.self="showCreateEmail = false">
    <div class="aura-card modal modal-sm">
      <div class="modal-head">
        <div><span class="kicker">E-Posta</span><h3 class="modal-title">Email Hesabı Ekle</h3></div>
        <button class="icon-btn" @click="showCreateEmail = false">×</button>
      </div>
      <div class="modal-body">
        <div class="email-domain-hint">{{ newEmail.email || 'kullanici' }}@{{ domain?.domain }}</div>
        <input v-model="newEmail.email" placeholder="kullanici" class="aura-input" autofocus @keyup.enter="createEmail" />
        <input v-model="newEmail.password" placeholder="Şifre" type="text" class="aura-input" style="margin-top:10px" />
        <select v-model="newEmail.quota" class="aura-input" style="margin-top:10px">
          <option :value="512">512 MB</option>
          <option :value="1024">1 GB</option>
          <option :value="2048">2 GB</option>
          <option :value="5120">5 GB</option>
        </select>
      </div>
      <div class="modal-foot">
        <button class="aura-btn aura-btn-ghost" @click="showCreateEmail = false">İptal</button>
        <button class="aura-btn aura-btn-primary" @click="createEmail">Oluştur</button>
      </div>
    </div>
  </div>

  <!-- Alias Ekleme Modal -->
  <div v-if="showCreateAlias" class="overlay" @click.self="showCreateAlias = false">
    <div class="aura-card modal modal-sm">
      <div class="modal-head">
        <div><span class="kicker">Alias</span><h3 class="modal-title">Park Edilmiş Domain Ekle</h3></div>
        <button class="icon-btn" @click="showCreateAlias = false">×</button>
      </div>
      <div class="modal-body">
        <input v-model="newAlias.alias" placeholder="site.net" class="aura-input" autofocus @keyup.enter="createAlias" />
        <select v-model="newAlias.type" class="aura-input" style="margin-top:10px">
          <option value="park">Parked (Aynı siteyi göster)</option>
          <option value="redirect">Redirect (Başka URL'ye yönlendir)</option>
        </select>
        <input v-if="newAlias.type === 'redirect'" v-model="newAlias.target" placeholder="https://hedef-site.com" class="aura-input" style="margin-top:10px" />
      </div>
      <div class="modal-foot">
        <button class="aura-btn aura-btn-ghost" @click="showCreateAlias = false">İptal</button>
        <button class="aura-btn aura-btn-primary" @click="createAlias">Ekle</button>
      </div>
    </div>
  </div>

  <!-- Veritabani Ekleme Modal -->
  <div v-if="showCreateDB" class="overlay" @click.self="showCreateDB = false">
    <div class="aura-card modal modal-sm">
      <div class="modal-head">
        <div><span class="kicker">Veritabanı</span><h3 class="modal-title">Veritabanı Ekle</h3></div>
        <button class="icon-btn" @click="showCreateDB = false">×</button>
      </div>
      <div class="modal-body">
        <input v-model="newDB.name" placeholder="Veritabanı adı" class="aura-input" autofocus @keyup.enter="createDB" />
        <input v-model="newDB.username" placeholder="Kullanıcı adı" class="aura-input" style="margin-top:10px" />
        <div style="display:flex;gap:8px;margin-top:10px">
          <input v-model="newDB.password" placeholder="Şifre" type="text" class="aura-input" />
          <button class="aura-btn aura-btn-ghost" type="button" @click="newDB.password = genPass()" style="white-space:nowrap">🎲 Üret</button>
        </div>
      </div>
      <div class="modal-foot">
        <button class="aura-btn aura-btn-ghost" @click="showCreateDB = false">İptal</button>
        <button class="aura-btn aura-btn-primary" @click="createDB">Oluştur</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.domain-detail { display: flex; flex-direction: column; gap: 14px; }

/* Header */
.detail-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; flex-wrap: wrap; padding: 16px; }
.head-left { display: flex; gap: 12px; min-width: 0; flex: 1; }
.back-btn { flex-shrink: 0; }
.head-main { min-width: 0; }
.head-title { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.brand-icon { width: 28px; height: 28px; border-radius: 8px; display: grid; place-items: center; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); color: var(--aura-accent); flex-shrink: 0; }
.brand-icon svg { width: 16px; height: 16px; }
.head-title h1 { font-size: 20px; font-weight: 720; letter-spacing: -0.02em; color: var(--aura-text); line-height: 1; }
.pill { display: inline-flex; align-items: center; gap: 6px; font-size: 11px; font-weight: 650; letter-spacing: 0.04em; padding: 5px 10px; border-radius: 999px; border: 1px solid var(--aura-border); }
.pill-on { background: var(--aura-accent-soft); border-color: color-mix(in srgb, var(--aura-accent) 18%, transparent); color: var(--aura-accent); }
.pill-off { background: var(--aura-bg-subtle); color: var(--aura-text-faint); }
.pill-dot { width: 6px; height: 6px; border-radius: 999px; background: currentColor; }
.head-meta { display: flex; gap: 16px; flex-wrap: wrap; margin-top: 8px; }
.meta-item { display: flex; gap: 6px; align-items: baseline; }
.meta-val { font-size: 12px; font-weight: 550; color: var(--aura-text); }
.meta-val.mono { font-family: ui-monospace, monospace; font-size: 11px; }
.meta-val.ok { color: var(--aura-success); }
.meta-val.muted { color: var(--aura-text-faint); }
.head-actions { display: flex; gap: 8px; flex-shrink: 0; }

.ssl-result, .error-card { padding: 12px 14px; display: flex; flex-direction: column; gap: 4px; }
.ssl-result p, .error-card p { font-size: 13px; color: var(--aura-text); }
.error-card { border-color: color-mix(in srgb, var(--aura-danger) 18%, var(--aura-border)); background: color-mix(in srgb, var(--aura-danger) 6%, var(--aura-surface)); }

.kicker { font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; color: var(--aura-text-faint); }
.muted { color: var(--aura-text-muted); font-size: 12px; }
.mono { font-family: ui-monospace, monospace; }

/* Tabs */
.tabs-wrap { display: flex; gap: 6px; padding: 6px; overflow-x: auto; }
.tab { display: inline-flex; align-items: center; gap: 7px; padding: 8px 14px; border-radius: 999px; border: 1px solid transparent; background: transparent; color: var(--aura-text-muted); font-size: 13px; font-weight: 500; cursor: pointer; white-space: nowrap; transition: all 0.15s; }
.tab:hover { background: var(--aura-bg-subtle); border-color: var(--aura-border); color: var(--aura-text); }
.tab.active { background: var(--aura-accent); color: var(--aura-accent-text); border-color: var(--aura-accent); box-shadow: 0 2px 8px var(--aura-accent-ring); }
.tab-icon svg { width: 14px; height: 14px; }
.tab.active .tab-icon { color: var(--aura-accent-text); }
.tab:not(.active) .tab-icon { color: var(--aura-text-faint); }

/* Loading */
.loading-card { padding: 40px; text-align: center; display: flex; flex-direction: column; gap: 8px; align-items: center; }

/* Tab panel */
.tab-panel { display: flex; flex-direction: column; gap: 14px; }

/* Quick grid — like Dashboard */
.quick-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
@media (max-width: 1100px) { .quick-grid { grid-template-columns: repeat(2, 1fr); } }
@media (max-width: 640px) { .quick-grid { grid-template-columns: 1fr; } }
.quick-card { text-align: left; padding: 14px 16px; display: flex; flex-direction: column; gap: 4px; cursor: pointer; position: relative; text-decoration: none; color: inherit; }
.quick-card:hover { border-color: color-mix(in srgb, var(--aura-accent) 22%, var(--aura-border)); }
.qc-icon { width: 28px; height: 28px; border-radius: 8px; display: grid; place-items: center; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); color: var(--aura-accent); }
.qc-icon svg { width: 16px; height: 16px; }
.qc-title { font-size: 13px; font-weight: 650; color: var(--aura-text); }
.qc-desc { font-size: 12px; color: var(--aura-text-muted); }
.qc-arrow { position: absolute; top: 14px; right: 12px; color: var(--aura-text-faint); font-size: 14px; transition: transform 0.15s; }
.quick-card:hover .qc-arrow { transform: translateX(2px); color: var(--aura-accent); }

/* Usage bar */
.usage-card { padding: 14px 18px; display: flex; flex-direction: column; gap: 10px; }
.usage-head { display: flex; justify-content: space-between; align-items: center; }
.usage-text { font-size: 13px; font-weight: 600; color: var(--aura-text); font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.usage-bar-track { height: 8px; background: var(--aura-bg-subtle); border-radius: 10px; overflow: hidden; border: 1px solid var(--aura-border); }
.usage-bar-fill { height: 100%; background: var(--aura-accent); border-radius: 10px; transition: width 0.5s ease; }
.usage-bar-fill.warn { background: #f59e0b; }
.usage-bar-fill.danger { background: var(--aura-danger); }

.detail-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
@media (max-width: 860px) { .detail-grid { grid-template-columns: 1fr; } }
.section { padding: 16px; display: flex; flex-direction: column; gap: 14px; }
.section-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
.section-head h3 { font-size: 14px; font-weight: 700; letter-spacing: -0.01em; color: var(--aura-text); margin-top: 2px; }
.section-head p { font-size: 12px; }
.php-select { display: flex; flex-direction: column; gap: 8px; max-width: 320px; }

.ext-list { margin-top: 4px; }
.ext-tags { display: flex; flex-wrap: wrap; gap: 4px; }
.ext-tag {
  padding: 2px 8px; border-radius: 6px; font-size: 11px; font-weight: 500;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  background: var(--aura-accent-soft); color: var(--aura-accent);
  border: 1px solid color-mix(in srgb, var(--aura-accent) 15%, transparent);
}
.ext-tag.more { background: var(--aura-bg-subtle); color: var(--aura-text-muted); border-color: var(--aura-border); }
.hint { font-size: 11px; }

.info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
@media (max-width: 640px) { .info-grid { grid-template-columns: 1fr; } }
.info-item { display: flex; flex-direction: column; gap: 4px; padding: 12px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); border-radius: 10px; }
.info-item strong { font-size: 13px; }
.info-item strong.ok { color: var(--aura-success); }
.info-item strong.muted { color: var(--aura-text-faint); }

.empty { text-align: center; padding: 32px; display: flex; flex-direction: column; gap: 6px; align-items: center; }
.data-list { display: flex; flex-direction: column; gap: 8px; }
.data-row { display: flex; justify-content: space-between; align-items: center; gap: 12px; padding: 12px 14px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); border-radius: 10px; }
.row-main { min-width: 0; }
.row-main strong { font-size: 13px; color: var(--aura-text); }
.row-main .mono { font-size: 11px; margin-top: 2px; display: block; }
.row-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.sm { padding: 6px 10px; font-size: 12px; }
.danger { color: var(--aura-danger); border-color: color-mix(in srgb, var(--aura-danger) 18%, var(--aura-border)); }
.danger:hover { background: color-mix(in srgb, var(--aura-danger) 8%, var(--aura-surface)); }

.inline-form { display: flex; gap: 8px; flex-wrap: wrap; padding: 12px; align-items: center; }
.inline-field { display: flex; align-items: center; gap: 6px; flex: 1; min-width: 160px; }
.inline-field input { flex: 1; }

.dns-table { border: 1px solid var(--aura-border); border-radius: 10px; overflow: hidden; }
.dns-head, .dns-row { display: grid; grid-template-columns: 80px 1fr 1fr 60px; gap: 8px; padding: 10px 12px; font-size: 13px; }
.dns-head { background: var(--aura-bg-subtle); border-bottom: 1px solid var(--aura-border); }
.dns-row { border-bottom: 1px solid color-mix(in srgb, var(--aura-border) 60%, transparent); }
.dns-row:last-child { border-bottom: none; }

.file-cta { display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: wrap; }
.log-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; }
@media (max-width: 640px) { .log-grid { grid-template-columns: 1fr; } }
.log-item { padding: 12px; display: flex; flex-direction: column; gap: 6px; background: var(--aura-bg-subtle); }
.log-item code { font-size: 11px; }

.aura-input { width: 100%; padding: 8px 12px; border: 1px solid var(--aura-border); border-radius: 10px; background: var(--aura-surface); color: var(--aura-text); font-size: 13px; transition: border-color 0.15s; }
.aura-input:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
.aura-btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 10px; font-size: 13px; font-weight: 600; cursor: pointer; border: 1px solid transparent; transition: all 0.15s; text-decoration: none; }
.aura-btn-primary { background: var(--aura-accent); color: var(--aura-accent-text); border-color: var(--aura-accent); }
.aura-btn-primary:hover { background: var(--aura-accent-hover); }
.aura-btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
.aura-btn-ghost { background: var(--aura-surface); color: var(--aura-text); border-color: var(--aura-border); }
.aura-btn-ghost:hover { background: var(--aura-bg-subtle); }

/* === Modal Base (scoped) === */
.overlay { position: fixed; inset: 0; background: rgba(15,23,42,0.45); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 16px; }
.modal { width: 100%; overflow: hidden; background: var(--aura-surface); border-radius: 14px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-sm { max-width: 460px; }
.modal-head { display: flex; justify-content: space-between; align-items: flex-start; padding: 16px 18px; border-bottom: 1px solid var(--aura-border); }
.modal-title { font-size: 15px; font-weight: 700; color: var(--aura-text); margin-top: 2px; }
.modal-body { padding: 18px; }
.modal-foot { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 18px; border-top: 1px solid var(--aura-border); }
.icon-btn { width: 32px; height: 32px; display: grid; place-items: center; border-radius: 8px; border: 1px solid var(--aura-border); background: var(--aura-surface); color: var(--aura-text-muted); cursor: pointer; font-size: 14px; }
.icon-btn:hover { background: var(--aura-surface-hover); color: var(--aura-text); }

/* === CMS Installer === */
.cms-form { display: flex; flex-direction: column; gap: 12px; }
.cms-options { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.cms-opt {
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  padding: 20px 14px; border: 2px solid var(--aura-border); border-radius: 14px;
  cursor: pointer; text-align: center; transition: all 0.2s; background: var(--aura-surface);
}
.cms-opt:hover { border-color: var(--aura-accent); background: var(--aura-accent-soft); transform: translateY(-1px); }
.cms-opt.active { border-color: var(--aura-accent); background: var(--aura-accent-soft); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
.cms-opt strong { font-size: 14px; font-weight: 600; color: var(--aura-text); }
.cms-opt small { font-size: 11px; color: var(--aura-text-muted); }
.cms-icon { font-size: 36px; line-height: 1; }

/* Success */
.cms-success { display: flex; flex-direction: column; align-items: center; gap: 14px; text-align: center; }
.cms-success h4 { font-size: 15px; font-weight: 600; color: var(--aura-text); max-width: 320px; }
.success-icon { font-size: 52px; line-height: 1; }

/* Info Grid */
.cms-info-grid {
  display: grid; grid-template-columns: 1fr 1fr; gap: 8px; width: 100%;
}
.cms-info-grid > div {
  display: flex; flex-direction: column; gap: 2px;
  padding: 10px 12px; background: var(--aura-bg-subtle); border-radius: 10px;
  border: 1px solid var(--aura-border); text-align: left;
}
.cms-info-grid span { font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.05em; color: var(--aura-text-faint); }
.cms-info-grid strong { font-size: 14px; color: var(--aura-text); }
.cms-info-grid code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px; color: var(--aura-accent); background: var(--aura-accent-soft);
  padding: 2px 6px; border-radius: 4px; word-break: break-all;
}

.alert { padding: 10px 14px; border-radius: 10px; background: #fef2f2; border: 1px solid #fecaca; color: #b91c1c; font-size: 13px; line-height: 1.4; }
[data-theme="dark"] .alert { background: #451a1a; border-color: #7f1d1d; color: #fecaca; }

.hint { font-size: 12px; color: var(--aura-text-muted); line-height: 1.5; }
.hint.muted { color: var(--aura-text-faint); }

/* === Subdomain Modal === */
.sub-form { display: flex; flex-direction: column; }
.sub-domain-input { display: flex; align-items: center; gap: 0; }
.sub-domain-input input { flex: 1; border-top-right-radius: 0; border-bottom-right-radius: 0; }
.domain-suffix {
  padding: 10px 14px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border);
  border-left: 0; border-radius: 0 10px 10px 0; font-size: 13px; color: var(--aura-text-muted);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace; white-space: nowrap;
}

/* Progress */
.progress-container { padding: 4px 0; }
.progress-steps { display: flex; flex-direction: column; gap: 0; }
.progress-step { display: flex; align-items: center; gap: 12px; padding: 10px 8px; border-radius: 8px; transition: all 0.2s; }
.progress-step.done { background: #f0fdf4; }
.progress-step.failed { background: #fef2f2; }
.progress-step.pending { opacity: 0.7; }
.progress-step.skipped { opacity: 0.5; }
[data-theme="dark"] .progress-step.done { background: #0d2818; }
[data-theme="dark"] .progress-step.failed { background: #451a1a; }

.step-indicator { width: 28px; height: 28px; border-radius: 50%; display: grid; place-items: center; flex-shrink: 0; }
.progress-step.done .step-indicator { background: #16a34a; color: white; }
.progress-step.failed .step-indicator { background: #dc2626; color: white; }
.progress-step.pending .step-indicator { background: var(--aura-accent-soft); border: 2px solid var(--aura-accent); }
.progress-step.skipped .step-indicator { background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); }

.step-icon { font-size: 13px; font-weight: 700; line-height: 1; }
.step-spinner {
  width: 14px; height: 14px; border: 2px solid var(--aura-accent); border-top-color: transparent;
  border-radius: 50%; animation: spin 0.6s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

.step-info { display: flex; flex-direction: column; gap: 1px; min-width: 0; }
.step-info strong { font-size: 13px; color: var(--aura-text); }
.step-detail { font-size: 11px; color: var(--aura-text-muted); word-break: break-all; }

/* Sub success */
.sub-success { display: flex; flex-direction: column; align-items: center; gap: 14px; text-align: center; }
.sub-success h4 { font-size: 15px; font-weight: 600; color: var(--aura-text); }
.sub-result-grid {
  display: flex; flex-direction: column; gap: 8px; width: 100%;
}
.sub-result-grid > div {
  display: flex; justify-content: space-between; align-items: center;
  padding: 10px 12px; background: var(--aura-bg-subtle); border-radius: 8px;
  border: 1px solid var(--aura-border);
}
.sub-result-grid span { font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.04em; color: var(--aura-text-faint); }
.sub-result-grid strong { font-size: 14px; color: var(--aura-text); }
.sub-result-grid code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px; color: var(--aura-accent); background: var(--aura-accent-soft);
  padding: 2px 6px; border-radius: 4px; word-break: break-all;
}

/* Email modal */
.email-domain-hint {
  padding: 8px 12px; background: var(--aura-bg-subtle); border-radius: 8px;
  font-size: 14px; font-weight: 500; color: var(--aura-text);
  text-align: center; margin-bottom: 10px;
  border: 1px solid var(--aura-border);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
}
</style>
