<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ref, onMounted, watch } from 'vue'
import { api } from '@/api/client'

const { t } = useI18n()

interface DNSRecord { id: number; type: string; name: string; value: string; ttl: number; priority?: number }
interface DomainInfo { id: number; domain: string }

const domains = ref<DomainInfo[]>([])
const selectedDomainId = ref(0)
const records = ref<DNSRecord[]>([])
const loading = ref(false)
const showAdd = ref(false)
const newRecord = ref({ type: 'A', name: '', value: '', ttl: 3600, priority: 0 })

const recordTypes = ['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS', 'SRV', 'CAA']

async function loadDomains() {
  try {
    const res = await api.get('/api/v1/domains')
    domains.value = res.data.domains || []
    if (domains.value.length > 0) selectedDomainId.value = domains.value[0].id
  } catch { }
}

async function loadRecords() {
  if (!selectedDomainId.value) return
  loading.value = true
  try {
    const res = await api.get('/api/v1/dns?domain_id=' + selectedDomainId.value)
    records.value = res.data.records || []
  } catch {
    records.value = []
  }
  finally { loading.value = false }
}

async function addRecord() {
  try {
    await api.post('/api/v1/dns', {
      domain_id: selectedDomainId.value,
      ...newRecord.value
    })
    showAdd.value = false
    newRecord.value = { type: 'A', name: '', value: '', ttl: 3600, priority: 0 }
    await loadRecords()
  } catch { }
}

async function deleteRecord(r: DNSRecord) {
  if (!confirm(r.type + ' ' + r.name + ' silinecek!')) return
  try { await api.delete('/api/v1/dns/' + r.id); await loadRecords() }
  catch { }
}

watch(selectedDomainId, loadRecords)
onMounted(async () => { await loadDomains(); await loadRecords() })
</script>

<template>
  <div class="dns-page">
    <div class="page-head">
      <div>
        <h2>{{ t('cloudflare.dnsManagement') }}</h2>
        <p>{{ t('auto.feab0b') }}</p>
      </div>
      <button class="aura-btn aura-btn-primary" :disabled="!selectedDomainId" @click="showAdd = true">{{ t('auto.113800') }}</button>
    </div>

    <div class="aura-card selector">
      <label class="field" style="flex:1; max-width:360px;">
        <span class="kicker">Domain</span>
        <select v-model="selectedDomainId" class="aura-select">
          <option v-for="d in domains" :key="d.id" :value="d.id">{{ d.domain }}</option>
        </select>
      </label>
      <div class="sel-meta">
        <span class="kicker">{{ records.length }} {{ t('common.record') }}</span>
        <button class="aura-btn aura-btn-ghost sm" @click="loadRecords">{{ t('common.refresh') }}</button>
      </div>
    </div>

    <div v-if="loading" class="state muted">{{ t('common.loading') }}</div>
    <div v-else-if="records.length === 0" class="aura-card empty">
      <div class="empty-icon">◍</div>
      <div class="empty-value">{{ t('auto.5ec4c6') }}</div>
      <p class="empty-desc">{{ t('auto.fb9c52') }}</p>
      <button class="aura-btn aura-btn-primary" :disabled="!selectedDomainId" @click="showAdd = true">{{ t('auto.113800') }}</button>
    </div>

    <div v-else class="aura-card table-wrap">
      <div class="dt-head">
        <span class="kicker">{{ t('common.type') }}</span>
        <span class="kicker">{{ t('common.name') }}</span>
        <span class="kicker">{{ t('common.value') }}</span>
        <span class="kicker">{{ t('common.ttl') }}</span>
        <span></span>
      </div>
      <div v-for="r in records" :key="r.id" class="dt-row">
        <span><span class="type-badge">{{ r.type }}</span></span>
        <span class="cell-name">{{ r.name || '@' }}</span>
        <span class="cell-value">{{ r.value }}{{ r.priority ? t('auto.6a65a9') + r.priority : '' }}</span>
        <span class="cell-ttl">{{ r.ttl }}s</span>
        <span class="cell-act"><button class="icon-btn danger" :title="t('common.delete')" @click="deleteRecord(r)">×</button></span>
      </div>
    </div>

    <div v-if="showAdd" class="overlay" @click.self="showAdd=false">
      <div class="aura-card modal">
        <div class="modal-head">
          <div>
            <span class="kicker">{{ t('auto.124fa1') }}</span>
            <h3 class="modal-title">{{ t('dns.addRecord') }}</h3>
          </div>
          <button class="icon-btn" @click="showAdd=false">×</button>
        </div>
        <div class="modal-body">
          <div class="form-grid">
            <label class="field">
              <span class="kicker">{{ t('common.type') }}</span>
              <select v-model="newRecord.type" class="aura-select">
                <option v-for="t in recordTypes" :key="t" :value="t">{{ t }}</option>
              </select>
            </label>
            <label class="field">
              <span class="kicker">{{ t('common.ttl') }}</span>
              <input v-model.number="newRecord.ttl" type="number" min="60" placeholder="3600" />
            </label>
          </div>
          <label class="field">
            <span class="kicker">{{ t('common.name') }}</span>
            <input v-model="newRecord.name" placeholder="www, mail gibi" />
          </label>
          <label class="field">
            <span class="kicker">{{ t('common.value') }}</span>
            <input v-model="newRecord.value" :placeholder="t('dns.recordValue')" />
          </label>
          <label v-if="newRecord.type === 'MX' || newRecord.type === 'SRV'" class="field">
            <span class="kicker">{{ t('auto.5bd24f') }}</span>
            <input v-model.number="newRecord.priority" type="number" min="0" max="100" placeholder="10" />
          </label>
        </div>
        <div class="modal-foot">
          <button class="aura-btn aura-btn-ghost" @click="showAdd=false">{{ t('common.cancel') }}</button>
          <button class="aura-btn aura-btn-primary" @click="addRecord">Ekle</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dns-page { display: flex; flex-direction: column; gap: 16px; }
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

