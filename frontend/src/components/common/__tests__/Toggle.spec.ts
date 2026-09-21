import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Toggle from '../Toggle.vue'

describe('Toggle', () => {
  it('exposes an accessible name and visible state when requested', async () => {
    const wrapper = mount(Toggle, {
      props: {
        modelValue: false,
        label: '启用设置',
        showState: true,
      },
    })

    expect(wrapper.get('[role="switch"]').attributes('aria-label')).toBe('启用设置')
    expect(wrapper.text()).toContain('关')

    await wrapper.get('[role="switch"]').trigger('click')
    await wrapper.setProps({ modelValue: true })

    expect(wrapper.text()).toContain('开')
    expect(wrapper.emitted('update:modelValue')).toEqual([[true]])
  })

  it('keeps caller attributes on the interactive button', () => {
    const wrapper = mount(Toggle, {
      attrs: {
        'data-testid': 'settings-toggle',
      },
      props: {
        modelValue: true,
      },
    })

    expect(wrapper.get('[role="switch"]').attributes('data-testid')).toBe('settings-toggle')
  })
})
