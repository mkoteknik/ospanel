<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface Service { name: string; display: string; icon: string; category: string; systemd: string; installed: boolean; active: boolean; enabled: boolean }

const services = ref<Service[]>([])
const loading = ref(true)
const actionLoading = ref('')
const categories = ['web','database','cache','email','dns','security','container']

const catNames: Record<string,string> = { web:'🌐 Web', database:'🗄️ Veritabanı', cache:'⚡ Cache', email:'📧 Email', dns:'🔧 DNS', security:'🛡️ Güvenlik', container:'🐳 Konteyner' }

async function load() {
  loading.value = true
  try { const res = await api.get('/api/v1/services'); services.value = res.data.services || [] } catch { }
  finally { loading.value = false }
}

async function doAction(service: string, action: string) {
  actionLoading.value = service + action
  try {
    await api.post('/api/v1/services/action', { service, action })
    await load()
  } catch { }
  finally { actionLoading.value = '' }
}

function getStatusIcon(s: Service): string {
  if (!s.installed) return '⬜'
  if (s.active) return '🟢'
  return '🔴'
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div><h2>⚙️ Sistem Yönetimi</h2><p>Servisleri görüntüle, başlat/durdur, tek tıkla kur.</p></div>
      <button class="btn-refresh" @click="load">🔄 Yenile</button>
    </div>

    <div v-if="loading" class="loading">Servisler kontrol ediliyor...</div>

    <div v-else v-for="cat in categories" :key="cat" class="category">
      <div class="cat-title">{{ catNames[cat] }}</div>
      <div class="service-grid">
        <div v-for="s in services.filter(s => s.category === cat)" :key="s.name" class="svc-card">
          <div class="svc-top">
            <span class="svc-icon">{{ s.icon }}</span>
            <div class="svc-info">
              <strong>{{ s.display }}</strong>
              <small>{{ s.systemd }}</small>
            </div>
            <span class="svc-status">{{ getStatusIcon(s) }}</span>
          </div>
          <div class="svc-actions">
            <template v-if="!s.installed">
              <button class="btn-install" :disabled="!!actionLoading" @click="doAction(s.name, 'install')">
                {{ actionLoading === s.name + 'install' ? '⏳ Kuruluyor...' : '📥 Kur' }}
              </button>
            </template>
            <template v-else>
              <button v-if="s.active" class="btn-stop" :disabled="!!actionLoading" @click="doAction(s.name, 'stop')">⏹ Durdur</button>
              <button v-else class="btn-start" :disabled="!!actionLoading" @click="doAction(s.name, 'start')">▶ Başlat</button>
              <button class="btn-restart" :disabled="!!actionLoading" @click="doAction(s.name, 'restart')">🔄</button>
              <button v-if="s.enabled" class="btn-disable" :disabled="!!actionLoading" @click="doAction(s.name, 'disable')" title="Boot'ta başlatma">🔽</button>
              <button v-else class="btn-enable" :disabled="!!actionLoading" @click="doAction(s.name, 'enable')" title="Boot'ta başlat">🔼</button>
            </template>
          </div>
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
.btn-refresh { padding: 10px 20px; background: white; border: 1px solid #ddd; border-radius: 8px; font-size: 14px; cursor: pointer; }
.btn-refresh:hover { background: #f5f5f5; }

.loading { text-align: center; padding: 60px; color: #888; }

.category { margin-bottom: 28px; }
.cat-title { font-size: 12px; text-transform: uppercase; color: #888; font-weight: 700; margin-bottom: 10px; letter-spacing: 1px; }

.service-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 12px; }

.svc-card { background: white; border-radius: 12px; padding: 16px 20px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); border: 1px solid #f0f0f0; }
.svc-top { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
.svc-icon { font-size: 24px; }
.svc-info { flex: 1; }
.svc-info strong { display: block; font-size: 14px; color: #1a1a2e; }
.svc-info small { font-size: 11px; color: #888; font-family: monospace; }
.svc-status { font-size: 14px; }

.svc-actions { display: flex; gap: 6px; }
.svc-actions button { padding: 6px 12px; border-radius: 6px; font-size: 12px; cursor: pointer; border: 1px solid #ddd; background: white; transition: all 0.15s; }
.svc-actions button:hover:not(:disabled) { background: #f0f0f0; }
.svc-actions button:disabled { opacity: 0.5; cursor: not-allowed; }

.btn-install { background: #0f3460 !important; color: white !important; border-color: #0f3460 !important; }
.btn-install:hover:not(:disabled) { background: #1a4a7a !important; }
.btn-start { color: #155724; border-color: #c3e6cb !important; }
.btn-start:hover:not(:disabled) { background: #d4edda !important; }
.btn-stop { color: #c0392b; border-color: #f5c6cb !important; }
.btn-stop:hover:not(:disabled) { background: #f8d7da !important; }
.btn-restart { color: #856404; border-color: #ffeeba !important; }
.btn-enable { color: #0f3460; }
.btn-disable { color: #888; }
</style>
