<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface Container {
  id: string; name: string; image: string; status: string; state: string; ports: string[]; created: string
}

const containers = ref<Container[]>([])
const stats = ref<any>({ installed: false })
const loading = ref(true)
const actionLoading = ref<string | null>(null)

async function load() {
  try {
    const res = await api.get('/api/v1/containers')
    containers.value = res.data.containers || []
  } catch { containers.value = [] }
  try {
    const r = await api.get('/api/v1/containers/stats')
    stats.value = r.data
  } catch { }
  finally { loading.value = false }
}

async function doAction(id: string, action: string) {
  actionLoading.value = id
  try { await api.post(`/api/v1/containers/${id}/${action}`); await load() }
  catch { }
  finally { actionLoading.value = null }
}

function getStateColor(state: string) {
  return state === 'running' ? '#d4edda' : state === 'exited' ? '#f8f9fa' : '#ffe0e0'
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>🐳 Konteyner Yönetimi</h2>
        <p>Docker / Podman konteynerlerini görüntüleyin ve yönetin.</p>
      </div>
      <button class="btn-refresh" @click="load">🔄 Yenile</button>
    </div>

    <div v-if="loading" class="loading">Konteynerler yükleniyor...</div>

    <div v-else-if="!stats.installed" class="not-installed">
      <div class="ni-icon">🐳</div>
      <h3>Docker / Podman kurulu değil</h3>
      <p>Konteyner desteği için Docker veya Podman kurulu olmalıdır.</p>
      <div class="install-cmds">
        <div class="cmd-block">
          <strong>Docker:</strong>
          <code>curl -fsSL https://get.docker.com | sudo bash</code>
        </div>
        <div class="cmd-block">
          <strong>Rootless Podman (önerilen):</strong>
          <code>sudo apt install podman -y</code>
        </div>
      </div>
    </div>

    <div v-else>
      <div class="stats-bar">
        <span>Toplam: <strong>{{ stats.total }}</strong></span>
        <span>Çalışan: <strong style="color:#155724">{{ stats.running }}</strong></span>
        <span>Socket: <code>{{ stats.socket }}</code></span>
      </div>

      <div v-if="containers.length === 0" class="empty-state">
        <p>Henüz konteyner yok.</p>
      </div>

      <div v-else class="container-grid">
        <div v-for="c in containers" :key="c.id" class="container-card">
          <div class="cc-header">
            <span class="cc-name">{{ c.name || c.id }}</span>
            <span class="cc-state" :style="{ background: getStateColor(c.state) }">
              {{ c.state }}
            </span>
          </div>
          <div class="cc-body">
            <div class="cc-row"><span>İmaj:</span> {{ c.image }}</div>
            <div class="cc-row"><span>ID:</span> <code>{{ c.id }}</code></div>
            <div class="cc-row"><span>Oluşturma:</span> {{ c.created }}</div>
            <div class="cc-row" v-if="c.ports.length">
              <span>Portlar:</span> {{ c.ports.join(', ') }}
            </div>
          </div>
          <div class="cc-actions">
            <button
              v-if="c.state !== 'running'" class="btn-sm btn-start"
              :disabled="actionLoading === c.id" @click="doAction(c.id, 'start')"
            >▶ Başlat</button>
            <button
              v-if="c.state === 'running'" class="btn-sm btn-stop"
              :disabled="actionLoading === c.id" @click="doAction(c.id, 'stop')"
            >⏹ Durdur</button>
            <button
              class="btn-sm" :disabled="actionLoading === c.id"
              @click="doAction(c.id, 'restart')"
            >🔄 Restart</button>
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

.btn-refresh {
  padding: 10px 20px; background: white; border: 1px solid #ddd;
  border-radius: 8px; font-size: 14px; cursor: pointer;
}
.btn-refresh:hover { background: #f5f5f5; }

.loading { text-align: center; padding: 60px; color: #888; }

.not-installed {
  text-align: center; padding: 60px 20px; background: white;
  border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04);
}
.ni-icon { font-size: 56px; margin-bottom: 16px; }
.not-installed h3 { margin: 0 0 8px; }
.not-installed p { color: #888; margin: 0 0 20px; }
.install-cmds { text-align: left; max-width: 600px; margin: 0 auto; display: flex; flex-direction: column; gap: 12px; }
.cmd-block strong { display: block; font-size: 14px; margin-bottom: 6px; }
.cmd-block code {
  display: block; padding: 10px 16px; background: #1a1a2e; color: #e0e0e0;
  border-radius: 8px; font-size: 13px; font-family: 'Consolas', monospace;
}

.stats-bar {
  display: flex; gap: 24px; padding: 12px 16px; background: white;
  border-radius: 10px; margin-bottom: 16px;
  font-size: 14px; color: #666;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
}
.stats-bar code { background: #f0f0f0; padding: 2px 6px; border-radius: 4px; font-size: 12px; }

.empty-state { text-align: center; padding: 40px; color: #888; }

.container-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 16px; }

.container-card {
  background: white; border-radius: 12px; padding: 20px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04); border: 1px solid #f0f0f0;
}

.cc-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.cc-name { font-size: 16px; font-weight: 700; color: #1a1a2e; }
.cc-state {
  padding: 3px 10px; border-radius: 20px; font-size: 11px; font-weight: 600;
  text-transform: uppercase;
}

.cc-body { margin-bottom: 16px; }
.cc-row { font-size: 13px; padding: 3px 0; color: #555; }
.cc-row span { color: #888; font-size: 11px; text-transform: uppercase; margin-right: 8px; }
.cc-row code { background: #f0f0f0; padding: 2px 6px; border-radius: 4px; font-size: 12px; }

.cc-actions { display: flex; gap: 8px; padding-top: 12px; border-top: 1px solid #f0f0f0; }
.btn-sm {
  flex: 1; padding: 8px; border: 1px solid #ddd; background: white;
  border-radius: 6px; font-size: 13px; cursor: pointer; text-align: center;
}
.btn-sm:hover { background: #f5f5f5; }
.btn-sm:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-start { color: #155724; border-color: #c3e6cb; }
.btn-start:hover { background: #d4edda; }
.btn-stop { color: #c0392b; border-color: #f5c6cb; }
.btn-stop:hover { background: #f8d7da; }
</style>
