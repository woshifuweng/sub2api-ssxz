<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      referrerpolicy="no-referrer"
      :src="homeContentUrl"
      class="h-screen w-full border-0"
      allowfullscreen
    />
    <div v-else v-html="renderedHomeContent"></div>
  </div>

  <div v-else class="home-gateway">
    <header class="home-nav">
      <RouterLink to="/home" class="home-brand" aria-label="SSXZ AI Gateway 首页">
        <strong>{{ siteName }}</strong>
        <span>AI Gateway</span>
      </RouterLink>

      <nav class="home-nav-links" aria-label="首页导航">
        <a href="#product">产品</a>
        <a href="#models">模型生态</a>
        <a href="#integration">接入方式</a>
        <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer">文档</a>
      </nav>

      <div class="home-nav-actions">
        <LocaleSwitcher />
        <RouterLink :to="isAuthenticated ? dashboardPath : '/login'" class="home-nav-login">
          {{ isAuthenticated ? '控制台' : '登录' }}
        </RouterLink>
        <RouterLink :to="primaryCtaPath" class="home-nav-cta">
          {{ isAuthenticated ? '进入控制台' : '开始使用' }}
        </RouterLink>
      </div>
    </header>

    <main>
      <section class="home-hero">
        <div class="hero-copy">
          <p class="hero-product-type">Developer AI Infrastructure Platform</p>
          <h1>SSXZ AI Gateway</h1>
          <p class="hero-positioning">
            一个 API Key，连接多模型、多供应商，统一管理额度、用量和状态。
          </p>
          <div class="hero-actions">
            <RouterLink :to="primaryCtaPath" class="home-button home-button-primary">
              进入控制台
              <Icon name="arrowRight" size="sm" />
            </RouterLink>
            <a href="#integration" class="home-button home-button-secondary">查看接入方式</a>
          </div>
          <div class="hero-capabilities" aria-label="核心能力">
            <span>API Key</span>
            <span>OpenAI Compatible</span>
            <span>多模型</span>
            <span>Usage</span>
            <span>Billing</span>
          </div>
        </div>

        <div
          ref="networkRef"
          class="gateway-network"
          :class="{ 'is-active': networkActive }"
          aria-label="多模型通过 SSXZ Gateway 统一接入"
        >
          <div class="network-frame">
            <div class="network-caption">
              <span>Unified routing</span>
              <strong>Provider network</strong>
            </div>

            <svg class="network-lines" viewBox="0 0 560 420" aria-hidden="true">
              <path class="network-line line-openai" d="M280 210 C280 142 256 104 160 82" />
              <path class="network-line line-claude" d="M280 210 C214 210 157 210 86 210" />
              <path class="network-line line-gemini" d="M280 210 C346 210 403 210 474 210" />
              <path class="network-line line-grok" d="M280 210 C280 278 304 316 400 338" />

              <path class="network-signal signal-openai" d="M160 82 C256 104 280 142 280 210" />
              <path class="network-signal signal-claude" d="M86 210 C157 210 214 210 280 210" />
              <path class="network-signal signal-gemini" d="M474 210 C403 210 346 210 280 210" />
              <path class="network-signal signal-grok" d="M400 338 C304 316 280 278 280 210" />
            </svg>

            <div class="provider-node node-openai">
              <span class="provider-logo provider-logo-light">
                <ModelIcon model="gpt-4" size="34px" />
              </span>
              <span>OpenAI</span>
            </div>
            <div class="provider-node node-claude">
              <span class="provider-logo">
                <ModelIcon model="claude" size="34px" />
              </span>
              <span>Claude</span>
            </div>
            <div class="provider-node node-gemini">
              <span class="provider-logo">
                <ModelIcon model="gemini" size="34px" />
              </span>
              <span>Gemini</span>
            </div>
            <div class="provider-node node-grok">
              <span class="provider-logo provider-logo-light">
                <ModelIcon model="grok" size="34px" />
              </span>
              <span>Grok</span>
            </div>

            <div class="gateway-core">
              <span>SSXZ</span>
              <small>Gateway</small>
            </div>

            <div class="network-status">
              <span class="network-status-dot" aria-hidden="true"></span>
              Unified endpoint
            </div>
          </div>
        </div>
      </section>

      <section id="product" class="home-section product-section">
        <div class="section-heading">
          <h2>把多供应商接入收敛为一套工作流</h2>
          <p>从创建 Key 到核对用量和账单，用户始终留在同一个 SSXZ 产品体系里。</p>
        </div>

        <div class="product-ledger">
          <RouterLink to="/app/keys" class="product-row">
            <span class="product-index">01</span>
            <div>
              <h3>API Keys</h3>
              <p>创建、分组、查看一次性密钥，并接入常用客户端。</p>
            </div>
            <Icon name="arrowRight" size="sm" />
          </RouterLink>
          <RouterLink to="/app/available-channels" class="product-row">
            <span class="product-index">02</span>
            <div>
              <h3>Models</h3>
              <p>按后端配置查看当前 Key 分组可用的模型与价格。</p>
            </div>
            <Icon name="arrowRight" size="sm" />
          </RouterLink>
          <RouterLink to="/app/usage" class="product-row">
            <span class="product-index">03</span>
            <div>
              <h3>Usage &amp; Billing</h3>
              <p>核对请求、Token、费用和余额，定位失败调用。</p>
            </div>
            <Icon name="arrowRight" size="sm" />
          </RouterLink>
        </div>
      </section>

      <section id="models" class="home-section model-section">
        <div class="section-heading section-heading-wide">
          <div>
            <h2>真实模型生态，统一入口管理</h2>
            <p>真实品牌标识保持原始形态，具体模型范围以后端配置和当前 Key 分组为准。</p>
          </div>
          <RouterLink to="/app/available-channels" class="section-link">
            查看模型与价格
            <Icon name="arrowRight" size="sm" />
          </RouterLink>
        </div>

        <div ref="providerStoriesRef" class="provider-stories">
          <article
            v-for="provider in providers"
            :key="provider.id"
            class="provider-story"
            :class="`provider-story--${provider.id}`"
            :data-provider="provider.id"
          >
            <div class="provider-story-visual" aria-hidden="true">
              <div class="provider-motion-stage">
                <span class="provider-motion-ring provider-motion-ring--outer"></span>
                <span class="provider-motion-ring provider-motion-ring--inner"></span>
                <span class="provider-motion-signal"></span>
                <span class="provider-motion-scan"></span>
                <div
                  class="provider-story-logo"
                  :class="{ 'is-light': provider.id === 'openai' || provider.id === 'grok' }"
                >
                  <span class="provider-story-logo-trace">
                    <ModelIcon :model="provider.iconModel" size="58%" />
                  </span>
                  <span class="provider-story-logo-color">
                    <ModelIcon :model="provider.iconModel" size="58%" />
                  </span>
                </div>
              </div>
              <div class="provider-path-labels">
                <span>{{ provider.owner }}</span>
                <span>SSXZ Gateway</span>
                <span>Unified API</span>
              </div>
            </div>

            <div class="provider-story-copy">
              <p class="provider-story-owner">{{ provider.owner }}</p>
              <h3>{{ provider.name }}</h3>
              <p class="provider-description">{{ provider.description }}</p>
              <ul class="provider-capability-list">
                <li v-for="capability in provider.capabilities" :key="capability">
                  <span aria-hidden="true"></span>
                  {{ capability }}
                </li>
              </ul>
              <div class="provider-config-row">
                <span>Endpoint</span>
                <code>{{ displayedApiBaseUrl }}</code>
              </div>
              <RouterLink to="/app/available-channels" class="provider-detail-link">
                查看当前可用模型
                <Icon name="arrowRight" size="sm" />
              </RouterLink>
            </div>
          </article>
        </div>
      </section>

      <section id="integration" class="home-section integration-section">
        <div class="integration-copy">
          <div class="section-heading">
            <h2>四步完成接入</h2>
            <p>创建 Key，设置 Base URL，选择后端已开放模型，然后在使用记录里核对调用。</p>
          </div>

          <ol class="integration-steps">
            <li><span>1</span><div><strong>创建 API Key</strong><small>按用途和模型组分配 Key。</small></div></li>
            <li><span>2</span><div><strong>设置 Base URL</strong><small>使用 SSXZ OpenAI-compatible 地址。</small></div></li>
            <li><span>3</span><div><strong>选择模型</strong><small>可用范围以后端配置为准。</small></div></li>
            <li><span>4</span><div><strong>查看用量</strong><small>核对请求、费用、延迟和失败记录。</small></div></li>
          </ol>

          <div class="client-list" aria-label="第三方客户端">
            <span>Cherry Studio</span>
            <span>Chatbox</span>
            <span>CC Switch</span>
            <span>Custom Client</span>
          </div>
        </div>

        <div class="code-panel">
          <div class="code-panel-head">
            <div>
              <span>Integration</span>
              <strong>{{ activeCodeTab.label }}</strong>
            </div>
            <button type="button" class="copy-button" @click="copyCode">
              <Icon name="clipboard" size="xs" />
              复制
            </button>
          </div>
          <div class="code-tabs" role="tablist" aria-label="代码示例">
            <button
              v-for="tab in codeTabs"
              :key="tab.id"
              type="button"
              :class="{ 'is-active': activeCodeTabId === tab.id }"
              role="tab"
              :aria-selected="activeCodeTabId === tab.id"
              @click="activeCodeTabId = tab.id"
            >
              {{ tab.label }}
            </button>
          </div>
          <pre><code>{{ activeCode }}</code></pre>
          <div class="base-url-row">
            <span>Base URL</span>
            <code>{{ displayedApiBaseUrl }}</code>
          </div>
        </div>
      </section>

      <section class="home-section developer-entry">
        <div>
          <h2>从真实控制台开始</h2>
          <p>创建 Key、查看模型、核对用量和管理账单都在同一套 SSXZ 控制台里完成。</p>
        </div>
        <div class="developer-actions">
          <RouterLink :to="primaryCtaPath" class="home-button home-button-primary">进入控制台</RouterLink>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="home-button home-button-secondary">
            打开文档
          </a>
          <RouterLink v-else to="/app/keys?guide=clients" class="home-button home-button-secondary">
            查看接入文档
          </RouterLink>
        </div>
      </section>
    </main>

    <footer class="home-footer">
      <div>
        <strong>{{ siteName }}</strong>
        <span>Developer AI Infrastructure Platform</span>
      </div>
      <nav aria-label="页脚导航">
        <a href="#models">Models</a>
        <a href="#integration">Integration</a>
        <RouterLink to="/app/channel-status">Status</RouterLink>
        <RouterLink to="/app/profile">Account</RouterLink>
      </nav>
      <span>&copy; {{ currentYear }}</span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { renderRichContent } from '@/utils/sanitize'
