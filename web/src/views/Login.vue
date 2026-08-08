<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'

const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const username = ref('')
const password = ref('')
const totpCode = ref('')
const need2FA = ref(false)
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  if (!username.value || !password.value) {
    error.value = t('auth.usernameRequired')
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await authStore.login(username.value, password.value, totpCode.value || undefined)
    // Backend may require 2FA
    if ((res as any)?.require_2fa) {
      need2FA.value = true
      error.value = t('auth.need2FA')
      return
    }
    router.push('/')
  } catch (err: any) {
    const data = err.response?.data
    if (data?.require_2fa === 'true' || data?.error?.includes('2FA')) {
      need2FA.value = true
      error.value = t('auth.require2FA')
    } else {
      error.value = data?.error || t('auth.loginFailed')
    }
  } finally {
    loading.value = false
  }
}

async function quickLogin() {
  username.value = 'admin'
  password.value = 'admin123'
  totpCode.value = ''
  need2FA.value = false
  loading.value = true
  error.value = ''
  try {
    await authStore.login('admin', 'admin123')
    router.push('/')
  } catch (err: any) {
    error.value = err.response?.data?.error || t('auth.loginFailed')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page" :data-theme="themeStore.resolved">
    <!-- Theme toggle — top right -->
    <button class="theme-toggle" @click="themeStore.toggle()" :title="themeStore.mode">
      <span v-if="themeStore.isDark">☀️</span>
      <span v-else>🌙</span>
      <small>{{ themeStore.mode === 'system' ? t('theme.system') : themeStore.isDark ? t('theme.dark') : t('theme.light') }}</small>
    </button>

    <!-- Subtle grid background -->
    <div class="bg-grid" aria-hidden="true"></div>
    <div class="bg-glow" aria-hidden="true"></div>

    <div class="aura-3d-wrap login-wrap">
      <div class="aura-3d-card login-card">
        <div class="login-head">
          <div class="logo-mark">
            <span class="logo-a">A</span>
          </div>
          <h1 class="brand">Aura Panel</h1>
          <p class="brand-sub">{{ t('auto.9e6b16') }}</p>
        </div>

        <form @submit.prevent="handleLogin" class="login-form">
          <div class="field">
            <label for="username">{{ t('auth.username') }}</label>
            <input
              id="username"
              v-model="username"
              type="text"
              autocomplete="username"
              placeholder="admin"
              :disabled="loading"
            />
          </div>

          <div class="field">
            <label for="password">{{ t('auth.password') }}</label>
            <input
              id="password"
              v-model="password"
              type="password"
              autocomplete="current-password"
              placeholder="••••••••••••"
              :disabled="loading"
            />
          </div>

          <div v-if="need2FA" class="field">
            <label for="totp">{{ t('auth.totp') }}</label>
            <input
              id="totp"
              v-model="totpCode"
              type="text"
              inputmode="numeric"
              maxlength="6"
              placeholder="123456"
              :disabled="loading"
            />
          </div>

          <div v-if="error" class="alert" role="alert">{{ error }}</div>

          <button type="submit" :disabled="loading" class="aura-btn aura-btn-primary login-primary">
            <span v-if="!loading">{{ t('auth.login') }}</span>
            <span v-else>{{ t('common.loading') }}</span>
          </button>

          <button type="button" class="aura-btn aura-btn-ghost" @click="quickLogin" :disabled="loading">
            {{ t('auth.quickLogin') }}
          </button>

          <p class="hint">{{ t('auth.brand') }} — admin / admin123</p>
        </form>

        <div class="login-foot">
          <span>Aura Panel v1.0</span>
          <span>•</span>
          <span>Go + Vue 3 + OpenLiteSpeed</span>
        </div>
      </div>

      <!-- 3D depth layer — subtle, not heavy -->
      <div class="card-depth" aria-hidden="true"></div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--aura-bg);
  position: relative;
  overflow: hidden;
  padding: 24px;
}

