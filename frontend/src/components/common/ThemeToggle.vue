<template>
  <button
    type="button"
    class="theme-toggle"
    :title="label"
    :aria-label="label"
    :aria-pressed="isDark"
    @click="toggleTheme"
  >
    <Icon :name="isDark ? 'sun' : 'moon'" size="sm" />
  </button>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { setSafeLocalStorageItem } from '@/utils/safeStorage'

const { t } = useI18n()
const isDark = ref(typeof document !== 'undefined' && document.documentElement.classList.contains('dark'))
const label = computed(() => (isDark.value ? t('nav.lightMode') : t('nav.darkMode')))

function syncThemeState() {
  isDark.value = document.documentElement.classList.contains('dark')
}

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  setSafeLocalStorageItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(syncThemeState)
</script>

<style scoped>
.theme-toggle {
  display: inline-flex;
  width: 2.2rem;
  height: 2.2rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  color: var(--ssxz-body, currentColor);
  transition: background-color 150ms ease, color 150ms ease;
}

.theme-toggle:hover {
  background: color-mix(in srgb, var(--ssxz-primary, currentColor) 10%, transparent);
  color: var(--ssxz-text, currentColor);
}
</style>
