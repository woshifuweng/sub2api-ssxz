<template>
  <AppSectionShell
    title="AI 工作台"
    subtitle="从一个问题、一段商品信息或一个创作想法开始，先在静态工作台里整理方向；真实聊天、作图和用量能力留到后续批次接入。"
    eyebrow="SSXZ AI 工作台"
    icon="sparkles"
  >
    <template #actions>
      <div class="app-workspace-actions">
        <RouterLink to="/app/developer" class="ssxz-btn-secondary rounded-full">
          <Icon name="terminal" size="sm" />
          开发者接入
        </RouterLink>
        <RouterLink to="/app/billing" class="ssxz-btn-primary rounded-full">
          <Icon name="creditCard" size="sm" />
          余额与账单
        </RouterLink>
      </div>
    </template>

    <section class="app-workspace-hero">
      <div class="app-workspace-hero-copy">
        <div class="hero-pill">统一 /app 体验准备中</div>
        <h1>把聊天、作图、开发者和账户入口收在一个工作台里。</h1>
        <p>
          当前页面只提供静态工作台骨架和本地草稿预览，不会自动请求聊天、作图、用量、计费或支付接口。
        </p>
      </div>

      <div class="app-workspace-composer">
        <textarea
          v-model="draft"
          rows="4"
          placeholder="记录一个问题、商品卖点、作图想法或接口接入计划..."
        />
        <div class="composer-footer">
          <div class="prompt-chip-row">
            <button
              v-for="prompt in promptChips"
              :key="prompt.label"
              type="button"
              class="prompt-chip"
              @click="applyPrompt(prompt.prompt)"
            >
              {{ prompt.label }}
            </button>
          </div>
          <button type="button" class="ssxz-btn-primary rounded-full" :disabled="!draft.trim()" @click="recordDraft">
            <Icon name="arrowUp" size="sm" />
            记录草稿
          </button>
        </div>
      </div>
    </section>

    <section v-if="localMessages.length" class="workspace-local-thread">
      <article v-for="message in localMessages" :key="message.id" class="local-message">
        <span class="local-message-role">{{ message.role }}</span>
        <p>{{ message.content }}</p>
      </article>
    </section>

    <section class="workspace-entry-grid" aria-label="工作台入口">
      <RouterLink v-for="entry in workspaceEntries" :key="entry.to" :to="entry.to" class="workspace-entry-card">
        <span class="workspace-entry-icon">
          <Icon :name="entry.icon" size="sm" />
        </span>
        <span class="workspace-entry-copy">
          <span class="workspace-entry-kicker">{{ entry.kicker }}</span>
          <span class="workspace-entry-title">{{ entry.title }}</span>
          <span class="workspace-entry-description">{{ entry.description }}</span>
        </span>
        <Icon name="arrowUp" size="xs" class="workspace-entry-arrow" />
      </RouterLink>
    </section>

    <section class="workspace-note-strip">
      <span class="note-dot" />
      <span>router 继续负责 /app* 登录拦截；本组件不弹登录框、不拼 returnTo、不触发真实请求。</span>
    </section>
  </AppSectionShell>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import AppSectionShell from '@/components/user/AppSectionShell.vue'

type IconName = InstanceType<typeof Icon>['$props']['name']

interface WorkspaceEntry {
  title: string
  kicker: string
  description: string
  icon: IconName
  to: string
}

interface LocalMessage {
  id: number
  role: '草稿'
  content: string
}

const draft = ref('')
const localMessages = ref<LocalMessage[]>([])

const promptChips = [
  { label: '商品卖点', prompt: '请整理一个商品的 5 个核心卖点，突出使用场景、差异化和购买理由。' },
  { label: '作图想法', prompt: '请把这段需求改写成适合作图的画面描述，包含主体、背景、光线和风格。' },
  { label: '接口计划', prompt: '请列出一次 API 接入前需要确认的模型、鉴权、限额、错误处理和上线检查。' }
]

const workspaceEntries: WorkspaceEntry[] = [
  {
    title: 'AI 聊天',
    kicker: '当前工作台',
    description: '先记录本地草稿；真实消息保存和模型回复留到后续接入。',
    icon: 'chat',
    to: '/app/chat'
  },
  {
    title: 'AI 作图',
    kicker: '静态入口',
    description: '进入作图工作区前的入口占位，不触发真实作图任务。',
    icon: 'sparkles',
    to: '/app/image'
  },
  {
    title: '开发者 API',
    kicker: '接入准备',
    description: '查看后续 API、模型、密钥和权限说明的承载入口。',
    icon: 'terminal',
    to: '/app/developer'
  },
  {
    title: '余额与账单',
    kicker: '账户入口',
    description: '仅保留导航入口，不触发支付、扣费或账单请求。',
    icon: 'creditCard',
    to: '/app/billing'
  },
  {
    title: '账户设置',
    kicker: '个人中心',
    description: '承载账户资料、偏好与安全设置入口。',
    icon: 'userCircle',
    to: '/app/account'
  }
]

function applyPrompt(prompt: string) {
  draft.value = prompt
}

function recordDraft() {
  const content = draft.value.trim()
  if (!content) return
  localMessages.value.unshift({
    id: Date.now(),
    role: '草稿',
    content
  })
  draft.value = ''
}
</script>

