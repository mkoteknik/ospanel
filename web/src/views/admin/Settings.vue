<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, reactive, onMounted } from 'vue'
import { api } from '@/api/client'

interface Setting { key: string; value: string; description: string }
interface Pkg { id: number; name: string; cpu_shares: number; memory_mb: number; nproc: number; disk_mb: number; max_domains: number; max_emails: number; max_db: number }

const settings = ref<Setting[]>([])
const packages = ref<Pkg[]>([])
const { t } = useI18n()
const loading = ref(false)
const savingKey = ref<string | null>(null)
const error = ref('')
const editValues = reactive<Record<string, string>>({})
const showPkgForm = ref(false)
const editingPkg = ref<Pkg | null>(null)
const newPkg = ref({ name: '', cpu_shares: 1024, memory_mb: 1024, nproc: 50, disk_mb: 5120, max_domains: 10, max_emails: 20, max_db: 10 })

async function loadSettings() {
  loading.value = true; error.value = ''
  try {
    const res = await api.get('/api/v1/admin/settings')
    settings.value = res.data.settings || []
    settings.value.forEach(s => { editValues[s.key] = s.value })
  } catch { error.value = t('admin.settings.loadFailed') }
  finally { loading.value = false }
}

async function saveSetting(key: string) {
  savingKey.value = key
  try { await api.put('/api/v1/admin/settings', { [key]: editValues[key] }) }
  catch { error.value = t('admin.settings.saveFailed') }
  finally { savingKey.value = null }
}

async function loadPackages() { try { const r = await api.get('/api/v1/admin/packages'); packages.value = r.data.packages || [] } catch { } }

async function createPkg() {
  try { await api.post('/api/v1/admin/packages', newPkg.value); showPkgForm.value = false; resetPkgForm(); loadPackages() } catch { }
}

async function updatePkg(id: number) {
  const p = packages.value.find(x => x.id === id)
  if (!p) return
  try { await api.put('/api/v1/admin/packages/' + id, p); editingPkg.value = null; loadPackages() } catch { }
}

async function deletePkg(id: number) {
  if (!confirm('Paket silinecek!')) return
  try { await api.delete('/api/v1/admin/packages/' + id); loadPackages() } catch { }
}

function editPkg(p: Pkg) { editingPkg.value = { ...p } }
function resetPkgForm() { newPkg.value = { name: '', cpu_shares: 1024, memory_mb: 1024, nproc: 50, disk_mb: 5120, max_domains: 10, max_emails: 20, max_db: 10 } }

