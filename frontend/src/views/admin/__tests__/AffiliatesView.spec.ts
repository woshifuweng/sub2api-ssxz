import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { affiliateAPI, settingsAPI, showError, showSuccess, routeQuery, copyToClipboard } =
  vi.hoisted(() => ({
    affiliateAPI: {
      listUsers: vi.fn(),
      lookupUsers: vi.fn(),
      getUserOverview: vi.fn(),
      updateUserSettings: vi.fn(),
      clearUserSettings: vi.fn(),
      batchSetRate: vi.fn()
    },
    settingsAPI: {
      getSettings: vi.fn()
    },
    showError: vi.fn(),
    showSuccess: vi.fn(),
    copyToClipboard: vi.fn(),
    routeQuery: {} as Record<string, string>
  }))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    affiliate: affiliateAPI,
    settings: settingsAPI
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess
  })
}))

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard
  })
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: routeQuery
  })
}))

import AffiliatesView from '../AffiliatesView.vue'

function mountView() {
  return mount(AffiliatesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<section><slot name="filters" /><slot name="table" /><slot name="pagination" /></section>'
        },
        DataTable: {
          props: ['data'],
          template: `
            <div>
              <div v-for="row in data" :key="row.user_id" class="row">
                <slot name="cell-user" :row="row" :value="row.user_id" />
                <slot name="cell-aff_code" :row="row" :value="row.aff_code" />
                <slot name="cell-aff_count" :row="row" :value="row.aff_count" />
                <slot name="cell-invitee_recharge_total" :row="row" :value="row.invitee_recharge_total" />
                <slot name="cell-accrued_rebate_total" :row="row" :value="row.accrued_rebate_total" />
                <slot name="cell-aff_frozen_quota" :row="row" :value="row.aff_frozen_quota" />
                <slot name="cell-aff_quota" :row="row" :value="row.aff_quota" />
                <slot name="cell-transferred_rebate_total" :row="row" :value="row.transferred_rebate_total" />
                <slot name="cell-aff_rebate_rate_percent" :row="row" :value="row.aff_rebate_rate_percent" />
                <slot name="cell-actions" :row="row" />
              </div>
            </div>
          `
        },
        Pagination: true,
        Icon: true
      }
    }
  })
}

const RATE_INPUT = 'input[placeholder="留空 = 不设专属比例，跟随全局"]'

const baseEntry = {
  user_id: 7,
  email: 'promoter@example.com',
  username: 'promoter',
  aff_code: 'SSXZ7',
  aff_code_custom: true,
  aff_rebate_rate_percent: 12 as number | null,
  aff_count: 3,
  aff_quota: 8.25,
  aff_frozen_quota: 2.5,
  aff_history_quota: 13,
  accrued_rebate_total: 13,
  transferred_rebate_total: 2.25,
  invitee_recharge_total: 100
}