import { sanitizeUrl } from '@/utils/url'
import { DEFAULT_SITE_NAME, normalizeSiteName } from '@/utils/brand'
import { resolvePublicApiBaseUrl } from '@/utils/publicApiBaseUrl'

const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() =>
  normalizeSiteName(appStore.cachedPublicSettings?.site_name || appStore.siteName || DEFAULT_SITE_NAME)
)
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const homeContentUrl = computed(() => sanitizeUrl(homeContent.value))
const renderedHomeContent = computed(() => renderRichContent(homeContent.value))
const isHomeContentUrl = computed(() => !!homeContentUrl.value)
const displayedApiBaseUrl = computed(() => {
  const configured = appStore.cachedPublicSettings?.api_base_url || appStore.apiBaseUrl
  const currentOrigin = typeof window !== 'undefined' ? window.location.origin : ''
  return resolvePublicApiBaseUrl(configured, currentOrigin)
})

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/app/dashboard'))
const primaryCtaPath = computed(() => (isAuthenticated.value ? dashboardPath.value : '/register'))
const currentYear = computed(() => new Date().getFullYear())

const providers = [
  {
    id: 'openai',
    name: 'OpenAI',
    owner: 'OpenAI',
    iconModel: 'gpt-4',
    description: '通过统一兼容接口接入后端已配置的 OpenAI 模型，并在同一处核对用量、费用与失败记录。',
    capabilities: ['统一 API Key', '兼容常用 OpenAI 客户端', '按 Key 分组控制模型范围']
  },
  {
    id: 'claude',
    name: 'Claude',
    owner: 'Anthropic',
    iconModel: 'claude',
    description: 'Claude 模型通过 SSXZ 的已配置通道提供，具体模型和可用范围由当前 Key 分组决定。',
    capabilities: ['保持统一接入方式', '记录请求用量与费用', '通道状态集中查看']
  },
  {
    id: 'gemini',
    name: 'Gemini',
    owner: 'Google',
    iconModel: 'gemini',
    description: '在统一 Key 和计量体系下使用后端开放的 Gemini 能力，无需为每个客户端维护多套凭据。',
    capabilities: ['多供应商凭据收敛', '统一余额与计量', '客户端配置保持一致']
  },
  {
    id: 'grok',
    name: 'Grok',
    owner: 'xAI',
    iconModel: 'grok',
    description: 'Grok 入口与其他供应商保持同一套接入方式，开放状态和模型范围以后端配置为准。',
    capabilities: ['统一 Base URL', '模型开放状态可见', '调用记录集中核对']
  }
] as const

