<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface Container { id: string; name: string; image: string; status: string; state: string; ports: string[]; created: string }
interface Template { id: string; name: string; icon: string; description: string; image: string; ports: string; category: string }

const containers = ref<Container[]>([])
const stats = ref<any>({ installed: false })
const templates = ref<Template[]>([])
const loading = ref(true)
const actionLoading = ref<string | null>(null)
const deployLoading = ref<string | null>(null)
const showDeploy = ref(false)
const selectedTemplate = ref<Template | null>(null)
const deployName = ref('')
const deployResult = ref('')

async function load() {
  try {
    const [cr, tr, sr] = await Promise.all([
      api.get('/api/v1/containers'),
      api.get('/api/v1/deploy/templates'),
      api.get('/api/v1/containers/stats'),
    ])
    containers.value = cr.data.containers || []
    templates.value = tr.data.templates || []
    stats.value = sr.data
  } catch { }
  finally { loading.value = false }
}

async function doAction(id: string, action: string) {
  actionLoading.value = id
  try { await api.post(`/api/v1/containers/${id}/${action}`); await load() }
  catch { }
  finally { actionLoading.value = null }
}

function openDeploy(template: Template) {
  selectedTemplate.value = template
  deployName.value = template.id + '-' + Math.random().toString(36).slice(2, 8)
  deployResult.value = ''
  showDeploy.value = true
}

async function deployApp() {
  if (!selectedTemplate.value) return
  deployLoading.value = selectedTemplate.value.id
  try {
    const res = await api.post('/api/v1/deploy', {
      template_id: selectedTemplate.value.id,
      name: deployName.value || undefined,
    })
    deployResult.value = JSON.stringify(res.data, null, 2)
    showDeploy.value = false
    await load()
  } catch (err: any) {
    deployResult.value = err.response?.data?.error || 'Deploy başarısız'
  }
  finally { deployLoading.value = null }
}

