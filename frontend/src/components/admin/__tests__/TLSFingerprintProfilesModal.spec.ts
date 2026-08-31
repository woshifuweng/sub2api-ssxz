import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import TLSFingerprintProfilesModal from '../TLSFingerprintProfilesModal.vue'

const { listProfiles } = vi.hoisted(() => ({
  listProfiles: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    tlsFingerprintProfiles: {
      list: listProfiles,
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }),
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

describe('TLSFingerprintProfilesModal', () => {
  beforeEach(() => {
    listProfiles.mockReset()
    listProfiles.mockResolvedValue([])
  })

  it('loads profiles when lazy-mounted already open', async () => {
    mount(TLSFingerprintProfilesModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show'],
            template: '<section v-if="show" role="dialog"><slot /><slot name="footer" /></section>',
          },
          ConfirmDialog: true,
          Icon: true,
        },
      },
    })

    await flushPromises()

    expect(listProfiles).toHaveBeenCalledTimes(1)
  })
})
