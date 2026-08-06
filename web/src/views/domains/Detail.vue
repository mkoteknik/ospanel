<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '@/api/client'

const route = useRoute()
const router = useRouter()
const domainId = ref(Number(route.params.id))

interface DomainInfo {
  id: number; domain: string; document_root: string; php_version: string
  ssl_enabled: boolean; force_https: boolean; status: string; created_at: string
}

const domain = ref<DomainInfo | null>(null)
const loading = ref(true)
const activeTab = ref('overview')
const error = ref('')

// Database
const databases = ref<any[]>([])
const showCreateDB = ref(false)
const newDB = ref({ name: '', username: '', password: '', charset: 'utf8mb4' })

// Email
const emails = ref<any[]>([])
const showCreateEmail = ref(false)
const newEmail = ref({ email: '', password: '', quota: 1024 })

// Actions
const changingPHP = ref(false)
const installingSSL = ref(false)
const sslResult = ref('')

async function loadDomain() {
  loading.value = true
  try {
    const res = await api.get(`/api/v1/domains/${domainId.value}`)
    domain.value = res.data.domain || res.data
    // Load related data
    await Promise.all([loadDatabases(), loadEmails()])
  } catch { error.value = 'Domain yüklenemedi'; router.push('/domains') }
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

async function loadEmails() {
  try {
    const res = await api.get('/api/v1/emails')
    emails.value = res.data.emails || []
  } catch { emails.value = [] }
}

async function createDB() {
  try {
    await api.post('/api/v1/databases', newDB.value)
    showCreateDB.value = false
    newDB.value = { name: '', username: '', password: '', charset: 'utf8mb4' }
    await loadDatabases()
  } catch (err: any) { error.value = err.response?.data?.error || 'Veritabanı oluşturulamadı' }
}

async function deleteDB(id: number) {
  if (!confirm('Veritabanı silinecek!')) return
  try { await api.delete(`/api/v1/databases/${id}`); await loadDatabases() } catch { }
}

async function changePHP(version: string) {
  changingPHP.value = true
  try {
    await api.put(`/api/v1/domains/${domainId.value}`, { php_version: version })
    domain.value!.php_version = version
  } catch { }
  finally { changingPHP.value = false }
}

async function installSSL() {
  installingSSL.value = true
  sslResult.value = ''
  try {
    const res = await api.post(`/api/v1/domains/${domainId.value}/ssl`, { type: 'lets_encrypt' })
    sslResult.value = res.data.message
  } catch (err: any) {
    sslResult.value = err.response?.data?.error || 'SSL kurulumu başarısız'
  }
  finally { installingSSL.value = false }
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
    })
    showCreateEmail.value = false
    newEmail.value = { email: '', password: '', quota: 1024 }
    await loadEmails()
  } catch (err: any) { error.value = err.response?.data?.error || 'Email oluşturulamadı' }
}

async function deleteEmail(id: number) {
  if (!confirm('Email silinecek!')) return
  try { await api.delete(`/api/v1/emails/${id}`); await loadEmails() } catch { }
}

onMounted(() => loadDomain())
watch(() => route.params.id, () => { domainId.value = Number(route.params.id); loadDomain() })
</script>

