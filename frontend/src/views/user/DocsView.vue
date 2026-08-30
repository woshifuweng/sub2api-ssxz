<template>
  <AppLayout>
    <div class="space-y-6">
      <header>
        <p class="text-xs font-semibold uppercase tracking-[0.18em] text-primary-600 dark:text-primary-400">
          {{ t('docs.eyebrow') }}
        </p>
        <h1 class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ t('docs.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('docs.description') }}</p>
      </header>

      <div class="grid gap-6 lg:grid-cols-[14rem_minmax(0,1fr)]">
        <nav class="card h-fit space-y-2 p-4 lg:sticky lg:top-24" :aria-label="t('docs.title')">
          <p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">
            {{ t('docs.clientAccess') }}
          </p>
          <a href="#cc-switch-guide" class="block rounded-xl bg-primary-50 p-3 text-sm font-medium text-primary-700 dark:bg-primary-950/40 dark:text-primary-300">
            {{ t('docs.ccSwitch') }}
            <span class="mt-1 block text-xs font-normal">{{ t('docs.ccSwitchHint') }}</span>
          </a>
          <RouterLink to="/app/keys" class="block rounded-xl p-3 text-sm text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-800">
            {{ t('docs.apiKeys') }}
            <span class="mt-1 block text-xs text-gray-500">{{ t('docs.apiKeysHint') }}</span>
          </RouterLink>
        </nav>

        <section class="card min-w-0 p-5 md:p-8">
          <CcSwitchGuide />
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import CcSwitchGuide from '@/components/docs/CcSwitchGuide.vue'

const { t } = useI18n()
</script>
