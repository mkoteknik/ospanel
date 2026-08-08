<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { api } from '@/api/client'

const router = useRouter()
const { t } = useI18n()

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
  { label: t('auto.f3936a'), value: '8.4' },
  { label: t('auto.b36db9'), value: '8.3' },
  { label: 'PHP 8.2', value: '8.2' },
]

async function loadDomains() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get('/api/v1/domains')
    domains.value = res.data.domains || []
  } catch {
    error.value = t('domains.loadFailed')
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
    error.value = err.response?.data?.error || t('domains.createFailed')
  } finally {
    creating.value = false
  }
}

async function deleteDomain(domain: Domain) {
  if (!confirm(t('domains.deleteConfirm', { domain: domain.domain }))) return
  deleting.value = domain.id
  try {
    await api.delete(`/api/v1/domains/${domain.id}`)
    await loadDomains()
  } catch {
    error.value = t('domains.deleteFailed')
  } finally {
    deleting.value = null
  }
}

function viewDetail(domain: Domain) {
  router.push('/domains/' + domain.id)
}

function getStatusBadge(status: string) {
  const map: Record<string, { text: string; class: string }> = {
    active: { text: t('domains.status.active'), class: 'badge-active' },
    inactive: { text: t('domains.status.inactive'), class: 'badge-inactive' },
    error: { text: t('domains.status.error'), class: 'badge-error' },
  }
  return map[status] || { text: status, class: '' }
}

onMounted(loadDomains)
</script>

<template>
  <div class="domains-page">
    <div class="page-header">
      <div>
        <h2>{{ t('domains.title') }}</h2>
        <p class="page-desc">{{ t('domains.desc') }}</p>
      </div>
      <button class="btn-add" @click="showCreate = true">+ {{ t('domains.newDomain') }}</button>
    </div>

    <!-- Error -->
    <div v-if="error" class="alert-error">{{ error }}</div>

    <!-- Loading -->
    <div v-if="loading" class="loading">{{ t('domains.loading') }}</div>

    <!-- Empty State -->
    <div v-else-if="domains.length === 0" class="empty-state">
      <div class="empty-icon">🌐</div>
      <h3>{{ t('domains.emptyTitle') }}</h3>
      <p>{{ t('domains.emptyDesc') }}</p>
      <button class="btn-add" @click="showCreate = true">+ {{ t('domains.firstDomain') }}</button>
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
            <span class="info-label">{{ t('domains.info.php') }}</span>
            <span class="info-value">{{ domain.php_version }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">{{ t('domains.info.ssl') }}</span>
            <span class="info-value">{{ domain.ssl_enabled ? t('domains.info.sslActive') : t('domains.info.sslNone') }}</span>
          </div>
          <div class="info-row">
            <span class="info-label">{{ t('domains.info.https') }}</span>
            <span class="info-value">{{ domain.force_https ? t('domains.info.httpsRequired') : t('domains.info.httpsOff') }}</span>
          </div>
        </div>
        <div class="card-actions">
          <button class="btn-sm" @click.stop="viewDetail(domain)">{{ t('domains.detail') }}</button>
          <button
            class="btn-sm btn-danger"
            :disabled="deleting === domain.id"
            @click.stop="deleteDomain(domain)"
          >
            {{ deleting === domain.id ? t('domains.deleting') : t('common.delete') }}
          </button>
        </div>
      </div>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate = false">
      <div class="modal">
        <div class="modal-header">
          <h3>+ {{ t('domains.newDomain') }}</h3>
          <button class="modal-close" @click="showCreate = false">✕</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>{{ t('domains.domainName') }}</label>
            <input
              v-model="newDomain.domain"
              type="text"
              :placeholder="t('domains.domainPlaceholder')"
              @keyup.enter="createDomain"
            />
            <span class="form-hint">{{ t('domains.domainHint') }}</span>
          </div>
          <div class="form-group">
            <label>{{ t('domains.phpVersion') }}</label>
            <select v-model="newDomain.php_version">
              <option v-for="v in phpVersions" :key="v.value" :value="v.value">
                {{ v.label }}
              </option>
            </select>
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn-cancel" @click="showCreate = false">{{ t('common.cancel') }}</button>
          <button class="btn-add" :disabled="creating || !newDomain.domain" @click="createDomain">
            {{ creating ? t('domains.creating') : t('domains.create') }}
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

.page-header h2 { margin: 0; font-size: 20px; font-weight: 720; letter-spacing: -0.02em; color: var(--aura-text); }
.page-desc { color: var(--aura-text-muted); margin: 4px 0 0; font-size: 13px; }

.btn-add {
  padding: 10px 20px;
  background: var(--aura-accent);
  color: var(--aura-accent-text);
  border: 1px solid var(--aura-accent);
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}
.btn-add:hover { background: var(--aura-accent-hover); border-color: var(--aura-accent-hover); }
.btn-add:disabled { background: var(--aura-bg-subtle); color: var(--aura-text-faint); border-color: var(--aura-border); cursor: not-allowed; }

.alert-error {
  background: color-mix(in srgb, var(--aura-danger) 10%, var(--aura-surface));
  color: var(--aura-danger);
  border: 1px solid color-mix(in srgb, var(--aura-danger) 18%, transparent);
  padding: 12px 16px;
  border-radius: 10px;
  margin-bottom: 16px;
  font-size: 14px;
}

.loading {
  text-align: center;
  padding: 60px;
  color: var(--aura-text-faint);
  font-size: 16px;
}

.empty-state {
  text-align: center;
  padding: 60px 20px;
  background: var(--aura-surface);
  border: 1px solid var(--aura-border);
  border-radius: 12px;
  box-shadow: var(--aura-shadow-sm);
}

.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-state h3 { margin: 0 0 8px; color: var(--aura-text); }
.empty-state p { color: var(--aura-text-muted); margin: 0 0 20px; }

.domain-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.domain-card {
  background: var(--aura-surface);
  border-radius: 12px;
  border: 1px solid var(--aura-border);
  box-shadow: var(--aura-shadow-sm);
  padding: 20px;
  cursor: pointer;
  transition: all 0.2s;
}

.domain-card:hover {
  border-color: color-mix(in srgb, var(--aura-accent) 18%, var(--aura-border));
  box-shadow: var(--aura-shadow);
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
  color: var(--aura-text);
}

.badge {
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}

.badge-active { background: var(--aura-accent-soft); color: var(--aura-accent); border: 1px solid color-mix(in srgb, var(--aura-accent) 18%, transparent); }
.badge-inactive { background: var(--aura-bg-subtle); color: var(--aura-text-faint); border: 1px solid var(--aura-border); }
.badge-error { background: color-mix(in srgb, var(--aura-danger) 10%, var(--aura-surface)); color: var(--aura-danger); border: 1px solid color-mix(in srgb, var(--aura-danger) 18%, transparent); }

.card-info { margin-bottom: 16px; }

.info-row {
  display: flex;
  justify-content: space-between;
  padding: 6px 0;
  border-bottom: 1px solid var(--aura-border);
  font-size: 14px;
}

.info-label { color: var(--aura-text-faint); font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; }
.info-value { color: var(--aura-text); font-weight: 500; }

.card-actions {
  display: flex;
  gap: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--aura-border);
}

