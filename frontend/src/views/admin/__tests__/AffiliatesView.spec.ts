import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { affiliateAPI, showError, showSuccess } = vi.hoisted(() => ({
  affiliateAPI: {
    listUsers: vi.fn(),
    lookupUsers: vi.fn(),
    updateUserSettings: vi.fn(),
    clearUserSettings: vi.fn(),
    batchSetRate: vi.fn()
  },
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    affiliate: affiliateAPI
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

describe('admin AffiliatesView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    affiliateAPI.listUsers.mockResolvedValue({
      items: [
        {
          user_id: 7,
          email: 'promoter@example.com',
          username: 'promoter',
          aff_code: 'SSXZ7',
          aff_code_custom: true,
          aff_rebate_rate_percent: 12,
          aff_count: 3
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
    affiliateAPI.updateUserSettings.mockResolvedValue({ user_id: 8 })
    affiliateAPI.clearUserSettings.mockResolvedValue({ user_id: 7 })
  })

  it('loads existing custom affiliate users', async () => {
    const wrapper = mountView()

    await flushPromises()

    expect(affiliateAPI.listUsers).toHaveBeenCalledWith(1, 20, '')
    expect(wrapper.text()).toContain('promoter@example.com')
    expect(wrapper.text()).toContain('SSXZ7')
    expect(wrapper.text()).toContain('3 人')
    expect(wrapper.text()).toContain('12%')
  })

  it('searches a user and saves affiliate settings', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.find('input[placeholder="输入邮箱、用户名或用户 ID"]').setValue('new@example.com')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(affiliateAPI.lookupUsers).toHaveBeenCalledWith('new@example.com')
    await wrapper.findAll('button').find((button) => button.text().includes('选择'))!.trigger('click')
    await wrapper.find('input[placeholder="例如 SSXZ2026"]').setValue('SSXZ8')
    await wrapper.find('input[placeholder="留空使用默认比例"]').setValue('15')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(affiliateAPI.updateUserSettings).toHaveBeenCalledWith(8, {
      aff_code: 'SSXZ8',
      aff_rebate_rate_percent: 15
    })
    expect(showSuccess).toHaveBeenCalledWith('推广返利设置已保存')
  })
})
