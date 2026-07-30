import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import Toggle from '../Toggle.vue'

describe('Toggle', () => {
  it('exposes a clear state and emits the next value', async () => {
    const wrapper = mount(Toggle, {
      props: {
        modelValue: false,
        ariaLabel: 'Enable feature'
      }
    })
    const toggle = wrapper.get('[role="switch"]')

    expect(toggle.attributes('aria-checked')).toBe('false')
    expect(toggle.attributes('data-state')).toBe('off')
    expect(toggle.attributes('data-size')).toBe('default')
    expect(toggle.attributes('data-variant')).toBe('primary')
    expect(toggle.attributes('aria-label')).toBe('Enable feature')
    expect(wrapper.find('[data-testid="toggle-icon-off"]').exists()).toBe(true)

    await toggle.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([[true]])
  })

  it('renders the enabled state', () => {
    const wrapper = mount(Toggle, {
      props: {
        modelValue: true
      }
    })

    const toggle = wrapper.get('[role="switch"]')
    expect(toggle.attributes('data-state')).toBe('on')
    expect(toggle.attributes('aria-label')).toBe('切换设置')
    expect(wrapper.find('[data-testid="toggle-icon-on"]').exists()).toBe(true)
  })

  it('supports compact and destructive presentation variants', () => {
    const wrapper = mount(Toggle, {
      props: {
        modelValue: true,
        size: 'sm',
        variant: 'destructive'
      }
    })

    const toggle = wrapper.get('[role="switch"]')
    expect(toggle.attributes('data-size')).toBe('sm')
    expect(toggle.attributes('data-variant')).toBe('destructive')
    expect(toggle.classes()).toContain('material-switch--small')
    expect(toggle.classes()).toContain('material-switch--destructive')
  })

  it('does not emit while disabled', async () => {
    const wrapper = mount(Toggle, {
      props: {
        modelValue: false,
        disabled: true
      }
    })

    await wrapper.get('[role="switch"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
})