onMounted(() => { loadSettings(); loadPackages() })
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>{{ t('admin.settings.title') }}</h2>
        <p>{{ t('admin.settings.desc') }}</p>
      </div>
      <button class="aura-btn aura-btn-ghost" @click="loadSettings">↻ {{ t('common.refresh') }}</button>
    </div>

    <div v-if="error" class="aura-card error-card">
      <span class="kicker" style="color:var(--aura-danger)">{{ t('common.error') }}</span>
      <p>{{ error }}</p>
    </div>

    <div v-if="loading" class="loading-state"><div class="spinner"></div><p>{{ t('common.loading') }}</p></div>

    <div v-else-if="settings.length === 0" class="aura-card empty"><p>{{ t('admin.settings.empty') }}</p></div>

    <div v-else class="settings-grid">
      <div v-for="s in settings" :key="s.key" class="aura-card setting-card">
        <div class="setting-info">
          <span class="kicker">{{ s.key }}</span>
          <span class="setting-desc">{{ s.description || s.key }}</span>
        </div>
        <div class="setting-input-row">
          <input v-model="editValues[s.key]" class="aura-input" :placeholder="s.key" />
          <button class="aura-btn aura-btn-primary sm" :disabled="savingKey===s.key" @click="saveSetting(s.key)">
            {{ savingKey===s.key ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Paket Yonetimi -->
    <div class="aura-card section" style="margin-top:20px">
      <div class="section-head" style="display:flex;justify-content:space-between;align-items:center;padding:16px;border-bottom:1px solid var(--aura-border)">
        <div><span class="kicker">Hosting</span><h3>Paket Yönetimi</h3></div>
        <button class="aura-btn aura-btn-primary" @click="showPkgForm = !showPkgForm">{{ showPkgForm ? 'İptal' : '+ Yeni Paket' }}</button>
      </div>

      <!-- Yeni paket formu -->
      <div v-if="showPkgForm" class="pkg-form" style="padding:16px;display:grid;grid-template-columns:1fr 1fr;gap:10px;border-bottom:1px solid var(--aura-border)">
        <input v-model="newPkg.name" placeholder="Paket Adı" class="aura-input" />
        <input v-model.number="newPkg.cpu_shares" placeholder="CPU Shares (1024=1core)" class="aura-input" type="number" />
        <input v-model.number="newPkg.memory_mb" placeholder="RAM (MB)" class="aura-input" type="number" />
        <input v-model.number="newPkg.nproc" placeholder="Max Process" class="aura-input" type="number" />
        <input v-model.number="newPkg.disk_mb" placeholder="Disk (MB)" class="aura-input" type="number" />
        <input v-model.number="newPkg.max_domains" placeholder="Max Domain" class="aura-input" type="number" />
        <input v-model.number="newPkg.max_emails" placeholder="Max Email" class="aura-input" type="number" />
        <input v-model.number="newPkg.max_db" placeholder="Max Veritabanı" class="aura-input" type="number" />
        <button class="aura-btn aura-btn-primary" @click="createPkg" style="grid-column:1/-1">Oluştur</button>
      </div>

      <!-- Paket listesi -->
      <div style="padding:0">
        <div v-for="p in packages" :key="p.id">
          <!-- Goruntuleme satiri -->
          <div v-if="editingPkg?.id !== p.id" style="display:flex;align-items:center;justify-content:space-between;padding:12px 16px;border-bottom:1px solid var(--aura-border);font-size:13px">
            <div style="display:flex;align-items:center;gap:16px;flex:1;min-width:0">
              <strong style="width:100px;flex-shrink:0">{{ p.name }}</strong>
              <span class="muted" style="font-size:11px">CPU:{{ (p.cpu_shares/1024).toFixed(1) }}c</span>
              <span class="muted" style="font-size:11px">RAM:{{ p.memory_mb }}MB</span>
              <span class="muted" style="font-size:11px">Disk:{{ (p.disk_mb/1024).toFixed(0) }}GB</span>
              <span class="muted" style="font-size:11px">Dom:{{ p.max_domains }}</span>
            </div>
            <div style="display:flex;gap:6px">
              <button class="aura-btn aura-btn-ghost sm" @click="editPkg(p)">Düzenle</button>
              <button class="aura-btn aura-btn-ghost sm" @click="deletePkg(p.id)" style="color:var(--aura-danger)">Sil</button>
            </div>
          </div>
          <!-- Duzenleme satiri -->
          <div v-else class="edit-row">
            <div class="edit-grid">
              <label>Paket Adı <input v-model="editingPkg!.name" class="aura-input" /></label>
              <label>CPU (shares) <input v-model.number="editingPkg!.cpu_shares" class="aura-input" type="number" /><small>1024 = 1 core</small></label>
              <label>RAM (MB) <input v-model.number="editingPkg!.memory_mb" class="aura-input" type="number" /></label>
              <label>Max Process <input v-model.number="editingPkg!.nproc" class="aura-input" type="number" /></label>
              <label>Disk (MB) <input v-model.number="editingPkg!.disk_mb" class="aura-input" type="number" /><small>{{ (editingPkg!.disk_mb / 1024).toFixed(1) }} GB</small></label>
              <label>Max Domain <input v-model.number="editingPkg!.max_domains" class="aura-input" type="number" /></label>
              <label>Max E-Posta <input v-model.number="editingPkg!.max_emails" class="aura-input" type="number" /></label>
              <label>Max Veritabanı <input v-model.number="editingPkg!.max_db" class="aura-input" type="number" /></label>
            </div>
            <div class="edit-actions">
              <button class="aura-btn aura-btn-primary sm" @click="updatePkg(editingPkg!.id)">✓ Kaydet</button>
              <button class="aura-btn aura-btn-ghost sm" @click="editingPkg = null">İptal</button>
            </div>
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
.aura-btn { display: inline-flex; align-items: center; gap: 6px; padding: 8px 14px; border-radius: 10px; font-size: 13px; font-weight: 600; cursor: pointer; border: 1px solid transparent; transition: all 0.15s; text-decoration: none; }
.aura-btn-primary { background: var(--aura-accent); color: var(--aura-accent-text); border-color: var(--aura-accent); }
.aura-btn-primary:hover { background: var(--aura-accent-hover); }
.aura-btn-ghost { background: var(--aura-surface); color: var(--aura-text); border-color: var(--aura-border); }
.aura-btn-ghost:hover { background: var(--aura-bg-subtle); }
.aura-btn.sm { padding: 4px 10px; font-size: 12px; }
.aura-card { background: var(--aura-surface); border-radius: 12px; box-shadow: var(--aura-shadow); border: 1px solid var(--aura-border); }
.error-card { padding: 12px 16px; border: 1px solid #fecaca; background: #fef2f2; }
.loading-state { text-align: center; padding: 60px; color: #888; }
.empty { padding: 40px; text-align: center; color: #888; }
.settings-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(380px, 1fr)); gap: 16px; }
.setting-card { padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.setting-info { display: flex; flex-direction: column; gap: 2px; }
.kicker { font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; color: var(--aura-text-faint); }
.setting-desc { font-size: 12px; color: var(--aura-text-muted); }
.setting-input-row { display: flex; gap: 8px; }
.aura-input { width: 100%; padding: 8px 12px; border: 1px solid var(--aura-border); border-radius: 10px; background: var(--aura-surface); color: var(--aura-text); font-size: 13px; }
.aura-input:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
h3 { font-size: 15px; font-weight: 600; color: var(--aura-text); margin: 4px 0 0; }
.muted { color: var(--aura-text-muted); }
.pkg-form input { width: 100%; }

.edit-row {
  padding: 16px; border-bottom: 2px solid var(--aura-accent);
  background: var(--aura-accent-soft); display: flex; flex-direction: column; gap: 12px;
}
.edit-grid {
  display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px;
}
.edit-grid label {
  display: flex; flex-direction: column; gap: 3px;
  font-size: 11px; font-weight: 600; color: var(--aura-text-muted);
  text-transform: uppercase; letter-spacing: 0.03em;
}
.edit-grid label small { font-size: 10px; color: var(--aura-text-faint); text-transform: none; font-weight: 400; }
.edit-grid input { padding: 6px 10px; font-size: 13px; }
.edit-actions { display: flex; gap: 8px; }
@media (max-width: 900px) { .edit-grid { grid-template-columns: repeat(2, 1fr); } }
</style>