type CodeTabId = 'curl' | 'node' | 'python' | 'env'

const codeTabs: Array<{ id: CodeTabId; label: string }> = [
  { id: 'curl', label: 'cURL' },
  { id: 'node', label: 'Node.js' },
  { id: 'python', label: 'Python' },
  { id: 'env', label: 'Environment' }
]
const activeCodeTabId = ref<CodeTabId>('curl')
const activeCodeTab = computed(() =>
  codeTabs.find(tab => tab.id === activeCodeTabId.value) || codeTabs[0]
)
const activeCode = computed(() => {
  const baseUrl = displayedApiBaseUrl.value.replace(/\/$/, '')
  switch (activeCodeTabId.value) {
    case 'node':
      return `const response = await fetch('${baseUrl}/chat/completions', {
  method: 'POST',
  headers: {
    Authorization: \`Bearer \${process.env.SSXZ_API_KEY}\`,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({
    model: process.env.SSXZ_MODEL,
    messages: [{ role: 'user', content: 'Hello' }]
  })
})`
    case 'python':
      return `import os
import requests

response = requests.post(
    "${baseUrl}/chat/completions",
    headers={"Authorization": f"Bearer {os.environ['SSXZ_API_KEY']}"},
    json={
        "model": os.environ["SSXZ_MODEL"],
        "messages": [{"role": "user", "content": "Hello"}],
    },
)`
    case 'env':
      return `SSXZ_API_KEY=sk-your-key
SSXZ_BASE_URL=${baseUrl}
SSXZ_MODEL=backend-configured-model`
    case 'curl':
    default:
      return `curl "${baseUrl}/chat/completions" \\
  -H "Authorization: Bearer $SSXZ_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"'$SSXZ_MODEL'","messages":[{"role":"user","content":"Hello"}]}'`
  }
})

const networkRef = ref<HTMLElement | null>(null)
const providerStoriesRef = ref<HTMLElement | null>(null)
const networkActive = ref(false)
let networkObserver: IntersectionObserver | null = null
let providerStoryObserver: IntersectionObserver | null = null

async function copyCode() {
  try {
    await navigator.clipboard.writeText(activeCode.value)
    appStore.showSuccess('代码已复制')
  } catch {
    appStore.showError('复制失败，请手动复制')
  }
}

onMounted(() => {
  document.documentElement.classList.add('dark')
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings()

  if ('IntersectionObserver' in window && networkRef.value) {
    networkObserver = new IntersectionObserver(
      entries => {
        if (entries.some(entry => entry.isIntersecting)) {
          networkActive.value = true
          networkObserver?.disconnect()
        }
      },
      { threshold: 0.35 }
    )
    networkObserver.observe(networkRef.value)
  } else {
    networkActive.value = true
  }

  const providerStories = providerStoriesRef.value?.querySelectorAll<HTMLElement>('.provider-story') || []
  if ('IntersectionObserver' in window && providerStories.length > 0) {
    providerStoryObserver = new IntersectionObserver(
      entries => {
        entries.forEach(entry => {
          if (!entry.isIntersecting) return
          entry.target.classList.add('is-visible')
          providerStoryObserver?.unobserve(entry.target)
        })
      },
      { threshold: 0.28, rootMargin: '0px 0px -8% 0px' }
    )
    providerStories.forEach(story => providerStoryObserver?.observe(story))
  } else {
    providerStories.forEach(story => story.classList.add('is-visible'))
  }
})

