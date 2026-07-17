<template>
  <component :is="pageShell" v-bind="pageShellProps">
    <div class="mx-auto max-w-2xl space-y-6">
      <div class="card redeem-balance-card">
        <div class="redeem-balance-card__icon" aria-hidden="true">
          <Icon name="creditCard" size="lg" />
        </div>
        <div class="min-w-0">
          <p class="redeem-balance-card__label">{{ t('redeem.currentBalance') }}</p>
          <p class="redeem-balance-card__value">
            ${{ user?.balance?.toFixed(2) || '0.00' }}
          </p>
          <p class="redeem-balance-card__meta">
            {{ t('redeem.concurrency') }}:
            {{ user?.unlimited_concurrency ? '∞' : (user?.concurrency || 0) }}
            {{ t('redeem.requests') }}
          </p>
        </div>
      </div>

      <div class="card">
        <div class="p-6">
          <form @submit.prevent="handleRedeem" class="space-y-5">
            <div>
              <label for="code" class="input-label">
                {{ t('redeem.redeemCodeLabel') }}
              </label>
              <div class="relative mt-1">
                <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
                  <Icon name="gift" size="md" class="text-gray-400 dark:text-dark-500" />
                </div>
                <input
                  id="code"
                  v-model="redeemCode"
                  type="text"
                  required
                  :placeholder="t('redeem.redeemCodePlaceholder')"
                  :disabled="submitting"
                  class="input py-3 pl-12 text-lg"
                />
              </div>
              <p class="input-hint">
                {{ t('redeem.redeemCodeHint') }}
              </p>
            </div>

            <button
              type="submit"
              :disabled="!redeemCode.trim() || submitting"
              class="btn btn-primary w-full py-3"
            >
              <Icon v-if="submitting" name="refresh" size="md" class="animate-spin" />
              <Icon v-else name="checkCircle" size="md" />
              {{ submitting ? t('redeem.redeeming') : t('redeem.redeemButton') }}
            </button>
          </form>
        </div>
      </div>

      <transition name="fade">
        <div
          v-if="redeemResult"
          class="card redeem-status-card redeem-status-card--success"
        >
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div
                class="redeem-status-card__icon redeem-status-card__icon--success"
              >
                <Icon name="checkCircle" size="md" class="text-emerald-600 dark:text-emerald-400" />
              </div>
              <div class="min-w-0 flex-1">
                <h3 class="redeem-status-card__title">
                  {{ t('redeem.redeemSuccess') }}
                </h3>
                <div class="mt-2 space-y-1 text-sm text-[var(--ssxz-text-secondary)]">
                  <p class="font-medium">
                    {{ getRedeemResultMessage(redeemResult) }}
                  </p>
                  <p v-if="redeemResult.type === 'balance'" class="text-xs">
                    {{ t('redeem.balanceRefreshHint') }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <transition name="fade">
        <div
          v-if="errorMessage"
          class="card redeem-status-card redeem-status-card--error"
        >
          <div class="p-6">
            <div class="flex items-start gap-4">
              <div
                class="redeem-status-card__icon redeem-status-card__icon--error"
              >
                <Icon
                  name="exclamationCircle"
                  size="md"
                  class="text-red-600 dark:text-red-400"
                />
              </div>
              <div class="min-w-0 flex-1">
                <h3 class="redeem-status-card__title">
                  {{ t('redeem.redeemFailed') }}
                </h3>
                <p class="mt-2 text-sm text-[var(--ssxz-text-secondary)]">
                  {{ errorMessage }}
                </p>
              </div>
            </div>
          </div>
        </div>
      </transition>

      <div
        class="card redeem-info-card"
      >
        <div class="p-6">
          <div class="flex items-start gap-4">
            <div
              class="redeem-info-card__icon"
            >
              <Icon name="infoCircle" size="md" class="text-primary-600 dark:text-primary-400" />
            </div>
            <div class="min-w-0 flex-1">
              <h3 class="text-sm font-semibold text-[var(--ssxz-text)]">
                {{ t('redeem.aboutCodes') }}
              </h3>
              <ul
                class="mt-2 list-inside list-disc space-y-1 text-sm text-[var(--ssxz-text-muted)]"
              >
                <li>{{ t('redeem.codeRule1') }}</li>
                <li>{{ t('redeem.codeRule2') }}</li>
                <li>
                  {{ t('redeem.codeRule3') }}
                  <span
                    v-if="contactInfo"
                    class="ml-1.5 inline-flex items-center rounded-md border border-[var(--ssxz-border)] bg-[var(--ssxz-surface-muted)] px-2 py-0.5 text-xs font-medium text-[var(--ssxz-text)]"
                  >
                    {{ contactInfo }}
                  </span>
                </li>
                <li>{{ t('redeem.codeRule4') }}</li>
              </ul>
            </div>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
            {{ t('redeem.recentActivity') }}
          </h2>
        </div>
        <div class="p-6">
          <div v-if="loadingHistory" class="flex items-center justify-center py-8">
            <Icon name="refresh" size="lg" class="animate-spin text-[var(--ssxz-accent)]" />
          </div>

          <div v-else-if="historyLoadFailed" class="empty-state py-8">
            <div
              class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-red-50 dark:bg-red-900/20"
            >
              <Icon name="exclamationCircle" size="xl" class="text-red-500 dark:text-red-400" />
            </div>
            <p class="text-sm text-gray-600 dark:text-dark-300">
              {{ t('redeem.historyLoadFailed') }}
            </p>
            <button
              type="button"
              data-testid="redeem-history-retry"
              class="btn btn-secondary mt-4"
              @click="fetchHistory"
            >
              {{ t('redeem.retryHistory') }}
            </button>
          </div>

          <div v-else-if="history.length > 0" class="space-y-3">
            <div
              v-for="item in history"
              :key="item.id"
              class="flex items-center justify-between rounded-xl bg-gray-50 p-4 dark:bg-dark-800"
            >
              <div class="flex items-center gap-4">
                <div
                  :class="[
                    'flex h-10 w-10 items-center justify-center rounded-xl',
                    'bg-[var(--ssxz-surface-muted)]'
                  ]"
                >
                  <Icon
                    v-if="isBalanceType(item.type)"
                    name="dollar"
                    size="md"
                    :class="
                      'text-[var(--ssxz-text-secondary)]'
                    "
                  />
                  <Icon
                    v-else-if="isSubscriptionType(item.type)"
                    name="badge"
                    size="md"
                    class="text-[var(--ssxz-text-secondary)]"
                  />
                  <Icon
                    v-else
                    name="bolt"
                    size="md"
                    :class="
                      'text-[var(--ssxz-text-secondary)]'
                    "
                  />
                </div>
                <div>
                  <p class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ getHistoryItemTitle(item) }}
                  </p>
                  <p class="text-xs text-gray-500 dark:text-dark-400">
                    {{ formatDateTime(item.used_at || item.created_at) }}
                  </p>
                </div>
              </div>
              <div class="text-right">
                <p
                  :class="[
                    'text-sm font-semibold',
                    'text-[var(--ssxz-text)]'
                  ]"
                >
                  {{ formatHistoryValue(item) }}
                </p>
                <p
                  v-if="!isAdminAdjustment(item.type)"
                  class="font-mono text-xs text-gray-400 dark:text-dark-500"
                >
                  {{ item.code.slice(0, 8) }}...
                </p>
                <p v-else class="text-xs text-gray-400 dark:text-dark-500">
                  {{ t('redeem.adminAdjustment') }}
                </p>
                <p
                  v-if="item.notes"
                  class="mt-1 max-w-[200px] truncate text-xs italic text-gray-500 dark:text-dark-400"
                  :title="item.notes"
                >
                  {{ item.notes }}
                </p>
              </div>
            </div>
          </div>

          <div v-else class="empty-state py-8">
            <div
              class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-gray-100 dark:bg-dark-800"
            >
              <Icon name="clock" size="xl" class="text-gray-400 dark:text-dark-500" />
            </div>
            <p class="text-sm text-gray-500 dark:text-dark-400">
              {{ t('redeem.historyWillAppear') }}
            </p>
          </div>
        </div>
      </div>
    </div>
  </component>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { redeemAPI, authAPI, type RedeemHistoryItem, type RedeemResult } from '@/api'
