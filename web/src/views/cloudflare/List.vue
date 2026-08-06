<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

const configured = ref(false)
const loading = ref(true)
const stats = ref<any>({})
const showConfig = ref(false)
const purging = ref(false)
const cfConfig = ref({ email: '', api_key: '' })
const zones = ref<{ id: string; name: string; status: string }[]>([])
const selectedZone = ref('')
const dnsRecords = ref<any[]>([])
const sslMode = ref('full')
const activeTab = ref('overview')

async function loadDNS() {
  if (!selectedZone.value) return
  try {
    const dnsRes = await api.get('/api/v1/cf/dns?zone_id=' + selectedZone.value)
    dnsRecords.value = dnsRes.data.records || []
  } catch { }
}

async function deleteDNS(id: string) {
  if (!confirm('DNS kaydı silinecek!')) return
  try { await api.delete('/api/v1/cf/dns?id=' + id + '&zone_id=' + selectedZone.value); await loadDNS() }
  catch { }
}

async function load() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/cf/status')
    stats.value = res.data
    configured.value = res.data.configured
    if (res.data.configured) {
      // Zone listesini al
      try {
        const zRes = await api.get('/api/v1/cf/zones')
        zones.value = zRes.data.zones || []
        if (zones.value.length > 0) {
          selectedZone.value = zones.value[0].id
          loadDNS()
        }
      } catch { }
    }
  } catch { }
  finally { loading.value = false }
}

async function saveConfig() {
  try {
    await api.post('/api/v1/cf/configure', cfConfig.value)
    showConfig.value = false
    await load()
  } catch { }
}

async function purgeAll() {
  if (!confirm('Tüm CloudFlare cache temizlenecek. Emin misiniz?')) return
  purging.value = true
  try { await api.post('/api/v1/cf/purge', {}); await load() }
  catch { }
  finally { purging.value = false }
}