onBeforeUnmount(() => {
  networkObserver?.disconnect()
  providerStoryObserver?.disconnect()
})
</script>

<style scoped>
.home-gateway {
  min-height: 100vh;
  overflow-x: hidden;
  background: var(--ssxz-bg);
  color: var(--ssxz-text);
  font-family: var(--ssxz-font-sans);
}

.home-nav {
  position: sticky;
  top: 0;
  z-index: 40;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  width: 100%;
  min-height: 64px;
  align-items: center;
  border-bottom: 1px solid var(--ssxz-border);
  background: color-mix(in srgb, var(--ssxz-bg) 92%, transparent);
  padding: 0 max(24px, calc((100vw - 1180px) / 2));
  backdrop-filter: blur(14px);
}

.home-brand {
  display: inline-flex;
  width: fit-content;
  align-items: baseline;
  gap: 0.55rem;
  color: var(--ssxz-text);
  text-decoration: none;
}

.home-brand strong {
  font-size: 0.95rem;
  font-weight: 700;
}

.home-brand span {
  color: var(--ssxz-text-muted);
  font-size: 0.76rem;
}

.home-nav-links,
.home-nav-actions {
  display: flex;
  align-items: center;
}

.home-nav-links {
  gap: 1.6rem;
}

.home-nav-links a,
.home-nav-login {
  color: var(--ssxz-text-muted);
  font-size: 0.84rem;
  font-weight: 500;
  text-decoration: none;
}

.home-nav-links a:hover,
.home-nav-login:hover {
  color: var(--ssxz-text);
}

.home-nav-actions {
  justify-content: flex-end;
  gap: 0.75rem;
}

.home-nav-actions :deep(button) {
  color: var(--ssxz-text-muted);
}

.home-nav-cta {
  display: inline-flex;
  min-height: 36px;
  align-items: center;
  border: 1px solid var(--ssxz-primary);
  border-radius: var(--ssxz-radius-button);
  background: var(--ssxz-primary);
  padding: 0 0.9rem;
  color: var(--ssxz-action-text);
  font-size: 0.82rem;
  font-weight: 600;
  text-decoration: none;
}

.home-nav-cta:hover {
  border-color: var(--ssxz-primary-hover);
  background: var(--ssxz-primary-hover);
}

main,
.home-footer {
  width: min(1180px, calc(100% - 48px));
  margin: 0 auto;
}

.home-hero {
  display: grid;
  min-height: 700px;
  grid-template-columns: minmax(0, 0.92fr) minmax(460px, 1.08fr);
  align-items: center;
  gap: 4rem;
  padding: 5.5rem 0 6rem;
}

.hero-copy {
  max-width: 580px;
}

.hero-product-type {
  margin: 0;
  color: var(--ssxz-accent);
  font-family: var(--ssxz-font-mono);
  font-size: 0.78rem;
}

.hero-copy h1 {
  margin: 1rem 0 0;
  max-width: 10ch;
  color: var(--ssxz-text);
  font-size: 3.5rem;
  font-weight: 680;
  line-height: 1.04;
}

.hero-positioning {
  max-width: 30rem;
  margin: 1.35rem 0 0;
  color: var(--ssxz-text-secondary);
  font-size: 1.08rem;
  line-height: 1.8;
}

.hero-actions,
.developer-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.hero-actions {
  margin-top: 2rem;
}

.home-button {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border-radius: var(--ssxz-radius-button);
  padding: 0 1rem;
  font-size: 0.9rem;
  font-weight: 600;
  text-decoration: none;
  transition: background-color 160ms ease, border-color 160ms ease, color 160ms ease;
}

.home-button:focus-visible,
.home-nav a:focus-visible,
.provider-option:focus-visible,
.code-tabs button:focus-visible,
.copy-button:focus-visible {
  outline: 0;
  box-shadow: var(--ssxz-focus-ring);
}

.home-button-primary {
  border: 1px solid var(--ssxz-primary);
  background: var(--ssxz-primary);
  color: var(--ssxz-action-text);
}

.home-button-primary:hover {
  border-color: var(--ssxz-primary-hover);
  background: var(--ssxz-primary-hover);
}

.home-button-secondary {
  border: 1px solid var(--ssxz-border-strong);
  background: var(--ssxz-surface);
  color: var(--ssxz-text);
}

.home-button-secondary:hover {
  background: var(--ssxz-surface-raised);
}

.hero-capabilities {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 1.25rem;
}

.hero-capabilities span {
  border: 1px solid var(--ssxz-border);
  border-radius: 999px;
  background: var(--ssxz-bg-subtle);
  padding: 0.35rem 0.6rem;
  color: var(--ssxz-text-muted);
  font-family: var(--ssxz-font-mono);
  font-size: 0.7rem;
}

.gateway-network {
  min-width: 0;
}

.network-frame {
  position: relative;
  min-height: 520px;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-bg-subtle);
}

.network-caption {
  position: absolute;
  top: 1.1rem;
  left: 1.1rem;
  z-index: 3;
  display: grid;
  gap: 0.15rem;
}

.network-caption span {
  color: var(--ssxz-text-subtle);
  font-family: var(--ssxz-font-mono);
  font-size: 0.68rem;
}

.network-caption strong {
  color: var(--ssxz-text-secondary);
  font-size: 0.84rem;
  font-weight: 550;
}

