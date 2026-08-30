<template>
  <FoundationDialog
    class="home-docs-dialog"
    :open="open"
    title="SSXZ 接入文档"
    description="从创建 API Key 到客户端连通，先完成下面这组核心步骤。"
    @update:open="emit('update:open', $event)"
  >
    <div class="home-docs-tabs" role="tablist" aria-label="接入文档章节">
      <button
        v-for="tab in tabs"
        :id="`home-docs-tab-${tab.id}`"
        :key="tab.id"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.id"
        :aria-controls="`home-docs-panel-${tab.id}`"
        :class="{ 'is-active': activeTab === tab.id }"
        @click="activeTab = tab.id"
      >
        <component :is="tab.icon" aria-hidden="true" />
        {{ tab.label }}
      </button>
    </div>

    <section
      v-if="activeTab === 'quickstart'"
      id="home-docs-panel-quickstart"
      role="tabpanel"
      aria-labelledby="home-docs-tab-quickstart"
      class="home-docs-panel"
    >
      <ol class="home-docs-steps">
        <li>
          <span class="home-docs-step-number">1</span>
          <div>
            <h3>创建并保存 API Key</h3>
            <p>在控制台选择实际要使用的分组。完整 Key 只交给你本人使用，不要发到聊天或截图中。</p>
            <RouterLink :to="createKeyPath" @click="closeDialog">前往 API Key</RouterLink>
          </div>
        </li>
        <li>
          <span class="home-docs-step-number">2</span>
          <div>
            <h3>填写统一接口地址</h3>
            <p>客户端使用当前站点自动生成的 Base URL，不需要手工猜域名或重复添加路径。</p>
            <div class="home-docs-copy-row">
              <code>{{ apiBaseUrl }}</code>
              <FoundationButton
                variant="ghost"
                size="sm"
                title="复制 Base URL"
                aria-label="复制 Base URL"
                @click="copy(apiBaseUrl)"
              >
                <Copy aria-hidden="true" />
                复制
              </FoundationButton>
            </div>
          </div>
        </li>
        <li>
          <span class="home-docs-step-number">3</span>
          <div>
            <h3>只选择当前 Key 可用的模型</h3>
            <p>以客户端模型下拉或 <code>/v1/models</code> 返回结果为准。首次接入先发一条短消息，再到用量页核对记录。</p>
            <RouterLink to="/app/usage" @click="closeDialog">查看用量与账单</RouterLink>
          </div>
        </li>
      </ol>
    </section>

    <section
      v-else-if="activeTab === 'clients'"
      id="home-docs-panel-clients"
      role="tabpanel"
      aria-labelledby="home-docs-tab-clients"
      class="home-docs-panel"
    >
      <div class="home-docs-clients">
        <details v-for="client in clientGuides" :key="client.id" :open="client.id === 'claude'">
          <summary>
            <span class="home-docs-client-icon">
              <component :is="client.icon" aria-hidden="true" />
            </span>
            <span>
              <strong>{{ client.name }}</strong>
              <small>{{ client.summary }}</small>
            </span>
            <ChevronDown class="home-docs-chevron" aria-hidden="true" />
          </summary>
          <div class="home-docs-client-body">
            <div v-for="file in client.files" :key="file.path" class="home-docs-code-block">
              <div class="home-docs-code-header">
                <span>{{ file.path }}</span>
                <FoundationButton
                  variant="ghost"
                  size="sm"
                  :title="`复制 ${file.path}`"
                  :aria-label="`复制 ${file.path}`"
                  @click="copy(file.content)"
                >
                  <Copy aria-hidden="true" />
                  复制
                </FoundationButton>
              </div>
              <CodeHighlight :code="file.content" :language="file.language" dense />
            </div>
          </div>
        </details>
      </div>
    </section>

    <section
      v-else-if="activeTab === 'billing'"
      id="home-docs-panel-billing"
      role="tabpanel"
      aria-labelledby="home-docs-tab-billing"
      class="home-docs-panel"
    >
      <div class="home-docs-billing">
        <header class="home-docs-billing-header">
          <span>Billing</span>
          <h3>计费规则</h3>
          <p>输入、输出和缓存按实际调用模型的对应单价分别计算，再应用当前 Key 所属用户组的倍率。</p>
        </header>

        <div class="home-docs-billing-formula">
          <strong>Token 计费公式</strong>
          <code>实际扣费 =（输入 Token × 输入单价 + 输出 Token × 输出单价 + 缓存费用）× 用户组倍率</code>
        </div>

        <ol class="home-docs-billing-rules">
          <li class="home-docs-billing-rule">
            <span>01</span>
            <div>
              <h4>输入与输出分开计费</h4>
              <p>输入和输出 Token 使用实际调用模型各自的单价；适用服务等级或长上下文价格时，会先据此确定单价。</p>
            </div>
          </li>
          <li class="home-docs-billing-rule">
            <span>02</span>
            <div>
              <h4>缓存单独计费</h4>
              <p>缓存创建与缓存读取按对应缓存单价单独计算；支持 5 分钟与 1 小时明细的模型会分别核算。</p>
            </div>
          </li>
          <li class="home-docs-billing-rule">
            <span>03</span>
            <div>
              <h4>最后应用用户组倍率</h4>
              <p>模型基础费用汇总后，再乘当前 Key 所属用户组倍率，形成该次请求的实际扣费。</p>
            </div>
          </li>
          <li class="home-docs-billing-rule">
            <span>04</span>
            <div>
              <h4>幂等与余额保护</h4>
              <p>相同 Request ID 与 API Key 的重复结算会被幂等去重，不会重复扣费；余额不足时最多扣至 0，并记录未结算差额。</p>
            </div>
          </li>
        </ol>
      </div>
    </section>

    <section
      v-else
      id="home-docs-panel-faq"
      role="tabpanel"
      aria-labelledby="home-docs-tab-faq"
      class="home-docs-panel"
    >
      <div class="home-docs-faq">
        <details v-for="(item, index) in faqItems" :key="item.question" :open="index === 0">
          <summary>
            <span>{{ item.question }}</span>
            <ChevronDown aria-hidden="true" />
          </summary>
          <p>{{ item.answer }}</p>
        </details>
      </div>
    </section>

    <template #footer>
      <a
        v-if="docUrl"
        :href="docUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="f0-button f0-button--outline"
      >
        扩展文档
        <ExternalLink aria-hidden="true" />
      </a>
      <FoundationButton variant="secondary" @click="closeDialog">关闭</FoundationButton>
      <RouterLink :to="createKeyPath" class="f0-button f0-button--default" @click="closeDialog">
        创建 API Key
        <ArrowRight aria-hidden="true" />
      </RouterLink>
    </template>
  </FoundationDialog>
