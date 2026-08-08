<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, onMounted, computed } from 'vue'
import { api } from '@/api/client'

interface Container { id: string; name: string; image: string; status: string; state: string; ports: string[]; created: string }
interface Template { id: string; name: string; icon: string; description: string; image: string; ports: string; category: string; env: string }

const containers = ref<Container[]>([])
const { t } = useI18n()
const stats = ref<any>({ installed: false, total: 0, running: 0 })
const templates = ref<Template[]>([])
const loading = ref(true)
const actionLoading = ref<string | null>(null)
const installLoading = ref(false)
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

async function installRuntime() {
  installLoading.value = true
  try {
    await api.post('/api/v1/services/action', { service: 'podman', action: 'install' })
    await load()
  } catch { }
  finally { installLoading.value = false }
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
    showDeploy.value = false
    await load()
  } catch (err: any) {
    deployResult.value = err.response?.data?.error || 'Deploy basarisiz'
  }
  finally { deployLoading.value = null }
}

const categories = computed(() => [...new Set(templates.value.map(t => t.category))])

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>{{ t('containers.title') }}</h2>
        <p>Docker/Podman konteynerleri ve one-click uygulama kurulumu.</p>
      </div>
      <div class="header-actions">
        <span v-if="stats.installed" class="runtime-badge">
          <span class="runtime-dot"></span> {{ t('containers.runtimeActiveWithCount', { running: stats.running, total: stats.total }) }}
        </span>
        <button class="btn-refresh" @click="load">🔄 Yenile</button>
      </div>
    </div>

    <!-- Runtime Kurulu Değil -->
    <div v-if="!loading && !stats.installed" class="install-prompt">
      <div class="prompt-icon">🐳</div>
      <h3>{{ t('containers.notInstalled') }}</h3>
      <p>{{ t('containers.notInstalledDesc') }}</p>
      <div class="prompt-actions">
        <button class="btn-install-runtime" :disabled="installLoading" @click="installRuntime">
          <span v-if="installLoading" class="btn-spinner"></span>
          {{ installLoading ? t('common.installing') : t('auto.4be928') }}
        </button>
        <code class="prompt-cmd">sudo apt install podman -y</code>
      </div>
    </div>

    <!-- One-Click Deploy -->
    <div v-if="stats.installed" class="deploy-section">
      <h3>⚡ One-Click Deploy</h3>
      <p class="section-desc">{{ t('auto.76f13c') }}</p>
      <div v-for="cat in categories" :key="cat" class="deploy-category">
        <div class="cat-label">{{ cat }}</div>
        <div class="template-grid">
          <div
            v-for="t in templates.filter(tp => tp.category === cat)"
            :key="t.id"
            class="template-card"
            @click="openDeploy(t)"
          >
            <span class="t-icon">{{ t.icon }}</span>
            <span class="t-name">{{ t.name }}</span>
            <span class="t-desc">{{ t.description }}</span>
            <span class="t-image">{{ t.image }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Konteyner Listesi -->
    <div v-if="containers.length > 0" class="container-section">
      <h3>📦 Konteynerler</h3>
      <div class="container-grid">
        <div
          v-for="c in containers"
          :key="c.id"
          class="container-card"
          :class="{ running: c.state === 'running' }"
        >
          <div class="cc-left">
            <div class="cc-icon-wrap" :class="{ pulse: c.state === 'running' }">
              <span class="cc-icon">📦</span>
              <span v-if="c.state === 'running'" class="cc-dot"></span>
            </div>
            <div class="cc-info">
              <span class="cc-name">{{ c.name || c.id }}</span>
              <span class="cc-image">{{ c.image }}</span>
              <code class="cc-id">{{ c.id }}</code>
            </div>
          </div>

          <div class="cc-meta">
            <span v-if="c.state === 'running'" class="status-tag running">{{ t('common.active') }}</span>
            <span v-else class="status-tag stopped">{{ t('cache.stopped') }}</span>
            <span v-if="c.ports.length" class="cc-ports">{{ c.ports.join(', ') }}</span>
            <span class="cc-created">{{ c.created }}</span>
          </div>

          <div class="cc-actions">
            <label class="toggle-switch" :class="{ loading: actionLoading === c.id }">
              <input
                type="checkbox"
                :checked="c.state === 'running'"
                :disabled="actionLoading === c.id"
                @change="doAction(c.id, c.state === 'running' ? 'stop' : 'start')"
              />
              <span class="toggle-slider"></span>
            </label>
            <button
              class="btn-icon"
              title="Restart"
              :disabled="actionLoading === c.id || c.state !== 'running'"
              @click="doAction(c.id, 'restart')"
            >🔄</button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>{{ t('common.loading') }}</p>
    </div>

    <!-- Deploy Modal -->
    <div v-if="showDeploy && selectedTemplate" class="modal-overlay" @click.self="showDeploy = false">
      <div class="modal">
        <div class="modal-header">
          <h3>{{ selectedTemplate.icon }} {{ selectedTemplate.name }}</h3>
          <button class="modal-close" @click="showDeploy = false">✕</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>{{ t('containers.containerName') }}</label>
            <input v-model="deployName" type="text" />
          </div>
          <div class="form-row">
            <div class="form-group">
              <label>{{ t('containers.image') }}</label>
              <input :value="selectedTemplate.image" disabled type="text" class="input-disabled" />
            </div>
            <div class="form-group">
              <label>Port</label>
              <input :value="selectedTemplate.ports" disabled type="text" class="input-disabled" />
            </div>
          </div>
          <pre v-if="deployResult" class="deploy-result">{{ deployResult }}</pre>
        </div>
        <div class="modal-footer">
          <button class="btn-cancel" @click="showDeploy = false">{{ t('common.cancel') }}</button>
          <button class="btn-deploy" :disabled="deployLoading === selectedTemplate.id" @click="deployApp">
            {{ deployLoading === selectedTemplate.id ? '⏳ Deploy ediliyor...' : '🚀 Deploy Et' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; color: #1a1a2e; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 13px; }
.header-actions { display: flex; align-items: center; gap: 12px; }
.runtime-badge { display: flex; align-items: center; gap: 6px; font-size: 12px; color: #155724; background: #d4edda; padding: 6px 12px; border-radius: 20px; font-weight: 600; }
.runtime-dot { width: 8px; height: 8px; background: #27ae60; border-radius: 50%; }
.btn-refresh { padding: 8px 16px; background: var(--aura-surface); border: 1px solid #e0e0e0; border-radius: 8px; font-size: 13px; cursor: pointer; color: #555; }
.btn-refresh:hover { background: #f5f5f5; }

/* Loading */
.loading-state { text-align: center; padding: 80px 0; }
.spinner { width: 36px; height: 36px; border: 3px solid #e0e0e0; border-top-color: #0f3460; border-radius: 50%; animation: spin 0.8s linear infinite; margin: 0 auto 16px; }
@keyframes spin { to { transform: rotate(360deg); } }

/* Install Prompt */
.install-prompt { text-align: center; padding: 60px 20px; background: var(--aura-surface); border-radius: 16px; border: 2px dashed #e0e0e0; }
.prompt-icon { font-size: 56px; margin-bottom: 12px; }
.install-prompt h3 { margin: 0 0 8px; font-size: 20px; color: #1a1a2e; }
.install-prompt p { color: #888; margin: 0 0 20px; }
.prompt-actions { display: flex; flex-direction: column; align-items: center; gap: 12px; }
.btn-install-runtime { display: flex; align-items: center; gap: 8px; padding: 14px 32px; background: linear-gradient(135deg, #0f3460, #1a4a7a); color: white; border: none; border-radius: 10px; font-size: 15px; font-weight: 600; cursor: pointer; transition: all 0.2s; }
.btn-install-runtime:hover:not(:disabled) { transform: translateY(-1px); box-shadow: 0 6px 20px rgba(15,52,96,0.3); }
.btn-install-runtime:disabled { opacity: 0.7; cursor: not-allowed; }
.prompt-cmd { padding: 10px 20px; background: #1a1a2e; color: #4ecb71; border-radius: 8px; font-family: 'Consolas', monospace; font-size: 13px; }

/* Deploy Section */
.deploy-section { margin-bottom: 32px; }
.deploy-section h3 { margin: 0 0 4px; font-size: 18px; }
.section-desc { color: #888; font-size: 13px; margin: 0 0 20px; }
.deploy-category { margin-bottom: 16px; }
.cat-label { font-size: 11px; text-transform: uppercase; color: #888; font-weight: 700; margin-bottom: 8px; letter-spacing: 1px; }
.template-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 10px; }
.template-card { background: var(--aura-surface); border-radius: 12px; padding: 16px; border: 1px solid var(--aura-border); cursor: pointer; transition: all 0.2s; display: flex; flex-direction: column; gap: 4px; }
.template-card:hover { border-color: #0f3460; box-shadow: 0 4px 16px rgba(0,0,0,0.08); transform: translateY(-2px); }
.t-icon { font-size: 26px; }
.t-name { font-size: 14px; font-weight: 700; color: #1a1a2e; }
.t-desc { font-size: 11px; color: #888; line-height: 1.4; }
.t-image { font-size: 10px; color: #aaa; font-family: monospace; margin-top: auto; padding-top: 8px; }

/* Container Section */
.container-section { margin-top: 24px; }
.container-section h3 { margin: 0 0 12px; font-size: 18px; }
.container-grid { display: flex; flex-direction: column; gap: 8px; }

.container-card { display: flex; align-items: center; gap: 16px; padding: 14px 20px; background: var(--aura-surface); border-radius: 10px; border: 1px solid var(--aura-border); transition: all 0.2s; }
.container-card.running { border-left: 3px solid #27ae60; background: #fafffe; }
.container-card:hover { border-color: #e0e0e0; box-shadow: 0 2px 8px rgba(0,0,0,0.04); }

.cc-left { display: flex; align-items: center; gap: 14px; min-width: 240px; }
.cc-icon-wrap { position: relative; width: 40px; height: 40px; display: flex; align-items: center; justify-content: center; background: #f5f6fa; border-radius: 10px; }
.cc-icon { font-size: 20px; }
.cc-dot { position: absolute; bottom: -2px; right: -2px; width: 9px; height: 9px; background: #27ae60; border: 2px solid white; border-radius: 50%; }
.pulse .cc-dot { animation: pulse-dot 2s infinite; }
@keyframes pulse-dot { 0%, 100% { box-shadow: 0 0 0 0 rgba(39,174,96,0.4); } 50% { box-shadow: 0 0 0 6px rgba(39,174,96,0); } }

.cc-info { display: flex; flex-direction: column; }
.cc-name { font-weight: 600; font-size: 14px; color: #1a1a2e; }
.cc-image { font-size: 11px; color: #888; }
.cc-id { font-size: 10px; color: #aaa; font-family: monospace; margin-top: 2px; }

.cc-meta { flex: 1; display: flex; align-items: center; gap: 12px; }
.status-tag { padding: 3px 10px; border-radius: 20px; font-size: 11px; font-weight: 600; }
.status-tag.running { background: #d4edda; color: #155724; }
.status-tag.stopped { background: #f8d7da; color: #721c24; }
.cc-ports { font-size: 11px; color: #555; font-family: monospace; background: #f0f0f0; padding: 3px 8px; border-radius: 4px; }
.cc-created { font-size: 11px; color: #aaa; }

.cc-actions { display: flex; align-items: center; gap: 8px; }

/* Toggle Switch */
.toggle-switch { position: relative; display: inline-block; width: 48px; height: 26px; cursor: pointer; }
.toggle-switch input { opacity: 0; width: 0; height: 0; }
.toggle-slider { position: absolute; inset: 0; background: #ccc; border-radius: 26px; transition: 0.3s; }
.toggle-slider::before { content: ''; position: absolute; height: 20px; width: 20px; left: 3px; bottom: 3px; background: var(--aura-surface); border-radius: 50%; transition: 0.3s; box-shadow: 0 1px 3px rgba(0,0,0,0.2); }
.toggle-switch input:checked + .toggle-slider { background: #27ae60; }
.toggle-switch input:checked + .toggle-slider::before { transform: translateX(22px); }
.toggle-switch.loading { opacity: 0.6; pointer-events: none; }

.btn-icon { width: 34px; height: 34px; display: flex; align-items: center; justify-content: center; border: 1px solid #e0e0e0; border-radius: 8px; background: var(--aura-surface); cursor: pointer; font-size: 14px; transition: all 0.15s; }
.btn-icon:hover:not(:disabled) { background: #f5f5f5; border-color: #ccc; }
.btn-icon:disabled { opacity: 0.4; cursor: not-allowed; }

/* Modal */
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: var(--aura-surface); border-radius: 14px; width: 90%; max-width: 500px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.form-group { margin-bottom: 16px; flex: 1; }
.form-group label { display: block; font-size: 13px; font-weight: 600; margin-bottom: 6px; color: var(--aura-text); }
.form-group input { width: 100%; padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; }
.form-group input:focus { outline: none; border-color: #0f3460; }
.input-disabled { background: #f8f8f8; color: #888; }
.form-row { display: flex; gap: 12px; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; cursor: pointer; font-size: 14px; }
.btn-deploy { padding: 10px 24px; background: linear-gradient(135deg, #0f3460, #1a4a7a); color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-deploy:hover:not(:disabled) { background: linear-gradient(135deg, #1a4a7a, #2563a0); }
.btn-deploy:disabled { opacity: 0.7; cursor: not-allowed; }
.deploy-result { background: #1a1a2e; color: #f85149; padding: 12px; border-radius: 8px; font-size: 12px; font-family: monospace; white-space: pre-wrap; margin-top: 12px; }
</style>
