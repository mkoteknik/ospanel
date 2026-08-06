<template>
  <div class="page">
    <h2>🔒 SSL Yönetimi</h2>
    <div class="info-banner">
      ℹ️ SSL sertifikaları, domain detay sayfasından yönetilir. Let's Encrypt entegrasyonu Linux sunucuda aktif olacak.
    </div>
    <div class="empty-state">
      <div class="empty-icon">🔒</div>
      <h3>SSL Yönetimi</h3>
      <p>Domainlerinize ücretsiz Let's Encrypt SSL sertifikası kurun.</p>
      <p class="sub">Domain detay sayfası → SSL Kur butonu ile kullanabilirsiniz.</p>
    </div>
  </div>
</template>

<style scoped>
.page { max-width: 1200px; }
h2 { margin: 0 0 16px; }
.info-banner { background: #e8f4fd; color: #0f3460; padding: 12px 16px; border-radius: 8px; margin-bottom: 16px; font-size: 14px; }
.empty-state { text-align: center; padding: 60px 20px; background: white; border-radius: 12px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }
.empty-icon { font-size: 48px; margin-bottom: 16px; }
.empty-state h3 { margin: 0 0 8px; }
.empty-state p { color: #888; margin: 0; }
.sub { font-size: 13px; color: #aaa; margin-top: 8px !important; }
</style>
