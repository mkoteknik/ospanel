<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { api } from '@/api/client'

const { t } = useI18n()

const olsInfo = ref<any>({})
const loading = ref(true)
const actionLoading = ref('')
const showPasswordModal = ref(false)
const newPass = ref('')
const newPassConfirm = ref('')

async function loadStatus() {
  loading.value = true
  try {
    const [infoRes] = await Promise.all([
      api.get('/api/v1/ols/info'),
    ])
    olsInfo.value = infoRes.data
  } catch { }
  finally { loading.value = false }
}

async function openWebAdmin() {
  try {
    const res = await api.get('/api/v1/ols/info')
    if (res.data.direct_url) {
      window.open(res.data.direct_url, '_blank')
    } else {
      window.open(res.data.ols_admin_url || 'http://' + location.hostname + ':7080', '_blank')
    }
  } catch { }
}

async function changePassword() {
  if (!newPass.value || newPass.value.length < 6) {
    alert(t('ols.passwordMin'))
    return
  }
  if (newPass.value !== newPassConfirm.value) {
    alert(t('ols.passwordMismatch'))
    return
  }
  actionLoading.value = 'password'
  try {
    await api.put('/api/v1/ols/password', { new_password: newPass.value })
    showPasswordModal.value = false
    newPass.value = ''
    newPassConfirm.value = ''
    alert(t('ols.passwordChanged'))
    await loadStatus()
  } catch { }
  finally { actionLoading.value = '' }
}

