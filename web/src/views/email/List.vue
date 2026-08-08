<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, onMounted, watch } from 'vue'
import { api } from '@/api/client'

const { t } = useI18n()

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
  if (!confirm(t('email.confirmDelete'))) return
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
  <div class="email-page">
    <div class="page-head">
      <div>
        <h2>{{ t('auto.efac85') }}</h2>
        <p>{{ t('auto.50fe98') }}</p>
      </div>
      <button class="aura-btn aura-btn-primary" :disabled="!selectedDomain" @click="showCreate = true">+ Email Ekle</button>
    </div>

    <!-- Domain selector -->
    <div class="aura-card selector">
      <div class="sel-group">
        <span class="kicker">Domain</span>
        <select v-model="selectedDomain" class="aura-select">
          <option v-for="d in domains" :key="d.id" :value="d.domain">{{ d.domain }}</option>
        </select>
      </div>
      <div class="sel-stats">
        <span class="sel-count">{{ emails.length }} hesap</span>
        <button class="aura-btn aura-btn-ghost sel-refresh" @click="loadEmails" :title="t('common.refresh')">↻</button>
      </div>
    </div>

    <div v-if="loading" class="state muted">{{ t('common.loading') }}</div>

    <div v-else-if="emails.length === 0" class="aura-card empty">
      <div class="empty-icon">✉</div>
      <div class="empty-value">{{ t('auto.1634c1') }}</div>
      <p class="empty-desc">{{ t('email.emptyDesc2', { domain: selectedDomain }) }}</p>
      <button class="aura-btn aura-btn-primary" @click="showCreate = true">{{ t('auto.68381a') }}</button>
    </div>

    <div v-else class="aura-card table-wrap">
      <div class="et-header">
        <span class="kicker">{{ t('common.email') }}</span>
        <span class="kicker">Kota</span>
        <span class="kicker">{{ t('common.status') }}</span>
        <span class="kicker">{{ t('auto.69ff4b') }}</span>
        <span></span>
      </div>
      <div v-for="e in emails" :key="e.id" class="et-row">
        <span class="et-email">{{ e.email }}</span>
        <span class="et-quota">{{ (e.quota / 1024).toFixed(1) }} GB</span>
        <span><span class="badge">{{ t('common.active') }}</span></span>
        <span class="et-date">{{ new Date(e.created_at).toLocaleDateString('tr-TR') }}</span>
        <span class="et-actions">
          <button class="icon-btn" title="Webmail" @click="openWebmail(e.email)">↗</button>
          <button class="icon-btn danger" :title="t('common.delete')" :disabled="actionLoading === 'del'+e.id" @click="deleteEmail(e.id)">×</button>
        </span>
      </div>
    </div>

    <!-- IMAP / POP3 / SMTP -->
    <div class="info-grid">
      <div class="aura-card info-card">
        <span class="kicker">IMAP</span>
        <code class="info-value">mail.{{ selectedDomain || 'site.com' }}</code>
        <span class="info-meta">Port 993 · SSL</span>
      </div>
      <div class="aura-card info-card">
        <span class="kicker">POP3</span>
        <code class="info-value">mail.{{ selectedDomain || 'site.com' }}</code>
        <span class="info-meta">Port 995 · SSL</span>
      </div>
      <div class="aura-card info-card">
        <span class="kicker">SMTP</span>
        <code class="info-value">mail.{{ selectedDomain || 'site.com' }}</code>
        <span class="info-meta">Port 587 · TLS</span>
      </div>
    </div>

    <!-- Create modal -->
    <div v-if="showCreate" class="overlay" @click.self="showCreate = false">
      <div class="aura-card modal">
        <div class="modal-head">
          <div>
            <span class="kicker">Yeni hesap</span>
            <h3 class="modal-title">@{{ selectedDomain }}</h3>
          </div>
          <button class="icon-btn" @click="showCreate = false">×</button>
        </div>
        <div class="modal-body">
          <div class="field">
            <label class="kicker">Email adresi</label>
            <div class="input-suffix">
              <input v-model="newEmail.local" :placeholder="t('auto.d370f6')" @keyup.enter="createEmail" />
              <span>@{{ selectedDomain }}</span>
            </div>
          </div>
          <div class="field">
            <label class="kicker">{{ t('common.password') }}</label>
            <div class="pw-row">
              <input v-model="newEmail.password" type="text" placeholder="••••••••" />
              <button class="aura-btn aura-btn-ghost" @click="generatePassword">{{ t('auto.59c356') }}</button>
            </div>
          </div>
          <div class="field">
            <label class="kicker">{{ t('common.quota') }}</label>
            <input v-model.number="newEmail.quota" type="number" min="100" max="102400" />
          </div>
        </div>
        <div class="modal-foot">
          <button class="aura-btn aura-btn-ghost" @click="showCreate = false">{{ t('common.cancel') }}</button>
          <button class="aura-btn aura-btn-primary" :disabled="actionLoading === 'create'" @click="createEmail">{{ t('common.create') }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.email-page { display: flex; flex-direction: column; gap: 16px; }
.page-head { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; flex-wrap: wrap; }
.page-head h2 { font-size: 20px; font-weight: 700; letter-spacing: -0.02em; color: var(--aura-text); }
.page-head p { margin-top: 4px; font-size: 13px; color: var(--aura-text-muted); max-width: 560px; line-height: 1.5; }

.kicker { font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; color: var(--aura-text-faint); }

.selector { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; padding: 14px 16px; }
.sel-group { flex: 1; max-width: 380px; display: flex; flex-direction: column; gap: 8px; }
.aura-select { width: 100%; padding: 10px 12px; border: 1px solid var(--aura-border); border-radius: 10px; font-size: 14px; background: var(--aura-surface); color: var(--aura-text); }
.aura-select:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
.sel-stats { display: flex; align-items: center; gap: 10px; }
.sel-count { font-size: 13px; color: var(--aura-text-muted); }
.sel-refresh { padding: 8px 12px; }

.state { text-align: center; padding: 40px; font-size: 13px; }
.state.muted { color: var(--aura-text-muted); }

.empty { text-align: center; padding: 32px 20px; display: flex; flex-direction: column; align-items: center; gap: 8px; }
.empty-icon { width: 44px; height: 44px; border-radius: 12px; display: grid; place-items: center; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); color: var(--aura-text-muted); font-size: 18px; }
.empty-value { font-size: 15px; font-weight: 650; color: var(--aura-text); }
.empty-desc { font-size: 13px; color: var(--aura-text-muted); margin-bottom: 8px; }

