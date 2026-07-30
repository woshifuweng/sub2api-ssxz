import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listAdminAgents, showError } = vi.hoisted(() => ({
  listAdminAgents: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/reseller', () => ({
  default: { listAdminAgents }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError })
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ path: '/admin/reseller/agents' })
}))

import AdminAgents from '../AdminAgents.vue'

describe('AdminAgents', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listAdminAgents.mockResolvedValue({
      items: [{
        user_id: 7,
        email: 'agent@example.com',
        username: 'Agent Seven',
        role: 'agent_manager',
        commission_rate: 0.125,
        aff_code: 'AGENT7',
        recruit_count: 3,
        aff_quota: 12,
        granted_at: '2026-07-30T00:00:00Z',
        granted_by: 1
      }],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
  })

  it('loads the admin agent list and keeps unavailable fields explicit', async () => {
    const wrapper = mount(AdminAgents, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          AdminPageHeader: { template: '<header><slot name="actions" /></header>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: {
            props: ['data'],
            template: `
              <div>
                <div v-for="row in data" :key="row.user_id">
                  <slot name="cell-user" :row="row" />
                  <slot name="cell-role" :row="row" />
                  <slot name="cell-manager" :row="row" />
                  <slot name="cell-total_commission" :row="row" />
                  <slot name="cell-commission_rate" :row="row" :value="row.commission_rate" />
                </div>
              </div>
            `
          },
          Pagination: true,
          LiquidButton: { template: '<button><slot /></button>' },
          Icon: true,
          RouterLink: { props: ['to'], template: '<a :href="to"><slot /></a>' }
        }
      }
    })

    await flushPromises()

    expect(listAdminAgents).toHaveBeenCalledWith(1, 20, '')
    expect(wrapper.text()).toContain('Agent Seven')
    expect(wrapper.text()).toContain('agent@example.com')
    expect(wrapper.text()).toContain('Agent Manager')
    expect(wrapper.text()).toContain('12.50%')
    expect(wrapper.text()).toContain('--')
  })
})