</template>

<script setup lang="ts">
import { computed, ref, toRef, type Component } from 'vue'
import { RouterLink } from 'vue-router'
import {
  ArrowRight,
  BookOpen,
  ChevronDown,
  CircleHelp,
  Code2,
  Copy,
  ExternalLink,
  PanelsTopLeft,
  ReceiptText,
  Rocket,
  Route,
  Sparkles,
  Terminal
} from '@lucide/vue'
import { FoundationButton, FoundationDialog } from '@/components/foundation'
import CodeHighlight from './CodeHighlight.vue'
import { normalizeApiBaseUrl, useCliConfigs } from './home-config'

type DocsTab = 'quickstart' | 'clients' | 'billing' | 'faq'

interface ClientGuide {
  id: string
  name: string
  summary: string
  icon: Component
  files: Array<{
    path: string
    content: string
    language: string
  }>
}

const props = defineProps<{
  open: boolean
  baseUrl: string
  docUrl: string
  createKeyPath: string
}>()

const emit = defineEmits<{
  copy: [text: string]
  'update:open': [value: boolean]
}>()

const tabs: Array<{ id: DocsTab; label: string; icon: Component }> = [
  { id: 'quickstart', label: '快速开始', icon: Rocket },
  { id: 'clients', label: '客户端配置', icon: Route },
  { id: 'billing', label: '计费规则', icon: ReceiptText },
  { id: 'faq', label: '常见问题', icon: CircleHelp }
]

