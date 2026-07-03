import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: vi.fn().mockResolvedValue(true)
  })
}))

import UseKeyModal from '../UseKeyModal.vue'

describe('UseKeyModal', () => {
  it('shows third-party client guidance by default', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    expect(wrapper.text()).toContain('keys.useKeyModal.cliTabs.thirdParty')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.title')
    expect(wrapper.text()).toContain('CC Switch')
    expect(wrapper.text()).toContain('Cherry Studio')
    expect(wrapper.text()).toContain('Chatbox')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.ccSwitchFields.baseUrl')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.ccSwitchFields.apiEndpoint')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.cherryStudioFields.provider')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.chatboxFields.apiHost')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.connectionChecklistTitle')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.connectionChecklistEndpoint')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.connectionChecklistRestart')
    expect(wrapper.find('[data-testid="third-party-connection-checklist"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="third-party-troubleshooting"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.troubleshootingTitle')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.troubleshooting401')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.troubleshooting503')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.troubleshootingUsageHint')
    expect(wrapper.text()).toContain('keys.useKeyModal.thirdParty.securityNote')
    expect(wrapper.text()).toContain('https://example.com/v1')
    expect(wrapper.text()).toContain('https://example.com/v1/models')
    expect(wrapper.find('pre code').exists()).toBe(false)
  })

  it('warns when the key cannot currently be used by clients', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com',
        platform: 'openai',
        keyStatus: 'quota_exhausted'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    expect(wrapper.find('[data-testid="key-status-warning"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('keys.useKeyModal.statusWarning.quotaExhaustedTitle')
    expect(wrapper.text()).toContain('keys.useKeyModal.statusWarning.quotaExhaustedDescription')
  })

  it('does not duplicate v1 in third-party connection guidance', () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    expect(wrapper.text()).toContain('https://example.com/v1')
    expect(wrapper.text()).toContain('https://example.com/v1/models')
    expect(wrapper.text()).not.toContain('/v1/v1')
  })

  it('renders updated GPT-5.4 mini/nano names in OpenCode config', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const opencodeTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.opencode')
    )

    expect(opencodeTab).toBeDefined()
    await opencodeTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('"name": "GPT-5.4 Mini"')
    expect(codeBlock.text()).toContain('"name": "GPT-5.4 Nano"')
  })

  it('uses the first allowed model in Codex CLI configs when the key is model-restricted', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai',
        allowedModels: ['gpt-4.1', 'gpt-4o-mini']
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const codexTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.codexCli')
    )

    expect(codexTab).toBeDefined()
    await codexTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('model_provider = "ssxz"')
    expect(codeBlock.text()).toContain('model = "gpt-4.1"')
    expect(codeBlock.text()).toContain('model_reasoning_effort = "medium"')
    expect(codeBlock.text()).toContain('[model_providers.ssxz]')
    expect(codeBlock.text()).toContain('name = "SSXZ API"')
    expect(codeBlock.text()).toContain('base_url = "https://example.com/v1"')
    expect(codeBlock.text()).toContain('wire_api = "responses"')
    expect(codeBlock.text()).toContain('requires_openai_auth = true')
    expect(codeBlock.text()).not.toContain('[model_providers.OpenAI]')
    expect(codeBlock.text()).not.toContain('[model_providers.custom]')
    expect(codeBlock.text()).not.toContain('model = "gpt-5.5"')
  })

  it('uses the SSXZ provider name in Codex WebSocket configs', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1',
        platform: 'openai',
        allowedModels: ['gpt-5.5']
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const codexWsTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.codexCliWs')
    )

    expect(codexWsTab).toBeDefined()
    await codexWsTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('model_provider = "ssxz"')
    expect(codeBlock.text()).toContain('[model_providers.ssxz]')
    expect(codeBlock.text()).toContain('name = "SSXZ API"')
    expect(codeBlock.text()).toContain('supports_websockets = true')
    expect(codeBlock.text()).not.toContain('[model_providers.OpenAI]')
  })

  it('uses the first allowed model in Gemini CLI configs when the key is model-restricted', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-test',
        baseUrl: 'https://example.com/v1beta',
        platform: 'gemini',
        allowedModels: ['gemini-2.5-pro', 'gemini-2.5-flash']
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const geminiTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.geminiCli')
    )

    expect(geminiTab).toBeDefined()
    await geminiTab!.trigger('click')
    await nextTick()

    const codeBlock = wrapper.find('pre code')
    expect(codeBlock.exists()).toBe(true)
    expect(codeBlock.text()).toContain('GEMINI_MODEL="gemini-2.5-pro"')
    expect(codeBlock.text()).not.toContain('GEMINI_MODEL="gemini-2.0-flash"')
  })

  it('does not render CLI configs when only a masked API key is available', async () => {
    const wrapper = mount(UseKeyModal, {
      props: {
        show: true,
        apiKey: 'sk-user-...1234',
        baseUrl: 'https://example.com/v1',
        platform: 'openai'
      },
      global: {
        stubs: {
          BaseDialog: {
            template: '<div><slot /><slot name="footer" /></div>'
          },
          Icon: {
            template: '<span />'
          }
        }
      }
    })

    const codexTab = wrapper.findAll('button').find((button) =>
      button.text().includes('keys.useKeyModal.cliTabs.codexCli')
    )

    expect(codexTab).toBeDefined()
    await codexTab!.trigger('click')
    await nextTick()

    expect(wrapper.find('[data-testid="full-key-missing-warning"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('keys.useKeyModal.fullKeyMissingTitle')
    expect(wrapper.find('pre code').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('sk-user-...1234')
  })
})