/* Grid — very subtle, not aurora blob */
.bg-grid {
  position: absolute; inset: 0;
  background-image:
    linear-gradient(var(--aura-border) 1px, transparent 1px),
    linear-gradient(90deg, var(--aura-border) 1px, transparent 1px);
  background-size: 40px 40px;
  opacity: 0.25;
  mask-image: radial-gradient(ellipse at center, black 40%, transparent 75%);
}
[data-theme="dark"] .bg-grid { opacity: 0.12; }

.bg-glow {
  position: absolute; top: -20%; right: -10%; width: 60%; height: 60%;
  background: radial-gradient(600px circle at center, var(--aura-accent-soft) 0%, transparent 70%);
  opacity: 0.6;
  pointer-events: none;
}

.theme-toggle {
  position: absolute; top: 16px; right: 16px;
  display: inline-flex; align-items: center; gap: 6px;
  padding: 8px 12px;
  border-radius: 999px;
  border: 1px solid var(--aura-border);
  background: var(--aura-surface);
  color: var(--aura-text-muted);
  font-size: 12px; font-weight: 500;
  cursor: pointer;
  box-shadow: var(--aura-shadow-sm);
}

.login-wrap {
  width: 100%; max-width: 420px;
  position: relative;
  z-index: 1;
}

.login-card {
  padding: 32px 28px 24px;
  width: 100%;
}

.card-depth {
  position: absolute; inset: 0;
  border-radius: var(--aura-radius-lg);
  background: var(--aura-accent-soft);
  transform: translateZ(-20px) translateY(8px) scale(0.97);
  opacity: 0.5;
  filter: blur(12px);
  z-index: -1;
}

.login-head {
  text-align: center;
  margin-bottom: 28px;
}

.logo-mark {
  width: 48px; height: 48px;
  margin: 0 auto 12px;
  border-radius: 12px;
  background: var(--aura-accent);
  color: var(--aura-accent-text);
  display: grid; place-items: center;
  font-size: 22px; font-weight: 800; letter-spacing: -0.02em;
  box-shadow: 0 4px 12px var(--aura-accent-ring), 0 1px 3px rgba(0,0,0,0.1);
  transform: translateZ(10px);
}

.brand {
  font-size: 22px; font-weight: 720; letter-spacing: -0.02em;
  color: var(--aura-text);
  line-height: 1.1;
}
.brand-sub {
  margin-top: 6px;
  font-size: 13px; color: var(--aura-text-muted);
  font-weight: 450;
}

.login-form { display: flex; flex-direction: column; gap: 14px; }

.field { display: flex; flex-direction: column; gap: 6px; }
.field label {
  font-size: 12px; font-weight: 600; letter-spacing: 0.02em;
  color: var(--aura-text);
  text-transform: none; /* avoid all-caps kicker slop */
}
.field input {
  padding: 11px 14px;
  border: 1px solid var(--aura-border);
  border-radius: 10px;
  font-size: 14px;
  background: var(--aura-surface);
  color: var(--aura-text);
  transition: border-color 0.15s, box-shadow 0.15s, background 0.15s;
}
.field input:placeholder { color: var(--aura-text-faint); }
.field input:focus {
  outline: none;
  border-color: var(--aura-accent);
  box-shadow: 0 0 0 3px var(--aura-accent-ring);
  background: var(--aura-surface);
}
.field input:disabled { opacity: 0.6; }

.alert {
  padding: 10px 12px;
  border-radius: 10px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  color: #b91c1c;
  font-size: 13px; line-height: 1.4;
}
[data-theme="dark"] .alert {
  background: #451a1a; border-color: #7f1d1d; color: #fecaca;
}

.login-primary { width: 100%; padding: 12px 16px; font-size: 14px; }

.hint {
  text-align: center;
  font-size: 12px; color: var(--aura-text-faint);
  margin-top: 2px;
}
.hint code {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 6px;
  background: var(--aura-bg-subtle);
  border: 1px solid var(--aura-border);
  color: var(--aura-text-muted);
}

.login-foot {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid var(--aura-border);
  display: flex; align-items: center; justify-content: center; gap: 8px;
  font-size: 11px; color: var(--aura-text-faint);
  letter-spacing: 0.01em;
}
</style>
