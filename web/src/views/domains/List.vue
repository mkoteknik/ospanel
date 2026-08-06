<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api/client'

const router = useRouter()

interface Domain {
  id: number
  domain: string
  document_root: string
  php_version: string
  ssl_enabled: boolean
  force_https: boolean
  status: string
  created_at: string
}

const domains = ref<Domain[]>([])
const loading = ref(true)
const error = ref('')
const showCreate = ref(false)
const newDomain = ref({ domain: '', php_version: '8.3' })
const creating = ref(false)
const deleting = ref<number | null>(null)

const phpVersions = [
  { label: 'PHP 8.4', value: '8.4' },
  { label: 'PHP 8.3 (Önerilen)', value: '8.3' },
  { label: 'PHP 8.2', value: '8.2' },
  { label: 'PHP 8.1', value: '8.1' },
  { label: 'PHP 8.0', value: '8.0' },
  { label: 'PHP 7.4', value: '7.4' },
]

async function loadDomains() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get('/api/v1/domains')
    domains.value = res.data.domains || []
  } catch {
    error.value = 'Domainler yüklenemedi'
    domains.value = []
  } finally {
    loading.value = false
  }
}

async function createDomain() {
  if (!newDomain.value.domain) return
  creating.value = true
  try {
    await api.post('/api/v1/domains', newDomain.value)
    showCreate.value = false
    newDomain.value = { domain: '', php_version: '8.3' }
    await loadDomains()
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Domain oluşturulamadı'
  } finally {
    creating.value = false
  }
}

async function deleteDomain(domain: Domain) {
  if (!confirm(`${domain.domain} silinecek. Emin misiniz?`)) return
  deleting.value = domain.id
  try {
    await api.delete(`/api/v1/domains/${domain.id}`)
    await loadDomains()
  } catch {
    error.value = 'Domain silinemedi'
  } finally {
    deleting.value = null
  }
}

function viewDetail(domain: Domain) {
  router.push('/domains/' + domain.id)
}

function getStatusBadge(status: string) {
  const map: Record<string, { text: string; class: string }> = {
    active: { text: 'Aktif', class: 'badge-active' },
    inactive: { text: 'Pasif', class: 'badge-inactive' },
    error: { text: 'Hata', class: 'badge-error' },
  }
  return map[status] || { text: status, class: '' }
}

onMounted(loadDomains)
</script>

