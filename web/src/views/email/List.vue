<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { api } from '@/api/client'

interface Domain { id: number; domain: string }
interface EmailAcc { id: number; email: string; quota: number; status: string; created_at: string }

const domains = ref<Domain[]>([])
const selectedDomain = ref('')
const emails = ref<EmailAcc[]>([])
const loading = ref(false)
const showCreate = ref(false)
const newEmail = ref({ local: '', password: '', quota: 1024 })
const actionLoading = ref('')

async function loadDomains() {
  try { const r = await api.get('/api/v1/domains'); domains.value = r.data.domains || []
    if (domains.value.length > 0 && !selectedDomain.value) selectedDomain.value = domains.value[0].domain
  } catch { }
}

async function loadEmails() {
  if (!selectedDomain.value) return
  loading.value = true
  try { const r = await api.get('/api/v1/emails?domain=' + selectedDomain.value)
    emails.value = r.data.emails || [] } catch { emails.value = [] }
  finally { loading.value = false }
}

async function createEmail() {
  if (!newEmail.value.local) return
  actionLoading.value = 'create'
  try {
    await api.post('/api/v1/emails', {
      email: newEmail.value.local + '@' + selectedDomain.value,
      password: newEmail.value.password,
      quota: newEmail.value.quota,
      domain: selectedDomain.value,
    })
    showCreate.value = false; newEmail.value = { local: '', password: '', quota: 1024 }; loadEmails()
  } catch { }
  finally { actionLoading.value = '' }
}

async function deleteEmail(id: number) {
  if (!confirm('Email hesabı silinecek!')) return
  actionLoading.value = 'del' + id
  try { await api.delete('/api/v1/emails/' + id); loadEmails() } catch { }
  finally { actionLoading.value = '' }
}

function openWebmail(email: string) {
  window.open('https://' + selectedDomain.value + '/webmail', '_blank')
}

function generatePassword() {
  const chars = 'abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789'
  let p = ''; for (let i = 0; i < 14; i++) p += chars[Math.floor(Math.random() * chars.length)]
  newEmail.value.password = p
}

