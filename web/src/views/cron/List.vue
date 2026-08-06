<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface CronJob { minute: string; hour: string; day: string; month: string; weekday: string; command: string; user?: string }
const jobs = ref<CronJob[]>([])
const loading = ref(true)
const showAdd = ref(false)
const newJob = ref({ minute: '*', hour: '*', day: '*', month: '*', weekday: '*', command: '' })

const presets = [
  { label: 'Her dakika', expr: '* * * * *' },
  { label: 'Her saat', expr: '0 * * * *' },
  { label: 'Her gün 03:00', expr: '0 3 * * *' },
  { label: 'Her Pazartesi 06:00', expr: '0 6 * * 1' },
  { label: 'Ayda bir (1.) 00:00', expr: '0 0 1 * *' },
]

async function load() {
  loading.value = true
  try { const res = await api.get('/api/v1/cron'); jobs.value = res.data.jobs || [] } catch { }
  finally { loading.value = false }
}

async function addJob() {
  try {
    await api.post('/api/v1/cron', newJob.value)
    showAdd.value = false
    newJob.value = { minute: '*', hour: '*', day: '*', month: '*', weekday: '*', command: '' }
    await load()
  } catch { }
}

async function deleteJob(cmd: string) {
  if (!confirm('Bu cron silinecek!')) return
  try { await api.delete('/api/v1/cron?command=' + encodeURIComponent(cmd)); await load() }
  catch { }
}

function applyPreset(preset: { label: string; expr: string }) {
  const parts = preset.expr.split(' ')
  newJob.value.minute = parts[0]; newJob.value.hour = parts[1]
  newJob.value.day = parts[2]; newJob.value.month = parts[3]
  newJob.value.weekday = parts[4]
}

function formatCron(j: CronJob): string {
  return `${j.minute} ${j.hour} ${j.day} ${j.month} ${j.weekday}`
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div><h2>⏰ Cron Jobs</h2><p>Zamanlanmış görevleri yönetin.</p></div>
      <button class="btn-primary" @click="showAdd = true">+ Cron Ekle</button>
    </div>

    <div v-if="loading" class="loading">Yükleniyor...</div>

    <div v-else-if="jobs.length === 0" class="empty">Henüz cron job yok. "Cron Ekle" ile yeni görev tanımlayın.</div>

    <div v-else class="cron-list">
      <div v-for="(j, i) in jobs" :key="i" class="cron-row">
        <code class="cron-expr">{{ formatCron(j) }}</code>
        <span class="cron-cmd">{{ j.command }}</span>
        <span v-if="j.user" class="cron-user">{{ j.user }}</span>
        <button class="btn-del-sm" @click="deleteJob(j.command)">🗑️</button>
      </div>
    </div>

    <!-- Add Modal -->
    <div v-if="showAdd" class="modal-overlay" @click.self="showAdd=false">
      <div class="modal">
        <div class="modal-header"><h3>⏰ Cron Ekle</h3><button class="modal-close" @click="showAdd=false">✕</button></div>
        <div class="modal-body">
          <div class="preset-list">
            <button v-for="p in presets" :key="p.label" class="preset-btn" @click="applyPreset(p)">{{ p.label }}</button>
          </div>
          <div class="cron-inputs">
            <div class="ci"><label>Dk</label><input v-model="newJob.minute" /></div>
            <div class="ci"><label>Saat</label><input v-model="newJob.hour" /></div>
            <div class="ci"><label>Gün</label><input v-model="newJob.day" /></div>
            <div class="ci"><label>Ay</label><input v-model="newJob.month" /></div>
            <div class="ci"><label>Hafta</label><input v-model="newJob.weekday" /></div>
          </div>
          <div class="form-group"><label>Komut</label><input v-model="newJob.command" placeholder="ör: /usr/bin/php /home/user/script.php" /></div>
          <div class="preview">📋 <code>{{ formatCron(newJob) }} {{ newJob.command || 'komut...' }}</code></div>
        </div>
        <div class="modal-footer"><button class="btn-cancel" @click="showAdd=false">İptal</button><button class="btn-primary" @click="addJob">Ekle</button></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 14px; }
.btn-primary { padding: 10px 20px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-primary:hover { background: #1a4a7a; }

.loading { text-align: center; padding: 60px; color: #888; }
.empty { text-align: center; padding: 40px; background: white; border-radius: 12px; color: #888; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }

.cron-list { display: flex; flex-direction: column; gap: 6px; }
.cron-row { display: flex; align-items: center; gap: 12px; padding: 12px 16px; background: white; border-radius: 8px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.cron-expr { font-family: 'Consolas',monospace; font-size: 13px; background: #f0f0f0; padding: 4px 10px; border-radius: 4px; white-space: nowrap; }
.cron-cmd { flex: 1; font-size: 13px; color: #333; font-family: monospace; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cron-user { font-size: 12px; color: #888; background: #f0f4ff; padding: 2px 8px; border-radius: 4px; }
.btn-del-sm { background: none; border: none; font-size: 16px; cursor: pointer; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; max-width: 550px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; cursor: pointer; }

.preset-list { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 16px; }
.preset-btn { padding: 6px 12px; background: #f0f4ff; border: 1px solid #d0d8f0; border-radius: 6px; font-size: 12px; cursor: pointer; }
.preset-btn:hover { background: #e0e8ff; }

.cron-inputs { display: grid; grid-template-columns: repeat(5, 1fr); gap: 8px; margin-bottom: 16px; }
.ci label { display: block; font-size: 11px; color: #888; margin-bottom: 4px; text-transform: uppercase; }
.ci input { width: 100%; padding: 8px; border: 2px solid #e0e0e0; border-radius: 6px; font-size: 14px; font-family: monospace; text-align: center; }
.ci input:focus { outline: none; border-color: #0f3460; }

.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 14px; font-weight: 600; margin-bottom: 6px; }
.form-group input { width: 100%; padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; font-family: monospace; }
.form-group input:focus { outline: none; border-color: #0f3460; }

.preview { padding: 10px 14px; background: #f8f9fa; border-radius: 8px; font-size: 13px; }
.preview code { font-family: 'Consolas',monospace; color: #0f3460; }
</style>
