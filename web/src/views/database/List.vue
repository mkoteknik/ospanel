<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

const { t } = useI18n()

interface Database {
  id: number; name: string; username: string; charset: string; status: string; created_at: string
}

const databases = ref<Database[]>([])
const loading = ref(true)
const showCreate = ref(false)
const newDB = ref({ name: '', username: '', password: '', charset: 'utf8mb4' })
const creating = ref(false)

const charsets = [
  { label: t('auto.f36e38'), value: 'utf8mb4' },
  { label: 'utf8', value: 'utf8' },
  { label: 'latin1', value: 'latin1' },
  { label: t('auto.fc1e92'), value: 'latin5' },
]

async function loadDBs() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/databases')
    databases.value = res.data.databases || []
  } catch { databases.value = [] }
  finally { loading.value = false }
}

async function createDB() {
  creating.value = true
  try {
    await api.post('/api/v1/databases', newDB.value)
    showCreate.value = false
    newDB.value = { name: '', username: '', password: '', charset: 'utf8mb4' }
    await loadDBs()
  } catch { }
  finally { creating.value = false }
}

async function deleteDB(id: number, name: string) {
  if (!confirm(t('database.confirmDeleteWithName', { name }))) return
  try { await api.delete(`/api/v1/databases/${id}`); await loadDBs() } catch { }
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text)
}

onMounted(loadDBs)
</script>