watch(selectedDomain, loadEmails)
onMounted(loadDomains)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div><h2>📧 E-Posta Yönetimi</h2><p>Domainlerinize email hesapları oluşturun. Postfix + Dovecot altyapısı.</p></div>
      <button class="btn-primary" @click="showCreate = true" :disabled="!selectedDomain">+ Email Ekle</button>
    </div>

    <!-- Domain Selector -->
    <div class="selector-bar">
      <div class="sel-group"><label>Domain</label>
        <select v-model="selectedDomain"><option v-for="d in domains" :key="d.id" :value="d.domain">{{ d.domain }}</option></select>
      </div>
      <div class="sel-stats"><span>{{ emails.length }} hesap</span><button class="btn-sm" @click="loadEmails">🔄</button></div>
    </div>

    <div v-if="loading" class="loading">Yükleniyor...</div>

    <div v-else-if="emails.length === 0" class="empty">
      <div class="empty-icon">📧</div>
      <h3>Henüz email hesabı yok</h3>
      <p>{{ selectedDomain }} için ilk email hesabını oluşturun.</p>
      <button class="btn-primary" @click="showCreate = true">+ Email Oluştur</button>
    </div>

    <div v-else class="email-table-wrap">
      <div class="email-table">
        <div class="et-header"><span>Email</span><span>Kota</span><span>Durum</span><span>Oluşturma</span><span></span></div>
        <div v-for="e in emails" :key="e.id" class="et-row">
          <span class="et-email">📧 {{ e.email }}</span>
          <span>{{ (e.quota / 1024).toFixed(1) }} GB</span>
          <span><span class="badge badge-on">🟢 Aktif</span></span>
          <span class="et-date">{{ new Date(e.created_at).toLocaleDateString('tr-TR') }}</span>
          <span class="et-actions">
            <button class="btn-icon" @click="openWebmail(e.email)" title="Webmail">🌐</button>
            <button class="btn-icon" @click="deleteEmail(e.id)" :disabled="actionLoading === 'del'+e.id" title="Sil">🗑️</button>
          </span>
        </div>
      </div>
    </div>

    <!-- IMAP/POP3 Bilgi -->
    <div class="info-cards">
      <div class="info-card"><strong>📨 IMAP</strong><code>mail.{{ selectedDomain || 'site.com' }}</code><span>Port: 993 (SSL)</span></div>
      <div class="info-card"><strong>📬 POP3</strong><code>mail.{{ selectedDomain || 'site.com' }}</code><span>Port: 995 (SSL)</span></div>
      <div class="info-card"><strong>📤 SMTP</strong><code>mail.{{ selectedDomain || 'site.com' }}</code><span>Port: 587 (TLS)</span></div>
    </div>

    <!-- Create Modal -->
    <div v-if="showCreate" class="modal-overlay" @click.self="showCreate=false">
      <div class="modal">
        <div class="modal-header"><h3>+ Email Hesabı - @{{ selectedDomain }}</h3><button class="modal-close" @click="showCreate=false">✕</button></div>
        <div class="modal-body">
          <div class="input-suffix"><input v-model="newEmail.local" placeholder="kullanıcı" @keyup.enter="createEmail" /><span>@{{ selectedDomain }}</span></div>
          <div class="form-group"><label>Şifre</label>
            <div class="pw-row"><input v-model="newEmail.password" type="text" /><button class="btn-sm" @click="generatePassword">🎲 Üret</button></div>
          </div>
          <div class="form-group"><label>Kota (MB)</label><input v-model.number="newEmail.quota" type="number" min="100" max="102400" /></div>
        </div>
        <div class="modal-footer"><button class="btn-cancel" @click="showCreate=false">İptal</button><button class="btn-primary" :disabled="actionLoading==='create'" @click="createEmail">✅ Oluştur</button></div>
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
.btn-primary:disabled { background: #999; cursor: not-allowed; }
.btn-sm { padding: 8px 14px; background: white; border: 1px solid #ddd; border-radius: 6px; font-size: 13px; cursor: pointer; }
.btn-sm:hover { background: #f5f5f5; }

.selector-bar { display: flex; justify-content: space-between; align-items: flex-end; margin-bottom: 20px; gap: 16px; background: white; padding: 16px 20px; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.sel-group { flex: 1; max-width: 400px; }
.sel-group label { display: block; font-size: 12px; font-weight: 600; color: #888; text-transform: uppercase; margin-bottom: 6px; }
.sel-group select { width: 100%; padding: 10px 14px; border: 2px solid #e0e0e0; border-radius: 8px; font-size: 14px; background: white; }
.sel-group select:focus { outline: none; border-color: #0f3460; }
.sel-stats { display: flex; align-items: center; gap: 10px; font-size: 14px; color: #888; }

.loading { text-align: center; padding: 60px; color: #888; }
.empty { text-align: center; padding: 60px 20px; background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty h3 { margin: 0 0 8px; }
.empty p { color: #888; margin: 0 0 20px; }

.email-table-wrap { background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); overflow: hidden; margin-bottom: 20px; }
.email-table { width: 100%; }
.et-header, .et-row { display: grid; grid-template-columns: 1fr 100px 80px 110px 60px; gap: 12px; padding: 12px 20px; font-size: 13px; align-items: center; }
.et-header { font-weight: 700; font-size: 11px; color: #888; text-transform: uppercase; background: #fafafa; border-bottom: 2px solid #e5e5e5; }
.et-row { border-bottom: 1px solid #f5f5f5; }
.et-row:hover { background: #f8f9fa; }
.et-email { font-weight: 500; color: #1a1a2e; }
.et-date { font-size: 12px; color: #888; }
.et-actions { display: flex; gap: 4px; }
.btn-icon { background: none; border: 1px solid #e0e0e0; border-radius: 4px; padding: 6px 8px; cursor: pointer; font-size: 14px; }
.btn-icon:hover:not(:disabled) { background: #f0f0f0; }

.badge { padding: 3px 10px; border-radius: 20px; font-size: 11px; font-weight: 600; }
.badge-on { background: #d4edda; color: #155724; }

.info-cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 12px; }
.info-card { background: white; border-radius: 12px; padding: 16px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); }
.info-card strong { display: block; font-size: 14px; margin-bottom: 4px; }
.info-card code { display: block; font-size: 13px; color: #0f3460; font-family: monospace; margin-bottom: 4px; }
.info-card span { font-size: 12px; color: #888; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: white; border-radius: 12px; width: 90%; max-width: 460px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); }
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

.input-suffix { display: flex; align-items: center; margin-bottom: 16px; }
.input-suffix input { flex: 1; padding: 10px 12px; border: 2px solid #e0e0e0; border-right: none; border-radius: 8px 0 0 8px; font-size: 14px; }
.input-suffix input:focus { outline: none; border-color: #0f3460; }
.input-suffix span { padding: 10px 14px; background: #f0f0f0; border: 2px solid #e0e0e0; border-left: none; border-radius: 0 8px 8px 0; font-size: 14px; color: #888; }

.pw-row { display: flex; gap: 8px; }
.pw-row input { flex: 1; }
</style>
