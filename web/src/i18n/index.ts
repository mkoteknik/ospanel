import { createI18n } from 'vue-i18n'
import tr from './locales/tr.json'
import en from './locales/en.json'

export type Locale = 'tr' | 'en'

const stored = (localStorage.getItem('locale') as Locale | null)
const browser = navigator.language.toLowerCase().startsWith('tr') ? 'tr' : 'en'
const defaultLocale: Locale = stored || (browser as Locale) || 'tr'

export const i18n = createI18n({
  legacy: false,
  locale: defaultLocale,
  fallbackLocale: 'en',
  messages: { tr, en },
  globalInjection: true,
})

export function setLocale(locale: Locale) {
  i18n.global.locale.value = locale
  localStorage.setItem('locale', locale)
  document.documentElement.lang = locale
}

export function getLocale(): Locale {
  return i18n.global.locale.value as Locale
}

// initialize html lang
document.documentElement.lang = defaultLocale
