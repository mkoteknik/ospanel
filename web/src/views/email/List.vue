<script setup lang="ts">
const emailFeatures = [
  { icon: '📧', title: 'Email Hesapları', desc: 'Domain başına sınırsız email hesabı. IMAP/POP3/SMTP desteği.' },
  { icon: '🔄', title: 'Forwarder (Yönlendirme)', desc: 'Email\'leri otomatik olarak başka adrese yönlendirin.' },
  { icon: '🤖', title: 'Otomatik Yanıtlayıcı', desc: 'Tatil modu, iş dışı mesajları için otomatik cevap.' },
  { icon: '🛡️', title: 'SpamAssassin', desc: 'Gelişmiş spam filtreleme. Puan tabanlı spam tespiti.' },
  { icon: '🔑', title: 'DKIM / SPF / DMARC', desc: 'Email güvenlik kayıtları otomatik yapılandırma.' },
  { icon: '📊', title: 'Email Kotası', desc: 'Hesap başına depolama sınırı. Kota aşımı uyarıları.' },
  { icon: '🌐', title: 'Webmail', desc: 'SnappyMail ile web tabanlı email erişimi. Modern arayüz.' },
  { icon: '📱', title: 'Mobil Senkronizasyon', desc: 'IMAP/SMTP ile tüm cihazlarda email senkronizasyonu.' },
]
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2>📧 E-Posta Yönetimi</h2>
      <p>Domainlerinize profesyonel email hesapları oluşturun. Dovecot + Postfix altyapısı.</p>
    </div>

    <div class="card-grid">
      <div v-for="f in emailFeatures" :key="f.title" class="email-card">
        <div class="em-icon">{{ f.icon }}</div>
        <h3>{{ f.title }}</h3>
        <p>{{ f.desc }}</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-header { margin-bottom: 24px; }
.page-header h2 { margin: 0; font-size: 22px; }
.page-header p { color: #888; margin: 4px 0 0; font-size: 14px; }

.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.email-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
  border: 1px solid #f0f0f0;
  transition: all 0.2s;
}
.email-card:hover { box-shadow: 0 4px 16px rgba(0,0,0,0.08); }

.em-icon { font-size: 32px; margin-bottom: 12px; }
.email-card h3 { margin: 0 0 8px; font-size: 16px; color: #1a1a2e; }
.email-card p { margin: 0; font-size: 13px; color: #777; line-height: 1.5; }
</style>