.selector { display: flex; justify-content: space-between; align-items: flex-end; gap: 16px; padding: 16px; flex-wrap: wrap; }
.sel-meta { display: flex; align-items: center; gap: 10px; }
.aura-select { width: 100%; padding: 10px 12px; border: 1px solid var(--aura-border); border-radius: 10px; font-size: 14px; background: var(--aura-surface); color: var(--aura-text); cursor: pointer; }
.aura-select:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
.sm { padding: 8px 12px; font-size: 12px; }

.table-wrap { overflow: hidden; padding: 0; }
.dt-head, .dt-row { display: grid; grid-template-columns: 80px 1fr 1fr 90px 40px; gap: 12px; padding: 12px 16px; align-items: center; }
.dt-head { background: var(--aura-bg-subtle); border-bottom: 1px solid var(--aura-border); }
.dt-row { border-bottom: 1px solid var(--aura-border); transition: background 0.12s; }
.dt-row:last-child { border-bottom: none; }
.dt-row:hover { background: var(--aura-surface-hover); }
.type-badge { display: inline-flex; padding: 3px 8px; background: var(--aura-accent-soft); color: var(--aura-accent); border-radius: 6px; font-weight: 700; font-size: 11px; font-family: ui-monospace, monospace; border: 1px solid color-mix(in srgb, var(--aura-accent) 14%, transparent); }
.cell-name { font-weight: 550; color: var(--aura-text); font-size: 13px; overflow: hidden; text-overflow: ellipsis; }
.cell-value { font-family: ui-monospace, monospace; font-size: 12px; color: var(--aura-text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cell-ttl { font-family: ui-monospace, monospace; font-size: 12px; color: var(--aura-text-faint); }
.cell-act { display: flex; justify-content: flex-end; }

.overlay { position: fixed; inset: 0; background: rgba(15,23,42,0.45); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 16px; }
.modal { width: 100%; max-width: 520px; overflow: hidden; }
.modal-head { display: flex; justify-content: space-between; align-items: flex-start; padding: 18px 20px; border-bottom: 1px solid var(--aura-border); }
.modal-title { font-size: 15px; font-weight: 700; color: var(--aura-text); margin-top: 2px; }
.modal-body { padding: 18px 20px; display: flex; flex-direction: column; gap: 14px; }
.modal-foot { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid var(--aura-border); }
.field { display: flex; flex-direction: column; gap: 6px; }
.field input { width: 100%; padding: 10px 12px; border: 1px solid var(--aura-border); border-radius: 10px; font-size: 14px; background: var(--aura-surface); color: var(--aura-text); }
.field input:focus { outline: none; border-color: var(--aura-accent); box-shadow: 0 0 0 3px var(--aura-accent-ring); }
.field input:placeholder { color: var(--aura-text-faint); }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.icon-btn { width: 30px; height: 30px; display: grid; place-items: center; border-radius: 8px; border: 1px solid var(--aura-border); background: var(--aura-surface); color: var(--aura-text-muted); cursor: pointer; font-size: 16px; line-height: 1; }
.icon-btn:hover { background: var(--aura-surface-hover); color: var(--aura-text); }
.icon-btn.danger:hover { background: color-mix(in srgb, var(--aura-danger) 8%, var(--aura-surface)); border-color: color-mix(in srgb, var(--aura-danger) 18%, var(--aura-border)); color: var(--aura-danger); }
</style>
