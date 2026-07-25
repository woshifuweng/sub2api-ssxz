import { computed, type Ref } from 'vue'
import { Cable, KeyRound, Network, ReceiptText } from '@lucide/vue'

export const SECTIONS = {
  HOME: 0,
  CLAUDE: 1,
  CODEX: 2,
  GEMINI: 3,
  FEATURES: 4
} as const

export type SectionIndex = (typeof SECTIONS)[keyof typeof SECTIONS]

export const sections = [
  { name: '首页' },
  { name: 'Claude Code' },
  { name: 'Codex CLI' },
  { name: 'Gemini CLI' },
  { name: '平台能力' }
] as const

export const featureCards = [
  {
    icon: KeyRound,
    title: '一个 Key，连接多模型',
    desc: 'OpenAI、Claude、Gemini 与 Grok 统一从一个入口接入，模型范围以当前 Key 实际可用目录为准。'
  },
  {
    icon: Cable,
    title: '主流客户端，即插即用',
    desc: 'Claude Code、Codex CLI、Gemini CLI、Cherry Studio 与 CC Switch 按标准接口接入。'
  },
  {
    icon: ReceiptText,
    title: '官方模型，按量计费',
    desc: '输入、输出、实际扣费与余额变化集中记录，可按模型和时间核对。'
  },
  {
    icon: Network,
    title: '多账号自动切换',
    desc: '同组账号统一参与调度，单个账号暂时不可用时继续选择组内可用账号。'
  }
] as const

export function normalizeApiBaseUrl(value: string): string {
  const trimmed = value.trim().replace(/\/+$/, '')
  if (!trimmed) return '/v1'
  return /\/v1$/i.test(trimmed) ? trimmed : `${trimmed}/v1`
}

function gatewayRoot(value: string): string {
  return normalizeApiBaseUrl(value).replace(/\/v1$/i, '') || '/'
}

export function useCliConfigs(baseUrl: Ref<string>) {
  const apiBaseUrl = computed(() => normalizeApiBaseUrl(baseUrl.value))
  const rootUrl = computed(() => gatewayRoot(baseUrl.value))

  const claudeConfig = computed(() => `{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "your-api-key",
    "ANTHROPIC_BASE_URL": "${rootUrl.value}",
    "ANTHROPIC_MODEL": "<当前 Key 可用模型>"
  }
}`)

  const codexConfig = computed(() => `model_provider = "ssxz"
model = "<当前 Key 可用模型>"
model_reasoning_effort = "high"
disable_response_storage = true

[model_providers.ssxz]
name = "SSXZ API"
base_url = "${apiBaseUrl.value}"
wire_api = "responses"
requires_openai_auth = true`)

  const codexAuthConfig = computed(() => `{
  "OPENAI_API_KEY": "your-api-key"
}`)

  const geminiEnvConfig = computed(() => `GOOGLE_GEMINI_BASE_URL=${rootUrl.value}
GEMINI_API_KEY=your-api-key
GEMINI_MODEL=<当前 Key 可用模型>`)

  const geminiSettingsConfig = computed(() => `{
  "ide": {
    "enabled": true
  },
  "security": {
    "auth": {
      "selectedType": "gemini-api-key"
    }
  }
}`)

  return {
    claudeConfig,
    codexConfig,
    codexAuthConfig,
    geminiEnvConfig,
    geminiSettingsConfig
  }
}

export const panelClasses = {
  commandPanel: 'rounded-lg border command-panel-surface',
  configPanel: 'rounded-lg border config-panel',
  panelHeader: 'px-4 py-2 panel-header',
  codeBody: 'code-panel-body',
  iconButtonSmall: [
    'flex items-center justify-center rounded-md border h-7 w-7',
    'border-[hsl(var(--border))]',
    'bg-transparent text-[hsl(var(--muted-foreground))]',
    'transition hover:bg-[hsl(var(--accent))] hover:text-[hsl(var(--foreground))]'
  ].join(' ')
} as const

export type ProviderLogoType = 'claude' | 'openai' | 'gemini'

export function getLogoType(section: number): ProviderLogoType {
  switch (section) {
    case SECTIONS.CLAUDE: return 'claude'
    case SECTIONS.GEMINI: return 'gemini'
    default: return 'openai'
  }
}

export function getLogoClass(section: number): string {
  switch (section) {
    case SECTIONS.CLAUDE: return 'text-[#D97757]'
    case SECTIONS.CODEX: return 'text-[#191919] dark:text-white'
    default: return ''
  }
}