<template>
  <div class="domains-page">
    <div class="page-header">
      <div>
        <h2>🌐 Domain Yönetimi</h2>
        <p class="page-desc">Web sitelerinizi yönetin, PHP sürümlerini değiştirin, SSL kurun.</p>
      </div>
      <button class="btn-add" @click="showCreate = true">+ Yeni Domain</button>
    </div>

    <!-- Error -->
    <div v-if="error" class="alert-error">{{ error }}</div>

    <!-- Loading -->
    <div v-if="loading" class="loading">Domainler yükleniyor...</div>

    <!-- Empty State -->
    <div v-else-if="domains.length === 0" class="empty-state">
      <div class="empty-icon">🌐</div>
      <h3>Henüz domain eklenmedi</h3>
      <p>İlk web sitenizi oluşturmak için "Yeni Domain" butonuna tıklayın.</p>
      <button class="btn-add" @click="showCreate = true">+ İlk Domaini Ekle</button>
    </div>

    <!-- Domain List -->
    <div v-else class="domain-grid">
      <div
        v-for="domain in domains"
        :key="domain.id"
        class="domain-card"
        @click="viewDetail(domain)"
      >
        <div class="card-top">
          <div class="domain-name">{{ domain.domain }}</div>
          <span :class="'badge ' + getStatusBadge(domain.status).class">
            {{ getStatusBadge(domain.status).text }}
          </span>
        </div>
        <div class="card-info">
          <div class="info-row">
            <span class="info-label">PHP</span>
            <span class="info-value">{{ domain.php_version }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">SSL</span>
            <span class="info-value">{{ domain.ssl_enabled ? '🔒 Aktif' : '🔓 Yok' }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">HTTPS</span>
            <span class="info-value">{{ domain.force_https ? '✅ Zorunlu' : '❌ Kapalı' }}</span>
          </div>
        </div>
        <div class="card-actions">
          <button class="btn-sm" @click.stop="viewDetail(domain)">🔍 Detay</button>
          <button
            class="btn-sm btn-danger"
            :disabled="deleting === domain.id"
            @click.stop="deleteDomain(domain)"
          >
            {{ deleting === domain.id ? 'Siliniyor...' : '🗑️ Sil' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
      <div class="modal">
        <div class="modal-header">
          <h3>+ Yeni Domain</h3>
          <button class="modal-close" @click="showCreate = false">✕</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>Domain Adı</label>
            <input
              v-model="newDomain.domain"
              type="text"
              placeholder="orn: site.com"
              @keyup.enter="createDomain"
            />
            <span class="form-hint">www kullanmadan yazın, örn: site.com</span>
          </div>
          <div class="form-group">
            <label>PHP Sürümü</label>
            <select v-model="newDomain.php_version">
              <option v-for="v in phpVersions" :key="v.value" :value="v.value">
                {{ v.label }}
              </option>
            </select>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-cancel" @click="showCreate = false">İptal</button>
          <button class="btn-add" :disabled="creating || !newDomain.domain" @click="createDomain">
            {{ creating ? 'Oluşturuluyor...' : '✅ Oluştur' }}
          </button>
        </div>
      </div>
    </div>

  </div>
</template>

<style scoped>
.domains-page { width: 100%; }

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.page-header h2 { margin: 0; }
.page-desc { color: #888; margin: 4px 0 0; font-size: 14px; }

.btn-add {
  padding: 10px 20px;
  background: #0f3460;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
}
.btn-add:hover { background: #1a4a7a; }
.btn-add:disabled { background: #999; cursor: not-allowed; }

.alert-error {
  background: #ffe0e0;
  color: #c0392b;
  padding: 12px 16px;
  border-radius: 8px;
  margin-bottom: 16px;
  font-size: 14px;
}

.loading {
  text-align: center;
  padding: 60px;
  color: #888;
  font-size: 16px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}

.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-state h3 { margin: 0 0 8px; color: #1a1a2e; }
.empty-state p { color: #888; margin: 0 0 20px; }

.domain-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.domain-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
  padding: 20px;
  cursor: pointer;
  transition: all 0.2s;
  border: 2px solid transparent;
}

.domain-card:hover {
  border-color: #0f3460;
  box-shadow: 0 4px 16px rgba(15, 52, 96, 0.12);
}

.card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.domain-name {
  font-size: 18px;
  font-weight: 700;
  color: #1a1a2e;
}

.badge {
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.badge-active { background: #d4edda; color: #155724; }
.badge-inactive { background: #f8f9fa; color: #666; }
.badge-error { background: #ffe0e0; color: #c0392b; }

.card-info { margin-bottom: 16px; }

.info-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
  border-bottom: 1px solid #f0f0f0;
  font-size: 14px;
}

.info-label { color: #888; }
.info-value { color: #333; font-weight: 500; }

.card-actions {
  display: flex;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

.btn-sm {
  flex: 1;
  padding: 8px;
  border: 1px solid #e0e0e0;
  background: white;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
  text-align: center;
}

.btn-sm:hover { background: #f5f5f5; }

.btn-danger {
  color: #c0392b;
  border-color: #f0d0d0;
}
.btn-danger:hover { background: #fff0f0; }

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: white;
  border-radius: 12px;
  width: 90%;
  max-width: 480px;
  box-shadow: 0 20px 60px rgba(0,0,0,0.3);
}

.modal-lg { max-width: 600px; }

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid #f0f0f0;
}

.modal-header h3 { margin: 0; }

.modal-close {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: #888;
  padding: 4px 8px;
}
.modal-close:hover { color: #333; }

.modal-body { padding: 24px; }

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 16px 24px;
  border-top: 1px solid #f0f0f0;
}

.btn-cancel {
  padding: 10px 20px;
  background: #f0f0f0;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
}

.btn-cancel:hover { background: #e0e0e0; }

.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 14px; font-weight: 600; margin-bottom: 6px; color: #333; }
.form-group input, .form-group select {
  width: 100%;
  padding: 10px 12px;
  border: 2px solid #e0e0e0;
  border-radius: 8px;
  font-size: 14px;
  background: white;
}
.form-group input:focus, .form-group select:focus {
  outline: none;
  border-color: #0f3460;
}
.form-hint { display: block; font-size: 12px; color: #888; margin-top: 4px; }

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.detail-item { display: flex; flex-direction: column; gap: 4px; }
.detail-label { font-size: 12px; color: #888; font-weight: 600; text-transform: uppercase; }
.detail-value { font-size: 15px; color: #333; }
.link { color: #0f3460; text-decoration: none; }
.link:hover { text-decoration: underline; }
</style>
