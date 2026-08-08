import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'system'

export const useThemeStore = defineStore('theme', () => {
  const mode = ref<ThemeMode>((localStorage.getItem('aura-theme') as ThemeMode) || 'system')
  const systemDark = ref(window.matchMedia('(prefers-color-scheme: dark)').matches)

  // Listen to system changes
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
    systemDark.value = e.matches
  })

  const isDark = computed(() => {
    if (mode.value === 'system') return systemDark.value
    return mode.value === 'dark'
  })

  const resolved = computed(() => (isDark.value ? 'dark' : 'light'))

  function setMode(m: ThemeMode) {
    mode.value = m
    localStorage.setItem('aura-theme', m)
    apply()
  }

  function toggle() {
    // light -> dark -> system -> light
    if (mode.value === 'light') setMode('dark')
    else if (mode.value === 'dark') setMode('system')
    else setMode('light')
  }

  function apply() {
    document.documentElement.setAttribute('data-theme', resolved.value)
    document.documentElement.style.colorScheme = resolved.value
  }

  watch(resolved, apply, { immediate: true })

  // Initial apply
  apply()

  return { mode, isDark, resolved, setMode, toggle, systemDark }
})