<template>
  <div class="db-page">
    <div class="page-head">
      <div>
        <h2>{{ t('backup.typeDatabase') }}</h2>
        <p>{{ t('auto.de2cb5') }}</p>
      </div>
      <button class="aura-btn aura-btn-primary" @click="showCreate = true">{{ t('auto.81a54b') }}</button>
    </div>

    <div v-if="loading" class="state muted">{{ t('common.loading') }}</div>

    <div v-else-if="databases.length === 0" class="aura-card empty">
      <div class="empty-icon">◧</div>
      <div class="empty-value">{{ t('database.empty') }}</div>
      <p class="empty-desc">{{ t('auto.6131f8') }}</p>
      <button class="aura-btn aura-btn-primary" @click="showCreate = true">{{ t('auto.6cbcd4') }}</button>
    </div>

    <div v-else class="db-grid">
      <div v-for="db in databases" :key="db.id" class="aura-card db-card">
        <div class="db-top">
          <span class="kicker">{{ t('backup.typeDatabase') }}</span>
          <span class="badge">{{ t('common.active') }}</span>
        </div>
        <div class="db-name">{{ db.name }}</div>
        <div class="db-meta">
          <div class="meta-row">
            <span class="kicker">{{ t('common.user') }}</span>
            <code class="meta-value">{{ db.username }}</code>
          </div>
          <div class="meta-row">
            <span class="kicker">Charset</span>
            <span class="meta-value plain">{{ db.charset }}</span>
          </div>
        </div>
        <div class="db-actions">
          <button class="aura-btn aura-btn-ghost sm" @click="copyToClipboard(db.name)">Kopyala</button>
          <a :href="'http://localhost:7080/phpmyadmin/index.php?db=' + db.name" target="_blank" class="aura-btn aura-btn-ghost sm">phpMyAdmin ↗</a>
          <button class="aura-btn aura-btn-ghost sm danger" @click="deleteDB(db.id, db.name)">{{ t('common.delete') }}</button>
        </div>
      </div>
    </div>

    <!-- Create modal -->
    <div v-if="showCreate" class="overlay" @click.self="showCreate = false">
      <div class="aura-card modal">
        <div class="modal-head">
          <div>
            <span class="kicker">{{ t('auto.ab08f5') }}</span>
            <h3 class="modal-title">{{ t('auto.6d34e0') }}</h3>
          </div>
          <button class="icon-btn" @click="showCreate = false">×</button>
        </div>
        <div class="modal-body">
          <label class="field">
            <span class="kicker">{{ t('database.nameLabel') }}</span>
            <input v-model="newDB.name" type="text" placeholder="orn: wordpress_db" />
          </label>
          <label class="field">
            <span class="kicker">{{ t('database.userLabel') }}</span>
            <input v-model="newDB.username" type="text" placeholder="orn: wp_user" />
          </label>
          <label class="field">
            <span class="kicker">{{ t('common.password') }}</span>
            <input v-model="newDB.password" type="text" :placeholder="t('database.passwordHint')" />
          </label>
          <label class="field">
            <span class="kicker">Karakter seti</span>
            <select v-model="newDB.charset" class="aura-select">
              <option v-for="c in charsets" :key="c.value" :value="c.value">{{ c.label }}</option>
            </select>
          </label>
        </div>
        <div class="modal-foot">
          <button class="aura-btn aura-btn-ghost" @click="showCreate = false">{{ t('common.cancel') }}</button>
          <button class="aura-btn aura-btn-primary" :disabled="creating" @click="createDB">{{ creating ? t('common.creating') : t('common.create') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.db-page { display: flex; flex-direction: column; gap: 16px; }
.page-head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
.page-head h2 { font-size: 20px; font-weight: 700; letter-spacing: -0.02em; color: var(--aura-text); }
.page-head p { margin-top: 4px; font-size: 13px; color: var(--aura-text-muted); max-width: 560px; }

.kicker { font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; color: var(--aura-text-faint); }

.state { text-align: center; padding: 40px; font-size: 13px; }
.state.muted { color: var(--aura-text-muted); }

.empty { text-align: center; padding: 32px 20px; display: flex; flex-direction: column; align-items: center; gap: 8px; }
.empty-icon { width: 44px; height: 44px; border-radius: 12px; display: grid; place-items: center; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); color: var(--aura-text-muted); font-size: 20px; }
.empty-value { font-size: 15px; font-weight: 650; color: var(--aura-text); }
.empty-desc { font-size: 13px; color: var(--aura-text-muted); margin-bottom: 8px; }

.db-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 14px; }
.db-card { padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.db-top { display: flex; align-items: center; justify-content: space-between; }
.badge { display: inline-flex; align-items: center; gap: 6px; padding: 4px 10px; border-radius: 999px; font-size: 11px; font-weight: 650; background: var(--aura-accent-soft); color: var(--aura-success); }
.badge::before { content: ''; width: 6px; height: 6px; border-radius: 999px; background: var(--aura-success); }
.db-name { font-size: 16px; font-weight: 700; letter-spacing: -0.015em; color: var(--aura-text); word-break: break-all; }
.db-meta { display: flex; flex-direction: column; gap: 8px; padding: 10px 12px; border-radius: 10px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); }
.meta-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.meta-value { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; font-weight: 550; color: var(--aura-text); background: var(--aura-surface); border: 1px solid var(--aura-border); padding: 4px 8px; border-radius: 8px; }
.meta-value.plain { background: transparent; border: none; padding: 0; font-family: inherit; font-size: 13px; color: var(--aura-text-muted); }
.db-actions { display: flex; gap: 8px; padding-top: 2px; }
.db-actions .sm { flex: 1; padding: 8px 10px; font-size: 13px; }
.db-actions .sm.danger { color: var(--aura-danger); }
.db-actions .sm.danger:hover { background: color-mix(in srgb, var(--aura-danger) 8%, var(--aura-surface)); border-color: color-mix(in srgb, var(--aura-danger) 18%, var(--aura-border)); }

.overlay { position: fixed; inset: 0; background: rgba(15,23,42,0.45); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 16px; }
.modal { width: 100%; max-width: 460px; overflow: hidden; }
.modal-head { display: flex; justify-content: space-between; align-items: flex-start; padding: 18px 20px; border-bottom: 1px solid var(--aura-border); }
.modal-title { font-size: 15px; font-weight: 700; color: var(--aura-text); margin-top: 2px; }
.modal-body { padding: 18px 20px; display: flex; flex-direction: column; gap: 14px; }
.modal-foot { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid var(--aura-border); }

.field { display: flex; flex-direction: column; gap: 6px; }
.field input, .aura-select { width: 100%; padding: 10px 12px; border: 1px solid var(--aura-border); border-radius: 10px; font-size: 14px; background: var(--aura-surface); color: var(--aura-text); }
.field input:focus, .aura-select:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
.field input:placeholder { color: var(--aura-text-faint); }

.icon-btn { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 8px; border: 1px solid var(--aura-border); background: var(--aura-surface); color: var(--aura-text-muted); cursor: pointer; }
.icon-btn:hover { background: var(--aura-surface-hover); color: var(--aura-text); }
</style>
