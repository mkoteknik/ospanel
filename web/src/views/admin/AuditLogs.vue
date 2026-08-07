<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface AuditLog { id: number; action: string; resource: string; details: string; ip: string; created_at: string }

const logs = ref<AuditLog[]>([])
const loading = ref(false)

async function loadLogs() {
  loading.value = true
  try {
    const res = await api.get('/api/v1/admin/audit-logs')
    logs.value = res.data.logs || []
  } catch { }
  finally { loading.value = false }
}

function actionIcon(action: string) {
  if (action.includes('login')) return '🔐'
  if (action.includes('create')) return '➕'
  if (action.includes('delete')) return '🗑️'
  if (action.includes('update')) return '✏️'
  return '📋'
}

onMounted(loadLogs)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>Denetim Kayıtları</h2>
        <p>Sistemdeki tüm işlemlerin kaydı.</p>
      </div>
      <button class="btn-sm" @click="loadLogs">Yenile</button>
    </div>

    <div v-if="loading" class="loading">Yükleniyor...</div>

    <div v-else-if="logs.length === 0" class="empty">Henüz denetim kaydı yok.</div>

    <div v-else class="log-list">
      <div v-for="l in logs" :key="l.id" class="log-row">
        <span class="log-icon">{{ actionIcon(l.action) }}</span>
        <span class="log-action">{{ l.action }}</span>
        <span class="log-resource">{{ l.resource }}</span>
        <span class="log-ip">{{ l.ip }}</span>
        <span class="log-time">{{ l.created_at?.split('T')[0] }} {{ l.created_at?.split('T')[1]?.substring(0,8) || '' }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 14px; }
.btn-sm { padding: 8px 16px; background: #0f3460; color: white; border: none; border-radius: 6px; font-size: 13px; cursor: pointer; }
.loading { text-align: center; padding: 60px; color: #888; }
.empty { text-align: center; padding: 60px; background: white; border-radius: 12px; color: #888; }
.log-list { background: white; border-radius: 12px; box-shadow: 0 1px 4px rgba(0,0,0,0.04); overflow: hidden; }
.log-row { display: grid; grid-template-columns: 40px 140px 1fr 130px 160px; gap: 12px; padding: 12px 20px; font-size: 13px; align-items: center; border-bottom: 1px solid #f5f5f5; }
.log-row:hover { background: #f8f9fa; }
.log-icon { font-size: 16px; }
.log-action { font-weight: 600; color: #1a1a2e; }
.log-resource { color: #555; font-family: monospace; font-size: 12px; }
.log-ip { color: #888; font-family: monospace; font-size: 12px; }
.log-time { color: #aaa; font-size: 11px; text-align: right; }
</style>
