import { mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import F0FoundationPreview from '../F0FoundationPreview.vue'

beforeAll(() => {
  Object.defineProperty(HTMLDialogElement.prototype, 'showModal', {
    configurable: true,
    value(this: HTMLDialogElement) {
      this.setAttribute('open', '')
    }
  })
  Object.defineProperty(HTMLDialogElement.prototype, 'close', {
    configurable: true,
    value(this: HTMLDialogElement) {
      this.removeAttribute('open')
      this.dispatchEvent(new Event('close'))
    }
  })
})

afterEach(() => {
  document.body.innerHTML = ''
})

describe('F0FoundationPreview', () => {
  it('renders the approved component families and switches the scoped theme', async () => {
    const wrapper = mount(F0FoundationPreview, { attachTo: document.body })

    expect(wrapper.get('.f0-foundation').attributes('data-theme')).toBe('light')
    expect(wrapper.findAll('.f0-card').length).toBeGreaterThanOrEqual(8)
    expect(wrapper.findAll('.f0-button').length).toBeGreaterThanOrEqual(10)
    expect(wrapper.findAll('[data-action-tier]')).toHaveLength(2)
    expect(wrapper.get('[data-action-tier="primary"]').text()).toContain('保存设置')
    expect(wrapper.get('[data-action-tier="utility"]').text()).toContain('删除条目')
    expect(wrapper.findAll('.f0-badge').length).toBeGreaterThanOrEqual(6)
    expect(wrapper.get('[data-testid="f0-icon-set"]').text()).toContain('动作')
    expect(wrapper.get('[data-testid="f0-icon-set"]').findAll('.model-icon')).toHaveLength(4)
    expect(wrapper.findAll('.f0-input-control')).toHaveLength(4)
    const inputIds = wrapper.findAll('.f0-input-control').map((input) => input.attributes('id'))
    expect(new Set(inputIds).size).toBe(inputIds.length)
    expect(wrapper.get(`label[for="${inputIds[0]}"]`).text()).toBe('显示名称')
    expect(wrapper.get('.f0-table').exists()).toBe(true)
    expect(wrapper.get('.f0-sidebar').exists()).toBe(true)

    await wrapper.get('[data-testid="theme-toggle"]').trigger('click')

    expect(wrapper.get('.f0-foundation').attributes('data-theme')).toBe('dark')
    expect(wrapper.get('.f0-foundation').classes()).toContain('f0-dark')
  })

  it('keeps input, dialog, and sidebar interactions functional', async () => {
    const wrapper = mount(F0FoundationPreview, { attachTo: document.body })
    const nameInput = wrapper.findAll<HTMLInputElement>('.f0-input-control')[0]

    await nameInput.setValue('更新后的名称')
    expect(nameInput.element.value).toBe('更新后的名称')

    const openButton = wrapper.findAll('button').find((button) => button.text() === '打开对话框')
    expect(openButton).toBeDefined()
    await openButton!.trigger('click')
    await nextTick()

    expect(wrapper.get('dialog').attributes()).toHaveProperty('open')
    await wrapper.get('button[aria-label="关闭对话框"]').trigger('click')
    await nextTick()
    expect(wrapper.get('dialog').attributes()).not.toHaveProperty('open')

    const customersItem = wrapper.findAll('.f0-sidebar-item').find((item) => item.text() === '客户管理')
    expect(customersItem).toBeDefined()
    await customersItem!.trigger('click')
    expect(customersItem!.attributes('aria-current')).toBe('page')
  })

  it('remains absent from production routing and global style entry points', () => {
    const routerSource = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
    const appMainSource = readFileSync(resolve(process.cwd(), 'src/main.ts'), 'utf8')
    const globalStyleSource = readFileSync(resolve(process.cwd(), 'src/style.css'), 'utf8')
    const foundationStyleSource = readFileSync(
      resolve(process.cwd(), 'src/foundation/foundation.css'),
      'utf8'
    )
    const packageSource = JSON.parse(
      readFileSync(resolve(process.cwd(), 'package.json'), 'utf8')
    ) as { dependencies: Record<string, string> }

    expect(routerSource).not.toContain('f0-preview')
    expect(appMainSource).not.toContain('f0-preview')
    expect(globalStyleSource).not.toContain('f0-foundation')
    expect(foundationStyleSource).toMatch(/\.f0-button \{[\s\S]*?height: 2\.25rem;/)
    expect(foundationStyleSource).toMatch(/\.f0-button--sm \{[\s\S]*?height: 2rem;/)
    expect(foundationStyleSource).toMatch(/\.f0-button--lg \{[\s\S]*?height: 2\.5rem;/)
    expect(foundationStyleSource).toContain('--primary: 210 24% 38%;')
    expect(foundationStyleSource).toContain('--brand-accent: 210 24% 38%;')
    expect(foundationStyleSource).not.toMatch(/--(?:primary|ring): 239\s/)
    expect(foundationStyleSource).toContain('--success: 153 60% 38%;')
    expect(foundationStyleSource).toContain('--destructive: 0 72% 51%;')
    expect(foundationStyleSource).toContain('--warning: 36 88% 48%;')
    const darkThemeBlock = foundationStyleSource.match(
      /\.f0-foundation\.f0-dark \{([\s\S]*?)\n\}/
    )?.[1]
    expect(darkThemeBlock).toBeDefined()
    expect(darkThemeBlock).toContain('--primary: 240 5% 96%;')
    expect(darkThemeBlock).toContain('--brand-accent: 210 24% 72%;')
    expect(darkThemeBlock).toContain('--background: 240 8.3% 4.7%;')
    expect(darkThemeBlock).toContain('--card: 0 0% 0% / 0;')
    expect(darkThemeBlock).toContain('--border: 240 7% 18%;')
    expect(darkThemeBlock).toContain('--button-shadow-hover: 0 0% 0% / 0.48;')
    expect(darkThemeBlock).not.toMatch(/\b(?:height|padding|gap):/)
    expect(packageSource.dependencies['@lucide/vue']).toBe('1.24.0')
    expect(packageSource.dependencies).not.toHaveProperty('lucide-vue-next')
  })
})
