<script setup lang="ts">
import { computed } from 'vue'
import { NConfigProvider, NGlobalStyle, lightTheme, darkTheme } from 'naive-ui'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()

const naiveTheme = computed(() => themeStore.isDark ? darkTheme : lightTheme)

// Naive theme overrides — Aura tokens
const themeOverrides = computed(() => ({
  common: {
    primaryColor: themeStore.isDark ? '#2dd4bf' : '#0d9488',
    primaryColorHover: themeStore.isDark ? '#5eead4' : '#0f766e',
    primaryColorPressed: themeStore.isDark ? '#14b8a6' : '#115e59',
    borderRadius: '12px',
    fontFamily: 'ui-sans-system, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
  },
  Card: {
    borderRadius: '12px',
  },
  Layout: {
    color: themeStore.isDark ? '#0f172a' : '#f8fafc',
  }
}))
</script>

<template>
  <n-config-provider :theme="naiveTheme" :theme-overrides="themeOverrides">
    <n-global-style />
    <router-view />
  </n-config-provider>
</template>
