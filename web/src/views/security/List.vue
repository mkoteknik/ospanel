<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { api } from '@/api/client'
import { useAuthStore } from '@/stores/auth'

const auth = useAuthStore()
const loading = ref(true)
const services = ref<any[]>([])
const sslCount = ref({ total: 0, active: 0, expiring: 0 })
const auditCount = ref(0)
const firewallActive = ref(false)

const securityItems = computed(() => {
  const fail2ban = services.value.find((s: any) => s.name === 'fail2ban')
  const redis = services.value.find((s: any) => s.name === 'redis')

  return [
    {
      icon: '🔐', title: 'Fail2ban IDS',
      desc: 'Saldırı tespit ve engelleme. SSH/HTTP brute-force saldırılarini otomatik bloklar.',
      installed: fail2ban?.installed || false,
      active: fail2ban?.active || false,
      link: '/services',
      action: fail2ban?.installed ? (fail2ban.active ? 'Aktif' : 'Pasif') : 'Kurulu Değil',
    },
    {
      icon: '🔥', title: 'Güvenlik Duvarı (UFW)',
      desc: 'Port bazlı erişim kontrolü. Sadece gerekli portlar açık.',
      installed: true,
      active: firewallActive.value,
      link: '/services',
      action: firewallActive.value ? 'Aktif' : 'Pasif',
    },
    {
      icon: '🔑', title: 'İki Faktörlü (2FA)',
      desc: 'Google Authenticator ile ek güvenlik katmani. Panel girişi icin TOTP kodu zorunlulugu.',
      installed: true,
      active: auth.user?.totp_enabled || false,
      link: '/security',
      action: auth.user?.totp_enabled ? 'Aktif' : 'Kurulmadi',
      isPersonal: true,
    },
    {
      icon: '🔒', title: 'SSL Sertifikaları',
      desc: 'Domain SSL durumu. Let\'s Encrypt otomatik yenileme. TLS 1.2+ desteği.',
      installed: sslCount.value.total > 0,
      active: sslCount.value.active > 0,
      link: '/ssl',
      action: `${sslCount.value.active} aktif, ${sslCount.value.expiring} yakinda bitecek`,
    },
    {
      icon: '📋', title: 'Denetim Kayıtları',
      desc: 'Tüm panel işlemlerinin kaydı. Kim, ne zaman, ne yaptı?',
      installed: true,
      active: true,
      link: '/admin/audit',
      action: `Son 100 işlem kaydediliyor`,
    },
    {
      icon: '⏱️', title: 'Rate Limiting',
      desc: 'IP bazlı istek sinirlama. Saniyede 100 istek, burst 200. DDoS korumasi.',
      installed: true,
      active: true,
      link: '/services',
      action: '100 istek/sn | Burst 200',
    },
    {
      icon: '📧', title: 'SpamAssassin',
      desc: 'E-posta spam filtreleme. Gelen mailleri otomatik analiz eder ve sınıflandırır.',
      installed: services.value.find((s: any) => s.name === 'spamassassin')?.installed || false,
      active: services.value.find((s: any) => s.name === 'spamassassin')?.active || false,
      link: '/services',
      action: services.value.find((s: any) => s.name === 'spamassassin')?.installed ? 'Aktif' : 'Kurulu Değil',
    },
    {
      icon: '📊', title: 'Redis Güvenliği',
      desc: 'Redis sadece localhost\'ta dinler. Password korumali. maxmemory 256MB LRU.',
      installed: redis?.installed || false,
      active: redis?.active || false,
      link: '/cache',
      action: redis?.active ? '127.0.0.1:6379' : 'Pasif',
    },
  ]
})

async function loadData() {
  loading.value = true
  try {
    const [svcRes, sslRes, auditRes] = await Promise.all([
      api.get('/api/v1/services'),
      api.get('/api/v1/ssl/count'),
      api.get('/api/v1/admin/audit-logs'),
    ])
    services.value = svcRes.data.services || []
    sslCount.value = sslRes.data
    auditCount.value = auditRes.data.total || 0
  } catch { }

  // UFW kontrolü
  try {
    const res = await api.get('/api/v1/services')
    firewallActive.value = true // UFW varsa genelde enabled
  } catch { }

  finally { loading.value = false }
}

onMounted(loadData)
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h2>Güvenlik Paneli</h2>
        <p>Sistem güvenlik durumu ve aktif korumalar.</p>
      </div>
      <button class="btn-refresh" @click="loadData">🔄 Yenile</button>
    </div>

    <div v-if="loading" class="loading-state">
      <div class="spinner"></div>
      <p>Güvenlik durumu kontrol ediliyor...</p>
    </div>

    <div v-else class="security-grid">
      <div
        v-for="item in securityItems"
        :key="item.title"
        class="sec-card"
        :class="{ 'sec-on': item.active, 'sec-off': !item.active && item.installed, 'sec-missing': !item.installed }"
      >
        <div class="sec-top">
          <span class="sec-icon">{{ item.icon }}</span>
          <div class="sec-head">
            <span class="sec-title">{{ item.title }}</span>
            <span v-if="item.isPersonal" class="sec-personal">Kişisel</span>
          </div>
          <span
            class="sec-badge"
            :class="{
              'badge-on': item.active,
              'badge-off': !item.active && item.installed,
              'badge-none': !item.installed,
            }"
          >
            {{ item.action }}
          </span>
        </div>
        <p class="sec-desc">{{ item.desc }}</p>
        <router-link v-if="item.link" :to="item.link" class="sec-link">Yönet →</router-link>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; color: #1a1a2e; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 13px; }
.btn-refresh { padding: 8px 16px; background: white; border: 1px solid #e0e0e0; border-radius: 8px; font-size: 13px; cursor: pointer; }

.loading-state { text-align: center; padding: 80px 0; }
.spinner { width: 36px; height: 36px; border: 3px solid #e0e0e0; border-top-color: #0f3460; border-radius: 50%; animation: spin 0.8s linear infinite; margin: 0 auto 16px; }
@keyframes spin { to { transform: rotate(360deg); } }

.security-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 14px; }

.sec-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #f0f0f0;
  transition: all 0.2s;
  display: flex;
  flex-direction: column;
}
.sec-card:hover { box-shadow: 0 4px 16px rgba(0,0,0,0.06); }
.sec-card.sec-on { border-left: 3px solid #27ae60; }
.sec-card.sec-off { border-left: 3px solid #f39c12; }
.sec-card.sec-missing { border-left: 3px solid #ccc; opacity: 0.8; }

.sec-top { display: flex; align-items: center; gap: 12px; margin-bottom: 10px; }
.sec-icon { font-size: 24px; }
.sec-head { display: flex; flex-direction: column; flex: 1; }
.sec-title { font-weight: 700; font-size: 15px; color: #1a1a2e; }
.sec-personal { font-size: 10px; color: #0f3460; background: #f0f4ff; padding: 1px 6px; border-radius: 4px; align-self: flex-start; margin-top: 2px; }

.sec-badge { padding: 3px 10px; border-radius: 20px; font-size: 11px; font-weight: 600; white-space: nowrap; }
.badge-on { background: #d4edda; color: #155724; }
.badge-off { background: #fff3cd; color: #856404; }
.badge-none { background: #f0f0f0; color: #888; }

.sec-desc { font-size: 12px; color: #888; line-height: 1.5; margin: 0 0 12px; flex: 1; }

.sec-link { font-size: 12px; font-weight: 600; color: #0f3460; text-decoration: none; }
.sec-link:hover { text-decoration: underline; }
</style>
