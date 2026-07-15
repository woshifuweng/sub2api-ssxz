import { mount } from '@vue/test-utils'
import { afterEach, beforeAll, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import HomeDocsDialog from '@/components/home/aether/HomeDocsDialog.vue'

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

describe('HomeDocsDialog', () => {
  it('keeps the core documentation available without an external doc URL', async () => {
    const wrapper = mount(HomeDocsDialog, {
      attachTo: document.body,
      props: {
        open: true,
        baseUrl: 'https://gateway.example/v1',
        docUrl: '',
        createKeyPath: '/app/keys'
      },
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :data-to="to"><slot /></a>'
          }
        }
      }
    })

    await nextTick()

    expect(wrapper.get('dialog').attributes()).toHaveProperty('open')
    expect(wrapper.text()).toContain('创建并保存 API Key')
    expect(wrapper.text()).toContain('https://gateway.example/v1')
    expect(wrapper.text()).toContain('只选择当前 Key 可用的模型')
    expect(wrapper.find('a[href]').exists()).toBe(false)

    await wrapper.get('button[aria-label="复制 Base URL"]').trigger('click')
    expect(wrapper.emitted('copy')).toEqual([['https://gateway.example/v1']])

    await wrapper.get('#home-docs-tab-clients').trigger('click')
    expect(wrapper.text()).toContain('Claude Code')
    expect(wrapper.text()).toContain('Codex CLI')
    expect(wrapper.text()).toContain('Gemini CLI')
    expect(wrapper.text()).toContain('Cherry Studio')
    expect(wrapper.text()).toContain('CC Switch')
    expect(wrapper.text()).toContain('your-api-key')
    expect(wrapper.text()).not.toContain('api.ssxzapi.com')

    await wrapper.get('#home-docs-tab-billing').trigger('click')
    expect(wrapper.get('#home-docs-panel-billing').attributes('aria-labelledby')).toBe(
      'home-docs-tab-billing'
    )
    expect(wrapper.text()).toContain('Billing')
    expect(wrapper.text()).toContain('计费规则')
    expect(wrapper.text()).toContain(
      '实际扣费 =（输入 Token × 输入单价 + 输出 Token × 输出单价 + 缓存费用）× 用户组倍率'
    )
    expect(wrapper.findAll('.home-docs-billing-rule')).toHaveLength(4)
    expect(wrapper.text()).toContain('缓存创建与缓存读取按对应缓存单价单独计算')
    expect(wrapper.text()).toContain('相同 Request ID 与 API Key 的重复结算会被幂等去重')
    expect(wrapper.text()).toContain('余额不足时最多扣至 0，并记录未结算差额')

    await wrapper.get('#home-docs-tab-faq').trigger('click')
    expect(wrapper.text()).toContain('出现 401 时先检查什么？')
    expect(wrapper.text()).toContain('在哪里核对余额和实际扣费？')
  })

  it('keeps an optional extension-doc link separate from the built-in core guide', async () => {
    const wrapper = mount(HomeDocsDialog, {
      attachTo: document.body,
      props: {
        open: true,
        baseUrl: 'https://gateway.example',
        docUrl: 'https://docs.example/ssxz',
        createKeyPath: '/register'
      },
      global: {
        stubs: {
          RouterLink: {
            props: ['to'],
            template: '<a :data-to="to"><slot /></a>'
          }
        }
      }
    })

    await nextTick()

    const extensionLink = wrapper.get('a[href="https://docs.example/ssxz"]')
    expect(extensionLink.text()).toContain('扩展文档')
    expect(wrapper.text()).toContain('快速开始')
  })
})