<template>
  <div class="domain-panel">
    <!-- Loading -->
    <div v-if="loading" class="loading">Domain yükleniyor...</div>

    <template v-else-if="domain">
      <!-- Header -->
      <div class="panel-header">
        <div class="header-left">
          <button class="back-btn" @click="router.push('/domains')">← Domainler</button>
          <div>
            <h1>🌐 {{ domain.domain }}</h1>
            <div class="header-meta">
              <span :class="'badge ' + (domain.status === 'active' ? 'badge-on' : 'badge-off')">
                {{ domain.status === 'active' ? '🟢 Aktif' : '🔴 Pasif' }}
              </span>
              <span>📂 {{ domain.document_root }}</span>
              <span>🐘 PHP {{ domain.php_version }}</span>
              <span>{{ domain.ssl_enabled ? '🔒 SSL Aktif' : '🔓 SSL Yok' }}</span>
            </div>
          </div>
        </div>
        <div class="header-actions">
          <a :href="'http://' + domain.domain" target="_blank" class="btn-outline">🌍 Siteyi Aç</a>
          <button class="btn-primary" @click="installSSL" :disabled="installingSSL">
            {{ installingSSL ? 'Kuruluyor...' : '🔒 SSL Kur' }}
          </button>
        </div>
      </div>

      <!-- SSL Sonuç -->
      <div v-if="sslResult" class="ssl-result">{{ sslResult }}</div>

      <!-- Tabs -->
      <div class="tabs">
        <button v-for="tab in ['overview','database','email','dns','files','logs']" :key="tab"
          :class="'tab ' + (activeTab === tab ? 'active' : '')"
          @click="activeTab = tab"
        >
          {{ { overview:'📊 Genel', database:'🗄️ Veritabanı', email:'📧 E-Posta', dns:'🔧 DNS', files:'📁 Dosyalar', logs:'📋 Loglar' }[tab] }}
        </button>
      </div>

      <!-- Tab: Genel -->
      <div v-if="activeTab === 'overview'" class="tab-content">
        <!-- Hızlı İşlemler -->
        <div class="quick-actions">
          <a :href="'http://localhost/adminer/?server=localhost&db=' + databases[0]?.name" target="_blank" class="action-card">
            <span class="ac-icon">🗄️</span>
            <span>Adminer</span>
          </a>
          <div class="action-card" @click="activeTab = 'files'">
            <span class="ac-icon">📁</span>
            <span>Dosyalar</span>
          </div>
          <div class="action-card" @click="activeTab = 'email'">
            <span class="ac-icon">📧</span>
            <span>E-Posta</span>
          </div>
          <div class="action-card" @click="activeTab = 'database'">
            <span class="ac-icon">🗄️</span>
            <span>Veritabanı</span>
          </div>
        </div>

        <!-- PHP Sürümü -->
        <div class="section">
          <h3>🐘 PHP Sürümü</h3>
          <div class="php-selector">
            <button v-for="v in ['7.4','8.0','8.1','8.2','8.3','8.4']" :key="v"
              :class="'php-btn ' + (domain.php_version === v ? 'active' : '')"
              :disabled="changingPHP"
              @click="changePHP(v)"
            >PHP {{ v }}</button>
          </div>
        </div>

        <!-- Bilgi -->
        <div class="section">
          <h3>📋 Domain Bilgisi</h3>
          <div class="info-grid">
            <div class="info-item"><span>Domain</span><strong>{{ domain.domain }}</strong></div>
            <div class="info-item"><span>Document Root</span><strong>{{ domain.document_root }}</strong></div>
            <div class="info-item"><span>PHP Sürümü</span><strong>{{ domain.php_version }}</strong></div>
            <div class="info-item"><span>HTTPS Zorunlu</span><strong>{{ domain.force_https ? '✅ Evet' : '❌ Hayır' }}</strong></div>
            <div class="info-item"><span>SSL</span><strong>{{ domain.ssl_enabled ? '🔒 Aktif' : '🔓 Yok' }}</strong></div>
            <div class="info-item"><span>Oluşturma</span><strong>{{ domain.created_at }}</strong></div>
          </div>
        </div>
      </div>

      <!-- Tab: Veritabanı -->
      <div v-if="activeTab === 'database'" class="tab-content">
        <div class="section-header">
          <h3>🗄️ Veritabanları</h3>
          <button class="btn-add-sm" @click="showCreateDB = true">+ Veritabanı Ekle</button>
        </div>
        <div v-if="databases.length === 0" class="empty">Henüz veritabanı yok.</div>
        <div v-else class="data-list">
          <div v-for="db in databases" :key="db.id" class="data-row">
            <div>
              <strong>{{ db.name }}</strong>
              <code>{{ db.username }}</code>
              <span class="muted">{{ db.charset }}</span>
            </div>
            <div class="row-actions">
              <a :href="'http://localhost/adminer/?server=localhost&username=' + db.username + '&db=' + db.name"
                target="_blank" class="btn-sm">🔗 Adminer</a>
              <button class="btn-sm btn-del" @click="deleteDB(db.id)">🗑️</button>
            </div>
          </div>
        </div>
        <!-- Create DB -->
        <div v-if="showCreateDB" class="inline-form">
          <input v-model="newDB.name" placeholder="DB adı" />
          <input v-model="newDB.username" placeholder="Kullanıcı" />
          <input v-model="newDB.password" placeholder="Şifre" type="text" />
          <button class="btn-add-sm" @click="createDB">Oluştur</button>
          <button class="btn-cancel-sm" @click="showCreateDB = false">İptal</button>
        </div>
      </div>

      <!-- Tab: Email -->
      <div v-if="activeTab === 'email'" class="tab-content">
        <div class="section-header">
          <h3>📧 E-Posta Hesapları</h3>
          <button class="btn-add-sm" @click="showCreateEmail = true">+ Email Ekle</button>
        </div>
        <div v-if="emails.length === 0" class="empty">Henüz email hesabı yok.</div>
        <div v-else class="data-list">
          <div v-for="e in emails" :key="e.id" class="data-row">
            <div>
              <strong>{{ e.email }}</strong>
              <span class="muted">Quota: {{ e.quota }}MB</span>
            </div>
            <div class="row-actions">
              <a :href="'https://' + domain.domain + '/webmail'" target="_blank" class="btn-sm">🌐 Webmail</a>
              <button class="btn-sm btn-del" @click="deleteEmail(e.id)">🗑️</button>
            </div>
          </div>
        </div>
        <div v-if="showCreateEmail" class="inline-form">
          <input v-model="newEmail.email" :placeholder="'kullanici@' + domain.domain" />
          <input v-model="newEmail.password" placeholder="Şifre" type="text" />
          <button class="btn-add-sm" @click="createEmail">Oluştur</button>
          <button class="btn-cancel-sm" @click="showCreateEmail = false">İptal</button>
        </div>
      </div>

      <!-- Tab: DNS -->
      <div v-if="activeTab === 'dns'" class="tab-content">
        <h3>🔧 DNS Kayıtları</h3>
        <div class="dns-records">
          <div class="dns-row header"><span>Tür</span><span>Ad</span><span>Değer</span><span>TTL</span></div>
          <div class="dns-row"><span>A</span><span>@</span><span>SUNUCU_IP</span><span>3600</span></div>
          <div class="dns-row"><span>CNAME</span><span>www</span><span>{{ domain.domain }}</span><span>3600</span></div>
          <div class="dns-row"><span>MX</span><span>@</span><span>mail.{{ domain.domain }}</span><span>3600</span></div>
        </div>
        <p class="muted">DNS kayıtları PowerDNS üzerinden yönetilir. Değişiklikler anında aktiftir.</p>
      </div>

      <!-- Tab: Dosyalar -->
      <div v-if="activeTab === 'files'" class="tab-content">
        <h3>📁 Dosya Yöneticisi</h3>
        <p>Document Root: <code>{{ domain.document_root }}</code></p>
        <button class="btn-primary" @click="router.push('/files?path=' + domain.document_root)">
          📁 Dosya Yöneticisinde Aç
        </button>
      </div>

      <!-- Tab: Loglar -->
      <div v-if="activeTab === 'logs'" class="tab-content">
        <h3>📋 Loglar</h3>
        <div class="log-section">
          <h4>Access Log</h4>
          <pre class="log-preview">$VH_ROOT/logs/access.log</pre>
        </div>
        <div class="log-section">
          <h4>Error Log</h4>
          <pre class="log-preview">/usr/local/lsws/logs/error.log</pre>
        </div>
        <p class="muted">Log dosyaları Linux sunucuda gerçek zamanlı görüntülenecektir.</p>
      </div>
    </template>
  </div>
