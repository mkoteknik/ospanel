<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

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
  if (!confirm('Bu yedekleme isi silinecek!')) return
  try { await api.delete('/api/v1/backups/' + id); await loadJobs() }
  catch { }
}

async function runBackup(id: number) {
  try {
    await api.post('/api/v1/backups/' + id + '/run')
    alert('Yedekleme başlatildi!')
  } catch { }
}

function statusClass(s: string) {
  return s === 'completed' ? 'status-ok' : s === 'failed' ? 'status-err' : 'status-pending'
}

onMounted(loadJobs)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>Yedekleme</h2>
        <p>Web sitelerinizi ve veritabanlarınizi guvenceye alin.</p>
      </div>
      <button class="btn-primary" @click="showAdd = true">+ Yedekleme Plani</button>
    </div>

    <div v-if="loading" class="loading">Yükleniyor...</div>

    <div v-else-if="jobs.length === 0" class="empty">
      <p>Henüz bir yedekleme plani yok.</p>
      <button class="btn-primary" @click="showAdd = true">Ilk Yedeklemeyi Oluştur</button>
    </div>

    <div v-else class="job-list">
      <div v-for="job in jobs" :key="job.id" class="job-card">
        <div class="job-info">
          <span class="job-type">{{ job.type === 'full' ? 'Tam' : job.type === 'database' ? 'Veritabanı' : job.type }}</span>
          <span class="job-schedule">{{ job.schedule || 'Manuel' }}</span>
          <span :class="'job-status ' + statusClass(job.status)">{{ job.status }}</span>
        </div>
        <div class="job-details">
          <small>Hedef: {{ job.destination }} | Saklama: {{ job.retention }} gun</small>
          <small v-if="job.last_run">Son calisma: {{ job.last_run }}</small>
        </div>
        <div class="job-actions">
          <button class="btn-sm" @click="runBackup(job.id)">Simdi Çalıştır</button>
          <button class="btn-sm-danger" @click="deleteJob(job.id)">Sil</button>
        </div>
      </div>
    </div>

    <div v-if="showAdd" class="modal-overlay" @click.self="showAdd=false">
      <div class="modal">
        <div class="modal-header"><h3>+ Yedekleme Plani</h3><button class="modal-close" @click="showAdd=false">X</button></div>
        <div class="modal-body">
          <div class="form-group"><label>Tur</label>
            <select v-model="newJob.type" class="sel">
              <option value="full">Tam Yedekleme</option>
              <option value="database">Sadece Veritabanı</option>
              <option value="incremental">Artimli</option>
            </select>
          </div>
          <div class="form-group"><label>Hedef</label>
            <select v-model="newJob.destination" class="sel">
              <option value="local">Lokal (/var/backups/ospanel)</option>
              <option value="s3">Amazon S3</option>
              <option value="ftp">FTP Sunucusu</option>
            </select>
          </div>
          <div class="form-group"><label>Zamanlama</label>
            <select v-model="newJob.schedule" class="sel">
              <option value="daily">Her gun (02:00)</option>
              <option value="12:00">Her gun 12:00</option>
              <option value="20:00">Her gun 20:00</option>
              <option value="">Manuel</option>
            </select>
          </div>
          <div class="form-group"><label>Saklama (gun)</label>
            <input v-model.number="newJob.retention" type="number" min="1" max="365" />
          </div>
        </div>
        <div class="modal-footer"><button class="btn-cancel" @click="showAdd=false">İptal</button><button class="btn-primary" @click="createJob">Oluştur</button></div>
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
.btn-sm { padding: 8px 14px; background: #0f3460; color: white; border: none; border-radius: 6px; font-size: 12px; cursor: pointer; }
.btn-sm-danger { padding: 8px 14px; background: white; color: #d32f2f; border: 1px solid #d32f2f; border-radius: 6px; font-size: 12px; cursor: pointer; }
.loading { text-align: center; padding: 60px; color: #888; }
.empty { text-align: center; padding: 60px; background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.empty p { color: #888; margin-bottom: 16px; }
.job-list { display: flex; flex-direction: column; gap: 12px; }
.job-card { background: white; border-radius: 12px; padding: 20px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); display: flex; justify-content: space-between; align-items: center; gap: 16px; }
.job-info { display: flex; align-items: center; gap: 12px; }
.job-type { font-weight: 700; font-size: 14px; color: #1a1a2e; }
.job-schedule { font-size: 12px; color: #888; background: #f0f0f0; padding: 2px 8px; border-radius: 4px; }
.job-status { font-size: 11px; padding: 3px 10px; border-radius: 20px; font-weight: 600; text-transform: uppercase; }
.status-ok { background: #d4edda; color: #155724; }
.status-err { background: #f8d7da; color: #721c24; }
.status-pending { background: #fff3cd; color: #856404; }
.job-details { display: flex; flex-direction: column; gap: 4px; }
.job-details small { color: #888; font-size: 12px; }
.job-actions { display: flex; gap: 8px; }
.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; max-width: 500px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; cursor: pointer; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 13px; font-weight: 600; margin-bottom: 6px; color: #333; }
.form-group input { width: 100%; padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; }
.form-group input:focus { outline: none; border-color: #0f3460; }
.sel { width: 100%; padding: 10px 14px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; background: white; cursor: pointer; }
.sel:focus { outline: none; border-color: #0f3460; }
</style>