describe('admin AffiliatesView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    for (const key of Object.keys(routeQuery)) {
      delete routeQuery[key]
    }
    affiliateAPI.listUsers.mockResolvedValue({
      items: [
        {
          user_id: 7,
          email: 'promoter@example.com',
          username: 'promoter',
          aff_code: 'SSXZ7',
          aff_code_custom: true,
          aff_rebate_rate_percent: 12,
          aff_count: 3,
          aff_quota: 8.25,
          aff_frozen_quota: 2.5,
          aff_history_quota: 13,
          accrued_rebate_total: 13,
          transferred_rebate_total: 2.25,
          invitee_recharge_total: 100
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    affiliateAPI.lookupUsers.mockResolvedValue([
      { id: 8, email: 'new@example.com', username: 'new-user' }
    ])
    affiliateAPI.getUserOverview.mockResolvedValue({
      user_id: 8,
      email: 'new@example.com',
      username: 'new-user',
      aff_code: 'SSXZ8',
      rebate_rate_percent: 5,
      rebate_rate_custom: false,
      invited_count: 0,
      rebated_invitee_count: 0,
      available_quota: 0,
      history_quota: 0
    })
    affiliateAPI.updateUserSettings.mockResolvedValue({ user_id: 8 })
    affiliateAPI.clearUserSettings.mockResolvedValue({ user_id: 7 })
    settingsAPI.getSettings.mockResolvedValue({ affiliate_rebate_rate: 5 })
  })

  it('loads existing custom affiliate users', async () => {
    const wrapper = mountView()

    await flushPromises()

    expect(affiliateAPI.listUsers).toHaveBeenCalledWith(1, 20, '')
    expect(wrapper.text()).toContain('数据来自已有邀请关系、订单和返利账本')
    expect(wrapper.text()).toContain('promoter@example.com')
    expect(wrapper.text()).toContain('SSXZ7')
    expect(wrapper.text()).toContain('3 人')
    expect(wrapper.text()).toContain('100.00 额度')
    expect(wrapper.text()).toContain('13.00 额度')
    expect(wrapper.text()).toContain('8.25 额度')
    expect(wrapper.text()).toContain('2.25 额度')
    expect(wrapper.text()).toContain('12%')
    expect(wrapper.text()).toContain('复制链接')
  })

  it('copies an existing promoter register link from the admin table', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text().includes('复制链接'))!.trigger('click')

    expect(copyToClipboard).toHaveBeenCalledWith(
      expect.stringContaining('/register?aff=SSXZ7'),
      '推广链接已复制'
    )
  })

  it('uses route query as an investigation search keyword', async () => {
    routeQuery.search = 'promoter@example.com'

    mountView()
    await flushPromises()

    expect(affiliateAPI.listUsers).toHaveBeenCalledWith(1, 20, 'promoter@example.com')
    expect(affiliateAPI.lookupUsers).toHaveBeenCalledWith('promoter@example.com')
  })

  it('searches a user and saves affiliate settings', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('input[placeholder="输入邮箱、用户名或用户 ID"]').setValue('new@example.com')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(affiliateAPI.lookupUsers).toHaveBeenCalledWith('new@example.com')
    await wrapper.findAll('button').find((button) => button.text().includes('选择'))!.trigger('click')
    await flushPromises()
    await wrapper.find('input[placeholder="例如 SSXZ2026"]').setValue('SSXZ8')
    await wrapper.find(RATE_INPUT).setValue('15')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(affiliateAPI.updateUserSettings).toHaveBeenCalledWith(8, {
      aff_code: 'SSXZ8',
      aff_rebate_rate_percent: 15
    })
    expect(showSuccess).toHaveBeenCalledWith('推广返利设置已保存')
  })

  // Regression: 0 vs NULL. An emptied input must clear the override (NULL -> follow the
  // global rate), and an explicit 0 must stay 0 (rebate disabled for that user).
  it('sends clear_rebate_rate when the rate input is emptied', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text().includes('编辑'))!.trigger('click')
    await flushPromises()

    // Row has an exclusive 12%, so the input is prefilled from it.
    expect((wrapper.find(RATE_INPUT).element as HTMLInputElement).value).toBe('12')

    await wrapper.find(RATE_INPUT).setValue('')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(affiliateAPI.updateUserSettings).toHaveBeenCalledWith(7, {
      aff_code: 'SSXZ7',
      clear_rebate_rate: true
    })
    const payload = affiliateAPI.updateUserSettings.mock.calls[0][1]
    expect(payload).not.toHaveProperty('aff_rebate_rate_percent')
  })

  it('sends an explicit 0 as 0 rather than clearing the override', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.findAll('button').find((button) => button.text().includes('编辑'))!.trigger('click')
    await flushPromises()

    await wrapper.find(RATE_INPUT).setValue('0')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(affiliateAPI.updateUserSettings).toHaveBeenCalledWith(7, {
      aff_code: 'SSXZ7',
      aff_rebate_rate_percent: 0
    })
    const payload = affiliateAPI.updateUserSettings.mock.calls[0][1]
    expect(payload).not.toHaveProperty('clear_rebate_rate')
  })

  it('renders unset and explicit-zero rates differently', async () => {
    affiliateAPI.listUsers.mockResolvedValue({
      items: [
        { ...baseEntry, user_id: 21, aff_rebate_rate_percent: null },
        { ...baseEntry, user_id: 22, aff_rebate_rate_percent: 0 }
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('未设置（跟随全局 5%）')
    expect(wrapper.text()).toContain('0%（已关闭返利）')
  })

  it('leaves the rate untouched when the current state could not be read', async () => {
    affiliateAPI.getUserOverview.mockRejectedValue({ status: 500, message: 'boom' })

    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('input[placeholder="输入邮箱、用户名或用户 ID"]').setValue('new@example.com')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()
    await wrapper.findAll('button').find((button) => button.text().includes('选择'))!.trigger('click')
    await flushPromises()

    await wrapper.find('input[placeholder="例如 SSXZ2026"]').setValue('SSXZ8')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(affiliateAPI.updateUserSettings).toHaveBeenCalledWith(8, { aff_code: 'SSXZ8' })
  })
})
