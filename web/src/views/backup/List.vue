<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

const { t } = useI18n()

interface BackupJob {
  id: number
  domain_id: number | null
  type: string
  destination: string
  schedule: string
  retention: number
  status: string
  last_run: string | null
  next_run: string | null
  created_at: string
}

const jobs = ref<BackupJob[]>([])
const loading = ref(false)
const showAdd = ref(false)
const newJob = ref({ type: 'full', destination: 'local', schedule: 'daily', retention: 7, domain_id: null as number | null })

async function loadJobs() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/backups')
    jobs.value = res.data.backups || []
  } catch { jobs.value = [] }
  finally { loading.value = false }
}

async function createJob() {
  try {
    await api.post('/api/v1/backups', newJob.value)
    showAdd.value = false
    newJob.value = { type: 'full', destination: 'local', schedule: 'daily', retention: 7, domain_id: null }
    await loadJobs()
  } catch { }
}

async function deleteJob(id: number) {
  if (!confirm(t('backup.confirmDelete'))) return
  try { await api.delete('/api/v1/backups/' + id); await loadJobs() }
  catch { }
}

async function runBackup(id: number) {
  try {
    await api.post('/api/v1/backups/' + id + '/run')
    alert(t('backup.started'))
  } catch { }
}

function statusClass(s: string) {
  return s === 'completed' ? 'tone-ok' : s === 'failed' ? 'tone-err' : 'tone-pending'
}

function typeLabel(val: string) {
  if (val === 'full') return t('common.full')
  if (val === 'database') return t('backup.typeDatabase')
  return val
}

onMounted(loadJobs)
</script>

<template>
  <div class="backup-page">
    <div class="page-head">
      <div>
        <h2>{{ t('backup.title') }}</h2>
        <p>{{ t('backup.subtitle') }}</p>
      </div>
      <button class="aura-btn aura-btn-primary" @click="showAdd = true">{{ t('backup.addPlan') }}</button>
    </div>

    <div v-if="loading" class="state muted">{{ t('common.loading') }}</div>

    <div v-else-if="jobs.length === 0" class="aura-card empty">
      <div class="empty-icon">◧</div>
      <div class="empty-value">{{ t('backup.empty') }}</div>
      <p class="empty-desc">{{ t('backup.emptyDesc') }}</p>
      <button class="aura-btn aura-btn-primary" @click="showAdd = true">{{ t('backup.createFirst') }}</button>
    </div>

    <div v-else class="job-list">
      <div v-for="job in jobs" :key="job.id" class="aura-card job-card">
        <div class="job-main">
          <div class="job-top">
            <span class="kicker">Plan #{{ job.id }}</span>
            <span :class="'pill ' + statusClass(job.status)">{{ job.status }}</span>
          </div>
          <div class="job-title">{{ typeLabel(job.type) }} yedek</div>
          <div class="job-meta">
            <span class="meta-chip">{{ job.schedule || 'Manuel' }}</span>
            <span class="kicker">Hedef: {{ job.destination }}</span>
            <span class="kicker">{{ t('backup.retention') }}</span>
          </div>
          <div v-if="job.last_run" class="job-foot">
            <span class="kicker">{{ t('backup.lastRun') }}</span>
            <span class="foot-value">{{ job.last_run }}</span>
          </div>
        </div>
        <div class="job-actions">
          <button class="aura-btn aura-btn-primary sm" @click="runBackup(job.id)">{{ t('backup.runNow') }}</button>
          <button class="aura-btn aura-btn-ghost sm danger" @click="deleteJob(job.id)">{{ t('common.delete') }}</button>
        </div>
      </div>
    </div>

    <div v-if="showAdd" class="overlay" @click.self="showAdd=false">
      <div class="aura-card modal">
        <div class="modal-head">
          <div>
            <span class="kicker">Yeni plan</span>
            <h3 class="modal-title">{{ t('backup.planTitle') }}</h3>
          </div>
          <button class="icon-btn" @click="showAdd=false">×</button>
        </div>
        <div class="modal-body">
          <label class="field">
            <span class="kicker">{{ t('common.type') }}</span>
            <select v-model="newJob.type" class="aura-select">
              <option value="full">Tam Yedekleme</option>
              <option value="database">{{ t('backup.typeDatabaseOnly') }}</option>
              <option value="incremental">{{ t('backup.typeIncremental') }}</option>
            </select>
          </label>
          <label class="field">
            <span class="kicker">Hedef</span>
            <select v-model="newJob.destination" class="aura-select">
              <option value="local">Lokal (/var/backups/ospanel)</option>
              <option value="s3">Amazon S3</option>
              <option value="ftp">FTP Sunucusu</option>
            </select>
          </label>
          <label class="field">
            <span class="kicker">Zamanlama</span>
            <select v-model="newJob.schedule" class="aura-select">
              <option value="daily">{{ t('backup.scheduleDaily2') }}</option>
              <option value="12:00">{{ t('backup.schedule12') }}</option>
              <option value="20:00">{{ t('backup.schedule20') }}</option>
              <option value="">Manuel</option>
            </select>
          </label>
          <label class="field">
            <span class="kicker">{{ t('backup.retentionLabel') }}</span>
            <input v-model.number="newJob.retention" type="number" min="1" max="365" />
          </label>
        </div>
        <div class="modal-foot">
          <button class="aura-btn aura-btn-ghost" @click="showAdd=false">{{ t('common.cancel') }}</button>
          <button class="aura-btn aura-btn-primary" @click="createJob">{{ t('common.create') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.backup-page { display: flex; flex-direction: column; gap: 16px; }
.page-head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
.page-head h2 { font-size: 20px; font-weight: 700; letter-spacing: -0.02em; color: var(--aura-text); }
.page-head p { margin-top: 4px; font-size: 13px; color: var(--aura-text-muted); max-width: 560px; }
.kicker { font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; color: var(--aura-text-faint); }
.state { text-align: center; padding: 32px; font-size: 13px; }
.state.muted { color: var(--aura-text-muted); }
.empty { text-align: center; padding: 32px 20px; display: flex; flex-direction: column; align-items: center; gap: 8px; }
.empty-icon { width: 44px; height: 44px; border-radius: 12px; display: grid; place-items: center; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); color: var(--aura-text-muted); font-size: 18px; }
.empty-value { font-size: 15px; font-weight: 650; color: var(--aura-text); }
.empty-desc { font-size: 13px; color: var(--aura-text-muted); margin-bottom: 4px; }

.job-list { display: flex; flex-direction: column; gap: 12px; }
.job-card { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 16px; flex-wrap: wrap; }
.job-main { display: flex; flex-direction: column; gap: 8px; min-width: 240px; flex: 1; }
.job-top { display: flex; align-items: center; gap: 8px; }
.job-title { font-size: 15px; font-weight: 700; letter-spacing: -0.015em; color: var(--aura-text); }
.job-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; }
.meta-chip { font-size: 11px; font-weight: 600; color: var(--aura-text-muted); background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); padding: 3px 8px; border-radius: 999px; }
.job-foot { display: flex; align-items: center; gap: 8px; }
.foot-value { font-size: 12px; color: var(--aura-text-muted); }
.job-actions { display: flex; gap: 8px; flex-shrink: 0; }
.sm { padding: 8px 12px; font-size: 12px; }
.sm.danger { color: var(--aura-danger); }
.sm.danger:hover { background: color-mix(in srgb, var(--aura-danger) 8%, var(--aura-surface)); border-color: color-mix(in srgb, var(--aura-danger) 18%, var(--aura-border)); }

