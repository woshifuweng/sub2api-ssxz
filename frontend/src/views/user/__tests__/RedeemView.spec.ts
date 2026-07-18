import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, redeemAPI, authAPI, authStore, appStore, subscriptionStore } = vi.hoisted(() => ({
  routeState: {
    path: '/app/redeem'
  },
  redeemAPI: {
    getHistory: vi.fn(),
    redeem: vi.fn()
  },
  authAPI: {
    getPublicSettings: vi.fn()
  },
  authStore: {
    user: {
      balance: 49.4,
      concurrency: 10
    },
    refreshUser: vi.fn()
  },
  appStore: {
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn()
  },
  subscriptionStore: {
    fetchActiveSubscriptions: vi.fn()
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const translations: Record<string, string> = {
        'redeem.errors.REDEEM_CODE_NOT_FOUND': '兑换码不存在或已失效，请检查后重试。',
        'redeem.errors.REDEEM_CODE_USED': '该兑换码已被使用。',
        'redeem.errors.REDEEM_RATE_LIMITED': '失败次数过多，请稍后再试。',
        'redeem.errors.REDEEM_CODE_LOCKED': '该兑换码正在处理中，请稍后重试。',
        'redeem.errors.REDEEM_CODE_INVALID': '该兑换码无效，请核对后再试。',
        'auth.completeVerification': '请完成验证',
        'redeem.failedToRedeem': '兑换失败，请检查兑换码后重试。',
        'redeem.historyLoadFailed': 'Failed to load redeem history. Please retry.',
        'redeem.retryHistory': 'Reload',
        'redeem.balanceRedeemResult': `Credited $${params?.amount ?? '0.00'} balance.`,
        'redeem.balanceRefreshHint': 'Displayed balance was refreshed from the backend account.',
        'redeem.concurrencyRedeemResult': `Added ${params?.amount ?? 0} concurrent request quota.`,
        'redeem.subscriptionRedeemResult': `Granted access to ${params?.groupName ?? 'subscription group'}.`,
        'redeem.subscriptionDays': `${params?.days} days`,
        'redeem.title': 'Redeem Code'
      }
      return translations[key] ?? key
    }
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/stores/subscriptions', () => ({
  useSubscriptionStore: () => subscriptionStore
}))

vi.mock('@/api', () => ({
  redeemAPI,
  authAPI
}))

vi.mock('@/utils/format', () => ({
  formatDateTime: (value: string) => value
}))

vi.mock('@/components/layout/AppLayout.vue', () => ({
  default: {
    name: 'AppLayout',
    template: '<main data-testid="app-layout"><slot /></main>'
  }
}))

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: '<main data-testid="app-section-shell"><h1>{{ title }}</h1><p>{{ subtitle }}</p><slot /></main>'
  }
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    props: ['name', 'size'],
    template: '<span class="icon-stub" />'
  }
}))

import RedeemView from '../RedeemView.vue'