.table-wrap { overflow: hidden; padding: 0; }
.et-header, .et-row { display: grid; grid-template-columns: 1fr 110px 90px 120px 80px; gap: 12px; padding: 12px 16px; align-items: center; }
.et-header { background: var(--aura-bg-subtle); border-bottom: 1px solid var(--aura-border); }
.et-row { border-bottom: 1px solid var(--aura-border); font-size: 13px; }
.et-row:last-child { border-bottom: none; }
.et-row:hover { background: var(--aura-surface-hover); }
.et-email { font-weight: 550; color: var(--aura-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.et-quota { color: var(--aura-text-muted); font-variant-numeric: tabular-nums; }
.et-date { font-size: 12px; color: var(--aura-text-muted); }
.et-actions { display: flex; gap: 6px; justify-content: flex-end; }

.badge { display: inline-flex; align-items: center; gap: 6px; padding: 4px 10px; border-radius: 999px; font-size: 11px; font-weight: 650; background: var(--aura-accent-soft); color: var(--aura-success); border: 1px solid transparent; }
.badge::before { content: ''; width: 6px; height: 6px; border-radius: 999px; background: var(--aura-success); }

.icon-btn { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 8px; border: 1px solid var(--aura-border); background: var(--aura-surface); color: var(--aura-text-muted); cursor: pointer; font-size: 14px; }
.icon-btn:hover { background: var(--aura-surface-hover); color: var(--aura-text); border-color: var(--aura-border-strong); }
.icon-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.icon-btn.danger:hover { color: var(--aura-danger); border-color: color-mix(in srgb, var(--aura-danger) 20%, var(--aura-border)); }

.info-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
@media (max-width: 860px) { .info-grid { grid-template-columns: 1fr; } .et-header, .et-row { grid-template-columns: 1fr 80px 90px; } .et-header span:nth-child(4), .et-row span:nth-child(4) { display: none; } }
.info-card { padding: 14px 16px; display: flex; flex-direction: column; gap: 6px; }
.info-value { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 13px; color: var(--aura-text); }
.info-meta { font-size: 12px; color: var(--aura-text-faint); }

.overlay { position: fixed; inset: 0; background: rgba(15,23,42,0.45); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 16px; }
.modal { width: 100%; max-width: 460px; overflow: hidden; }
.modal-head { display: flex; justify-content: space-between; align-items: flex-start; padding: 18px 20px; border-bottom: 1px solid var(--aura-border); }
.modal-title { font-size: 15px; font-weight: 700; color: var(--aura-text); margin-top: 2px; }
.modal-body { padding: 18px 20px; display: flex; flex-direction: column; gap: 14px; }
.modal-foot { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid var(--aura-border); }

.field { display: flex; flex-direction: column; gap: 6px; }
.field input { width: 100%; padding: 10px 12px; border: 1px solid var(--aura-border); border-radius: 10px; font-size: 14px; background: var(--aura-surface); color: var(--aura-text); }
.field input:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
.field input:placeholder { color: var(--aura-text-faint); }
.input-suffix { display: flex; align-items: stretch; }
.input-suffix input { flex: 1; border-radius: 10px 0 0 10px; }
.input-suffix span { display: inline-flex; align-items: center; padding: 0 12px; background: var(--aura-bg-subtle); border: 1px solid var(--aura-border); border-left: none; border-radius: 0 10px 10px 0; font-size: 13px; color: var(--aura-text-muted); }
.pw-row { display: flex; gap: 8px; }
.pw-row input { flex: 1; }
</style>