.network-lines {
  position: absolute;
  inset: 2.9rem 0 0;
  width: 100%;
  height: calc(100% - 2.9rem);
}

.network-line,
.network-signal {
  fill: none;
  vector-effect: non-scaling-stroke;
}

.network-line {
  stroke: var(--ssxz-border-strong);
  stroke-width: 1.25;
  stroke-dasharray: 420;
  stroke-dashoffset: 420;
}

.network-signal {
  stroke: var(--ssxz-accent);
  stroke-width: 1.75;
  stroke-linecap: round;
  stroke-dasharray: 4 196;
  stroke-dashoffset: 200;
  opacity: 0;
}

.gateway-network.is-active .network-line {
  animation: network-line-draw 1.35s cubic-bezier(0.22, 1, 0.36, 1) forwards;
}

.gateway-network.is-active .line-claude { animation-delay: 120ms; }
.gateway-network.is-active .line-gemini { animation-delay: 240ms; }
.gateway-network.is-active .line-grok { animation-delay: 360ms; }
.gateway-network.is-active .network-signal {
  animation: network-signal-flow 6s linear 1.8s infinite;
}
.gateway-network.is-active .signal-claude { animation-delay: 3.1s; }
.gateway-network.is-active .signal-gemini { animation-delay: 4.4s; }
.gateway-network.is-active .signal-grok { animation-delay: 5.7s; }

.provider-node,
.gateway-core {
  position: absolute;
  z-index: 4;
}

.provider-node {
  display: grid;
  width: 92px;
  justify-items: center;
  gap: 0.5rem;
  opacity: 0;
  transform: scale(0.96);
}

.gateway-network.is-active .provider-node {
  animation: network-node-enter 520ms ease-out forwards;
}

.gateway-network.is-active .node-claude { animation-delay: 520ms; }
.gateway-network.is-active .node-gemini { animation-delay: 660ms; }
.gateway-network.is-active .node-grok { animation-delay: 800ms; }

.provider-node > span:last-child {
  color: var(--ssxz-text-muted);
  font-size: 0.75rem;
  font-weight: 550;
}

.provider-logo {
  display: grid;
  place-items: center;
  border: 1px solid var(--ssxz-border-strong);
  background: var(--ssxz-surface);
}

.provider-logo {
  width: 56px;
  height: 56px;
  border-radius: 14px;
}

.provider-logo img {
  display: block;
  width: 60%;
  height: 60%;
  object-fit: contain;
}

.provider-logo-light {
  background: #f8fafc;
}

.node-openai { top: 70px; left: calc(50% - 46px); }
.node-claude { top: calc(50% - 46px); left: 26px; }
.node-gemini { top: calc(50% - 46px); right: 26px; }
.node-grok { bottom: 40px; left: calc(50% - 46px); }

.gateway-core {
  top: calc(50% - 46px);
  left: calc(50% - 54px);
  display: grid;
  width: 108px;
  height: 92px;
  place-content: center;
  border: 1px solid color-mix(in srgb, var(--ssxz-primary) 48%, var(--ssxz-border));
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface-raised);
  opacity: 0;
  text-align: center;
}

.gateway-network.is-active .gateway-core {
  animation: network-node-enter 460ms ease-out 280ms forwards;
}

.gateway-core::after {
  content: "";
  position: absolute;
  inset: -7px;
  border: 1px solid color-mix(in srgb, var(--ssxz-primary) 18%, transparent);
  border-radius: 14px;
  animation: gateway-ring 4.8s ease-in-out infinite;
}

.gateway-core span {
  color: var(--ssxz-text);
  font-size: 0.95rem;
  font-weight: 700;
}

.gateway-core small {
  margin-top: 0.15rem;
  color: var(--ssxz-text-muted);
  font-family: var(--ssxz-font-mono);
  font-size: 0.66rem;
}

.network-status {
  position: absolute;
  right: 1rem;
  bottom: 0.9rem;
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  color: var(--ssxz-text-subtle);
  font-family: var(--ssxz-font-mono);
  font-size: 0.66rem;
}

.network-status-dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 50%;
  background: var(--ssxz-success);
}

.home-section {
  border-top: 1px solid var(--ssxz-border);
  padding: 6.5rem 0;
  scroll-margin-top: 64px;
}

.section-heading {
  max-width: 680px;
}

.section-heading h2,
.developer-entry h2 {
  margin: 0;
  color: var(--ssxz-text);
  font-size: 2rem;
  font-weight: 620;
  line-height: 1.18;
}

.section-heading p,
.developer-entry p {
  margin: 0.75rem 0 0;
  color: var(--ssxz-text-muted);
  font-size: 0.96rem;
  line-height: 1.75;
}

.section-heading-wide {
  display: flex;
  max-width: none;
  align-items: end;
  justify-content: space-between;
  gap: 2rem;
}

.section-link,
.provider-detail-link {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  color: var(--ssxz-accent);
  font-size: 0.84rem;
  font-weight: 550;
  text-decoration: none;
}

.product-ledger {
  margin-top: 2.25rem;
  border-top: 1px solid var(--ssxz-border);
}

.product-row {
  display: grid;
  min-height: 112px;
  grid-template-columns: 56px minmax(0, 1fr) auto;
  align-items: center;
  gap: 1rem;
  border-bottom: 1px solid var(--ssxz-border);
  color: inherit;
  text-decoration: none;
}

.product-row:hover {
  background: var(--ssxz-bg-subtle);
}

.product-index {
  color: var(--ssxz-text-subtle);
  font-family: var(--ssxz-font-mono);
  font-size: 0.72rem;
}

