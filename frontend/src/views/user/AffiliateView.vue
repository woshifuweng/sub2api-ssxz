<template>
  <component :is="pageShell" v-bind="pageShellProps">
    <div class="space-y-6">
      <div v-if="loading" class="flex justify-center py-12">
        <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
      </div>

      <div v-else-if="errorMessage" class="card p-6 text-center">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">推广中心暂时无法加载</h2>
        <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-gray-500 dark:text-gray-400">
          刷新后会重新获取推广码和邀请记录。已有记录会继续保留。
        </p>
        <button
          type="button"
          class="btn btn-primary mt-5"
          data-testid="affiliate-retry"
          @click="loadAffiliateDetail()"
        >
          重新加载
        </button>
      </div>

      <template v-else-if="detail">
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <div class="card p-5">
            <p class="text-sm text-gray-500">当前活动比例</p>
            <p class="mt-2 text-2xl font-semibold text-primary-600">{{ formattedRate }}%</p>
            <p class="mt-1 text-xs text-gray-500">实际奖励以系统记录为准</p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500">邀请人数</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ detail.aff_count }}</p>
            <p class="mt-1 text-xs text-gray-500">通过专属链接注册的人数</p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500">可结算额度</p>
            <p class="mt-2 text-2xl font-semibold text-emerald-600">{{ formatCurrency(detail.aff_quota) }}</p>
            <p class="mt-1 text-xs text-gray-500">可转入账户余额</p>
          </div>
          <div class="card p-5">
            <p class="text-sm text-gray-500">累计奖励</p>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatCurrency(detail.aff_history_quota) }}</p>
            <p v-if="detail.aff_frozen_quota > 0" class="mt-1 text-xs text-amber-600">
              待确认：{{ formatCurrency(detail.aff_frozen_quota) }}
            </p>
          </div>
        </div>

        <div class="card p-6">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">专属邀请</h2>
          <p class="mt-1 text-sm text-gray-500">复制专属链接发给客户，对方通过链接注册后会自动计入邀请记录。有效使用后，奖励会按活动规则进入可结算额度。</p>

          <div class="mt-5 grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">推广码</p>
              <div class="flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                <code class="flex-1 truncate text-sm font-semibold text-gray-900 dark:text-white">{{ detail.aff_code }}</code>
                <button class="btn btn-secondary btn-sm" data-testid="copy-affiliate-code" @click="copyValue(detail.aff_code, '推广码已复制')">
                  <Icon name="copy" size="sm" />
                  <span>复制</span>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <p class="text-sm font-medium text-gray-700 dark:text-gray-300">推广链接</p>
              <div class="flex items-center gap-2 rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-900">
                <code class="flex-1 truncate text-sm text-gray-700 dark:text-gray-300">{{ inviteLink }}</code>
                <button class="btn btn-secondary btn-sm" data-testid="copy-affiliate-link" @click="copyValue(inviteLink, '推广链接已复制')">
                  <Icon name="copy" size="sm" />
                  <span>复制</span>
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="card p-6">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h3 class="text-base font-semibold text-gray-900 dark:text-white">结算到余额</h3>
              <p class="mt-1 text-sm text-gray-500">将可结算额度转入账户余额，到账后可直接用于站内调用。待确认额度会在满足规则后再进入可结算额度。</p>
            </div>
            <button class="btn btn-primary" :disabled="transferring || detail.aff_quota <= 0" @click="transferQuota">
              <Icon v-if="transferring" name="refresh" size="sm" class="animate-spin" />
              <Icon v-else name="dollar" size="sm" />
              <span>{{ transferring ? '结算中...' : '转入余额' }}</span>
            </button>
          </div>
        </div>

        <div class="card p-6">
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">邀请记录</h3>
          <p class="mt-1 text-sm text-gray-500">这里显示通过你推广链接注册的客户，以及已计入的累计奖励。具体到账以系统记录为准。</p>
          <div v-if="detail.invitees.length === 0" class="mt-4 rounded-xl border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500 dark:border-dark-700">
            暂无邀请记录。
          </div>
          <div v-else class="mt-4 overflow-x-auto">
            <table class="w-full min-w-[560px] text-left text-sm">
              <thead>
                <tr class="border-b border-gray-200 text-gray-500 dark:border-dark-700">
                  <th class="px-3 py-2 font-medium">用户</th>
                  <th class="px-3 py-2 font-medium">名称</th>
                  <th class="px-3 py-2 text-right font-medium">已计奖励</th>
                  <th class="px-3 py-2 font-medium">注册时间</th>
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
const { copyToClipboard } = useClipboard()

const useWorkbenchShell = computed(() => route.path === '/app/affiliate')
const pageShell = computed(() => useWorkbenchShell.value ? AppSectionShell : AppLayout)
const pageShellProps = computed(() => useWorkbenchShell.value
  ? {
      title: '推广中心',
      subtitle: '复制专属链接，查看邀请人数、奖励和可转入余额。',
      eyebrow: '账户运营',
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
    errorMessage.value = extractApiErrorMessage(error, '推广数据加载失败，请稍后重试')
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
    appStore.showSuccess(`已转入余额：${formatCurrency(resp.transferred_quota)}`)
    await Promise.all([
      loadAffiliateDetail(true),
      authStore.refreshUser().catch(() => undefined),
    ])
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, '结算失败，请稍后重试'))
  } finally {
    transferring.value = false
  }
}

onMounted(() => {
  void loadAffiliateDetail()
})
</script>