.pill { display: inline-flex; padding: 3px 10px; border-radius: 999px; font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.02em; border: 1px solid transparent; }
.tone-ok { background: color-mix(in srgb, var(--aura-success) 12%, var(--aura-surface)); color: var(--aura-success); border-color: color-mix(in srgb, var(--aura-success) 18%, transparent); }
.tone-err { background: color-mix(in srgb, var(--aura-danger) 10%, var(--aura-surface)); color: var(--aura-danger); border-color: color-mix(in srgb, var(--aura-danger) 18%, transparent); }
.tone-pending { background: color-mix(in srgb, var(--aura-warning) 12%, var(--aura-surface)); color: var(--aura-warning); border-color: color-mix(in srgb, var(--aura-warning) 18%, transparent); }

.overlay { position: fixed; inset: 0; background: rgba(15,23,42,0.45); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 16px; }
.modal { width: 100%; max-width: 480px; overflow: hidden; }
.modal-head { display: flex; justify-content: space-between; align-items: flex-start; padding: 18px 20px; border-bottom: 1px solid var(--aura-border); }
.modal-title { font-size: 15px; font-weight: 700; color: var(--aura-text); margin-top: 2px; }
.modal-body { padding: 18px 20px; display: flex; flex-direction: column; gap: 14px; }
.modal-foot { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid var(--aura-border); }
.field { display: flex; flex-direction: column; gap: 6px; }
.field input, .aura-select { width: 100%; padding: 10px 12px; border: 1px solid var(--aura-border); border-radius: 10px; font-size: 14px; background: var(--aura-surface); color: var(--aura-text); }
.field input:focus, .aura-select:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
.icon-btn { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 8px; border: 1px solid var(--aura-border); background: var(--aura-surface); color: var(--aura-text-muted); cursor: pointer; }
.icon-btn:hover { background: var(--aura-surface-hover); color: var(--aura-text); }
@media (max-width: 640px) { .job-card { flex-direction: column; align-items: stretch; } .job-actions { width: 100%; } .job-actions .sm { flex: 1; justify-content: center; } }
</style>
