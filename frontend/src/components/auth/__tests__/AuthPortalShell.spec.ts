import { mount, flushPromises } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AuthOrbitVisual from '../AuthOrbitVisual.vue'
import AuthPortalShell from '../AuthPortalShell.vue'

const appStore = vi.hoisted(() => ({
  siteName: 'SSXZ AI',
  cachedPublicSettings: { site_subtitle: '智能服务控制台' },
  fetchPublicSettings: vi.fn().mockResolvedValue(null)
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

beforeEach(() => {
  document.documentElement.classList.remove('dark')
  localStorage.clear()
  vi.clearAllMocks()
})

afterEach(() => {
  document.body.innerHTML = ''
  document.documentElement.classList.remove('dark')
})

describe('AuthPortalShell', () => {
  it('keeps affiliate and safe redirect query values when switching tabs', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/home', component: { template: '<div />' } },
        { path: '/login', component: { template: '<div />' } },
        { path: '/register', component: { template: '<div />' } }
      ]
    })
    await router.push('/register?aff=AFF-2026&promo=WELCOME&returnTo=/app/keys')
    await router.isReady()

    const wrapper = mount(AuthPortalShell, {
      attachTo: document.body,
      props: { activeTab: 'register' },
      global: {
        plugins: [router],
        stubs: { AuthOrbitVisual: true }
      }
    })

    expect(wrapper.get('[data-testid="auth-tab-register"]').attributes('aria-current')).toBe('page')
    await wrapper.get('[data-testid="auth-tab-login"]').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/login')
    expect(router.currentRoute.value.query).toMatchObject({
      aff: 'AFF-2026',
      promo: 'WELCOME',
      returnTo: '/app/keys'
    })
  })

  it('changes only the scoped F0 theme while keeping the global preference in sync', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/home', component: { template: '<div />' } },
        { path: '/login', component: { template: '<div />' } },
        { path: '/register', component: { template: '<div />' } }
      ]
    })
    await router.push('/login')
    await router.isReady()

    const wrapper = mount(AuthPortalShell, {
      attachTo: document.body,
      props: { activeTab: 'login' },
      global: {
        plugins: [router],
        stubs: { AuthOrbitVisual: true }
      }
    })

    expect(wrapper.get('.f0-foundation').attributes('data-theme')).toBe('light')
    await wrapper.get('[data-testid="auth-theme-toggle"]').trigger('click')
    await flushPromises()

    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(wrapper.get('.f0-foundation').attributes('data-theme')).toBe('dark')
    expect(localStorage.getItem('theme')).toBe('dark')
  })
})

describe('AuthOrbitVisual', () => {
  it('renders 18 known local AI brand icons without image or fallback nodes', () => {
    const wrapper = mount(AuthOrbitVisual)

    expect(wrapper.findAll('.auth-orbit-node')).toHaveLength(18)
    expect(wrapper.findAll('.model-icon')).toHaveLength(18)
    expect(wrapper.findAll('.model-icon-fallback')).toHaveLength(0)
    expect(wrapper.find('img').exists()).toBe(false)
  })

  it('keeps C2 free of the rejected React and blue-glow dependencies', () => {
    const orbitSource = readFileSync(
      resolve(process.cwd(), 'src/components/auth/AuthOrbitVisual.vue'),
      'utf8'
    )
    const shellSource = readFileSync(
      resolve(process.cwd(), 'src/components/auth/AuthPortalShell.vue'),
      'utf8'
    )
    const linuxDoSource = readFileSync(
      resolve(process.cwd(), 'src/components/auth/LinuxDoOAuthSection.vue'),
      'utf8'
    )

    for (const source of [orbitSource, shellSource]) {
      expect(source).not.toContain('motion/react')
      expect(source).not.toContain('react-icons')
      expect(source).not.toContain('next/image')
      expect(source).not.toContain('#3b82f6')
      expect(source).not.toContain('<img')
    }
    expect(linuxDoSource).not.toContain('<svg')
  })
})