import AppLayout from '@/components/layout/AppLayout.vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useSubscriptionStore } from '@/stores/subscriptions'
import { extractI18nErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()
const subscriptionStore = useSubscriptionStore()

const user = computed(() => authStore.user)
const useWorkbenchShell = computed(() => route.path === '/app/redeem')
const pageShell = computed(() => (useWorkbenchShell.value ? AppSectionShell : AppLayout))
const pageShellProps = computed(() =>
  useWorkbenchShell.value
    ? {
        title: t('redeem.title'),
        subtitle: t('redeem.description'),
        eyebrow: t('redeem.accountBilling'),
        icon: 'gift'
      }
    : {}
)

const redeemCode = ref('')
const submitting = ref(false)
const redeemResult = ref<RedeemResult | null>(null)
const errorMessage = ref('')
const history = ref<RedeemHistoryItem[]>([])
const loadingHistory = ref(false)
const historyLoadFailed = ref(false)
const contactInfo = ref('')

const isBalanceType = (type: string) => type === 'balance' || type === 'admin_balance'
const isSubscriptionType = (type: string) => type === 'subscription'
const isAdminAdjustment = (type: string) => type === 'admin_balance' || type === 'admin_concurrency'

const getHistoryItemTitle = (item: RedeemHistoryItem) => {
  if (item.type === 'balance') return t('redeem.balanceAddedRedeem')
  if (item.type === 'admin_balance') {
    return item.value >= 0 ? t('redeem.balanceAddedAdmin') : t('redeem.balanceDeductedAdmin')
  }
  if (item.type === 'concurrency') return t('redeem.concurrencyAddedRedeem')
  if (item.type === 'admin_concurrency') {
    return item.value >= 0 ? t('redeem.concurrencyAddedAdmin') : t('redeem.concurrencyReducedAdmin')
  }
  if (item.type === 'subscription') return t('redeem.subscriptionAssigned')
  return t('common.unknown')
}

const formatHistoryValue = (item: RedeemHistoryItem) => {
  if (isBalanceType(item.type)) {
    const sign = item.value >= 0 ? '+' : ''
    return `${sign}$${item.value.toFixed(2)}`
  }
  if (isSubscriptionType(item.type)) {
    const days = item.validity_days || Math.round(item.value)
    const groupName = item.group?.name || ''
    return groupName ? `${days}${t('redeem.days')} - ${groupName}` : `${days}${t('redeem.days')}`
  }
  const sign = item.value >= 0 ? '+' : ''
  return `${sign}${item.value} ${t('redeem.requests')}`
}

const getRedeemResultMessage = (item: RedeemResult) => {
  if (item.type === 'balance') {
    return t('redeem.balanceRedeemResult', { amount: item.value.toFixed(2) })
  }
  if (item.type === 'concurrency') {
    return t('redeem.concurrencyRedeemResult', { amount: item.value })
  }
  if (item.type === 'subscription') {
    const groupName = item.group?.name || t('redeem.subscriptionAssigned')
    const days = item.validity_days
      ? ` (${t('redeem.subscriptionDays', { days: item.validity_days })})`
      : ''
    return `${t('redeem.subscriptionRedeemResult', { groupName })}${days}`
  }
  return t('redeem.codeRedeemSuccess')
}

const fetchHistory = async () => {
  loadingHistory.value = true
  historyLoadFailed.value = false

  try {
    history.value = await redeemAPI.getHistory()
  } catch {
    history.value = []
    historyLoadFailed.value = true
  } finally {
    loadingHistory.value = false
  }
}

const handleRedeem = async () => {
  if (submitting.value) return

  const code = redeemCode.value.trim()
  if (!code) {
    appStore.showError(t('redeem.pleaseEnterCode'))
    return
  }

  submitting.value = true
  errorMessage.value = ''
  redeemResult.value = null

  try {
    const result = await redeemAPI.redeem(code)
    redeemResult.value = result

    await authStore.refreshUser()

    if (result.type === 'subscription') {
      try {
        await subscriptionStore.fetchActiveSubscriptions(true)
      } catch {
        appStore.showWarning(t('redeem.subscriptionRefreshFailed'))
      }
    }

    redeemCode.value = ''
    await fetchHistory()
    appStore.showSuccess(t('redeem.codeRedeemSuccess'))
  } catch (error) {
    errorMessage.value = extractI18nErrorMessage(
      error,
      t,
      'redeem.errors',
      t('redeem.failedToRedeem')
    )
    appStore.showError(t('redeem.redeemFailed'))
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  await fetchHistory()
  try {
    const settings = await authAPI.getPublicSettings()
    contactInfo.value = settings.contact_info || ''
  } catch {
    contactInfo.value = ''
  }
})
</script>

<style scoped>
.redeem-balance-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem 1.5rem;
}

