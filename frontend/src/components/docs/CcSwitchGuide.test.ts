import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import CcSwitchGuide from './CcSwitchGuide.vue'

describe('beginner guide', () => {
  it('numbers every stage and makes official downloads prominent', () => {
    const wrapper = mount(CcSwitchGuide)
    expect(wrapper.findAll('.guide-content h2').map(h => h.text())).toEqual([
      '第一步：下载并安装 CC Switch', '第二步：下载并安装 Codex',
      '第三步：在本站创建 API 密钥', '第四步：导入 SSXZ 并启用', '第五步：发送消息，确认连接成功'
    ])
    expect(wrapper.text()).not.toContain('再安装')
    expect(wrapper.findAll('a.guide-download')).toHaveLength(2)
    for (const link of wrapper.findAll('a.guide-download')) {
      expect(link.attributes('target')).toBe('_blank')
      expect(link.attributes('rel')).toContain('noopener')
      expect(link.text()).toContain('官方入口')
    }
  })
  beforeEach(() => {
    HTMLDialogElement.prototype.showModal = vi.fn(function(this: HTMLDialogElement) { this.setAttribute('open', '') })
    HTMLDialogElement.prototype.close = vi.fn(function(this: HTMLDialogElement) { this.removeAttribute('open') })
  })
  it('starts with Windows Codex and a direct official download page', () => {
    const wrapper = mount(CcSwitchGuide)
    expect(wrapper.text()).toContain('Codex · Windows')
    expect(wrapper.find('a[href="https://ccswitch.io/zh/download"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('https://api.ssxzapi.com/v1')
    expect(wrapper.text()).not.toContain('irm https://claude.ai')
  })
  it('switches client and operating system without mixing addresses or commands', async () => {
    const wrapper = mount(CcSwitchGuide)
    await wrapper.findAll('button').find(b => b.text() === 'Claude Code')!.trigger('click')
    expect(wrapper.text()).toContain('irm https://claude.ai/install.ps1 | iex')
    expect(wrapper.find('.guide-content').text()).not.toContain('https://api.ssxzapi.com/v1')
    await wrapper.findAll('button').find(b => b.text() === 'Mac')!.trigger('click')
    expect(wrapper.text()).toContain('curl -fsSL https://claude.ai/install.sh | bash')
    expect(wrapper.text()).not.toContain('irm https://claude.ai/install.ps1')
  })
  it('has keyboard-accessible real image links and a closeable preview', async () => {
    const wrapper = mount(CcSwitchGuide)
    const link = wrapper.find('a[data-guide-image]')
    expect(link.attributes('rel')).toContain('noopener')
    await link.trigger('click')
    expect(wrapper.find('dialog').attributes('open')).toBeDefined()
    expect(wrapper.find('dialog img').exists()).toBe(true)
    await wrapper.find('dialog button').trigger('click')
    expect(wrapper.find('dialog').attributes('open')).toBeUndefined()
  })
  it('keeps troubleshooting collapsed and image workflow separate', async () => {
    const wrapper = mount(CcSwitchGuide)
    expect(wrapper.findAll('details').length).toBeGreaterThan(3)
    expect(wrapper.find('details').attributes('open')).toBeUndefined()
    await wrapper.findAll('button').find(b => b.text() === '图片工作台')!.trigger('click')
    expect(wrapper.find('a[href="/app/image"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('不要连续重试')
    expect(wrapper.findAll('fieldset')).toHaveLength(1)
    expect(wrapper.findAll('.guide-content h2').map(h => h.text())).toEqual([
      '第一步：打开图片工作台', '第二步：选择图片来源和模型',
      '第三步：填写要求，点击生成', '第四步：查看图片和费用'
    ])
  })
})