onMounted(loadStatus)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>{{ t('ols.title') }}</h2>
        <p>{{ t('ols.subtitle') }}</p>
      </div>
      <div class="header-actions">
        <button class="btn-refresh" @click="loadStatus">🔄 {{ t('ols.refresh') }}</button>
        <button class="btn-primary" @click="openWebAdmin">🖥️ {{ t('ols.webadmin') }}</button>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>{{ t('ols.checking') }}</p>
    </div>

    <div v-else>
      <!-- Üst Kartlar -->
      <div class="top-grid">
        <div class="info-card" :class="olsInfo.has_password ? 'ok' : 'warn'">
          <span class="ic-icon">🔗</span>
          <div>
            <span class="ic-val">{{ olsInfo.has_password ? t('ols.connected') : t('ols.notConnected') }}</span>
            <span class="ic-lbl">{{ t('ols.connection') }}</span>
          </div>
        </div>
        <div class="info-card">
          <span class="ic-icon">👤</span>
          <div>
            <span class="ic-val">{{ olsInfo.username || 'admin' }}</span>
            <span class="ic-lbl">{{ t('ols.adminUser') }}</span>
          </div>
        </div>
        <div class="info-card">
          <span class="ic-icon">🌐</span>
          <div>
            <span class="ic-val">{{ olsInfo.ols_admin_url || 'localhost:7080' }}</span>
            <span class="ic-lbl">{{ t('ols.adminUrl') }}</span>
          </div>
        </div>
      </div>

      <!-- Hızlı İşlemler -->
      <div class="section">
        <h3>⚡ {{ t('ols.quick') }}</h3>
        <div class="quick-actions">
          <button class="qa-card" @click="openWebAdmin">
            <span class="qa-icon">🖥️</span>
            <span class="qa-title">{{ t('ols.webadmin') }}</span>
            <span class="qa-desc">{{ t('ols.webadminDesc') }}</span>
          </button>
          <button class="qa-card" @click="showPasswordModal = true">
            <span class="qa-icon">🔑</span>
            <span class="qa-title">{{ t('ols.changePassword') }}</span>
            <span class="qa-desc">{{ t('ols.changePasswordDesc') }}</span>
          </button>
          <router-link to="/domains" class="qa-card">
            <span class="qa-icon">🌐</span>
            <span class="qa-title">{{ t('ols.domainMgmt') }}</span>
            <span class="qa-desc">{{ t('ols.domainMgmtDesc') }}</span>
          </router-link>
          <router-link to="/logs" class="qa-card">
            <span class="qa-icon">📋</span>
            <span class="qa-title">{{ t('ols.errorLogs') }}</span>
            <span class="qa-desc">{{ t('ols.errorLogsDesc') }}</span>
          </router-link>
        </div>
      </div>

      <!-- PHP Handler -->
      <div class="section">
        <h3>{{ t('ols.phpVersions') }}</h3>
        <div class="php-grid">
          <div class="php-card">
            <span class="php-ver">8.2</span>
            <span class="php-path">/usr/local/lsws/lsphp82/bin/lsphp</span>
          </div>
          <div class="php-card active">
            <span class="php-ver">8.3</span>
            <span class="php-path">/usr/local/lsws/lsphp83/bin/lsphp</span>
            <span class="php-default">{{ t('ols.default') }}</span>
          </div>
          <div class="php-card">
            <span class="php-ver">8.4</span>
            <span class="php-path">/usr/local/lsws/lsphp84/bin/lsphp</span>
          </div>
        </div>
      </div>

      <!-- Yapılandırma -->
      <div class="section">
        <h3>⚙️ {{ t('ols.important') }}</h3>
        <div class="config-grid">
          <div class="config-item">
            <div class="cfg-left">
              <span class="cfg-label">{{ t('ols.gzip') }}</span>
              <span class="cfg-desc">{{ t('ols.gzipDesc') }}</span>
            </div>
            <span class="cfg-val on">{{ t('ols.active') }}</span>
          </div>
          <div class="config-item">
            <div class="cfg-left">
              <span class="cfg-label">{{ t('ols.brotli') }}</span>
              <span class="cfg-desc">{{ t('ols.brotliDesc') }}</span>
            </div>
            <span class="cfg-val on">{{ t('ols.active') }}</span>
          </div>
          <div class="config-item">
            <div class="cfg-left">
              <span class="cfg-label">{{ t('ols.htaccess') }}</span>
              <span class="cfg-desc">{{ t('ols.htaccessDesc') }}</span>
            </div>
            <span class="cfg-val on">{{ t('ols.active') }}</span>
          </div>
          <div class="config-item">
            <div class="cfg-left">
              <span class="cfg-label">{{ t('ols.watchdog') }}</span>
              <span class="cfg-desc">{{ t('ols.watchdogDesc') }}</span>
            </div>
            <span class="cfg-val on">{{ t('ols.active') }}</span>
          </div>
          <div class="config-item">
            <div class="cfg-left">
              <span class="cfg-label">{{ t('ols.letsencrypt') }}</span>
              <span class="cfg-desc">{{ t('ols.letsencryptDesc') }}</span>
            </div>
            <span class="cfg-val on">{{ t('ols.active') }}</span>
          </div>
          <div class="config-item">
            <div class="cfg-left">
              <span class="cfg-label">{{ t('ols.fail2ban') }}</span>
              <span class="cfg-desc">{{ t('ols.fail2banDesc') }}</span>
            </div>
            <span class="cfg-val on">{{ t('ols.active') }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Şifre Modal -->
    <div v-if="showPasswordModal" class="modal-overlay" @click.self="showPasswordModal = false">
      <div class="modal">
        <div class="modal-header"><h3>🔑 {{ t('ols.passwordModal') }}</h3><button class="modal-close" @click="showPasswordModal = false">✕</button></div>
        <div class="modal-body">
          <div class="form-group"><label>{{ t('ols.newPassword') }}</label><input v-model="newPass" type="password" :placeholder="t('ols.passwordHint')" /></div>
          <div class="form-group"><label>{{ t('ols.confirmPassword') }}</label><input v-model="newPassConfirm" type="password" :placeholder="t('ols.passwordConfirmHint')" /></div>
        </div>
        <div class="modal-footer"><button class="btn-cancel" @click="showPasswordModal = false">{{ t('common.cancel') }}</button><button class="btn-primary" :disabled="actionLoading === 'password'" @click="changePassword">{{ t('common.save') }}</button></div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 28px; }