.redeem-balance-card__icon,
.redeem-info-card__icon {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 3rem;
  height: 3rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.85rem;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-text-secondary);
}

.redeem-balance-card__label {
  color: var(--ssxz-text-muted);
  font-size: 0.82rem;
  font-weight: 650;
}

.redeem-balance-card__value {
  margin-top: 0.15rem;
  color: var(--ssxz-text);
  font-size: clamp(1.8rem, 4vw, 2.35rem);
  font-weight: 760;
  letter-spacing: -0.02em;
  line-height: 1.1;
}

.redeem-balance-card__meta {
  margin-top: 0.35rem;
  color: var(--ssxz-text-muted);
  font-size: 0.78rem;
}

.redeem-status-card,
.redeem-info-card {
  padding: 1.25rem 1.5rem;
}

.redeem-status-card {
  border-left-width: 3px;
}

.redeem-status-card--success {
  border-left-color: var(--ssxz-success);
}

.redeem-status-card--error {
  border-left-color: var(--ssxz-danger);
}

.redeem-status-card__icon {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.5rem;
  border-radius: 0.75rem;
}

.redeem-status-card__icon--success {
  background: color-mix(in srgb, var(--ssxz-success) 14%, transparent);
  color: var(--ssxz-success);
}

.redeem-status-card__icon--error {
  background: color-mix(in srgb, var(--ssxz-danger) 14%, transparent);
  color: var(--ssxz-danger);
}

.redeem-status-card__title {
  color: var(--ssxz-text);
  font-size: 0.9rem;
  font-weight: 720;
}

.redeem-info-card {
  border-left: 3px solid var(--ssxz-border-strong);
}

@media (max-width: 640px) {
  .redeem-balance-card,
  .redeem-status-card,
  .redeem-info-card {
    padding: 1rem;
  }
}

.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