.product-row h3 {
  margin: 0;
  color: var(--ssxz-text);
  font-size: 1rem;
  font-weight: 600;
}

.product-row p {
  margin: 0.35rem 0 0;
  color: var(--ssxz-text-muted);
  font-size: 0.86rem;
}

.product-row > svg {
  color: var(--ssxz-text-subtle);
}

.provider-stories {
  display: grid;
  margin-top: 3rem;
  border-top: 1px solid var(--ssxz-border);
}

.provider-story {
  display: grid;
  min-height: min(76vh, 760px);
  grid-template-columns: minmax(0, 1.08fr) minmax(340px, 0.92fr);
  align-items: center;
  gap: clamp(3rem, 7vw, 7rem);
  border-bottom: 1px solid var(--ssxz-border);
  padding: clamp(4rem, 8vh, 7rem) 0;
}

.provider-story:nth-child(even) .provider-story-visual {
  order: 2;
}

.provider-story-visual,
.provider-story-copy {
  min-width: 0;
  opacity: 0;
  transform: translateY(24px);
  transition:
    opacity 720ms ease,
    transform 720ms cubic-bezier(0.22, 1, 0.36, 1);
}

.provider-story-copy {
  transition-delay: 140ms;
}

.provider-story.is-visible .provider-story-visual,
.provider-story.is-visible .provider-story-copy {
  opacity: 1;
  transform: translateY(0);
}

.provider-motion-stage {
  position: relative;
  display: grid;
  width: min(100%, 500px);
  aspect-ratio: 1;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: 18px;
  background: var(--ssxz-bg-subtle);
}

.provider-motion-stage::after {
  content: "";
  position: absolute;
  inset: 12%;
  border: 1px solid color-mix(in srgb, var(--ssxz-primary) 12%, transparent);
  border-radius: 50%;
}

.provider-motion-ring,
.provider-motion-signal,
.provider-motion-scan {
  position: absolute;
  border-radius: 50%;
  pointer-events: none;
}

.provider-motion-ring {
  border: 1px solid var(--ssxz-border-strong);
}

.provider-motion-ring--outer {
  width: 68%;
  height: 68%;
  border-top-color: color-mix(in srgb, var(--ssxz-primary) 64%, var(--ssxz-border));
}

.provider-motion-ring--inner {
  width: 48%;
  height: 48%;
  border-right-color: color-mix(in srgb, var(--ssxz-accent) 58%, var(--ssxz-border));
}

.provider-motion-signal {
  width: 9px;
  height: 9px;
  background: var(--ssxz-accent);
  box-shadow: 0 0 0 5px color-mix(in srgb, var(--ssxz-accent) 12%, transparent);
  offset-path: circle(34% at 50% 50%);
  opacity: 0;
}

.provider-story.is-visible .provider-motion-signal {
  animation: provider-signal-loop 8s linear 1.1s infinite;
}

.provider-motion-scan {
  width: 58%;
  height: 58%;
  background: conic-gradient(from 0deg, transparent 0 78%, color-mix(in srgb, var(--ssxz-accent) 18%, transparent) 90%, transparent 100%);
  opacity: 0;
}

.provider-story-logo {
  position: relative;
  z-index: 3;
  display: grid;
  width: 34%;
  min-width: 126px;
  aspect-ratio: 1;
  place-items: center;
  border: 1px solid var(--ssxz-border-strong);
  border-radius: 28px;
  background: var(--ssxz-surface);
  box-shadow: 0 24px 70px rgb(0 0 0 / 0.28);
}

.provider-story-logo.is-light {
  background: #f8fafc;
}

.provider-story-logo-trace,
.provider-story-logo-color {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
}

.provider-story-logo img {
  display: block;
  width: 58%;
  height: 58%;
  object-fit: contain;
}

.provider-story-logo-trace {
  filter: grayscale(1) contrast(0.65) opacity(0.35);
  clip-path: inset(0 0 0 0);
}

.provider-story-logo-color {
  clip-path: inset(0 100% 0 0);
}

.provider-story.is-visible .provider-story-logo-color {
  animation: provider-logo-reveal 1.45s cubic-bezier(0.22, 1, 0.36, 1) 260ms forwards;
}

.provider-story.is-visible .provider-story-logo-trace {
  animation: provider-trace-fade 1.6s ease 220ms forwards;
}

.provider-path-labels {
  display: grid;
  width: min(100%, 500px);
  grid-template-columns: repeat(3, 1fr);
  border: 1px solid var(--ssxz-border);
  border-top: 0;
  background: var(--ssxz-surface);
}

.provider-path-labels span {
  min-width: 0;
  border-right: 1px solid var(--ssxz-border);
  padding: 0.75rem;
  color: var(--ssxz-text-subtle);
  font-family: var(--ssxz-font-mono);
  font-size: 0.65rem;
  overflow: hidden;
  text-align: center;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-path-labels span:last-child {
  border-right: 0;
}

.provider-story-owner {
  margin: 0;
  color: var(--ssxz-accent);
  font-family: var(--ssxz-font-mono);
  font-size: 0.72rem;
}

.provider-story-copy h3 {
  margin: 0.65rem 0 0;
  color: var(--ssxz-text);
  font-size: clamp(2.4rem, 5vw, 4.5rem);
  font-weight: 620;
  line-height: 1;
}

.provider-description {
  max-width: 36rem;
  margin: 1.35rem 0 0;
  color: var(--ssxz-text-muted);
  font-size: 1rem;
  line-height: 1.8;
}

.provider-capability-list {
  display: grid;
  gap: 0.72rem;
  margin: 1.75rem 0 0;
  padding: 0;
  list-style: none;
}

.provider-capability-list li {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
}

.provider-capability-list li span {
  width: 5px;
  height: 5px;
  flex: 0 0 auto;
  border-radius: 50%;
  background: var(--ssxz-primary);
}

.provider-config-row {
  display: grid;
  margin-top: 1.8rem;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-button);
  background: var(--ssxz-surface-code);
}

