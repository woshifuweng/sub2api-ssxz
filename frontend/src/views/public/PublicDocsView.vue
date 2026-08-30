<template>
  <div class="min-h-screen bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-gray-100">
    <header class="glass sticky top-0 z-20 border-b border-gray-200 dark:border-dark-700">
      <div class="mx-auto flex h-16 max-w-7xl items-center justify-between gap-4 px-4 md:px-6">
        <RouterLink to="/home" class="flex items-center gap-3 font-semibold">
          <img :src="siteLogo || '/logo.svg'" class="h-9 w-9 rounded-xl object-contain" alt="" />
          <span>{{ siteName }}</span>
        </RouterLink>
        <div class="flex items-center gap-2">
          <RouterLink to="/home" class="btn btn-ghost btn-sm">{{ t('docs.backHome') }}</RouterLink>
          <RouterLink :to="authStore.isAuthenticated ? '/app/docs' : '/login'" class="btn btn-primary btn-sm">
            {{ authStore.isAuthenticated ? t('docs.start') : t('docs.login') }}
          </RouterLink>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-7xl space-y-8 px-4 py-10 md:px-6 md:py-16">
      <header class="max-w-3xl">
        <p class="text-xs font-semibold uppercase tracking-[0.18em] text-primary-600 dark:text-primary-400">
          {{ t('docs.eyebrow') }}
        </p>
        <h1 class="mt-2 text-3xl font-bold md:text-4xl">{{ t('docs.title') }}</h1>
        <p class="mt-3 text-base text-gray-600 dark:text-gray-300">{{ t('docs.publicDescription') }}</p>
      </header>

      <section class="card p-5 md:p-8">
        <CcSwitchGuide />
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import CcSwitchGuide from '@/components/docs/CcSwitchGuide.vue'

const { t } = useI18n()
const appStore = useAppStore()
const authStore = useAuthStore()
const siteName = computed(() => appStore.siteName)
const siteLogo = computed(() => appStore.siteLogo)
</script>
