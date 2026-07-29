import { defineComponent } from 'vue'
import { shallowMount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { authState, appState, routeState } = vi.hoisted(() => ({
  authState: {
    isAdmin: true,
    isSimpleMode: false,
    user: {
      username: 'admin',
      email: 'admin@example.test',
      role: 'admin',
      balance: 0
    },
    logout: vi.fn()
  },
  appState: {
    contactInfo: '',
    docUrl: '',
    cachedPublicSettings: { custom_menu_items: [] },
    toggleMobileSidebar: vi.fn()
  },
  routeState: {
    name: 'AdminDashboard',
    params: {},
    meta: { title: 'Admin dashboard' }
  }
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ push: vi.fn() }),
  useRoute: () => routeState
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => appState,
  useAuthStore: () => authState,
  useOnboardingStore: () => ({ replay: vi.fn() })
}))

vi.mock('@/stores/adminSettings', () => ({
  useAdminSettingsStore: () => ({ customMenuItems: [] })
}))

import AppHeader from '../AppHeader.vue'

const VersionBadgeStub = defineComponent({
  name: 'VersionBadge',
  props: { runtimeActionsEnabled: Boolean },
  template: '<div data-testid="version-badge" :data-runtime-enabled="String(runtimeActionsEnabled)" />'
})

function mountHeader() {
  return shallowMount(AppHeader, {
    global: {
      plugins: [createPinia()],
      stubs: {
        VersionBadge: VersionBadgeStub,
        RouterLink: {
          props: ['to'],
          template: '<a class="router-link-stub" :href="to"><slot /></a>'
        }
      }
    }
  })
}

describe('AppHeader version control', () => {
  beforeEach(() => {
    authState.isAdmin = true
  })

  it('mounts the managed VersionBadge in the admin header', () => {
    const wrapper = mountHeader()

    expect(wrapper.get('[data-testid="version-badge"]').attributes('data-runtime-enabled')).toBe('false')
  })

  it('does not mount the VersionBadge for regular users', () => {
    authState.isAdmin = false
    const wrapper = mountHeader()

    expect(wrapper.find('[data-testid="version-badge"]').exists()).toBe(false)
  })

  it('keeps the public documentation in the top header without relying on doc_url', () => {
    appState.docUrl = ''
    const wrapper = mountHeader()

    expect(wrapper.get('.header-action-link').attributes('href')).toBe('/docs')
  })
})
