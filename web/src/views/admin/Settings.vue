<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface Setting { key: string; value: string; description: string }

const settings = ref<Setting[]>([])
const loading = ref(false)

async function loadSettings() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/admin/settings')
    settings.value = res.data.settings || []
  } catch { }
  finally { loading.value = false }
}

async function saveSetting(key: string, value: string) {
  try {
    await api.put('/api/v1/admin/settings', { [key]: value })
    alert('Ayar kaydedildi!')
  } catch { }
}

onMounted(loadSettings)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>Sistem Ayarlari</h2>
        <p>Panel konfigurasyonunu yonetin.</p>
      </div>
    </div>

    <div v-if="loading" class="loading">Yukleniyor...</div>

    <div v-else class="settings-grid">
      <div v-for="s in settings" :key="s.key" class="setting-card">
        <div class="setting-info">
          <label>{{ s.description || s.key }}</label>
          <small>{{ s.key }}</small>
        </div>
        <div class="setting-input-row">
          <input :value="s.value" :id="'input-' + s.key" class="setting-input" />
          <button class="btn-sm" @click="saveSetting(s.key, (document.getElementById('input-' + s.key) as HTMLInputElement).value)">Kaydet</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 14px; }
.loading { text-align: center; padding: 60px; color: #888; }
.settings-grid { display: flex; flex-direction: column; gap: 12px; }
.setting-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); display: flex; justify-content: space-between; align-items: center; gap: 20px; }
.setting-info label { display: block; font-weight: 600; font-size: 14px; color: #1a1a2e; }
.setting-info small { font-size: 11px; color: #aaa; font-family: monospace; }
.setting-input-row { display: flex; gap: 8px; align-items: center; }
.setting-input { padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; width: 250px; font-family: monospace; }
.setting-input:focus { outline: none; border-color: #0f3460; }
.btn-sm { padding: 10px 18px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 13px; cursor: pointer; }
.btn-sm:hover { background: #1a4a7a; }
</style>
