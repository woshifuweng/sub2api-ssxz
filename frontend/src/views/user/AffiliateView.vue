<template>
  <component :is="pageShell" v-bind="pageShellProps">
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else-if="errorMessage" class="card p-6 text-center">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.loadFailedTitle') }}</h2>
        <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-gray-500 dark:text-gray-400">
          {{ t('affiliate.loadFailedBody') }}
        </p>
        <button
          type="button"
          class="btn btn-primary mt-5"
          data-testid="affiliate-retry"
          @click="loadAffiliateDetail()"
        >
          {{ t('affiliate.reload') }}
        </button>
      </div>

      <template v-else-if="detail">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <p class="text-sm text-gray-500">{{ t('affiliate.currentRate') }}</p>
            <p class="mt-2 text-2xl font-semibold text-primary-600">{{ formattedRate }}%</p>
            <p class="mt-1 text-xs text-gray-500">{{ t('affiliate.rateHint') }}</p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500">{{ t('affiliate.inviteCount') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ detail.aff_count }}</p>
            <p class="mt-1 text-xs text-gray-500">{{ t('affiliate.inviteCountHint') }}</p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500">{{ t('affiliate.availableQuota') }}</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600">{{ formatCurrency(detail.aff_quota) }}</p>
            <p class="mt-1 text-xs text-gray-500">{{ t('affiliate.availableQuotaHint') }}</p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500">{{ t('affiliate.totalRewards') }}</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatCurrency(detail.aff_history_quota) }}</p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1 text-xs text-amber-600">
              {{ t('affiliate.pending', { amount: formatCurrency(detail.aff_frozen_quota) }) }}
            </p>
          </div>
        </div>

        <div class="card p-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.exclusiveInvite') }}</h2>
          <p class="mt-1 text-sm text-gray-500">{{ t('affiliate.inviteDescription') }}</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.code') }}</p>
              <div class="flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                <code class="flex-1 truncate text-sm font-semibold text-gray-900 dark:text-white">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm" data-testid="copy-affiliate-code" @click="copyValue(detail.aff_code, t('affiliate.codeCopied'))">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copy') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('affiliate.link') }}</p>
              <div class="flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                <code class="flex-1 truncate text-sm text-gray-700 dark:text-gray-300">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm" data-testid="copy-affiliate-link" @click="copyValue(inviteLink, t('affiliate.linkCopied'))">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copy') }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.settleTitle') }}</h3>
              <p class="mt-1 text-sm text-gray-500">{{ t('affiliate.settleDescription') }}</p>
            </div>
            <button class="btn btn-primary" :disabled="transferring || detail.aff_quota <= 0" @click="transferQuota">
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.settling') : t('affiliate.transfer') }}</span>
            </button>
          </div>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('affiliate.recordsTitle') }}</h3>
          <p class="mt-1 text-sm text-gray-500">{{ t('affiliate.recordsDescription') }}</p>
          <div v-if="detail.invitees.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700">
            {{ t('affiliate.noRecords') }}
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700">
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.user') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.name') }}</th>
                  <th class="px-3 py-2 text-right font-medium">{{ t('affiliate.creditedReward') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.registeredAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in detail.invitees" :key="item.user_id" class="border-b border-gray-100 last:border-b-0 dark:border-dark-800">
                  <td class="px-3 py-3 text-gray-900 dark:text-white">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3 text-right font-medium text-emerald-600">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 text-gray-700 dark:text-gray-300">{{ formatDateTime(item.created_at) || '-' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </template>
    </div>
  </component>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import userAPI from '@/api/user'
import type { UserAffiliateDetail } from '@/types'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useClipboard } from '@/composables/useClipboard'
import { formatCurrency, formatDateTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const appStore = useAppStore()
const authStore = useAuthStore()
const route = useRoute()
const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const useWorkbenchShell = computed(() => route.path === '/app/affiliate')
const pageShell = computed(() => useWorkbenchShell.value ? AppSectionShell : AppLayout)
const pageShellProps = computed(() => useWorkbenchShell.value
  ? {
      title: t('affiliate.title'),
      subtitle: t('affiliate.subtitle'),
      eyebrow: t('affiliate.eyebrow'),
      icon: 'gift'
    }
  : {})

const loading = ref(true)
const transferring = ref(false)
const detail = ref<UserAffiliateDetail | null>(null)
const errorMessage = ref('')

const inviteLink = computed(() => {
  if (!detail.value) return ''
  if (typeof window === 'undefined') return `/register?aff=${encodeURIComponent(detail.value.aff_code)}`
  return `${window.location.origin}/register?aff=${encodeURIComponent(detail.value.aff_code)}`
})

const formattedRate = computed(() => {
  const rate = detail.value?.effective_rebate_rate_percent ?? 0
  const rounded = Math.round(rate * 100) / 100
  return Number.isInteger(rounded) ? String(rounded) : rounded.toString()
})

async function loadAffiliateDetail(silent = false): Promise<void> {
  if (!silent) loading.value = true
  try {
    detail.value = await userAPI.getAffiliateDetail()
    errorMessage.value = ''
  } catch (error) {
    detail.value = null
    errorMessage.value = extractApiErrorMessage(error, t('affiliate.loadFailed'))
    appStore.showError(errorMessage.value)
  } finally {
    if (!silent) loading.value = false
  }
}

async function copyValue(value: string, message: string): Promise<void> {
  if (!value) return
  await copyToClipboard(value, message)
}

async function transferQuota(): Promise<void> {
  if (!detail.value || detail.value.aff_quota <= 0 || transferring.value) return
  transferring.value = true
  try {
    const resp = await userAPI.transferAffiliateQuota()
    appStore.showSuccess(t('affiliate.transferSuccess', { amount: formatCurrency(resp.transferred_quota) }))
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('affiliate.transferFailed')))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
