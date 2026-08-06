<template>
  <component :is="pageShell" v-bind="pageShellProps">
    <div class="affiliate-workbench">
      <div v-if="loading" class="card affiliate-loading-state">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else-if="errorMessage" class="card affiliate-feedback-card p-6 text-center">
        <div class="affiliate-feedback-card__icon" aria-hidden="true">
          <Icon name="exclamationCircle" size="md" />
        </div>
        <h2 class="mt-3 text-base font-semibold text-[var(--ssxz-text)]">{{ t('affiliate.loadFailedTitle') }}</h2>
        <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-[var(--ssxz-text-muted)]">
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
          <div class="card affiliate-stat-card">
            <p class="affiliate-stat-card__label">{{ t('affiliate.currentRate') }}</p>
            <p class="affiliate-stat-card__value">{{ formattedRate }}%</p>
            <p class="affiliate-stat-card__hint">{{ t('affiliate.rateHint') }}</p>
          </div>
          <div class="card affiliate-stat-card">
            <p class="affiliate-stat-card__label">{{ t('affiliate.inviteCount') }}</p>
            <p class="affiliate-stat-card__value">{{ detail.aff_count }}</p>
            <p class="affiliate-stat-card__hint">{{ t('affiliate.inviteCountHint') }}</p>
          </div>
          <div class="card affiliate-stat-card">
            <p class="affiliate-stat-card__label">{{ t('affiliate.availableQuota') }}</p>
            <p class="affiliate-stat-card__value">{{ formatCurrency(detail.aff_quota) }}</p>
            <p class="affiliate-stat-card__hint">{{ t('affiliate.availableQuotaHint') }}</p>
          </div>
          <div class="card affiliate-stat-card">
            <p class="affiliate-stat-card__label">{{ t('affiliate.totalRewards') }}</p>
            <p class="affiliate-stat-card__value">{{ formatCurrency(detail.aff_history_quota) }}</p>
            <p v-if="detail.aff_frozen_quota > 0" class="affiliate-stat-card__pending">
              {{ t('affiliate.pending', { amount: formatCurrency(detail.aff_frozen_quota) }) }}
            </p>
          </div>
        </div>

        <div class="card affiliate-panel p-6">
          <h2 class="text-base font-semibold text-[var(--ssxz-text)]">{{ t('affiliate.exclusiveInvite') }}</h2>
          <p class="mt-1 text-sm text-[var(--ssxz-text-muted)]">{{ t('affiliate.inviteDescription') }}</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-[var(--ssxz-text-secondary)]">{{ t('affiliate.code') }}</p>
              <div class="affiliate-copy-row">
                <code class="flex-1 truncate text-sm font-semibold text-[var(--ssxz-text)]">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm" data-testid="copy-affiliate-code" @click="copyValue(detail.aff_code, t('affiliate.codeCopied'))">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copy') }}</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-[var(--ssxz-text-secondary)]">{{ t('affiliate.link') }}</p>
              <div class="affiliate-copy-row">
                <code class="flex-1 truncate text-sm text-[var(--ssxz-text-secondary)]">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm" data-testid="copy-affiliate-link" @click="copyValue(inviteLink, t('affiliate.linkCopied'))">
                  <Icon name="copy" size="sm" />
                  <span>{{ t('affiliate.copy') }}</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="card affiliate-panel p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-[var(--ssxz-text)]">{{ t('affiliate.settleTitle') }}</h3>
              <p class="mt-1 text-sm text-[var(--ssxz-text-muted)]">{{ t('affiliate.settleDescription') }}</p>
            </div>
            <button class="btn btn-primary" :disabled="transferring || detail.aff_quota <= 0" @click="transferQuota">
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? t('affiliate.settling') : t('affiliate.transfer') }}</span>
            </button>
          </div>
        </div>

        <div class="card affiliate-panel p-6">
          <h3 class="text-base font-semibold text-[var(--ssxz-text)]">{{ t('affiliate.recordsTitle') }}</h3>
          <p class="mt-1 text-sm text-[var(--ssxz-text-muted)]">{{ t('affiliate.recordsDescription') }}</p>
          <div v-if="detail.invitees.length === 0" class="affiliate-empty-state mt-4">
            <div class="affiliate-empty-state__icon"><Icon name="users" size="lg" /></div>
            <strong>{{ t('affiliate.noRecordsTitle', '暂无邀请记录') }}</strong>
            <span>{{ t('affiliate.noRecords') }}</span>
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[760px] text-left text-sm">
              <thead>
                <tr class="border-b border-[var(--ssxz-border)] text-[var(--ssxz-text-muted)]">
                  <th class="px-3 py-2 font-medium">#ID</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.user') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.name') }}</th>
                  <th class="px-3 py-2 font-medium">状态</th>
                  <th class="px-3 py-2 text-right font-medium">总充值</th>
                  <th class="px-3 py-2 text-right font-medium">总消费</th>
                  <th class="px-3 py-2 text-right font-medium">{{ t('affiliate.creditedReward') }}</th>
                  <th class="px-3 py-2 font-medium">{{ t('affiliate.registeredAt') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in detail.invitees" :key="item.user_id" class="border-b border-[var(--ssxz-border)] last:border-b-0">
                  <td class="px-3 py-3 text-[var(--ssxz-text-muted)] tabular-nums">#{{ item.user_id }}</td>
                  <td class="px-3 py-3 text-[var(--ssxz-text)]">{{ item.email || '-' }}</td>
                  <td class="px-3 py-3 text-[var(--ssxz-text-secondary)]">{{ item.username || '-' }}</td>
                  <td class="px-3 py-3">
                    <span :class="statusBadgeClass(item.status)">{{ statusLabel(item.status) }}</span>
                  </td>
                  <td class="px-3 py-3 text-right font-medium text-[var(--ssxz-text)]">{{ formatCurrency(item.total_recharge ?? 0) }}</td>
                  <td class="px-3 py-3 text-right font-medium text-[var(--ssxz-text)]">{{ formatCurrency(item.total_consumption ?? 0) }}</td>
                  <td class="px-3 py-3 text-right font-medium text-[var(--ssxz-text)]">{{ formatCurrency(item.total_rebate) }}</td>
                  <td class="px-3 py-3 text-[var(--ssxz-text-secondary)]">{{ formatDateTime(item.created_at) || '-' }}</td>
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

function statusLabel(status?: string): string {
  if (status === 'disabled') return '已禁用'
  return '正常'
}

function statusBadgeClass(status?: string): string {
  const base = 'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium'
  if (status === 'disabled') return `${base} bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400`
  return `${base} bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400`
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

<style scoped>
.affiliate-workbench {
  display: grid;
  gap: 1.5rem;
}

.affiliate-workbench :deep(.card) {
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow-card);
}

.affiliate-loading-state {
  display: grid;
  min-height: 14rem;
  place-items: center;
}

.affiliate-stat-card {
  min-height: 8rem;
  padding: 1.25rem;
}

.affiliate-stat-card__label,
.affiliate-stat-card__hint {
  color: var(--ssxz-text-muted);
}

.affiliate-stat-card__label {
  font-size: 0.82rem;
  font-weight: 650;
}

.affiliate-stat-card__value {
  margin-top: 0.55rem;
  color: var(--ssxz-text);
  font-size: 1.7rem;
  font-weight: 760;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.affiliate-stat-card__hint,
.affiliate-stat-card__pending {
  margin-top: 0.4rem;
  font-size: 0.75rem;
  line-height: 1.45;
}

.affiliate-stat-card__pending {
  color: var(--ssxz-warning);
}

.affiliate-panel {
  min-width: 0;
}

.affiliate-copy-row {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 0.5rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-button);
  background: var(--ssxz-surface-code);
  padding: 0.5rem 0.65rem;
}

.affiliate-empty-state {
  display: grid;
  min-height: 14rem;
  place-items: center;
  align-content: center;
  gap: 0.75rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface);
  color: var(--ssxz-text-muted);
  padding: 1.5rem;
  text-align: center;
}

.affiliate-empty-state__icon {
  display: grid;
  width: 3.75rem;
  height: 3.75rem;
  place-items: center;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-primary-soft);
  color: var(--ssxz-action);
}

.affiliate-empty-state strong {
  color: var(--ssxz-text);
  font-size: 1rem;
  font-weight: 700;
}

.affiliate-empty-state span {
  max-width: 28rem;
  font-size: 0.875rem;
  line-height: 1.65;
}

.affiliate-feedback-card {
  border-left: 3px solid var(--ssxz-danger);
}

.affiliate-feedback-card__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 0.75rem;
  background: color-mix(in srgb, var(--ssxz-danger) 14%, transparent);
  color: var(--ssxz-danger);
}

@media (max-width: 640px) {
  .affiliate-stat-card {
    min-height: 0;
    padding: 1rem;
  }

  .affiliate-copy-row {
    align-items: stretch;
    flex-direction: column;
  }

  .affiliate-copy-row .btn {
    width: 100%;
  }
}
</style>