.provider-config-row span {
  border-bottom: 1px solid var(--ssxz-border);
  padding: 0.55rem 0.75rem;
  color: var(--ssxz-text-subtle);
  font-size: 0.66rem;
}

.provider-config-row code {
  overflow: hidden;
  padding: 0.72rem 0.75rem;
  color: var(--ssxz-text-secondary);
  font-family: var(--ssxz-font-mono);
  font-size: 0.72rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.provider-detail-link {
  margin-top: 1.5rem;
}

.provider-story--openai.is-visible .provider-motion-ring--outer {
  animation: provider-ring-rotate 18s linear infinite;
}

.provider-story--claude.is-visible .provider-story-logo {
  animation: provider-breathe 4.8s ease-in-out 1.6s infinite;
}

.provider-story--gemini.is-visible .provider-motion-scan {
  animation: provider-highlight-sweep 5.5s ease-in-out 1.4s infinite;
}

.provider-story--grok.is-visible .provider-motion-scan {
  animation: provider-radar-scan 8s linear 1.2s infinite;
}

.integration-section {
  display: grid;
  grid-template-columns: minmax(0, 0.8fr) minmax(520px, 1.2fr);
  gap: 4rem;
}

.integration-steps {
  display: grid;
  gap: 0;
  margin: 2rem 0 0;
  padding: 0;
  list-style: none;
}

.integration-steps li {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr);
  gap: 0.85rem;
  border-top: 1px solid var(--ssxz-border);
  padding: 1rem 0;
}

.integration-steps li:last-child {
  border-bottom: 1px solid var(--ssxz-border);
}

.integration-steps li > span {
  color: var(--ssxz-accent);
  font-family: var(--ssxz-font-mono);
  font-size: 0.72rem;
}

.integration-steps li div {
  display: grid;
  gap: 0.2rem;
}

.integration-steps strong {
  color: var(--ssxz-text);
  font-size: 0.86rem;
  font-weight: 600;
}

.integration-steps small {
  color: var(--ssxz-text-muted);
  font-size: 0.78rem;
}

.client-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 1.25rem;
}

.client-list span {
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-button);
  background: var(--ssxz-bg-subtle);
  padding: 0.42rem 0.6rem;
  color: var(--ssxz-text-muted);
  font-size: 0.72rem;
}

.code-panel {
  align-self: start;
  overflow: hidden;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-card);
  background: var(--ssxz-surface);
}

.code-panel-head,
.base-url-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.code-panel-head {
  min-height: 64px;
  border-bottom: 1px solid var(--ssxz-border);
  padding: 0 1rem;
}

.code-panel-head > div {
  display: grid;
  gap: 0.15rem;
}

.code-panel-head span,
.base-url-row span {
  color: var(--ssxz-text-subtle);
  font-size: 0.68rem;
}

.code-panel-head strong {
  color: var(--ssxz-text-secondary);
  font-size: 0.82rem;
  font-weight: 550;
}

.copy-button {
  display: inline-flex;
  min-height: 34px;
  align-items: center;
  gap: 0.4rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-button);
  background: var(--ssxz-surface-raised);
  padding: 0 0.7rem;
  color: var(--ssxz-text-muted);
  font-size: 0.72rem;
}

.copy-button:hover {
  border-color: var(--ssxz-border-strong);
  color: var(--ssxz-text);
}

.code-tabs {
  display: flex;
  gap: 0.25rem;
  border-bottom: 1px solid var(--ssxz-border);
  padding: 0.5rem;
}

.code-tabs button {
  min-height: 32px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  padding: 0 0.7rem;
  color: var(--ssxz-text-muted);
  font-family: var(--ssxz-font-mono);
  font-size: 0.68rem;
}

.code-tabs button:hover,
.code-tabs button.is-active {
  background: var(--ssxz-surface-raised);
  color: var(--ssxz-text);
}

.code-panel pre {
  min-height: 270px;
  margin: 0;
  overflow: auto;
  background: var(--ssxz-surface-code);
  padding: 1.25rem;
}

.code-panel code {
  color: var(--ssxz-text-secondary);
  font-family: var(--ssxz-font-mono);
  font-size: 0.76rem;
  line-height: 1.75;
  white-space: pre-wrap;
}

.base-url-row {
  min-height: 54px;
  border-top: 1px solid var(--ssxz-border);
  padding: 0 1rem;
}

.base-url-row code {
  overflow: hidden;
  color: var(--ssxz-accent);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.developer-entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 3rem;
}

.developer-entry > div:first-child {
  max-width: 620px;
}

.home-footer {
  display: grid;
  min-height: 120px;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 2rem;
  border-top: 1px solid var(--ssxz-border);
  color: var(--ssxz-text-subtle);
  font-size: 0.74rem;
}

.home-footer > div {
  display: grid;
  gap: 0.2rem;
}

.home-footer strong {
  color: var(--ssxz-text-secondary);
  font-weight: 600;
}

.home-footer nav {
  display: flex;
  gap: 1.25rem;
}

.home-footer a {
  color: var(--ssxz-text-muted);
  text-decoration: none;
}

.home-footer a:hover {
  color: var(--ssxz-text);
}

@keyframes network-line-draw {
  to { stroke-dashoffset: 0; }
}

