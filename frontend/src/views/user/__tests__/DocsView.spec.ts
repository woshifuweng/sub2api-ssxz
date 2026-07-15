import { mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { describe, expect, it, vi } from 'vitest'

vi.mock('@/components/user/AppSectionShell.vue', () => ({
  default: {
    name: 'AppSectionShell',
    props: ['title', 'subtitle', 'eyebrow', 'icon'],
    template: '<main data-testid="app-section-shell"><slot /></main>'
  }
}))

vi.mock('@/components/foundation', () => ({
  FoundationProvider: {
    name: 'FoundationProvider',
    props: ['theme'],
    template: '<div data-testid="docs-foundation"><slot /></div>'
  },
  FoundationCard: {
    name: 'FoundationCard',
    template: '<section data-testid="docs-card"><slot /></section>'
  }
}))

import DocsView from '../DocsView.vue'

function mountView() {
  return mount(DocsView, {
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>'
        }
      }
    }
  })
}

describe('DocsView', () => {
  it('publishes the approved CC Switch guide and all seven real screenshots', () => {
    const wrapper = mountView()
    const guide = wrapper.get('[data-testid="cc-switch-guide"]')
    const images = guide.findAll('img')

    expect(guide.text()).toContain('用 CC Switch 一键接入 SSXZ')
    expect(guide.text()).toContain('第 1 步：在 SSXZ 点“导入到 CCS”')
    expect(guide.text()).toContain('第 2 步：确认打开并导入')
    expect(guide.text()).toContain('第 3 步：切到 SSXZ，开始使用')
    expect(guide.text()).not.toContain('当前为审核草稿')
    expect(images).toHaveLength(7)
    expect(images.every((image) => Boolean(image.attributes('src')))).toBe(true)
    expect(images.every((image) => !image.attributes('src').startsWith('./assets/'))).toBe(true)
  })

  it('shows the verified official repository and keeps secrets out of the published source', () => {
    const wrapper = mountView()
    const officialLink = wrapper.get('a[href="https://github.com/farion1231/cc-switch"]')
    const tutorialSource = readFileSync('../docs/教程/CC-Switch一键接入SSXZ.md', 'utf-8')

    expect(officialLink.text()).toContain('CC Switch 官方 GitHub')
    expect(tutorialSource).not.toMatch(/sk-[A-Za-z0-9_-]{16,}/)
    expect(tutorialSource).not.toContain('当前为审核草稿')
  })

  it('registers the authenticated customer documentation route', () => {
    const routerSource = readFileSync('src/router/index.ts', 'utf-8')

    expect(routerSource).toContain("path: '/app/docs'")
    expect(routerSource).toContain("component: () => import('@/views/user/DocsView.vue')")
    expect(routerSource).toMatch(/path: '\/app\/docs'[\s\S]*requiresAuth: true[\s\S]*requiresAdmin: false/)
  })
})