const categories = [...new Set(templates.value.map(t => t.category))]

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>🐳 Konteyner Yönetimi</h2>
        <p>Docker/Podman konteynerleri ve one-click uygulama kurulumu.</p>
      </div>
      <button class="btn-refresh" @click="load">🔄 Yenile</button>
    </div>

    <!-- One-Click Deploy -->
    <div v-if="stats.installed" class="deploy-section">
      <h3>⚡ One-Click Deploy</h3>
      <div v-for="cat in categories" :key="cat" class="deploy-category">
        <div class="cat-title">{{ cat }}</div>
        <div class="template-grid">
          <div v-for="t in templates.filter(t => t.category === cat)" :key="t.id"
            class="template-card" @click="openDeploy(t)">
            <div class="tc-icon">{{ t.icon }}</div>
            <div class="tc-name">{{ t.name }}</div>
            <div class="tc-desc">{{ t.description }}</div>
            <div class="tc-meta">{{ t.image }} | :{{ t.ports.split(':')[0] }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Container List -->
    <div v-if="containers.length > 0" class="container-grid" style="margin-top:24px">
      <div v-for="c in containers" :key="c.id" class="container-card">
        <div class="cc-header">
          <span class="cc-name">{{ c.name || c.id }}</span>
          <span class="cc-state" :style="{ background: c.state === 'running' ? '#d4edda' : '#f8f9fa' }">{{ c.state }}</span>
        </div>
        <div class="cc-body">
          <div class="cc-row"><span>İmaj:</span> {{ c.image }}</div>
          <div class="cc-row"><span>ID:</span> <code>{{ c.id }}</code></div>
          <div class="cc-row"><span>Oluşturma:</span> {{ c.created }}</div>
          <div class="cc-row" v-if="c.ports.length"><span>Portlar:</span> {{ c.ports.join(', ') }}</div>
        </div>
        <div class="cc-actions">
          <button v-if="c.state !== 'running'" class="btn-sm btn-start" :disabled="actionLoading === c.id" @click="doAction(c.id, 'start')">▶ Başlat</button>
          <button v-if="c.state === 'running'" class="btn-sm btn-stop" :disabled="actionLoading === c.id" @click="doAction(c.id, 'stop')">⏹ Durdur</button>
          <button class="btn-sm" :disabled="actionLoading === c.id" @click="doAction(c.id, 'restart')">🔄 Restart</button>
        </div>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading">Yükleniyor...</div>

    <!-- Not Installed -->
    <div v-if="!loading && !stats.installed" class="not-installed">
      <div class="ni-icon">🐳</div>
      <h3>Docker / Podman kurulu değil</h3>
      <p>Konteyner desteği ve one-click deploy için Docker veya Podman kurulu olmalıdır.</p>
      <code>sudo apt install podman -y</code>
    </div>

    <!-- Deploy Modal -->
    <div v-if="showDeploy && selectedTemplate" class="modal-overlay" @click.self="showDeploy = false">
      <div class="modal">
        <div class="modal-header">
          <h3>{{ selectedTemplate.icon }} {{ selectedTemplate.name }} Deploy</h3>
          <button class="modal-close" @click="showDeploy = false">✕</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>Konteyner Adı</label>
            <input v-model="deployName" type="text" />
          </div>
          <div class="form-group">
            <label>İmaj</label>
            <input :value="selectedTemplate.image" disabled type="text" />
          </div>
          <div class="form-group">
            <label>Port</label>
            <input :value="selectedTemplate.ports" disabled type="text" />
          </div>
          <pre v-if="deployResult" class="deploy-result">{{ deployResult }}</pre>
        </div>
        <div class="modal-footer">
          <button class="btn-cancel" @click="showDeploy = false">İptal</button>
          <button class="btn-add" :disabled="deployLoading === selectedTemplate.id" @click="deployApp">
            {{ deployLoading === selectedTemplate.id ? 'Deploy ediliyor...' : '🚀 Deploy' }}
          </button>
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
.not-installed { text-align: center; padding: 60px 20px; background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.not-installed code { display: block; margin: 12px auto 0; padding: 12px 20px; background: #1a1a2e; color: #e0e0e0; border-radius: 8px; font-family: 'Consolas',monospace; font-size: 14px; max-width: 500px; }
.ni-icon { font-size: 56px; margin-bottom: 16px; }
.not-installed h3 { margin: 0 0 8px; }

.deploy-section { margin-bottom: 24px; }
.deploy-section h3 { margin: 0 0 16px; font-size: 18px; }
.deploy-category { margin-bottom: 20px; }
.cat-title { font-size: 12px; text-transform: uppercase; color: #888; font-weight: 700; margin-bottom: 8px; letter-spacing: 1px; }

.template-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 12px; }
.template-card { background: white; border-radius: 12px; padding: 16px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); border: 1px solid #f0f0f0; cursor: pointer; transition: all 0.2s; }
.template-card:hover { box-shadow: 0 4px 16px rgba(0,0,0,0.08); border-color: #0f3460; transform: translateY(-2px); }
.tc-icon { font-size: 28px; margin-bottom: 8px; }
.tc-name { font-size: 15px; font-weight: 700; color: #1a1a2e; margin-bottom: 4px; }
.tc-desc { font-size: 12px; color: #888; margin-bottom: 8px; line-height: 1.4; }
.tc-meta { font-size: 11px; color: #aaa; font-family: monospace; }

.container-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 16px; }
.container-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); border: 1px solid #f0f0f0; }
.cc-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.cc-name { font-size: 16px; font-weight: 700; color: #1a1a2e; }
.cc-state { padding: 3px 10px; border-radius: 20px; font-size: 11px; font-weight: 600; text-transform: uppercase; }
.cc-body { margin-bottom: 16px; }
.cc-row { font-size: 13px; padding: 3px 0; color: #555; }
.cc-row span { color: #888; font-size: 11px; text-transform: uppercase; margin-right: 8px; }
.cc-row code { background: #f0f0f0; padding: 2px 6px; border-radius: 4px; font-size: 12px; }
.cc-actions { display: flex; gap: 8px; padding-top: 12px; border-top: 1px solid #f0f0f0; }
.btn-sm { flex: 1; padding: 8px; border: 1px solid #ddd; background: white; border-radius: 6px; font-size: 13px; cursor: pointer; text-align: center; }
.btn-sm:hover { background: #f5f5f5; }
.btn-sm:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-start { color: #155724; border-color: #c3e6cb; }
.btn-start:hover { background: #d4edda; }
.btn-stop { color: #c0392b; border-color: #f5c6cb; }
.btn-stop:hover { background: #f8d7da; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; max-width: 500px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; cursor: pointer; }
.btn-add { padding: 10px 20px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-add:hover { background: #1a4a7a; }
.btn-add:disabled { background: #999; cursor: not-allowed; }

.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 14px; font-weight: 600; margin-bottom: 6px; }
.form-group input { width: 100%; padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; }
.form-group input:focus { outline: none; border-color: #0f3460; }
.deploy-result { background: #1a1a2e; color: #4ecb71; padding: 12px; border-radius: 8px; font-size: 12px; font-family: monospace; white-space: pre-wrap; margin-top: 12px; }
</style>
