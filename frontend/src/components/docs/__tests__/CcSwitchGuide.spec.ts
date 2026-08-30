import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import zh from '@/i18n/locales/zh'
import CcSwitchGuide from '../CcSwitchGuide.vue'

describe('CcSwitchGuide', () => {
  it('renders the verified three-step guide and seven screenshots', () => {
    const wrapper = mount(CcSwitchGuide, {
      global: {
        plugins: [createI18n({ legacy: false, locale: 'zh', messages: { zh } })]
      }
    })

    expect(wrapper.text()).toContain('第 1 步')
    expect(wrapper.text()).toContain('第 2 步')
    expect(wrapper.text()).toContain('第 3 步')
    expect(wrapper.findAll('img')).toHaveLength(7)
    expect(wrapper.get('a[href="https://github.com/farion1231/cc-switch"]')).toBeTruthy()
  })
})
