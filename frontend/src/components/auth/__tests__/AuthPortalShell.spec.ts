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
  it('keeps affiliate and safe redirect values without carrying the removed promo field', async () => {
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
      returnTo: '/app/keys'
    })
    expect(router.currentRoute.value.query).not.toHaveProperty('promo')
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
    expect(wrapper.getComponent(AuthOrbitVisual).props('theme')).toBe('light')
    await wrapper.get('[data-testid="auth-theme-toggle"]').trigger('click')
    await flushPromises()

    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(wrapper.get('.f0-foundation').attributes('data-theme')).toBe('dark')
    expect(wrapper.getComponent(AuthOrbitVisual).props('theme')).toBe('dark')
    expect(localStorage.getItem('theme')).toBe('dark')
  })

  it('uses the shared SSXZ mark in the authentication header', async () => {
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
      props: { activeTab: 'login' },
      global: { plugins: [router], stubs: { AuthOrbitVisual: true } }
    })

    expect(wrapper.get('.auth-portal-brand [data-testid="brand-logo"]').exists()).toBe(true)
    expect(wrapper.find('.auth-portal-brand-mark svg').exists()).toBe(false)
  })
})

describe('AuthOrbitVisual', () => {
  it('renders 18 known local AI brand icons without image or fallback nodes', () => {
    const wrapper = mount(AuthOrbitVisual, { props: { theme: 'dark' } })

    expect(wrapper.findAll('.auth-orbit-node')).toHaveLength(18)
    expect(wrapper.findAll('.model-icon')).toHaveLength(18)
    expect(wrapper.findAll('.model-icon-fallback')).toHaveLength(0)
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.get('.auth-orbit-core [data-testid="brand-logo"]').exists()).toBe(true)
    expect(wrapper.get('.auth-orbit-core [data-testid="brand-logo"]').classes()).toContain('brand-logo--theme-dark')
    expect(wrapper.get('.auth-orbit-core').text()).not.toContain('SSXZ')
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

  it('keeps orbit provider icons transparent on the outline-dark surface', () => {
    const orbitSource = readFileSync(
      resolve(process.cwd(), 'src/components/auth/AuthOrbitVisual.vue'),
      'utf8'
    )

    expect(orbitSource).toMatch(
      /\.auth-orbit-node-face\s*\{[^}]*background:\s*transparent;/s
    )
    expect(orbitSource).toMatch(
      /\.auth-orbit-node-face :deep\(\.model-icon path\)\s*\{[^}]*paint-order:\s*stroke fill;/s
    )
    expect(orbitSource).toContain("const darkOrbitModels = new Set(['gpt-5.5', 'grok-4', 'kimi-k2'])")
    expect(orbitSource).toMatch(
      /\.auth-orbit-node-face--dark-icon :deep\(\.model-icon path\)\s*\{[^}]*stroke-width:\s*0\.9px;/s
    )
    expect(orbitSource).not.toContain('background: hsl(0 0% 98%);')
  })
})
