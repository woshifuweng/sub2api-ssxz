<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
        <div>
          <h1 class="text-xl font-semibold text-gray-900 dark:text-white">支付配置</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            管理充值入口、支付方式和支付渠道。订单入账以后端账本为准。
          </p>
        </div>
        <div class="flex items-center gap-2">
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadAll">
            <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
            刷新
          </button>
          <button type="button" class="btn btn-primary" :disabled="savingConfig || loading" @click="saveConfig">
            保存配置
          </button>
        </div>
      </div>

      <div v-if="loading && !configLoaded" class="flex items-center justify-center py-12">
        <LoadingSpinner />
      </div>

      <template v-else>
        <section class="card p-5">
          <div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_260px]">
            <div class="space-y-5">
              <div class="flex items-center justify-between rounded-lg border border-gray-200 p-4 dark:border-dark-700">
                <div>
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">充值支付入口</h2>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    关闭后普通用户不会看到正式充值入口，管理员仍可进入本页配置。
                  </p>
                </div>
                <label class="inline-flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-300">
                  <input v-model="form.enabled" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                  {{ form.enabled ? '已开启' : '已关闭' }}
                </label>
              </div>

              <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
                <label class="block">
                  <span class="input-label">最小充值金额</span>
                  <input v-model.number="form.min_amount" type="number" min="0" step="0.01" class="input" />
                </label>
                <label class="block">
                  <span class="input-label">最大充值金额</span>
                  <input v-model.number="form.max_amount" type="number" min="0" step="0.01" class="input" />
                </label>
                <label class="block">
                  <span class="input-label">每日充值上限</span>
                  <input v-model.number="form.daily_limit" type="number" min="0" step="0.01" class="input" />
                </label>
                <label class="block">
                  <span class="input-label">订单超时分钟</span>
                  <input v-model.number="form.order_timeout_minutes" type="number" min="1" step="1" class="input" />
                </label>
                <label class="block">
                  <span class="input-label">最大待支付订单</span>
                  <input v-model.number="form.max_pending_orders" type="number" min="1" step="1" class="input" />
                </label>
                <label class="block">
                  <span class="input-label">到账倍率</span>
                  <input v-model.number="form.balance_recharge_multiplier" type="number" min="0.01" step="0.01" class="input" />
                </label>
                <label class="block">
                  <span class="input-label">充值手续费率 %</span>
                  <input v-model.number="form.recharge_fee_rate" type="number" min="0" max="100" step="0.01" class="input" />
                </label>
                <label class="block">
                  <span class="input-label">余额充值</span>
                  <select v-model="balanceMode" class="input">
                    <option value="enabled">允许余额充值</option>
                    <option value="disabled">禁用余额充值</option>
                  </select>
                </label>
              </div>

              <div>
                <p class="input-label">启用支付方式</p>
                <div class="mt-2 flex flex-wrap gap-2">
                  <label
                    v-for="option in paymentTypeOptions"
                    :key="option.value"
                    class="inline-flex cursor-pointer items-center gap-2 rounded-lg border px-3 py-2 text-sm transition-colors"
                    :class="isPaymentTypeEnabled(option.value)
                      ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/20 dark:text-primary-200'
                      : 'border-gray-200 text-gray-600 hover:border-gray-300 dark:border-dark-700 dark:text-gray-300'"
                  >
                    <input
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                      :checked="isPaymentTypeEnabled(option.value)"
                      @change="togglePaymentType(option.value)"
                    />
                    {{ option.label }}
                  </label>
                </div>
              </div>

              <label class="block">
                <span class="input-label">充值页说明</span>
                <textarea
                  v-model="form.help_text"
                  rows="3"
                  class="input"
                  placeholder="例如：充值会增加账户额度，用量和扣费记录以后端账本为准。"
                />
              </label>
            </div>

            <aside class="rounded-lg border border-gray-200 p-4 dark:border-dark-700">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">当前状态</p>
              <dl class="mt-3 space-y-3 text-sm">
                <div>
                  <dt class="text-gray-500 dark:text-gray-400">支付入口</dt>
                  <dd :class="form.enabled ? 'text-green-600 dark:text-green-400' : 'text-gray-500 dark:text-gray-400'">
                    {{ form.enabled ? '已开启' : '已关闭' }}
                  </dd>
                </div>
                <div>
                  <dt class="text-gray-500 dark:text-gray-400">支付方式</dt>
                  <dd class="text-gray-900 dark:text-white">{{ enabledPaymentTypeLabels || '未选择' }}</dd>
                </div>
                <div>
                  <dt class="text-gray-500 dark:text-gray-400">支付渠道</dt>
                  <dd class="text-gray-900 dark:text-white">{{ enabledProviderCount }} / {{ providers.length }}</dd>
                </div>
              </dl>
            </aside>
          </div>
        </section>

        <PaymentProviderList
          :providers="providers"
          :loading="providersLoading"
          :can-create="true"
          :enabled-payment-types="form.enabled_payment_types"
          :all-payment-types="allPaymentTypes"
          redirect-label="跳转支付"
          @refresh="loadProviders"
          @create="openCreateProvider"
          @edit="openEditProvider"
          @delete="confirmDeleteProvider"
          @toggle-field="toggleProviderField"
          @toggle-type="toggleProviderType"
          @reorder="reorderProviders"
        />
      </template>

      <PaymentProviderDialog
        ref="providerDialogRef"
        :show="showProviderDialog"
        :saving="savingProvider"
        :editing="editingProvider"
        :all-key-options="providerKeyOptions"
        :enabled-key-options="providerKeyOptions"
        :all-payment-types="allPaymentTypes"
        redirect-label="跳转支付"
        @close="showProviderDialog = false"
        @save="saveProvider"
      />

      <ConfirmDialog
        :show="!!deletingProvider"
        title="删除支付渠道"
        :message="deletingProvider ? `确定删除 ${deletingProvider.name}？` : ''"
        confirm-text="删除"
        danger
        @confirm="deleteProvider"
        @cancel="deletingProvider = null"
      />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { useAppStore } from '@/stores/app'