<style scoped>
.app-workspace-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.app-workspace-hero {
  display: grid;
  gap: 1.25rem;
}

.app-workspace-hero-copy {
  max-width: 54rem;
}

.hero-pill {
  display: inline-flex;
  border: 1px solid var(--ssxz-border);
  border-radius: 9999px;
  background: var(--ssxz-surface-muted);
  padding: 0.5rem 0.85rem;
  color: var(--ssxz-body);
  font-size: 0.75rem;
  font-weight: 720;
}

.app-workspace-hero h1 {
  margin-top: 1rem;
  max-width: 46rem;
  color: var(--ssxz-text);
  font-size: clamp(2rem, 4vw, 3.5rem);
  font-weight: 760;
  line-height: 1.05;
}

.app-workspace-hero p {
  margin-top: 1rem;
  max-width: 44rem;
  color: var(--ssxz-muted);
  font-size: 0.95rem;
  line-height: 1.8;
}

.app-workspace-composer {
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-2xl);
  background: var(--ssxz-surface-raised);
  box-shadow: var(--ssxz-shadow);
  padding: 0.85rem;
}

.app-workspace-composer textarea {
  min-height: 8rem;
  width: 100%;
  resize: vertical;
  border: 0;
  background: transparent;
  color: var(--ssxz-text);
  font-size: 1rem;
  line-height: 1.7;
  outline: none;
}

.app-workspace-composer textarea::placeholder {
  color: var(--ssxz-subtle);
}

.composer-footer {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-top: 1px solid var(--ssxz-border);
  padding-top: 0.75rem;
}

.prompt-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.prompt-chip {
  border: 1px solid var(--ssxz-border);
  border-radius: 9999px;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-body);
  padding: 0.45rem 0.75rem;
  font-size: 0.78rem;
  font-weight: 680;
  transition: border-color 0.16s ease, color 0.16s ease, background-color 0.16s ease;
}

.prompt-chip:hover {
  border-color: color-mix(in srgb, var(--ssxz-primary) 42%, var(--ssxz-border));
  background: var(--ssxz-active-bg);
  color: var(--ssxz-primary);
}

.workspace-local-thread {
  display: grid;
  gap: 0.75rem;
}

.local-message {
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-xl);
  background: var(--ssxz-surface-muted);
  padding: 0.9rem;
}

.local-message-role {
  color: var(--ssxz-primary);
  font-size: 0.75rem;
  font-weight: 760;
}

.local-message p {
  margin-top: 0.35rem;
  color: var(--ssxz-body);
  line-height: 1.7;
  white-space: pre-wrap;
}

.workspace-entry-grid {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 0.85rem;
}

.workspace-entry-card {
  display: grid;
  min-height: 13rem;
  grid-template-rows: auto 1fr auto;
  gap: 0.8rem;
  border: 1px solid var(--ssxz-border);
  border-radius: var(--ssxz-radius-xl);
  background: color-mix(in srgb, var(--ssxz-surface-raised) 88%, transparent);
  box-shadow: var(--ssxz-shadow-sm);
  color: var(--ssxz-text);
  padding: 1rem;
  text-decoration: none;
  transition: border-color 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
}

.workspace-entry-card:hover {
  border-color: color-mix(in srgb, var(--ssxz-primary) 42%, var(--ssxz-border));
  box-shadow: var(--ssxz-shadow);
  transform: translateY(-1px);
}

.workspace-entry-icon {
  display: inline-flex;
  height: 2.35rem;
  width: 2.35rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--ssxz-border);
  border-radius: 0.85rem;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-primary);
}

.workspace-entry-copy {
  min-width: 0;
}

.workspace-entry-kicker {
  display: block;
  color: var(--ssxz-muted);
  font-size: 0.72rem;
  font-weight: 760;
}

.workspace-entry-title {
  display: block;
  margin-top: 0.25rem;
  font-size: 1rem;
  font-weight: 760;
}

.workspace-entry-description {
  display: block;
  margin-top: 0.45rem;
  color: var(--ssxz-muted);
  font-size: 0.82rem;
  line-height: 1.65;
}

.workspace-entry-arrow {
  color: var(--ssxz-subtle);
  rotate: 45deg;
}

.workspace-note-strip {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  border: 1px solid var(--ssxz-border);
  border-radius: 9999px;
  background: var(--ssxz-surface-muted);
  color: var(--ssxz-muted);
  padding: 0.75rem 1rem;
  font-size: 0.82rem;
}

.note-dot {
  height: 0.55rem;
  width: 0.55rem;
  flex: 0 0 auto;
  border-radius: 9999px;
  background: var(--ssxz-success);
  box-shadow: 0 0 0 4px color-mix(in srgb, var(--ssxz-success) 18%, transparent);
}

@media (max-width: 1120px) {
  .workspace-entry-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .app-workspace-actions,
  .composer-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .workspace-entry-grid {
    grid-template-columns: 1fr;
  }

  .workspace-entry-card {
    min-height: 0;
  }

  .workspace-note-strip {
    align-items: flex-start;
    border-radius: var(--ssxz-radius-xl);
  }
}
</style>
