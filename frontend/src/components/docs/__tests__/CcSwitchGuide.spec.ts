import { mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

import CcSwitchGuide from '../CcSwitchGuide.vue'

describe('CcSwitchGuide', () => {
  it('renders the approved guide from the single markdown source with all seven screenshots', () => {
    const wrapper = mount(CcSwitchGuide)
    const images = wrapper.findAll('img')
    const tutorialSource = readFileSync('../docs/教程/CC-Switch一键接入SSXZ.md', 'utf-8')

    expect(wrapper.text()).toContain('用 CC Switch 一键接入 SSXZ')
    expect(wrapper.text()).toContain('第 1 步：在 SSXZ 点“导入到 CCS”')
    expect(wrapper.text()).toContain('第 2 步：确认打开并导入')
    expect(wrapper.text()).toContain('第 3 步：切到 SSXZ，开始使用')
    expect(wrapper.get('a[href="https://github.com/farion1231/cc-switch"]').text()).toContain(
      'CC Switch 官方 GitHub'
    )
    expect(images).toHaveLength(7)
    expect(images.every((image) => Boolean(image.attributes('src')))).toBe(true)
    expect(images.every((image) => !image.attributes('src').startsWith('./assets/'))).toBe(true)
    expect(tutorialSource).not.toMatch(/sk-[A-Za-z0-9_-]{16,}/)
    expect(tutorialSource).not.toContain('当前为审核草稿')
  })
})