const activeTab = ref<DocsTab>('quickstart')
const apiBaseUrl = computed(() => normalizeApiBaseUrl(props.baseUrl))
const {
  claudeConfig,
  codexConfig,
  codexAuthConfig,
  geminiEnvConfig,
  geminiSettingsConfig
} = useCliConfigs(toRef(props, 'baseUrl'))

const cherryStudioConfig = computed(() => `API 类型: OpenAI Compatible
API 地址: ${apiBaseUrl.value}
API Key: your-api-key
模型: <当前 Key 可用模型>`)

const ccSwitchConfig = computed(() => `入口: SSXZ 控制台 > API Key
操作: 选择目标 Key > 导入到 CC Switch
Base URL: ${apiBaseUrl.value}
模型: <当前 Key 可用模型>`)

const clientGuides = computed<ClientGuide[]>(() => [
  {
    id: 'claude',
    name: 'Claude Code',
    summary: '设置认证令牌、站点根地址与当前 Key 可用模型。',
    icon: Code2,
    files: [{ path: '~/.claude/settings.json', content: claudeConfig.value, language: 'json' }]
  },
  {
    id: 'codex',
    name: 'Codex CLI',
    summary: '保留 Responses 协议，新增 SSXZ provider。',
    icon: Terminal,
    files: [
      { path: '~/.codex/config.toml', content: codexConfig.value, language: 'toml' },
      { path: '~/.codex/auth.json', content: codexAuthConfig.value, language: 'json' }
    ]
  },
  {
    id: 'gemini',
    name: 'Gemini CLI',
    summary: '通过环境变量连接当前站点，并保留 Gemini CLI 的本地设置。',
    icon: Sparkles,
    files: [
      { path: '~/.gemini/.env', content: geminiEnvConfig.value, language: 'plaintext' },
      { path: '~/.gemini/settings.json', content: geminiSettingsConfig.value, language: 'json' }
    ]
  },
  {
    id: 'cherry',
    name: 'Cherry Studio',
    summary: '按 OpenAI Compatible 类型填写地址、Key 与模型。',
    icon: PanelsTopLeft,
    files: [{ path: '服务商配置', content: cherryStudioConfig.value, language: 'plaintext' }]
  },
  {
    id: 'ccswitch',
    name: 'CC Switch',
    summary: '优先从 API Key 页面一键导入，避免手工填错。',
    icon: BookOpen,
    files: [{ path: '一键导入', content: ccSwitchConfig.value, language: 'plaintext' }]
  }
])

const faqItems = [
  {
    question: '客户端里应该选择哪个模型？',
    answer: '只选择当前 API Key 的模型列表中可见的型号。不同分组和上游账号支持范围不同，以客户端下拉或 /v1/models 返回结果为准。'
  },
  {
    question: '出现 401 时先检查什么？',
    answer: '先确认使用的是 SSXZ API Key、Base URL 没有重复拼接 /v1，并检查 Key 是否启用、过期或受 IP 白名单限制。'
  },
  {
    question: '为什么客户端能连接但模型调用失败？',
    answer: '先换回当前 Key 列表中明确可见的模型；若仍失败，到用量页查看请求结果，并联系站点管理员核对该分组的上游账号状态。'
  },
  {
    question: '在哪里核对余额和实际扣费？',
    answer: '在控制台的用量与账单页面按时间和模型查看。首次接入建议先发一条短消息，再核对对应记录和余额变化。'
  }
]

function copy(text: string): void {
  emit('copy', text)
}

function closeDialog(): void {
  emit('update:open', false)
}
</script>

<style scoped>
.home-docs-dialog {
  width: min(62rem, calc(100vw - 2rem));
}

.home-docs-dialog :deep(.f0-dialog-content) {
  gap: 1.25rem;
  padding-top: 1rem;
}

.home-docs-tabs {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.375rem;
  padding: 0.375rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--muted));
}

