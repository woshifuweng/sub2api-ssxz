<template>
  <AppLayout>
    <div class="space-y-6">
      <header class="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.18em] text-primary-600 dark:text-primary-400">
            {{ t('reseller.eyebrow') }}
          </p>
          <h1 class="mt-1 text-2xl font-bold text-gray-900 dark:text-white">{{ title }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ description }}</p>
        </div>

        <nav class="flex flex-wrap gap-2" :aria-label="t('nav.reseller')">
          <RouterLink
            v-for="item in navigation"
            :key="item.to"
            :to="item.to"
            class="btn btn-sm"
            :class="route.path === item.to ? 'btn-primary' : 'btn-secondary'"
          >
            {{ item.label }}
          </RouterLink>
        </nav>
      </header>

      <slot />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useResellerNavRole } from '@/composables/useResellerNavRole'

defineProps<{
  title: string
  description: string
}>()

const { t } = useI18n()
const route = useRoute()
const { isManager } = useResellerNavRole()

const navigation = computed(() => [
  { to: '/app/reseller', label: t('reseller.nav.dashboard') },
  { to: '/app/reseller/withdrawals', label: t('reseller.nav.withdrawals') },
  { to: '/app/reseller/recruits', label: t('reseller.nav.recruits') },
  { to: '/app/reseller/commission', label: t('reseller.nav.commission') },
  { to: '/app/reseller/invite', label: t('reseller.nav.invite') },
  ...(isManager.value
    ? [{ to: '/app/reseller/manager', label: t('reseller.nav.manager') }]
    : [])
])
</script>
