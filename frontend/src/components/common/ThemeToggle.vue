<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Moon, Sun } from '@lucide/vue'
import { useI18n } from 'vue-i18n'
import { cn } from '@/lib/utils'
import { setSafeLocalStorageItem } from '@/utils/safeStorage'

const { t } = useI18n()
const isDark = ref(
  typeof document !== 'undefined' && document.documentElement.classList.contains('dark')
)
const label = computed(() => (isDark.value ? t('nav.lightMode') : t('nav.darkMode')))

function syncThemeState(): void {
  isDark.value = document.documentElement.classList.contains('dark')
}

function toggle(): void {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  setSafeLocalStorageItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(syncThemeState)
</script>

<template>
  <button
    type="button"
    :class="
      cn(
        'relative grid h-8 w-16 shrink-0 grid-cols-2 items-center rounded-full border p-1 transition-colors duration-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2 dark:focus-visible:ring-offset-dark-900',
        isDark ? 'border-zinc-700 bg-zinc-950' : 'border-zinc-300 bg-white'
      )
    "
    role="button"
    :aria-label="label"
    :aria-pressed="isDark"
    :title="label"
    @click="toggle"
    @keydown.enter.prevent="toggle"
    @keydown.space.prevent="toggle"
  >
    <span
      class="absolute left-1 top-1 h-6 w-6 rounded-full shadow-sm ring-1 ring-black/10 transition-transform duration-200 ease-out"
      :class="isDark ? 'translate-x-0 bg-zinc-800' : 'translate-x-8 bg-zinc-200'"
      aria-hidden="true"
    />
    <span class="relative z-10 flex h-6 w-6 items-center justify-center">
      <Moon
        class="h-4 w-4 transition-colors"
        :class="isDark ? 'text-white' : 'text-zinc-400'"
        :stroke-width="1.75"
        aria-hidden="true"
      />
    </span>
    <span class="relative z-10 flex h-6 w-6 items-center justify-center">
      <Sun
        class="h-4 w-4 transition-colors"
        :class="isDark ? 'text-zinc-500' : 'text-zinc-800'"
        :stroke-width="1.75"
        aria-hidden="true"
      />
    </span>
  </button>
</template>
