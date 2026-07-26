import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { routeState, userAPI, appStore, authStore, clipboard } = vi.hoisted(() => ({
  routeState: {
    path: '/app/affiliate'
  },
  userAPI: {
    getAffiliateDetail: vi.fn(),
    transferAffiliateQuota: vi.fn()
  },
  appStore: {
    showError: vi.fn(),
    showSuccess: vi.fn()
  },
  authStore: {
    refreshUser: vi.fn()
  },
  clipboard: {
    copyToClipboard: vi.fn()
  }
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, string>) => {
      const messages: Record<string, string> = {
        'affiliate.title': '邀请返利',
        'affiliate.subtitle': '复制专属链接，查看邀请人数、奖励和可转入余额。',
        'affiliate.eyebrow': '账户运营',
        'affiliate.loadFailedTitle': '邀请返利暂时无法加载',
        'affiliate.loadFailedBody': '刷新后会重新获取推广码和邀请记录。已有记录会继续保留。',
        'affiliate.reload': '重新加载',
        'affiliate.currentRate': '当前活动比例',
        'affiliate.rateHint': '实际奖励以系统记录为准',
        'affiliate.inviteCount': '邀请人数',
        'affiliate.inviteCountHint': '通过专属链接注册的人数',
        'affiliate.availableQuota': '可结算额度',
        'affiliate.availableQuotaHint': '可转入账户余额',
        'affiliate.totalRewards': '累计奖励',
        'affiliate.pending': `待确认：${params?.amount ?? ''}`,
        'affiliate.exclusiveInvite': '专属邀请',
        'affiliate.inviteDescription': '复制专属链接发给客户，对方通过链接注册后会自动计入邀请记录。有效使用后，奖励会按活动规则进入可结算额度。',
        'affiliate.code': '推广码',
        'affiliate.link': '推广链接',
        'affiliate.copy': '复制',
        'affiliate.codeCopied': '推广码已复制',
        'affiliate.linkCopied': '推广链接已复制',
        'affiliate.settleTitle': '结算到余额',
        'affiliate.settleDescription': '将可结算额度转入账户余额。',
        'affiliate.settling': '结算中...',
        'affiliate.transfer': '转入余额',
        'affiliate.recordsTitle': '邀请记录',
        'affiliate.recordsDescription': '邀请记录说明',
        'affiliate.noRecords': '暂无邀请记录。',
        'affiliate.user': '用户',
        'affiliate.name': '名称',
        'affiliate.creditedReward': '已计奖励',
        'affiliate.registeredAt': '注册时间',
        'affiliate.loadFailed': '推广数据加载失败，请稍后重试',
        'affiliate.transferSuccess': `已转入余额：${params?.amount ?? ''}`,
        'affiliate.transferFailed': '结算失败，请稍后重试'
      }
      return messages[key] ?? key
    }
  })
}))

vi.mock('@/api/user', () => ({
  default: userAPI
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => clipboard
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: () => 'safe error'
}))

vi.mock('@/utils/format', () => ({
  formatCurrency: (value: number) => `$${value.toFixed(2)}`,
  formatDateTime: () => '2026-06-19 12:00:00'
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
    template: '<span />'
  }
}))

import AffiliateView from '../AffiliateView.vue'

