<script setup lang="ts">
const recordTypes = [
  { type: 'A', icon: '📍', title: 'A Kaydı', desc: 'Domaini IPv4 adresine yönlendirir.', example: 'site.com → 1.2.3.4' },
  { type: 'AAAA', icon: '📍', title: 'AAAA Kaydı', desc: 'Domaini IPv6 adresine yönlendirir.', example: 'site.com → ::1' },
  { type: 'CNAME', icon: '🔗', title: 'CNAME Kaydı', desc: 'Domaini başka bir domaine alias olarak bağlar.', example: 'www → site.com' },
  { type: 'MX', icon: '📧', title: 'MX Kaydı', desc: 'Email sunucusunu belirler. Öncelik değeri ile.', example: 'site.com → mail.site.com' },
  { type: 'TXT', icon: '📝', title: 'TXT Kaydı', desc: 'Metin tabanlı kayıt. SPF, DKIM, doğrulama için.', example: 'v=spf1 mx ~all' },
  { type: 'NS', icon: '🌐', title: 'NS Kaydı', desc: 'Name server yetkilendirmesi.', example: 'site.com → ns1.site.com' },
  { type: 'SRV', icon: '🔧', title: 'SRV Kaydı', desc: 'Belirli servislerin host ve port bilgisi.', example: '_sip._tcp → sip.sunucu.com' },
  { type: 'CAA', icon: '🔒', title: 'CAA Kaydı', desc: 'Hangi CA\'nın SSL verebileceğini belirler.', example: 'site.com → letsencrypt.org' },
]
</script>

<template>
  <div class="page">
    <div class="page-header">
      <h2>🔧 DNS Yönetimi</h2>
      <p>Domain DNS kayıtlarınızı yönetin. BIND9 entegrasyonu ile anında aktif.</p>
    </div>

    <div class="card-grid">
      <div v-for="r in recordTypes" :key="r.type" class="dns-card">
        <div class="dns-type">{{ r.type }}</div>
        <div class="dns-icon">{{ r.icon }}</div>
        <h3>{{ r.title }}</h3>
        <p>{{ r.desc }}</p>
        <code>{{ r.example }}</code>
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

.dns-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 4px rgba(0,0,0,0.04);
  border: 1px solid #f0f0f0;
  position: relative;
  transition: all 0.2s;
}
.dns-card:hover { box-shadow: 0 4px 16px rgba(0,0,0,0.08); }

.dns-type {
  position: absolute; top: 20px; right: 20px;
  font-size: 14px; font-weight: 700; color: #0f3460;
  background: #e8f4fd; padding: 4px 10px; border-radius: 6px;
}

.dns-icon { font-size: 28px; margin-bottom: 12px; }
.dns-card h3 { margin: 0 0 8px; font-size: 16px; color: #1a1a2e; }
.dns-card p { margin: 0 0 12px; font-size: 13px; color: #777; line-height: 1.5; }
.dns-card code {
  display: block; padding: 8px 12px; background: #f8f9fa; border-radius: 6px;
  font-size: 12px; color: #555; font-family: 'Consolas', monospace;
}
</style>