</template>

<style scoped>
.domain-panel { width: 100%; }
.loading { text-align: center; padding: 60px; color: #888; }

.panel-header {
  display: flex; justify-content: space-between; align-items: flex-start;
  margin-bottom: 24px; flex-wrap: wrap; gap: 16px;
}
.header-left { display: flex; align-items: flex-start; gap: 12px; }
.back-btn { background: none; border: none; color: #0f3460; cursor: pointer; font-size: 14px; white-space: nowrap; padding: 8px 0; }
.back-btn:hover { text-decoration: underline; }
.panel-header h1 { margin: 0 0 8px; font-size: 24px; }
.header-meta { display: flex; gap: 16px; font-size: 13px; color: #666; flex-wrap: wrap; }
.header-actions { display: flex; gap: 8px; }

.badge { padding: 3px 10px; border-radius: 20px; font-size: 12px; font-weight: 600; }
.badge-on { background: #d4edda; color: #155724; }
.badge-off { background: #f8f9fa; color: #888; }

.btn-outline { padding: 10px 20px; border: 2px solid #0f3460; color: #0f3460; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; text-decoration: none; background: white; }
.btn-outline:hover { background: #f0f4ff; }
.btn-primary { padding: 10px 20px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-primary:hover { background: #1a4a7a; }
.btn-primary:disabled { background: #999; }

.ssl-result { background: #e8f4fd; color: #0f3460; padding: 12px 16px; border-radius: 8px; margin-bottom: 16px; font-size: 14px; }

.tabs { display: flex; gap: 4px; margin-bottom: 24px; background: white; border-radius: 12px; padding: 6px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); overflow-x: auto; }
.tab { padding: 10px 20px; border: none; background: none; border-radius: 8px; font-size: 14px; cursor: pointer; white-space: nowrap; color: #666; }
.tab.active { background: #0f3460; color: white; }
.tab:hover:not(.active) { background: #f0f0f0; }

.tab-content { background: white; border-radius: 12px; padding: 24px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }

.quick-actions { display: grid; grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); gap: 12px; margin-bottom: 24px; }
.action-card { background: #f8f9fa; border-radius: 12px; padding: 20px; text-align: center; cursor: pointer; transition: all 0.2s; border: 2px solid transparent; text-decoration: none; color: #333; display: block; }
.action-card:hover { border-color: #0f3460; background: #f0f4ff; }
.ac-icon { display: block; font-size: 28px; margin-bottom: 8px; }

.section { margin-bottom: 24px; }
.section h3 { margin: 0 0 12px; font-size: 16px; }
.section-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.section-header h3 { margin: 0; }

.php-selector { display: flex; gap: 8px; flex-wrap: wrap; }
.php-btn { padding: 10px 18px; border: 2px solid #e0e0e0; background: white; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; transition: all 0.2s; }
.php-btn.active { border-color: #0f3460; background: #0f3460; color: white; }
.php-btn:hover:not(.active):not(:disabled) { border-color: #0f3460; }
.php-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.info-item { display: flex; flex-direction: column; gap: 4px; padding: 12px; background: #f8f9fa; border-radius: 8px; }
.info-item span { font-size: 11px; color: #888; text-transform: uppercase; }
.info-item strong { font-size: 15px; color: #1a1a2e; }

.empty { text-align: center; padding: 40px; color: #888; }

.data-list { display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px; }
.data-row { display: flex; justify-content: space-between; align-items: center; padding: 12px 16px; background: #f8f9fa; border-radius: 8px; }
.data-row strong { display: block; font-size: 14px; }
.data-row code { background: #e0e0e0; padding: 2px 6px; border-radius: 4px; font-size: 12px; margin-left: 8px; }
.muted { font-size: 12px; color: #888; margin-left: 8px; }
.row-actions { display: flex; gap: 6px; }
.btn-sm { padding: 6px 12px; border: 1px solid #ddd; background: white; border-radius: 6px; font-size: 12px; cursor: pointer; text-decoration: none; color: #333; }
.btn-sm:hover { background: #f5f5f5; }
.btn-del { color: #c0392b; border-color: #f0d0d0; }
.btn-del:hover { background: #fff0f0; }
.btn-add-sm { padding: 8px 16px; background: #0f3460; color: white; border: none; border-radius: 6px; font-size: 13px; cursor: pointer; }
.btn-cancel-sm { padding: 8px 16px; background: #f0f0f0; border: none; border-radius: 6px; font-size: 13px; cursor: pointer; }

.inline-form { display: flex; gap: 8px; flex-wrap: wrap; padding: 16px; background: #f8f9fa; border-radius: 8px; margin-top: 12px; }
.inline-form input { padding: 8px 12px; border: 2px solid #e0e0e0; border-radius: 6px; font-size: 13px; flex: 1; min-width: 120px; }
.inline-form input:focus { outline: none; border-color: #0f3460; }

.dns-records { margin-bottom: 16px; }
.dns-row { display: grid; grid-template-columns: 80px 1fr 1fr 60px; gap: 8px; padding: 8px 0; border-bottom: 1px solid #f0f0f0; font-size: 14px; }
.dns-row.header { font-weight: 700; font-size: 12px; color: #888; text-transform: uppercase; }

.log-section { margin-bottom: 16px; }
.log-section h4 { margin: 0 0 8px; }
.log-preview { background: #1a1a2e; color: #4ecb71; padding: 12px 16px; border-radius: 8px; font-size: 13px; font-family: 'Consolas',monospace; }
</style>