describe('AffiliateView', () => {
  beforeEach(() => {
    routeState.path = '/app/affiliate'
    userAPI.getAffiliateDetail.mockReset()
    userAPI.transferAffiliateQuota.mockReset()
    appStore.showError.mockReset()
    appStore.showSuccess.mockReset()
    authStore.refreshUser.mockReset()
    clipboard.copyToClipboard.mockReset()
    userAPI.getAffiliateDetail.mockResolvedValue({
      aff_code: 'INVITE123',
      effective_rebate_rate_percent: 12.5,
      aff_count: 2,
      aff_quota: 3,
      aff_history_quota: 5,
      aff_frozen_quota: 0,
      invitees: []
    })
  })

  it('renders inside the user workbench shell on /app/affiliate', async () => {
    const wrapper = mount(AffiliateView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(false)
    expect(wrapper.text()).toContain('邀请返利')
    expect(wrapper.text()).toContain('复制专属链接，查看邀请人数、奖励和可转入余额。')
    expect(wrapper.text()).toContain('专属邀请')
    expect(wrapper.text()).toContain('当前活动比例')
    expect(wrapper.text()).toContain('实际奖励以系统记录为准')
    expect(wrapper.text()).toContain('INVITE123')
    expect(wrapper.find('.affiliate-empty-state .btn').exists()).toBe(false)
    expect(wrapper.findAll('[data-testid="copy-affiliate-link"]')).toHaveLength(1)
    expect(userAPI.getAffiliateDetail).toHaveBeenCalledTimes(1)
  })

  it('shows customer-friendly invite tracking and copies the bound register link', async () => {
    userAPI.getAffiliateDetail.mockResolvedValue({
      aff_code: 'INVITE123',
      effective_rebate_rate_percent: 12.5,
      aff_count: 2,
      aff_quota: 3,
      aff_history_quota: 5,
      aff_frozen_quota: 1.25,
      invitees: [
        {
          user_id: 17,
          email: 'new***@example.com',
          username: 'new user',
          created_at: '2026-06-19T04:00:00Z',
          total_rebate: 2.5
        }
      ]
    })

    const wrapper = mount(AffiliateView)
    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('通过专属链接注册的人数')
    expect(text).toContain('可转入账户余额')
    expect(text).toContain('待确认：$1.25')
    expect(text).toContain('有效使用后，奖励会按活动规则进入可结算额度。')
    expect(text).toContain('已计奖励')
    expect(text).toContain('new***@example.com')
    expect(text).toContain('$2.50')
    expect(text).toContain('/register?aff=INVITE123')

    await wrapper.get('[data-testid="copy-affiliate-link"]').trigger('click')

    expect(clipboard.copyToClipboard).toHaveBeenCalledWith(
      expect.stringContaining('/register?aff=INVITE123'),
      '推广链接已复制'
    )
  })

  it('keeps mobile-stackable copy rows and copies both the code and the bound link', async () => {
    const longCode = 'affiliate-code-that-is-long-enough-to-overflow-a-mobile-viewport'
    userAPI.getAffiliateDetail.mockResolvedValue({
      aff_code: longCode,
      effective_rebate_rate_percent: 10,
      aff_count: 0,
      aff_quota: 0,
      aff_history_quota: 0,
      aff_frozen_quota: 0,
      invitees: []
    })

    const wrapper = mount(AffiliateView)
    await flushPromises()

    const copyRows = wrapper.findAll('.affiliate-copy-row')
    expect(copyRows).toHaveLength(2)
    for (const row of copyRows) {
      expect(row.find('code').exists()).toBe(true)
      expect(row.find('button').exists()).toBe(true)
    }

    await wrapper.get('[data-testid="copy-affiliate-code"]').trigger('click')
    await wrapper.get('[data-testid="copy-affiliate-link"]').trigger('click')
    await flushPromises()

    expect(clipboard.copyToClipboard).toHaveBeenNthCalledWith(1, longCode, '推广码已复制')
    expect(clipboard.copyToClipboard).toHaveBeenNthCalledWith(
      2,
      `${window.location.origin}/register?aff=${encodeURIComponent(longCode)}`,
      '推广链接已复制'
    )
  })

  it('keeps the legacy layout when used outside the app workbench', async () => {
    routeState.path = '/affiliate'

    const wrapper = mount(AffiliateView)
    await flushPromises()

    expect(wrapper.find('[data-testid="app-layout"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="app-section-shell"]').exists()).toBe(false)
  })

  it('shows a retryable fallback when affiliate data cannot be loaded', async () => {
    userAPI.getAffiliateDetail
      .mockRejectedValueOnce(new Error('network down'))
      .mockResolvedValueOnce({
        aff_code: 'RETRY123',
        effective_rebate_rate_percent: 8,
        aff_count: 1,
        aff_quota: 0,
        aff_history_quota: 0,
        aff_frozen_quota: 0,
        invitees: []
      })

    const wrapper = mount(AffiliateView)
    await flushPromises()

    expect(wrapper.text()).toContain('邀请返利暂时无法加载')
    expect(wrapper.text()).toContain('刷新后会重新获取推广码和邀请记录')
    expect(wrapper.find('[data-testid="affiliate-retry"]').exists()).toBe(true)

    await wrapper.get('[data-testid="affiliate-retry"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('RETRY123')
    expect(userAPI.getAffiliateDetail).toHaveBeenCalledTimes(2)
  })
})
