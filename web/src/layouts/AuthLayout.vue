<script setup lang="ts">
</script>

<template>
  <div class="auth-layout">
    <div class="auth-container">
      <div class="auth-header">
        <h1>Aura Panel</h1>
        <p>Modern Hosting Kontrol Paneli</p>
      </div>
      <div class="auth-content">
        <slot />
      </div>
    </div>
  </div>
</template>

<style scoped>
.auth-layout {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.auth-container {
  width: 100%;
  max-width: 420px;
  padding: 20px;
}

.auth-header {
  text-align: center;
  color: white;
  margin-bottom: 30px;
}

.auth-header h1 {
  font-size: 32px;
  font-weight: 700;
  margin: 0;
}

.auth-header p {
  margin: 5px 0 0;
  opacity: 0.9;
}

.auth-content {
  background: white;
  border-radius: 12px;
  padding: 30px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}
</style>
