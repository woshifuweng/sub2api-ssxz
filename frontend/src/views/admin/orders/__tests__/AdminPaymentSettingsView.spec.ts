import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AdminPaymentSettingsView from '../AdminPaymentSettingsView.vue'

const { getConfig, updateConfig, getProviders, showError, showSuccess, fetchPublicSettings } = vi.hoisted(() => ({
  getConfig: vi.fn(),
  updateConfig: vi.fn(),
  getProviders: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  fetchPublicSettings: vi.fn()
}))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: {
    getConfig,
    updateConfig,
    getProviders,
    createProvider: vi.fn(),
    updateProvider: vi.fn(),
    deleteProvider: vi.fn()
  },
  default: {
    getConfig,
    updateConfig,
    getProviders,
    createProvider: vi.fn(),
    updateProvider: vi.fn(),
    deleteProvider: vi.fn()
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    fetchPublicSettings
  })
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const PaymentProviderListStub = {
  props: ['providers', 'loading', 'canCreate', 'enabledPaymentTypes'],
  template: '<div class="provider-list-stub">{{ providers.length }} providers <span v-for="provider in providers" :key="provider.id">{{ provider.name }}</span></div>'
}
const PaymentProviderDialogStub = {
  template: '<div />',
  methods: {
    reset: vi.fn(),
    loadProvider: vi.fn()
  }
}

function mountView() {
  return mount(AdminPaymentSettingsView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        PaymentProviderList: PaymentProviderListStub,
        PaymentProviderDialog: PaymentProviderDialogStub,
        ConfirmDialog: true,
        LoadingSpinner: true,
        Icon: true
      }
    }
  })
}

describe('AdminPaymentSettingsView', () => {
  beforeEach(() => {
    getConfig.mockReset()
    updateConfig.mockReset()
    getProviders.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    fetchPublicSettings.mockReset()

    getConfig.mockResolvedValue({
      data: {
        enabled: false,
        min_amount: 1,
        max_amount: 200,
        daily_limit: 1000,
        order_timeout_minutes: 30,
        max_pending_orders: 3,
        enabled_payment_types: ['easypay', 'stripe'],
        balance_disabled: false,
        balance_recharge_multiplier: 1,
        recharge_fee_rate: 0,
        load_balance_strategy: 'round-robin',
        product_name_prefix: '',
        product_name_suffix: '',
        help_image_url: '',
        help_text: '充值会增加账户额度'
      }
    })
    getProviders.mockResolvedValue({
      data: [
        {
          id: 1,
          provider_key: 'easypay',
          name: 'EasyPay',
          config: {},
          supported_types: ['alipay'],
          enabled: true,
          payment_mode: 'qrcode',
          refund_enabled: false,
          allow_user_refund: false,
          limits: '',
          sort_order: 0
        }
      ]
    })
    updateConfig.mockResolvedValue({ data: { message: 'ok' } })
  })

  it('loads payment config and providers even when payment is disabled', async () => {
    const wrapper = mountView()

    await flushPromises()

    expect(getConfig).toHaveBeenCalledOnce()
    expect(getProviders).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('已关闭')
    expect(wrapper.text()).toContain('EasyPay')
    expect(wrapper.text()).toContain('1 providers')
  })

  it('saves the existing config through the admin payment API', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(updateConfig).toHaveBeenCalledWith(expect.objectContaining({
      enabled: false,
      min_amount: 1,
      max_amount: 200,
      enabled_payment_types: ['easypay', 'stripe'],
      help_text: '充值会增加账户额度'
    }))
    expect(showSuccess).toHaveBeenCalledWith('支付配置已保存')
    expect(fetchPublicSettings).toHaveBeenCalledWith(true)
  })
})
