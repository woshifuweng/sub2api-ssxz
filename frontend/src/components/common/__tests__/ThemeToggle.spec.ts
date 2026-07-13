import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'nav.lightMode': 'Light Mode',
      'nav.darkMode': 'Dark Mode'
    })[key] ?? key
  })
}))

import ThemeToggle from '../ThemeToggle.vue'

describe('ThemeToggle', () => {
  beforeEach(() => {
    document.documentElement.classList.remove('dark')
    localStorage.clear()
  })

  it('persists the selected theme and updates the document root', async () => {
    const wrapper = mount(ThemeToggle)
    const button = wrapper.get('button')

    expect(button.attributes('aria-label')).toBe('Dark Mode')
    await button.trigger('click')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(localStorage.getItem('theme')).toBe('dark')

    await button.trigger('click')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(localStorage.getItem('theme')).toBe('light')
  })
})
