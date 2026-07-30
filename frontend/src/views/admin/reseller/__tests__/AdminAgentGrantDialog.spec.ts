import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listUsers, grantAdminRole, showWarning, showError, showSuccess } = vi.hoisted(() => ({
  listUsers: vi.fn(),
  grantAdminRole: vi.fn(),
  showWarning: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn()
}))

vi.mock('@/api/admin/users', () => ({
  list: listUsers
}))

vi.mock('@/api/reseller', () => ({
  default: { grantAdminRole }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showWarning, showError, showSuccess })
}))

import AdminAgentGrantDialog from '../AdminAgentGrantDialog.vue'

describe('AdminAgentGrantDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listUsers.mockResolvedValue({
      items: [{
        id: 18,
        username: 'Agent Candidate',
        email: 'candidate@example.com',
        status: 'active'
      }],
      total: 1,
      page: 1,
      page_size: 10,
      pages: 1
    })
    grantAdminRole.mockResolvedValue({ user_id: 18, role: 'agent_manager' })
  })

  it('searches active users, requires an explicit selection, and grants the selected role', async () => {
    const wrapper = mount(AdminAgentGrantDialog, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>'
          },
          LiquidButton: { template: '<button><slot /></button>' },
          Icon: true
        }
      }
    })

    await wrapper.get('#agent-user-search').setValue('candidate@example.com')
    await wrapper.get('[data-testid="search-users"]').trigger('click')
    await flushPromises()

    expect(listUsers).toHaveBeenCalledWith(1, 10, {
      search: 'candidate@example.com',
      status: 'active'
    })
    expect(grantAdminRole).not.toHaveBeenCalled()

    await wrapper.get('[data-testid="user-result-18"]').trigger('click')
    await wrapper.get('select').setValue('agent_manager')
    await wrapper.get('input[placeholder="例如：渠道来源或负责人"]').setValue('华东渠道')
    await wrapper.get('[data-testid="grant-role"]').trigger('click')
    await flushPromises()

    expect(grantAdminRole).toHaveBeenCalledWith(18, {
      role: 'agent_manager',
      notes: '华东渠道'
    })
    expect(showSuccess).toHaveBeenCalled()
    expect(wrapper.emitted('granted')?.[0]?.[0]).toMatchObject({
      id: 18,
      email: 'candidate@example.com'
    })
  })
})