.home-docs-tabs button {
  display: inline-flex;
  min-width: 0;
  min-height: 2.5rem;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border: 1px solid transparent;
  border-radius: calc(var(--radius) - 0.125rem);
  padding: 0 0.75rem;
  color: hsl(var(--muted-foreground));
  background: transparent;
  font-size: 0.8125rem;
  font-weight: 620;
  cursor: pointer;
}

.home-docs-tabs button:hover,
.home-docs-tabs button.is-active {
  color: hsl(var(--foreground));
  background: hsl(var(--card));
  border-color: hsl(var(--border));
}

.home-docs-tabs svg,
.home-docs-copy-row svg,
.home-docs-code-header svg,
.home-docs-dialog :deep(.f0-dialog-footer svg) {
  width: 1rem;
  height: 1rem;
}

.home-docs-panel {
  min-height: 23rem;
}

.home-docs-steps {
  display: grid;
  gap: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.home-docs-steps li {
  display: grid;
  grid-template-columns: 2rem minmax(0, 1fr);
  gap: 1rem;
  padding: 1.125rem 0;
  border-bottom: 1px solid hsl(var(--border) / 0.7);
}

.home-docs-steps li:last-child {
  border-bottom: 0;
}

.home-docs-step-number,
.home-docs-client-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  color: hsl(var(--foreground));
  background: hsl(var(--muted));
}

.home-docs-step-number {
  width: 2rem;
  height: 2rem;
  font-size: 0.75rem;
  font-weight: 700;
}

.home-docs-steps h3,
.home-docs-billing h3,
.home-docs-billing h4,
.home-docs-billing p,
.home-docs-steps p,
.home-docs-faq p {
  margin: 0;
}

.home-docs-steps h3 {
  font-size: 0.9375rem;
  line-height: 1.4rem;
}

.home-docs-steps p,
.home-docs-faq p {
  margin-top: 0.35rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.8125rem;
  line-height: 1.45rem;
}

.home-docs-steps a {
  display: inline-flex;
  margin-top: 0.6rem;
  color: hsl(var(--foreground));
  font-size: 0.8125rem;
  font-weight: 650;
  text-underline-offset: 0.2rem;
}

.home-docs-copy-row {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  margin-top: 0.7rem;
  padding: 0.45rem 0.5rem 0.45rem 0.75rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--muted));
}

.home-docs-copy-row code {
  min-width: 0;
  overflow: hidden;
  font-size: 0.8125rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.home-docs-clients,
.home-docs-faq {
  display: grid;
  gap: 0.625rem;
}

.home-docs-clients details,
.home-docs-faq details {
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--card));
}

.home-docs-clients summary,
.home-docs-faq summary {
  display: flex;
  min-height: 3.5rem;
  align-items: center;
  gap: 0.75rem;
  padding: 0.7rem 0.875rem;
  cursor: pointer;
  list-style: none;
}

.home-docs-clients summary::-webkit-details-marker,
.home-docs-faq summary::-webkit-details-marker {
  display: none;
}

.home-docs-client-icon {
  width: 2.25rem;
  height: 2.25rem;
  flex: 0 0 auto;
}

.home-docs-client-icon svg,
.home-docs-faq summary svg,
.home-docs-chevron {
  width: 1rem;
  height: 1rem;
}

.home-docs-clients summary > span:nth-child(2) {
  display: grid;
  min-width: 0;
  flex: 1;
}

.home-docs-clients summary strong,
.home-docs-faq summary span {
  font-size: 0.875rem;
  font-weight: 650;
}

.home-docs-clients summary small {
  margin-top: 0.15rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  line-height: 1.1rem;
}

.home-docs-chevron,
.home-docs-faq summary svg {
  flex: 0 0 auto;
  color: hsl(var(--muted-foreground));
  transition: transform 180ms ease;
}

.home-docs-clients details[open] .home-docs-chevron,
.home-docs-faq details[open] summary svg {
  transform: rotate(180deg);
}

.home-docs-client-body {
  display: grid;
  gap: 0.75rem;
  padding: 0 0.875rem 0.875rem;
}