import { adminPaymentAPI, type AdminPaymentConfig, type UpdatePaymentConfigRequest } from '@/api/admin/payment'
import type { ProviderInstance } from '@/types/payment'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Icon from '@/components/icons/Icon.vue'
import PaymentProviderDialog from '@/components/payment/PaymentProviderDialog.vue'
import PaymentProviderList from '@/components/payment/PaymentProviderList.vue'
import type { TypeOption } from '@/components/payment/providerConfig'

type ProviderPayload = {
  provider_key: string
  name: string
  supported_types: string[]
  enabled: boolean
  payment_mode: string
  refund_enabled: boolean
  allow_user_refund: boolean
  config: Record<string, string>
  limits: string
}

type PaymentProviderDialogExpose = {
  reset: (defaultKey: string) => void
  loadProvider: (provider: ProviderInstance) => void
}

const appStore = useAppStore()

const paymentTypeOptions: TypeOption[] = [
  { value: 'easypay', label: '易支付' },
  { value: 'alipay', label: '支付宝' },
  { value: 'wxpay', label: '微信支付' },
  { value: 'stripe', label: 'Stripe' }
]

const allPaymentTypes: TypeOption[] = [
  ...paymentTypeOptions,
  { value: 'card', label: '银行卡' },
  { value: 'link', label: 'Link' }
]

const providerKeyOptions = paymentTypeOptions

const defaultConfig: AdminPaymentConfig = {
  enabled: false,
  min_amount: 1,
  max_amount: 0,
  daily_limit: 0,
  order_timeout_minutes: 30,
  max_pending_orders: 3,
  enabled_payment_types: [],
  balance_disabled: false,
  balance_recharge_multiplier: 1,
  recharge_fee_rate: 0,
  load_balance_strategy: 'round-robin',
  product_name_prefix: '',
  product_name_suffix: '',
  help_image_url: '',
  help_text: ''
}

const form = reactive<AdminPaymentConfig>({ ...defaultConfig })
const providers = ref<ProviderInstance[]>([])
const loading = ref(false)
const configLoaded = ref(false)
const providersLoading = ref(false)
const savingConfig = ref(false)
const savingProvider = ref(false)
const showProviderDialog = ref(false)
const editingProvider = ref<ProviderInstance | null>(null)
const deletingProvider = ref<ProviderInstance | null>(null)
const providerDialogRef = ref<PaymentProviderDialogExpose | null>(null)

const balanceMode = computed({
  get: () => (form.balance_disabled ? 'disabled' : 'enabled'),
  set: (value: string) => {
    form.balance_disabled = value === 'disabled'
  }
})

const enabledProviderCount = computed(() => providers.value.filter((provider) => provider.enabled).length)

const enabledPaymentTypeLabels = computed(() =>
  paymentTypeOptions
    .filter((option) => form.enabled_payment_types.includes(option.value))
    .map((option) => option.label)
    .join('、')
)

function applyConfig(config: Partial<AdminPaymentConfig>) {
  Object.assign(form, {
    ...defaultConfig,
    ...config,
    enabled_payment_types: Array.isArray(config.enabled_payment_types)
      ? [...config.enabled_payment_types]
      : []
  })
}

async function loadConfig() {
  const response = await adminPaymentAPI.getConfig()
  applyConfig(response.data)
  configLoaded.value = true
}

async function loadProviders() {
  providersLoading.value = true
  try {
    const response = await adminPaymentAPI.getProviders()
    providers.value = response.data || []
  } finally {
    providersLoading.value = false
  }
}

