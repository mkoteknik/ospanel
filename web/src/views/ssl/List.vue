<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'

interface Domain { id: number; domain: string; ssl_enabled: boolean; php_version: string; status: string; document_root: string }

const router = useRouter()
const domains = ref<Domain[]>([])
const loading = ref(true)
const installingSSL = ref<number | null>(null)

async function load() {
  loading.value = true
  try { const r = await api.get('/api/v1/domains'); domains.value = r.data.domains || [] } catch { }
  finally { loading.value = false }
}

async function installSSL(domain: Domain) {
  installingSSL.value = domain.id
  try {
    await api.post('/api/v1/domains/' + domain.id + '/ssl', { type: 'lets_encrypt', email: 'admin@' + domain.domain })
    await load()
  } catch { }
  finally { installingSSL.value = null }
}

function goDomain(id: number) { router.push('/domains/' + id) }

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div><h2>🔒 SSL Yönetimi</h2><p>Domainlerin SSL durumunu görüntüleyin ve yönetin.</p></div>
      <button class="btn-refresh" @click="load">🔄 Yenile</button>
    </div>

    <div v-if="loading" class="loading">Yükleniyor...</div>

    <div v-else-if="domains.length === 0" class="empty">
      <div class="empty-icon">🔒</div>
      <h3>Henüz domain yok</h3>
      <p>SSL yönetimi için önce domain ekleyin.</p>
    </div>

    <div v-else class="ssl-grid">
      <div v-for="d in domains" :key="d.id" class="ssl-card" :class="{ 'ssl-active': d.ssl_enabled }">
        <div class="ssl-top">
          <div class="ssl-domain">{{ d.domain }}</div>
          <span :class="'ssl-badge ' + (d.ssl_enabled ? 'on' : 'off')">
            {{ d.ssl_enabled ? '🔒 SSL Aktif' : '🔓 SSL Yok' }}
          </span>
        </div>
        <div class="ssl-info">
          <div class="ssl-row"><span>PHP</span> {{ d.php_version }}</div>
          <div class="ssl-row"><span>DocRoot</span> <code>{{ d.document_root }}</code></div>
          <div class="ssl-row"><span>Durum</span> <span :class="d.status === 'active' ? 'text-green' : 'text-red'">{{ d.status }}</span></div>
        </div>
        <div class="ssl-actions">
          <button class="btn-sm" @click="goDomain(d.id)">🔍 Domain Panel</button>
          <button v-if="!d.ssl_enabled" class="btn-install" :disabled="installingSSL === d.id" @click="installSSL(d)">
            {{ installingSSL === d.id ? '⏳ Kuruluyor...' : '🔒 SSL Kur' }}
          </button>
          <span v-else class="ssl-done">✅ Sertifikalı</span>
        </div>
      </div>
    </div>

    <div class="info-banner">
      💡 <strong>CloudFlare kullanıyorsanız:</strong> CloudFlare → SSL/TLS → Full (strict) yapın. Ücretsiz Edge sertifikası otomatik aktif olur.
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 14px; }
.btn-refresh { padding: 10px 20px; background: white; border: 1px solid #ddd; border-radius: 8px; font-size: 14px; cursor: pointer; }
.btn-refresh:hover { background: #f5f5f5; }

.loading { text-align: center; padding: 60px; color: #888; }
.empty { text-align: center; padding: 60px 20px; background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty h3 { margin: 0 0 8px; }
.empty p { color: #888; margin: 0; }

.ssl-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 16px; margin-bottom: 20px; }

.ssl-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); border: 1px solid #f0f0f0; transition: all 0.2s; }
.ssl-card.ssl-active { border-left: 4px solid #27ae60; }
.ssl-card:hover { box-shadow: 0 4px 16px rgba(0,0,0,0.08); }

.ssl-top { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
.ssl-domain { font-size: 17px; font-weight: 700; color: #1a1a2e; }
.ssl-badge { padding: 5px 12px; border-radius: 20px; font-size: 12px; font-weight: 600; }
.ssl-badge.on { background: #d4edda; color: #155724; }
.ssl-badge.off { background: #f8f9fa; color: #888; }

.ssl-info { margin-bottom: 16px; }
.ssl-row { display: flex; justify-content: space-between; padding: 5px 0; font-size: 13px; border-bottom: 1px solid #f5f5f5; }
.ssl-row span { color: #888; font-size: 11px; text-transform: uppercase; }
.ssl-row code { font-size: 11px; background: #f0f0f0; padding: 2px 6px; border-radius: 4px; max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.text-green { color: #155724 !important; font-weight: 600; }
.text-red { color: #c0392b !important; }

.ssl-actions { display: flex; gap: 8px; padding-top: 12px; border-top: 1px solid #f0f0f0; }
.btn-sm { flex: 1; padding: 8px; border: 1px solid #ddd; background: white; border-radius: 6px; font-size: 13px; cursor: pointer; text-align: center; }
.btn-sm:hover { background: #f5f5f5; }
.btn-install { flex: 1; padding: 8px; background: #0f3460; color: white; border: none; border-radius: 6px; font-size: 13px; font-weight: 600; cursor: pointer; }
.btn-install:hover:not(:disabled) { background: #1a4a7a; }
.btn-install:disabled { background: #999; cursor: not-allowed; }
.ssl-done { flex: 1; text-align: center; font-size: 14px; padding: 8px; }

.info-banner { background: #e8f4fd; color: #0f3460; padding: 14px 20px; border-radius: 10px; font-size: 14px; }
.info-banner strong { color: #1a1a2e; }
</style>
