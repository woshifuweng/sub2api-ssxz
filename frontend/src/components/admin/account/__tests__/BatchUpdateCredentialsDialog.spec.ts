import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import BatchUpdateCredentialsDialog from '../BatchUpdateCredentialsDialog.vue'
import { adminAPI } from '@/api/admin'

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: { batchUpdateCredentials: vi.fn() }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${JSON.stringify(params)}` : key
    })
  }
})

const BaseDialogStub = {
  props: ['show'],
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
}

const ConfirmDialogStub = {
  props: ['show', 'message'],
  emits: ['confirm', 'cancel'],
  template: `
    <div v-if="show" data-testid="confirmation">
      <span>{{ message }}</span>
      <button data-testid="confirm" @click="$emit('confirm')">confirm</button>
    </div>
  `
}

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template: '<div data-testid="select-stub">{{ modelValue }}</div>'
}

function mountDialog() {
  return mount(BatchUpdateCredentialsDialog, {
    props: { show: true, accountIds: [30, 34] },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        Select: SelectStub
      }
    }
  })
}

describe('BatchUpdateCredentialsDialog', () => {
  beforeEach(() => {
    vi.mocked(adminAPI.accounts.batchUpdateCredentials).mockReset()
    vi.mocked(adminAPI.accounts.batchUpdateCredentials).mockResolvedValue({
      success: 2,
      failed: 0,
      results: []
    })
  })

  it('never includes the entered value in confirmation and submits after confirmation', async () => {
    const wrapper = mountDialog()
    const hiddenValue = 'account-uuid-not-for-display'

    await wrapper.get('[data-testid="batch-credential-value"]').setValue(hiddenValue)
    await wrapper.get('#batch-update-credentials-form').trigger('submit.prevent')

    expect(adminAPI.accounts.batchUpdateCredentials).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="confirmation"]').text()).not.toContain(hiddenValue)

    await wrapper.get('[data-testid="confirm"]').trigger('click')
    await flushPromises()

    expect(adminAPI.accounts.batchUpdateCredentials).toHaveBeenCalledTimes(1)
    expect(adminAPI.accounts.batchUpdateCredentials).toHaveBeenCalledWith({
      account_ids: [30, 34],
      field: 'account_uuid',
      value: hiddenValue
    })
    expect(wrapper.emitted('updated')).toHaveLength(1)
    expect(wrapper.emitted('completed')).toHaveLength(1)
    expect((wrapper.get('[data-testid="batch-credential-value"]').element as HTMLInputElement).value).toBe('')
  })

  it('sends null when clearing an allowed credential field', async () => {
    const wrapper = mountDialog()

    await wrapper.get('[data-testid="batch-credential-clear"]').setValue(true)
    await wrapper.get('#batch-update-credentials-form').trigger('submit.prevent')
    await wrapper.get('[data-testid="confirm"]').trigger('click')
    await flushPromises()

    expect(adminAPI.accounts.batchUpdateCredentials).toHaveBeenCalledWith({
      account_ids: [30, 34],
      field: 'account_uuid',
      value: null
    })
  })
})
