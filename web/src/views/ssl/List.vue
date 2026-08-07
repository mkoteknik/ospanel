<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

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
  if (!confirm('SSL sertifikasi silinecek!')) return
  try { await api.delete('/api/v1/ssl/' + domainId); await loadSSL() }
  catch { }
}

function daysClass(days: number) {
  if (days > 60) return 'days-ok'
  if (days > 30) return 'days-warn'
  return 'days-critical'
}

onMounted(loadSSL)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>SSL Sertifikalari</h2>
        <p>Domain SSL durumlarini goruntuleyin, yenileyin ve yonetin.</p>
      </div>
      <button class="btn-sm" @click="loadSSL">Yenile</button>
    </div>

    <div v-if="loading" class="loading">Yukleniyor...</div>

    <div v-else-if="sslDomains.length === 0" class="empty">
      <p>Henuz SSL kurulu domain yok. Domain detay sayfasindan SSL kurabilirsiniz.</p>
    </div>

    <div v-else class="ssl-grid">
      <div v-for="d in sslDomains" :key="d.domain_id" class="ssl-card" :class="{ 'ssl-active': d.ssl_enabled && d.cert }">
        <div class="ssl-domain">{{ d.domain }}</div>
        <div class="ssl-status-row">
          <span v-if="d.ssl_enabled && d.cert" :class="'days-badge ' + daysClass(d.cert.days_left)">
            {{ d.cert.days_left }} gun
          </span>
          <span v-else class="days-badge days-none">SSL Yok</span>
        </div>
        <div v-if="d.cert" class="ssl-info">
          <small>{{ d.cert.issuer }}</small>
          <small>Bitis: {{ d.cert.expires_at?.split('T')[0] || '-' }}</small>
        </div>
        <div class="ssl-actions" v-if="d.cert">
          <button class="btn-sm" @click="renewCert(d.domain_id)">Yenile</button>
          <button class="btn-sm-danger" @click="deleteCert(d.domain_id)">Sil</button>
        </div>
        <div v-else class="ssl-actions">
          <router-link :to="'/domains/' + d.domain_id" class="btn-sm">SSL Kur</router-link>
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
.btn-sm { padding: 8px 16px; background: #0f3460; color: white; border: none; border-radius: 6px; font-size: 13px; cursor: pointer; text-decoration: none; display: inline-block; }
.btn-sm:hover { background: #1a4a7a; }
.btn-sm-danger { padding: 8px 16px; background: white; color: #d32f2f; border: 1px solid #d32f2f; border-radius: 6px; font-size: 13px; cursor: pointer; }
.loading { text-align: center; padding: 60px; color: #888; }
.empty { text-align: center; padding: 60px; background: white; border-radius: 12px; color: #888; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.ssl-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 16px; }
.ssl-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); border: 2px solid #f0f0f0; transition: all 0.2s; }
.ssl-card.ssl-active { border-color: #d4edda; }
.ssl-domain { font-weight: 700; font-size: 16px; margin-bottom: 8px; color: #1a1a2e; }
.ssl-status-row { margin-bottom: 8px; }
.days-badge { display: inline-block; padding: 4px 12px; border-radius: 20px; font-size: 12px; font-weight: 700; }
.days-ok { background: #d4edda; color: #155724; }
.days-warn { background: #fff3cd; color: #856404; }
.days-critical { background: #f8d7da; color: #721c24; }
.days-none { background: #f0f0f0; color: #888; }
.ssl-info { display: flex; flex-direction: column; gap: 4px; margin-bottom: 12px; }
.ssl-info small { color: #888; font-size: 12px; }
.ssl-actions { display: flex; gap: 8px; }
</style>
