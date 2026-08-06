<template>
  <div class="page">
    <h2>🛡️ Güvenlik</h2>
    <div class="info-banner">
      ℹ️ Fail2ban, ModSecurity, IP engelleme ve güvenlik duvarı yönetimi Linux sunucuda aktif olacak.
    </div>
    <div class="empty-state">
      <div class="empty-icon">🛡️</div>
      <h3>Güvenlik Ayarları</h3>
      <p>Sunucu güvenliğini buradan yönetin.</p>
      <div class="feature-list">
        <span>🔐 Fail2ban entegrasyonu</span>
        <span>🛡️ ModSecurity WAF</span>
        <span>🚫 IP Kara/Beyaz Liste</span>
        <span>🔑 2FA (İki Faktörlü Kimlik Doğrulama)</span>
        <span>📋 Giriş geçmişi ve audit log</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.page { max-width: 1200px; }
h2 { margin: 0 0 16px; }
.info-banner { background: #e8f4fd; color: #0f3460; padding: 12px 16px; border-radius: 8px; margin-bottom: 16px; font-size: 14px; }
.empty-state { text-align: center; padding: 60px 20px; background: white; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-state h3 { margin: 0 0 16px; }
.empty-state p { color: #888; margin: 0; }
.feature-list { display: flex; flex-direction: column; gap: 8px; margin-top: 16px; }
.feature-list span { padding: 8px 16px; background: #f8f9fa; border-radius: 8px; font-size: 14px; }
</style>
