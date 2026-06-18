import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { User } from '@/types'

const messages: Record<string, string> = {
  'profile.administrator': '管理员',
  'profile.user': '普通用户',
  'profile.statusActive': '正常',
  'profile.statusDisabled': '已停用'
}

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => messages[key] || key
  })
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: {
    name: 'Icon',
    template: '<span />'
  }
}))

import ProfileInfoCard from '../ProfileInfoCard.vue'

function makeUser(overrides: Partial<User> = {}): User {
  return {
    id: 8,
    username: '测试用户',
    email: 'user@example.test',
    role: 'user',
    balance: 49.4,
    concurrency: 10,
    status: 'active',
    allowed_groups: null,
    created_at: '2026-05-01T00:00:00Z',
    updated_at: '2026-05-01T00:00:00Z',
    ...overrides
  }
}

describe('ProfileInfoCard', () => {
  it('shows customer-facing role and active status labels', () => {
    const wrapper = mount(ProfileInfoCard, {
      props: { user: makeUser() }
    })

    const text = wrapper.text()
    expect(text).toContain('普通用户')
    expect(text).toContain('正常')
    expect(text).not.toMatch(/\bactive\b/)
  })

  it('shows the disabled status in Chinese', () => {
    const wrapper = mount(ProfileInfoCard, {
      props: { user: makeUser({ status: 'disabled' }) }
    })

    expect(wrapper.text()).toContain('已停用')
    expect(wrapper.text()).not.toMatch(/\bdisabled\b/)
  })
})