.home-docs-code-block {
  min-width: 0;
}

.home-docs-code-header {
  display: flex;
  min-height: 2.25rem;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  color: hsl(var(--muted-foreground));
  font-family: var(--font-mono, "Cascadia Code", monospace);
  font-size: 0.75rem;
}

.home-docs-billing {
  display: grid;
  gap: 1rem;
}

.home-docs-billing-header > span {
  color: hsl(var(--muted-foreground));
  font-size: 0.6875rem;
  font-weight: 720;
  letter-spacing: 0;
  text-transform: uppercase;
}

.home-docs-billing-header h3 {
  margin-top: 0.2rem;
  font-size: 1.125rem;
  line-height: 1.6rem;
}

.home-docs-billing-header p {
  margin-top: 0.3rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.8125rem;
  line-height: 1.45rem;
}

.home-docs-billing-formula {
  display: grid;
  gap: 0.5rem;
  padding: 0.875rem 1rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--muted));
}

.home-docs-billing-formula strong {
  font-size: 0.75rem;
  font-weight: 680;
}

.home-docs-billing-formula code {
  overflow-x: auto;
  color: hsl(var(--foreground));
  font-family: var(--font-mono, "Cascadia Code", monospace);
  font-size: 0.8125rem;
  line-height: 1.45rem;
  white-space: nowrap;
}

.home-docs-billing-rules {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

.home-docs-billing-rule {
  display: grid;
  min-width: 0;
  grid-template-columns: 2rem minmax(0, 1fr);
  gap: 0.75rem;
  padding: 0.875rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--card));
}

.home-docs-billing-rule > span {
  color: hsl(var(--muted-foreground));
  font-family: var(--font-mono, "Cascadia Code", monospace);
  font-size: 0.75rem;
  font-weight: 720;
  line-height: 1.35rem;
}

.home-docs-billing-rule h4 {
  font-size: 0.8125rem;
  line-height: 1.35rem;
}

.home-docs-billing-rule p {
  margin-top: 0.25rem;
  color: hsl(var(--muted-foreground));
  font-size: 0.75rem;
  line-height: 1.3rem;
}

.home-docs-faq details {
  padding: 0 0.875rem;
}

.home-docs-faq summary {
  padding-right: 0;
  padding-left: 0;
}

.home-docs-faq summary span {
  flex: 1;
}

.home-docs-faq p {
  margin: -0.25rem 0 0.9rem;
}

.home-docs-dialog :deep(.f0-dialog-footer a) {
  text-decoration: none;
}

@media (max-width: 640px) {
  .home-docs-dialog {
    width: calc(100vw - 1rem);
    max-height: calc(100dvh - 1rem);
  }

  .home-docs-dialog :deep(.f0-dialog-header) {
    padding: 1rem 1rem 0;
  }

  .home-docs-dialog :deep(.f0-dialog-content) {
    padding: 1rem;
  }

  .home-docs-dialog :deep(.f0-dialog-footer) {
    padding: 0 1rem 1rem;
  }

  .home-docs-tabs button {
    min-height: 2.75rem;
    padding: 0 0.35rem;
    font-size: 0.75rem;
  }

  .home-docs-tabs button svg {
    display: none;
  }

  .home-docs-panel {
    min-height: 0;
  }

  .home-docs-steps li {
    grid-template-columns: 1.75rem minmax(0, 1fr);
    gap: 0.75rem;
  }

  .home-docs-step-number {
    width: 1.75rem;
    height: 1.75rem;
  }

  .home-docs-billing-formula code {
    white-space: normal;
  }

  .home-docs-billing-rules {
    grid-template-columns: 1fr;
  }

  .home-docs-client-icon {
    display: none;
  }

  .home-docs-dialog :deep(.f0-dialog-footer) {
    display: grid;
    grid-template-columns: 1fr;
  }

  .home-docs-dialog :deep(.f0-dialog-footer > *) {
    width: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-docs-chevron,
  .home-docs-faq summary svg {
    transition: none;
  }
}
</style>
