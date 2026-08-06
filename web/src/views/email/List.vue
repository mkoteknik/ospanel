<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '@/api/client'

interface Email {
  id: number; email: string; quota: number; forward_to: string; status: string; created_at: string
}

const emails = ref<Email[]>([])
const loading = ref(true)
const showCreate = ref(false)
const newEmail = ref({ email: '', password: '', quota: 1024, domain_id: 0 })

async function load() {
  loading.value = true
  try {
    // Email listesi için domain seçimi gerekli, şimdilik boş
    const res = await api.get('/api/v1/domains')
    const domains = res.data.domains || []
    emails.value = []
  } catch { }
  finally { loading.value = false }
}

async function create() {
  try {
    await api.post('/api/v1/emails', newEmail.value)
    showCreate.value = false
    newEmail.value = { email: '', password: '', quota: 1024, domain_id: 0 }
    load()
  } catch { }
}

onMounted(load)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>📧 E-Posta Yönetimi</h2>
        <p class="page-desc">Domainlerinize e-posta hesapları oluşturun (Dovecot + Postfix).</p>
      </div>
      <button class="btn-add" @click="showCreate = true">+ E-Posta Ekle</button>
    </div>

    <div class="info-banner">
      ℹ️ E-posta yönetimi için sunucuda Dovecot ve Postfix kurulu olmalıdır.
      Kurulum scripti bu servisleri otomatik olarak yapılandıracaktır.
    </div>

    <div v-if="loading" class="loading">Yükleniyor...</div>

    <div v-else class="empty-state">
      <div class="empty-icon">📧</div>
      <h3>Email modülü hazır</h3>
      <p>Linux sunucuda Dovecot/Postfix aktif olduğunda email hesapları burada yönetilecek.</p>
    </div>
  </div>
</template>

<style scoped>
.page { max-width: 1200px; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { margin: 0; }
.page-desc { color: #888; margin: 4px 0 0; font-size: 14px; }
.btn-add { padding: 10px 20px; background: #0f3460; color: white; border: none; border-radius: 8px; font-size: 14px; font-weight: 600; cursor: pointer; }
.btn-add:hover { background: #1a4a7a; }
.info-banner { background: #e8f4fd; color: #0f3460; padding: 12px 16px; border-radius: 8px; margin-bottom: 16px; font-size: 14px; }
.loading { text-align: center; padding: 60px; color: #888; }
.empty-state { text-align: center; padding: 60px 20px; background: white; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-state h3 { margin: 0 0 8px; }
.empty-state p { color: #888; margin: 0; }
</style>
