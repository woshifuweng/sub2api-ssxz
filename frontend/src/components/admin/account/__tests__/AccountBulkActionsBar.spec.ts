import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountBulkActionsBar from '../AccountBulkActionsBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

describe('AccountBulkActionsBar', () => {
  it('emits the dedicated batch credential event', async () => {
    const wrapper = mount(AccountBulkActionsBar, {
      props: { selectedIds: [30, 34] },
      global: { stubs: { Icon: true } }
    })

    const button = wrapper.findAll('button').find((item) =>
      item.text().includes('admin.accounts.bulkActions.updateCredentials')
    )
    expect(button).toBeTruthy()

    await button!.trigger('click')
    expect(wrapper.emitted('batch-credentials')).toHaveLength(1)
  })
})