@keyframes network-signal-flow {
  0% { opacity: 0; stroke-dashoffset: 200; }
  8% { opacity: 0.9; }
  30% { opacity: 0.9; }
  40%, 100% { opacity: 0; stroke-dashoffset: 0; }
}

@keyframes network-node-enter {
  to { opacity: 1; transform: scale(1); }
}

@keyframes gateway-ring {
  0%, 100% { opacity: 0.45; transform: scale(0.99); }
  50% { opacity: 0.8; transform: scale(1.015); }
}

@keyframes provider-logo-reveal {
  to { clip-path: inset(0 0 0 0); }
}

@keyframes provider-trace-fade {
  0%, 58% { opacity: 1; }
  100% { opacity: 0; }
}

@keyframes provider-ring-rotate {
  to { transform: rotate(360deg); }
}

@keyframes provider-signal-loop {
  0% { offset-distance: 0%; opacity: 0; }
  8%, 86% { opacity: 0.8; }
  100% { offset-distance: 100%; opacity: 0; }
}

@keyframes provider-breathe {
  0%, 100% { box-shadow: 0 24px 70px rgb(0 0 0 / 0.28); }
  50% { box-shadow: 0 24px 70px rgb(0 0 0 / 0.28), 0 0 0 10px rgb(217 119 87 / 0.07); }
}

@keyframes provider-highlight-sweep {
  0%, 20% { opacity: 0; transform: rotate(-38deg); }
  44%, 58% { opacity: 0.8; }
  82%, 100% { opacity: 0; transform: rotate(38deg); }
}

@keyframes provider-radar-scan {
  0% { opacity: 0; transform: rotate(0deg); }
  10%, 90% { opacity: 0.72; }
  100% { opacity: 0; transform: rotate(360deg); }
}

@media (max-width: 1024px) {
  .home-nav {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .home-nav-links {
    display: none;
  }

  .home-hero {
    grid-template-columns: 1fr;
    gap: 2.5rem;
    padding-top: 4rem;
  }

  .hero-copy {
    max-width: 680px;
  }

  .gateway-network {
    width: min(100%, 680px);
  }

  .provider-story {
    min-height: 0;
    grid-template-columns: 1fr;
    gap: 2.5rem;
    padding: 5rem 0;
  }

  .provider-story:nth-child(even) .provider-story-visual {
    order: 0;
  }

  .provider-story-visual {
    width: min(100%, 680px);
  }

  .provider-motion-stage,
  .provider-path-labels {
    width: 100%;
  }

  .provider-story-copy {
    max-width: 680px;
  }

  .integration-section {
    grid-template-columns: 1fr;
  }

  .code-panel {
    width: 100%;
  }
}

@media (max-width: 720px) {
  .home-nav {
    padding: 0 16px;
  }

  .home-brand span,
  .home-nav-login,
  .home-nav-actions :deep(button) {
    display: none;
  }

  main,
  .home-footer {
    width: min(100% - 32px, 1180px);
  }

  .home-hero {
    min-height: auto;
    padding: 3.5rem 0 4rem;
  }

  .hero-copy h1 {
    font-size: 2.5rem;
  }

  .hero-positioning {
    font-size: 1rem;
  }

  .network-frame {
    min-height: 380px;
  }

  .network-lines { top: 2.4rem; }

  .node-openai { top: 58px; }
  .node-claude { left: 8px; }
  .node-gemini { right: 8px; }
  .node-grok { bottom: 25px; }

  .provider-logo {
    width: 48px;
    height: 48px;
  }

  .gateway-core {
    left: calc(50% - 48px);
    width: 96px;
    height: 84px;
  }

  .home-section {
    padding: 4.5rem 0;
  }

  .section-heading h2,
  .developer-entry h2 {
    font-size: 1.55rem;
  }

  .section-heading-wide,
  .developer-entry {
    align-items: flex-start;
    flex-direction: column;
  }

  .provider-stories {
    margin-top: 2rem;
  }

  .provider-story {
    gap: 2rem;
    padding: 4rem 0;
  }

  .provider-motion-stage {
    border-radius: 12px;
  }

  .provider-story-logo {
    min-width: 92px;
    border-radius: 20px;
  }

  .provider-story-copy h3 {
    font-size: 2.6rem;
  }

  .provider-path-labels span {
    padding-inline: 0.45rem;
    font-size: 0.58rem;
  }

  .product-row {
    grid-template-columns: 36px minmax(0, 1fr) auto;
  }

  .code-panel pre {
    min-height: 220px;
  }

  .home-footer {
    grid-template-columns: 1fr;
    gap: 1rem;
    padding: 2rem 0;
  }

  .home-footer nav {
    flex-wrap: wrap;
  }
}

@media (prefers-reduced-motion: reduce) {
  .gateway-network .network-line {
    stroke-dashoffset: 0;
  }

  .gateway-network .network-signal {
    display: none;
  }

  .gateway-network .provider-node,
  .gateway-network .gateway-core {
    opacity: 1;
    transform: none;
    animation: none;
  }

  .gateway-core::after {
    animation: none;
  }

  .provider-story-visual,
  .provider-story-copy {
    opacity: 1;
    transform: none;
    transition: none;
  }

  .provider-story-logo-color {
    clip-path: inset(0);
  }

  .provider-story-logo-trace,
  .provider-motion-signal,
  .provider-motion-scan {
    display: none;
  }

  .provider-story .provider-motion-ring,
  .provider-story .provider-story-logo,
  .provider-story .provider-story-logo-color {
    animation: none !important;
  }
}
</style>