.page-header h2 { margin: 0; font-size: 22px; color: #1a1a2e; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 13px; }
.header-actions { display: flex; gap: 8px; }
.btn-primary { padding: 10px 20px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-primary:hover { background: #1a4a7a; }
.btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
.btn-refresh { padding: 8px 16px; background: white; border: 1px solid #e0e0e0; border-radius: 8px; font-size: 13px; cursor: pointer; }

.loading-state { text-align: center; padding: 80px 0; }
.spinner { width: 36px; height: 36px; border: 3px solid #e0e0e0; border-top-color: #0f3460; border-radius: 50%; animation: spin 0.8s linear infinite; margin: 0 auto 16px; }
@keyframes spin { to { transform: rotate(360deg); } }

.top-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 12px; margin-bottom: 28px; }
.info-card { background: white; border-radius: 12px; padding: 16px 20px; display: flex; align-items: center; gap: 14px; border: 1px solid #f0f0f0; }
.info-card.ok { border-left: 3px solid #27ae60; }
.info-card.warn { border-left: 3px solid #f39c12; }
.ic-icon { font-size: 24px; }
.ic-val { display: block; font-weight: 700; font-size: 14px; color: #1a1a2e; }
.ic-lbl { font-size: 11px; color: #888; }

.section { margin-bottom: 28px; }
.section h3 { margin: 0 0 14px; font-size: 16px; color: #1a1a2e; }

.quick-actions { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 10px; }
.qa-card { background: white; border: 1px solid #f0f0f0; border-radius: 12px; padding: 18px; cursor: pointer; text-decoration: none; display: flex; flex-direction: column; gap: 4px; transition: all 0.2s; text-align: left; color: inherit; font-family: inherit; font-size: inherit; }
.qa-card:hover { border-color: #0f3460; box-shadow: 0 4px 12px rgba(0,0,0,0.06); transform: translateY(-1px); }
.qa-icon { font-size: 24px; }
.qa-title { font-weight: 700; font-size: 14px; color: #1a1a2e; }
.qa-desc { font-size: 12px; color: #888; }

.php-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 10px; }
.php-card { background: white; border: 1px solid #f0f0f0; border-radius: 12px; padding: 16px; position: relative; }
.php-card.active { border-color: #0f3460; background: #f8faff; }
.php-ver { display: block; font-size: 20px; font-weight: 700; color: #1a1a2e; margin-bottom: 4px; }
.php-path { font-size: 11px; color: #888; font-family: monospace; }
.php-default { position: absolute; top: 12px; right: 12px; font-size: 10px; background: #0f3460; color: white; padding: 2px 8px; border-radius: 10px; }

.config-grid { display: flex; flex-direction: column; gap: 8px; }
.config-item { background: white; border: 1px solid #f0f0f0; border-radius: 10px; padding: 14px 18px; display: flex; justify-content: space-between; align-items: center; }
.cfg-label { display: block; font-weight: 600; font-size: 13px; color: #1a1a2e; }
.cfg-desc { font-size: 11px; color: #888; }
.cfg-val { padding: 3px 10px; border-radius: 20px; font-size: 11px; font-weight: 600; }
.cfg-val.on { background: #d4edda; color: #155724; }
.cfg-val.off { background: #f8d7da; color: #721c24; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 14px; width: 90%; max-width: 440px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
.modal-header { display: flex; justify-content: space-between; align-items: center; padding: 20px 24px; border-bottom: 1px solid #f0f0f0; }
.modal-header h3 { margin: 0; }
.modal-close { background: none; border: none; font-size: 20px; cursor: pointer; color: #888; }
.modal-body { padding: 24px; }
.modal-footer { display: flex; justify-content: flex-end; gap: 8px; padding: 16px 24px; border-top: 1px solid #f0f0f0; }
.btn-cancel { padding: 10px 20px; background: #f0f0f0; border: none; border-radius: 8px; cursor: pointer; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; font-size: 13px; font-weight: 600; margin-bottom: 6px; }
.form-group input { width: 100%; padding: 10px 12px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; }
.form-group input:focus { outline: none; border-color: #0f3460; }
</style>
