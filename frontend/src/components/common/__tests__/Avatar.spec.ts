import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import Avatar from '../Avatar.vue'

describe('Avatar', () => {
  it('uses a neutral fallback with initials when no image is available', () => {
    const wrapper = mount(Avatar, { props: { name: 'Visual User' } })

    expect(wrapper.classes()).toContain('ssxz-avatar--fallback')
    expect(wrapper.text()).toBe('V')
    expect(wrapper.find('img').exists()).toBe(false)
  })

  it('renders the supplied avatar image', () => {
    const wrapper = mount(Avatar, {
      props: { name: 'Visual User', src: 'data:image/png;base64,valid' }
    })

    expect(wrapper.find('img').attributes('src')).toContain('data:image/png')
    expect(wrapper.classes()).not.toContain('ssxz-avatar--fallback')
  })
})