async function loadAll() {
  loading.value = true
  try {
    await Promise.all([loadConfig(), loadProviders()])
  } catch (error) {
    appStore.showError('支付配置加载失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

function isPaymentTypeEnabled(type: string): boolean {
  return form.enabled_payment_types.includes(type)
}

function togglePaymentType(type: string) {
  if (isPaymentTypeEnabled(type)) {
    form.enabled_payment_types = form.enabled_payment_types.filter((item) => item !== type)
    return
  }
  form.enabled_payment_types = [...form.enabled_payment_types, type]
}

function positiveNumber(value: number | undefined, fallback: number): number {
  if (typeof value !== 'number' || Number.isNaN(value) || value <= 0) return fallback
  return value
}

function nonNegativeNumber(value: number | undefined): number {
  if (typeof value !== 'number' || Number.isNaN(value) || value < 0) return 0
  return value
}

function configPayload(): UpdatePaymentConfigRequest {
  return {
    enabled: !!form.enabled,
    min_amount: nonNegativeNumber(form.min_amount),
    max_amount: nonNegativeNumber(form.max_amount),
    daily_limit: nonNegativeNumber(form.daily_limit),
    order_timeout_minutes: Math.round(positiveNumber(form.order_timeout_minutes, 30)),
    max_pending_orders: Math.round(positiveNumber(form.max_pending_orders, 3)),
    enabled_payment_types: [...form.enabled_payment_types],
    balance_disabled: !!form.balance_disabled,
    balance_recharge_multiplier: positiveNumber(form.balance_recharge_multiplier, 1),
    recharge_fee_rate: nonNegativeNumber(form.recharge_fee_rate),
    load_balance_strategy: form.load_balance_strategy || 'round-robin',
    product_name_prefix: form.product_name_prefix || '',
    product_name_suffix: form.product_name_suffix || '',
    help_image_url: form.help_image_url || '',
    help_text: form.help_text || ''
  }
}

async function saveConfig() {
  savingConfig.value = true
  try {
    await adminPaymentAPI.updateConfig(configPayload())
    appStore.showSuccess('支付配置已保存')
    await loadConfig()
    await appStore.fetchPublicSettings?.(true)
  } catch (error) {
    appStore.showError('支付配置保存失败')
  } finally {
    savingConfig.value = false
  }
}

async function openCreateProvider() {
  editingProvider.value = null
  showProviderDialog.value = true
  await nextTick()
  providerDialogRef.value?.reset('easypay')
}

async function openEditProvider(provider: ProviderInstance) {
  editingProvider.value = provider
  showProviderDialog.value = true
  await nextTick()
  providerDialogRef.value?.loadProvider(provider)
}

async function saveProvider(payload: ProviderPayload) {
  savingProvider.value = true
  try {
    if (editingProvider.value) {
      await adminPaymentAPI.updateProvider(editingProvider.value.id, payload)
    } else {
      await adminPaymentAPI.createProvider(payload)
    }
    appStore.showSuccess('支付渠道已保存')
    showProviderDialog.value = false
    await loadProviders()
  } catch (error) {
    appStore.showError('支付渠道保存失败')
  } finally {
    savingProvider.value = false
  }
}

async function updateProvider(provider: ProviderInstance, patch: Partial<ProviderInstance>) {
  await adminPaymentAPI.updateProvider(provider.id, patch)
  await loadProviders()
}

async function toggleProviderField(provider: ProviderInstance, field: 'enabled' | 'refund_enabled' | 'allow_user_refund') {
  try {
    await updateProvider(provider, { [field]: !provider[field] })
  } catch (error) {
    appStore.showError('支付渠道更新失败')
  }
}

async function toggleProviderType(provider: ProviderInstance, type: string) {
  const supported = provider.supported_types.includes(type)
    ? provider.supported_types.filter((item) => item !== type)
    : [...provider.supported_types, type]
  try {
    await updateProvider(provider, { supported_types: supported })
  } catch (error) {
    appStore.showError('支付渠道更新失败')
  }
}

async function reorderProviders(updates: { id: number; sort_order: number }[]) {
  try {
    await Promise.all(updates.map((item) => adminPaymentAPI.updateProvider(item.id, { sort_order: item.sort_order })))
    await loadProviders()
  } catch (error) {
    appStore.showError('支付渠道排序失败')
  }
}

function confirmDeleteProvider(provider: ProviderInstance) {
  deletingProvider.value = provider
}

async function deleteProvider() {
  if (!deletingProvider.value) return
  const providerID = deletingProvider.value.id
  try {
    await adminPaymentAPI.deleteProvider(providerID)
    appStore.showSuccess('支付渠道已删除')
    deletingProvider.value = null
    await loadProviders()
  } catch (error) {
    appStore.showError('支付渠道删除失败')
  }
}

onMounted(loadAll)
</script>