async function changeSSL(mode: string) {
  try {
    await api.post('/api/v1/cf/ssl', { mode })
    sslMode.value = mode
  } catch { }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>☁️ CloudFlare</h2>
        <p>DNS, Cache, SSL ve Güvenlik yönetimi.</p>
      </div>
      <div class="header-actions">
        <button v-if="!configured" class="btn-primary" @click="showConfig = true">🔑 Bağlan</button>
        <button v-if="configured" class="btn-action" @click="showConfig = true">⚙️ Ayarlar</button>
        <button v-if="configured" class="btn-warn" :disabled="purging" @click="purgeAll">
          {{ purging ? 'Temizleniyor...' : '🗑️ Cache Temizle' }}
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading">CloudFlare'a bağlanıyor...</div>

    <!-- Yapılandırılmamış -->
    <div v-else-if="!configured" class="not-configured">
      <div class="nc-icon">☁️</div>
      <h3>CloudFlare henüz yapılandırılmadı</h3>
      <p>CloudFlare API anahtarınızı ekleyerek DNS, cache ve güvenlik özelliklerini panelden yönetin.</p>
      <button class="btn-primary" @click="showConfig = true">🔑 CloudFlare Bağla</button>
    </div>

    <template v-else>
      <!-- Zone Selector -->
      <div class="zone-bar" v-if="zones.length > 0">
        <label>Domain:</label>
        <select v-model="selectedZone" @change="loadDNS">
          <option v-for="z in zones" :key="z.id" :value="z.id">{{ z.name }} ({{ z.status }})</option>
        </select>
        <span class="zone-count">{{ zones.length }} domain</span>
      </div>

      <!-- Stats -->
      <div class="stats-row">
        <div class="stat-card"><span class="st-icon">🌐</span><div><strong>{{ dnsRecords.length }}</strong><small>DNS Kaydı</small></div></div>
        <div class="stat-card"><span class="st-icon">🛡️</span><div><strong>{{ sslMode }}</strong><small>SSL Modu</small></div></div>
        <div class="stat-card"><span class="st-icon">⚡</span><div><strong>Aktif</strong><small>Cache</small></div></div>
        <div class="stat-card"><span class="st-icon">🔥</span><div><strong>Aktif</strong><small>WAF</small></div></div>
      </div>

      <!-- Tabs -->
      <div class="tabs">
        <button :class="'tab ' + (activeTab==='overview'?'active':'')" @click="activeTab='overview'">📊 Genel</button>
        <button :class="'tab ' + (activeTab==='dns'?'active':'')" @click="activeTab='dns'">🔧 DNS</button>
        <button :class="'tab ' + (activeTab==='ssl'?'active':'')" @click="activeTab='ssl'">🔒 SSL</button>
      </div>

      <!-- Overview -->
      <div v-if="activeTab==='overview'" class="tab-content">
        <div class="ssl-section">
          <h3>🔒 SSL/TLS Modu</h3>
          <div class="ssl-options">
            <button v-for="m in ['off','flexible','full','strict']" :key="m"
              :class="'ssl-btn ' + (sslMode===m?'active':'')" @click="changeSSL(m)">
              {{ {off:'Kapalı',flexible:'Esnek',full:'Tam',strict:'Sıkı'}[m] }}
            </button>
          </div>
        </div>
        <div class="quick-section">
          <h3>⚡ Hızlı İşlemler</h3>
          <div class="quick-grid">
            <div class="quick-card" @click="purgeAll"><span>🗑️</span><strong>Tüm Cache Temizle</strong><p>Tüm dosyalar için CF önbelleğini sıfırla</p></div>
            <div class="quick-card" @click="activeTab='dns'"><span>🔧</span><strong>DNS Yönetimi</strong><p>DNS kayıtlarını görüntüle ve düzenle</p></div>
          </div>
        </div>
      </div>

      <!-- DNS -->
      <div v-if="activeTab==='dns'" class="tab-content">
        <div class="dns-table">
          <div class="dns-header"><span>Tür</span><span>Ad</span><span>Değer</span><span>TTL</span><span>Proxy</span><span></span></div>
          <div v-for="r in dnsRecords" :key="r.id" class="dns-row">
            <span class="dns-type">{{ r.type }}</span>
            <span>{{ r.name }}</span>
            <span class="dns-val">{{ r.content }}</span>
            <span>{{ r.ttl }}</span>
            <span>{{ r.proxied ? '🟠' : '⬜' }}</span>
            <button class="btn-del-sm" @click="deleteDNS(r.id)">🗑️</button>
          </div>
        </div>
      </div>

      <!-- SSL -->
      <div v-if="activeTab==='ssl'" class="tab-content">
        <div class="info-card">
          <h3>CloudFlare SSL Sertifikaları</h3>
          <p><strong>Origin Certificate:</strong> Sunucu ile CloudFlare arası şifreleme. 15 yıl geçerli, ücretsiz.</p>
          <p><strong>Edge Certificate:</strong> Ziyaretçi ile CloudFlare arası şifreleme. Otomatik yönetilir.</p>
          <p><strong>Always HTTPS:</strong> Tüm HTTP trafiğini otomatik HTTPS'e yönlendirir.</p>
        </div>
      </div>
    </template>

    <!-- Config Modal -->
    <div v-if="showConfig" class="modal-overlay" @click.self="showConfig=false">
      <div class="modal">
        <div class="modal-header"><h3>☁️ CloudFlare Yapılandırma</h3><button class="modal-close" @click="showConfig=false">✕</button></div>
        <div class="modal-body">
          <div class="form-group"><label>CloudFlare Email</label><input v-model="cfConfig.email" placeholder="admin@site.com" /></div>
          <div class="form-group"><label>Global API Key</label><input v-model="cfConfig.api_key" type="password" placeholder="CloudFlare → My Profile → API Tokens" /></div>
          <p style="font-size:12px;color:#888">API Key: CloudFlare profil → API Tokens → Global API Key. Zone'lar otomatik çekilecek.</p>
        </div>
        <div class="modal-footer"><button class="btn-cancel" @click="showConfig=false">İptal</button><button class="btn-primary" @click="saveConfig">💾 Kaydet</button></div>
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

.btn-primary { padding: 10px 20px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-primary:hover { background: #1a4a7a; }
.btn-action { padding: 10px 20px; background: white; border: 1px solid #ddd; border-radius: 8px; font-size: 14px; cursor: pointer; }
.btn-action:hover { background: #f5f5f5; }
.btn-warn { padding: 10px 20px; background: #e74c3c; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-warn:hover { background: #c0392b; }
.btn-warn:disabled { background: #999; }

.loading { text-align: center; padding: 60px; color: #888; }

.not-configured { text-align: center; padding: 60px 20px; background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.nc-icon { font-size: 56px; margin-bottom: 16px; }
.not-configured h3 { margin: 0 0 8px; }
.not-configured p { color: #888; margin: 0 0 20px; }

.zone-bar { display: flex; align-items: center; gap: 12px; margin-bottom: 20px; background: white; padding: 12px 20px; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.zone-bar label { font-size: 13px; font-weight: 600; color: #333; }
.zone-bar select { padding: 8px 14px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; background: white; min-width: 250px; cursor: pointer; }
.zone-bar select:focus { outline: none; border-color: #0f3460; }
.zone-count { font-size: 12px; color: #888; }

.stats-row { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; margin-bottom: 24px; }
.stat-card { background: white; border-radius: 12px; padding: 16px; display: flex; align-items: center; gap: 14px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.st-icon { font-size: 28px; }
.stat-card strong { display: block; font-size: 20px; color: #1a1a2e; }
.stat-card small { font-size: 12px; color: #888; }

.tabs { display: flex; gap: 4px; margin-bottom: 20px; background: white; border-radius: 10px; padding: 5px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.tab { padding: 10px 20px; border: none; background: none; border-radius: 8px; font-size: 14px; cursor: pointer; color: #666; }
.tab.active { background: #0f3460; color: white; }
.tab:hover:not(.active) { background: #f0f0f0; }

.tab-content { background: white; border-radius: 12px; padding: 24px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }

.ssl-section { margin-bottom: 24px; }
.ssl-section h3 { margin: 0 0 12px; }
.ssl-options { display: flex; gap: 8px; }
.ssl-btn { padding: 10px 20px; border: 2px solid #e0e0e0; background: white; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; text-transform: capitalize; }
.ssl-btn.active { border-color: #0f3460; background: #0f3460; color: white; }
.ssl-btn:hover:not(.active) { border-color: #0f3460; }

.quick-section h3 { margin: 0 0 12px; }
.quick-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 12px; }
.quick-card { background: #f8f9fa; border-radius: 10px; padding: 20px; cursor: pointer; border: 2px solid transparent; transition: all 0.2s; }
.quick-card:hover { border-color: #0f3460; background: #f0f4ff; }
.quick-card span { font-size: 24px; display: block; margin-bottom: 8px; }
.quick-card strong { display: block; font-size: 14px; color: #1a1a2e; margin-bottom: 4px; }
.quick-card p { margin: 0; font-size: 12px; color: #888; }

.dns-table { width: 100%; }
.dns-header, .dns-row { display: grid; grid-template-columns: 60px 1fr 1fr 60px 50px 40px; gap: 8px; padding: 10px 14px; font-size: 13px; align-items: center; }
.dns-header { font-weight: 700; font-size: 11px; color: #888; text-transform: uppercase; border-bottom: 2px solid #e5e5e5; }
.dns-row { border-bottom: 1px solid #f5f5f5; }
.dns-type { font-weight: 700; color: #0f3460; }
.dns-val { font-family: monospace; font-size: 12px; color: #555; overflow: hidden; text-overflow: ellipsis; }
.btn-del-sm { background: none; border: none; font-size: 14px; cursor: pointer; padding: 4px; }

.info-card { background: #f8f9fa; border-radius: 10px; padding: 20px; }
.info-card h3 { margin: 0 0 12px; }
.info-card p { margin: 8px 0; font-size: 14px; color: #555; }
.info-card strong { color: #1a1a2e; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; max-width: 480px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; cursor: pointer; }

.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 14px; font-weight: 600; margin-bottom: 6px; }
.form-group input { width: 100%; padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; }
.form-group input:focus { outline: none; border-color: #0f3460; }
</style>