describe('RedeemView', () => {
  beforeEach(() => {
    routeState.path = '/app/redeem'
    redeemAPI.getHistory.mockReset()
    redeemAPI.redeem.mockReset()
    authAPI.getPublicSettings.mockReset()
    authStore.refreshUser.mockReset()
    appStore.showError.mockReset()
    appStore.showSuccess.mockReset()
    appStore.showWarning.mockReset()
    subscriptionStore.fetchActiveSubscriptions.mockReset()
    redeemAPI.getHistory.mockResolvedValue([])
    authAPI.getPublicSettings.mockResolvedValue({ contact_info: '' })
  })

  it('renders the redeem form inside the user workbench shell on /app/redeem', async () => {
    const wrapper = mount(RedeemView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('redeem.redeemCodeLabel')
    expect(redeemAPI.getHistory).toHaveBeenCalledTimes(1)
  })

  it('keeps the legacy layout when used outside the app workbench', async () => {
    routeState.path = '/redeem'

    const wrapper = mount(RedeemView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(false)
  })

  it('explains a successful balance redeem using the backend DTO shape', async () => {
    redeemAPI.redeem.mockResolvedValue({
      id: 1,
      code: 'BAL-123',
      type: 'balance',
      value: 12.5,
      status: 'used',
      used_at: '2026-07-02T00:00:00Z',
      created_at: '2026-07-02T00:00:00Z'
    })

    const wrapper = mount(RedeemView)
    await flushPromises()

    await wrapper.get('input#code').setValue(' BAL-123 ')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(redeemAPI.redeem).toHaveBeenCalledWith('BAL-123')
    expect(authStore.refreshUser).toHaveBeenCalledTimes(1)
    expect(redeemAPI.getHistory).toHaveBeenCalledTimes(2)
    expect(appStore.showSuccess).toHaveBeenCalledWith('redeem.codeRedeemSuccess')
    expect((wrapper.get('input#code').element as HTMLInputElement).value).toBe('')
    expect(wrapper.text()).toContain('Credited $12.50 balance.')
    expect(wrapper.text()).toContain('Displayed balance was refreshed from the backend account.')
    expect(wrapper.text()).not.toContain('undefined')
  })

  it('keeps the entered code and shows a friendly mapped failure message', async () => {
    redeemAPI.redeem.mockRejectedValue({
      reason: 'REDEEM_CODE_NOT_FOUND',
      message: 'redeem code not found'
    })

    const wrapper = mount(RedeemView)
    await flushPromises()

    await wrapper.get('input#code').setValue('BAD-CODE')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(redeemAPI.redeem).toHaveBeenCalledTimes(1)
    expect((wrapper.get('input#code').element as HTMLInputElement).value).toBe('BAD-CODE')
    expect(wrapper.text()).toContain('兑换码不存在或已失效，请检查后重试。')
    expect(wrapper.text()).not.toContain('redeem code not found')
    expect(appStore.showError).toHaveBeenCalledWith('redeem.redeemFailed')
  })

  it('maps the production interceptor 404 shape without exposing its raw message', async () => {
    redeemAPI.redeem.mockRejectedValue({
      status: 404,
      code: 404,
      message: 'redeem code not found'
    })

    const wrapper = mount(RedeemView)
    await flushPromises()

    await wrapper.get('input#code').setValue('MISSING-CODE')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toContain('兑换码不存在或已失效，请检查后重试。')
    expect(wrapper.text()).not.toContain('redeem code not found')
  })

  it('maps the production interceptor 409 shape without exposing its raw message', async () => {
    redeemAPI.redeem.mockRejectedValue({
      status: 409,
      code: 409,
      message: 'redeem code already used'
    })

    const wrapper = mount(RedeemView)
    await flushPromises()

    await wrapper.get('input#code').setValue('USED-CODE')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toContain('该兑换码已被使用。')
    expect(wrapper.text()).not.toContain('redeem code already used')
  })

  it('maps stable expired, disabled, Turnstile, rate-limit and lock reasons', async () => {
    const cases = [
      ['REDEEM_CODE_EXPIRED', '兑换码不存在或已失效，请检查后重试。'],
      ['REDEEM_CODE_DISABLED', '兑换码不存在或已失效，请检查后重试。'],
      ['REDEEM_CODE_INACTIVE', '兑换码不存在或已失效，请检查后重试。'],
      ['TURNSTILE_VERIFICATION_FAILED', '请完成验证'],
      ['REDEEM_RATE_LIMITED', '失败次数过多，请稍后再试。'],
      ['REDEEM_CODE_LOCKED', '该兑换码正在处理中，请稍后重试。']
    ] as const

    for (const [reason, expected] of cases) {
      redeemAPI.redeem.mockRejectedValueOnce({
        status: 400,
        reason,
        message: `internal ${reason}`
      })

      const wrapper = mount(RedeemView)
      await flushPromises()
      await wrapper.get('input#code').setValue('TEST-CODE')
      await wrapper.get('form').trigger('submit.prevent')
      await flushPromises()

      expect(wrapper.text()).toContain(expected)
      expect(wrapper.text()).not.toContain(`internal ${reason}`)
      wrapper.unmount()
    }
  })

  it('maps the production interceptor 429 shape to a friendly rate-limit message', async () => {
    redeemAPI.redeem.mockRejectedValue({
      status: 429,
      code: 429,
      message: 'too many failed attempts, please try again later'
    })

    const wrapper = mount(RedeemView)
    await flushPromises()

    await wrapper.get('input#code').setValue('RATE-LIMITED')
    await wrapper.get('form').trigger('submit.prevent')
    await flushPromises()

    expect(wrapper.text()).toContain('失败次数过多，请稍后再试。')
    expect(wrapper.text()).not.toContain('too many failed attempts')
  })

  it('never exposes raw network, server or unknown error details', async () => {
    const cases = [
      { status: 0, message: 'Network error. Please check your connection.' },
      { status: 500, reason: 'INTERNAL_FAILURE', message: 'database stack trace' },
      new Error('unexpected internal exception')
    ]

    for (const error of cases) {
      redeemAPI.redeem.mockRejectedValueOnce(error)

      const wrapper = mount(RedeemView)
      await flushPromises()
      await wrapper.get('input#code').setValue('UNKNOWN-FAILURE')
      await wrapper.get('form').trigger('submit.prevent')
      await flushPromises()

      expect(wrapper.text()).toContain('兑换失败，请检查兑换码后重试。')
      expect(wrapper.text()).not.toContain('Network error')
      expect(wrapper.text()).not.toContain('database stack trace')
      expect(wrapper.text()).not.toContain('unexpected internal exception')
      expect(wrapper.text()).not.toContain('INTERNAL_FAILURE')
      wrapper.unmount()
    }
  })

  it('prevents duplicate redeem submissions while a request is pending', async () => {
    let resolveRedeem: (value: unknown) => void = () => {}
    redeemAPI.redeem.mockReturnValue(
      new Promise(resolve => {
        resolveRedeem = resolve
      })
    )

    const wrapper = mount(RedeemView)
    await flushPromises()

    await wrapper.get('input#code').setValue('ONCE-ONLY')
    await wrapper.get('form').trigger('submit.prevent')
    await wrapper.get('form').trigger('submit.prevent')

    expect(redeemAPI.redeem).toHaveBeenCalledTimes(1)
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()

    resolveRedeem({
      id: 2,
      code: 'ONCE-ONLY',
      type: 'balance',
      value: 1,
      status: 'used',
      used_at: '2026-07-02T00:00:00Z',
      created_at: '2026-07-02T00:00:00Z'
    })
    await flushPromises()
  })

  it('shows a retry action when redemption history fails to load', async () => {
    redeemAPI.getHistory
      .mockRejectedValueOnce(new Error('network down'))
      .mockResolvedValueOnce([
        {
          id: 3,
          code: 'HISTORY-OK',
          type: 'balance',
          value: 5,
          status: 'used',
          used_at: '2026-07-02T00:00:00Z',
          created_at: '2026-07-02T00:00:00Z'
        }
      ])

    const wrapper = mount(RedeemView)
    await flushPromises()

    expect(wrapper.text()).toContain('Failed to load redeem history. Please retry.')
    expect(wrapper.text()).toContain('Reload')

    await wrapper.get('[data-testid="redeem-history-retry"]').trigger('click')
    await flushPromises()

    expect(redeemAPI.getHistory).toHaveBeenCalledTimes(2)
    expect(wrapper.text()).toContain('HISTORY')
    expect(wrapper.text()).not.toContain('Failed to load redeem history. Please retry.')
  })
})
