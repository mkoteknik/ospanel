<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface Database {
  id: number; name: string; username: string; charset: string; status: string; created_at: string
}

const databases = ref<Database[]>([])
const loading = ref(true)
const showCreate = ref(false)
const newDB = ref({ name: '', username: '', password: '', charset: 'utf8mb4' })
const creating = ref(false)

const charsets = [
  { label: 'utf8mb4 (Önerilen)', value: 'utf8mb4' },
  { label: 'utf8', value: 'utf8' },
  { label: 'latin1', value: 'latin1' },
  { label: 'latin5 (Türkçe)', value: 'latin5' },
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
  if (!confirm(`${name} veritabanı silinecek. Emin misiniz?`)) return
  try { await api.delete(`/api/v1/databases/${id}`); await loadDBs() } catch { }
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text)
}

onMounted(loadDBs)
</script>

<template>
  <div class="db-page">
    <div class="page-header">
      <div>
        <h2>🗄️ Veritabanı Yönetimi</h2>
        <p class="page-desc">MySQL/MariaDB veritabanları oluşturun ve yönetin.</p>
      </div>
      <button class="btn-add" @click="showCreate = true">+ Yeni Veritabanı</button>
    </div>

    <div v-if="loading" class="loading">Yükleniyor...</div>

    <div v-else-if="databases.length === 0" class="empty-state">
      <div class="empty-icon">🗄️</div>
      <h3>Henüz veritabanı yok</h3>
      <p>İlk veritabanınızı oluşturun.</p>
      <button class="btn-add" @click="showCreate = true">+ Veritabanı Oluştur</button>
    </div>

    <div v-else class="db-grid">
      <div v-for="db in databases" :key="db.id" class="db-card">
        <div class="db-name">🗄️ {{ db.name }}</div>
        <div class="db-info">
          <div class="db-row"><span>Kullanıcı:</span> <code>{{ db.username }}</code></div>
          <div class="db-row"><span>Charset:</span> {{ db.charset }}</div>
          <div class="db-row"><span>Durum:</span> <span class="badge badge-active">Aktif</span></div>
        </div>
        <div class="db-actions">
          <button class="btn-sm" @click="copyToClipboard(db.name)">📋 Kopyala</button>
          <a
            :href="'http://localhost:7080/phpmyadmin/index.php?db=' + db.name"
            target="_blank"
            class="btn-sm"
          >🔗 phpMyAdmin</a>
          <button class="btn-sm btn-danger" @click="deleteDB(db.id, db.name)">🗑️</button>
        </div>
      </div>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
      <div class="modal">
        <div class="modal-header">
          <h3>+ Yeni Veritabanı</h3>
          <button class="modal-close" @click="showCreate = false">✕</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>Veritabanı Adı</label>
            <input v-model="newDB.name" type="text" placeholder="orn: wordpress_db" />
          </div>
          <div class="form-group">
            <label>Kullanıcı Adı</label>
            <input v-model="newDB.username" type="text" placeholder="orn: wp_user" />
          </div>
          <div class="form-group">
            <label>Şifre</label>
            <input v-model="newDB.password" type="text" placeholder="Güçlü bir şifre" />
          </div>
          <div class="form-group">
            <label>Karakter Seti</label>
            <select v-model="newDB.charset">
              <option v-for="c in charsets" :key="c.value" :value="c.value">{{ c.label }}</option>
            </select>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-cancel" @click="showCreate = false">İptal</button>
          <button class="btn-add" :disabled="creating" @click="createDB">
            {{ creating ? 'Oluşturuluyor...' : 'Oluştur' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.db-page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { margin: 0; }
.page-desc { color: #888; margin: 4px 0 0; font-size: 14px; }

.btn-add { padding: 10px 20px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-add:hover { background: #1a4a7a; }
.btn-add:disabled { background: #999; cursor: not-allowed; }

.loading { text-align: center; padding: 60px; color: #888; }
.empty-state { text-align: center; padding: 60px 20px; background: white; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-state h3 { margin: 0 0 8px; }
.empty-state p { color: #888; margin: 0 0 20px; }

.db-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 16px; }
.db-card { background: white; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); padding: 20px; }
.db-name { font-size: 18px; font-weight: 700; margin-bottom: 12px; }
.db-info { margin-bottom: 16px; }
.db-row { font-size: 14px; padding: 4px 0; color: #555; }
.db-row span { color: #888; font-size: 12px; text-transform: uppercase; }
.db-row code { background: #f0f0f0; padding: 2px 6px; border-radius: 4px; font-size: 13px; }

.db-actions { display: flex; gap: 8px; padding-top: 12px; border-top: 1px solid #f0f0f0; }
.btn-sm { flex: 1; padding: 8px; border: 1px solid #e0e0e0; background: white; border-radius: 6px; font-size: 13px; cursor: pointer; text-align: center; text-decoration: none; color: #333; }
.btn-sm:hover { background: #f5f5f5; }
.btn-danger { color: #c0392b; border-color: #f0d0d0; }
.btn-danger:hover { background: #fff0f0; }

.badge { padding: 4px 10px; border-radius: 20px; font-size: 12px; font-weight: 600; }
.badge-active { background: #d4edda; color: #155724; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; max-width: 480px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; cursor: pointer; }

.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 14px; font-weight: 600; margin-bottom: 6px; }
.form-group input, .form-group select { width: 100%; padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; }
.form-group input:focus, .form-group select:focus { outline: none; border-color: #0f3460; }
</style>