.btn-sm {
  flex: 1;
  padding: 8px;
  border: 1px solid var(--aura-border);
  background: var(--aura-surface);
  color: var(--aura-text);
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.15s;
  text-align: center;
}

.btn-sm:hover { background: var(--aura-bg-subtle); }

.btn-danger {
  color: var(--aura-danger);
  border-color: color-mix(in srgb, var(--aura-danger) 18%, var(--aura-border));
}
.btn-danger:hover { background: color-mix(in srgb, var(--aura-danger) 8%, var(--aura-surface)); }

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
  background: var(--aura-surface);
  border: 1px solid var(--aura-border);
  border-radius: 12px;
  width: 90%;
  max-width: 480px;
  box-shadow: var(--aura-shadow-lg);
}

.modal-lg { max-width: 600px; }

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--aura-border);
}

.modal-header h3 { margin: 0; color: var(--aura-text); }

.modal-close {
  background: none;
  border: none;
  font-size: 20px;
  cursor: pointer;
  color: var(--aura-text-faint);
  padding: 4px 8px;
}
.modal-close:hover { color: var(--aura-text); }

.modal-body { padding: 24px; }

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 16px 24px;
  border-top: 1px solid var(--aura-border);
  background: var(--aura-bg-subtle);
  border-radius: 0 0 12px 12px;
}

.btn-cancel {
  padding: 10px 20px;
  background: var(--aura-bg-subtle);
  color: var(--aura-text);
  border: 1px solid var(--aura-border);
  border-radius: 8px;
  font-size: 14px;
  cursor: pointer;
}

.btn-cancel:hover { background: var(--aura-surface-hover); }

.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 13px; font-weight: 600; margin-bottom: 6px; color: var(--aura-text); }
.form-group input, .form-group select {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--aura-border);
  border-radius: 10px;
  font-size: 14px;
  background: var(--aura-surface);
  color: var(--aura-text);
}
.form-group input:focus, .form-group select:focus {
  outline: none;
  border-color: var(--aura-accent);
  box-shadow: 0 0 0 3px var(--aura-accent-ring);
}
.form-hint { display: block; font-size: 12px; color: var(--aura-text-faint); margin-top: 4px; }

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
