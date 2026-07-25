import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'

describe('AdminPageHeader', () => {
  it('renders the shared title, description, and action slot', () => {
    const wrapper = mount(AdminPageHeader, {
      props: {
        title: '用户管理',
        description: '查看全站用户账号与余额'
      },
      slots: {
        actions: '<button type="button">刷新</button>'
      }
    })

    expect(wrapper.get('h1').text()).toBe('用户管理')
    expect(wrapper.get('p').text()).toBe('查看全站用户账号与余额')
    expect(wrapper.get('.admin-page-header__actions button').text()).toBe('刷新')
  })

  it('omits optional regions when no description or actions are provided', () => {
    const wrapper = mount(AdminPageHeader, {
      props: { title: '管理控制台' }
    })

    expect(wrapper.find('p').exists()).toBe(false)
    expect(wrapper.find('.admin-page-header__actions').exists()).toBe(false)
  })
})
