<template>
  <section ref="workspace" class="image-tools" :class="{ 'is-expanded': expanded }" aria-label="图片创作工具">
    <header class="tools-header">
      <div class="tools-tabs" role="tablist" aria-label="选择图片工具">
        <button v-for="tool in tools" :id="`tab-${tool.id}`" :key="tool.id" type="button" role="tab"
          :aria-selected="activeTool === tool.id" :aria-controls="`panel-${tool.id}`"
          :class="{ selected: activeTool === tool.id }" @click="selectTool(tool.id)">
          {{ tool.name }}
        </button>
      </div>
      <button type="button" class="tools-button" :aria-pressed="expanded" @click="expanded = !expanded">
        {{ expanded ? '退出全屏' : '展开工作区' }}
      </button>
    </header>

    <div v-if="!profile" class="tools-connect">
      <h2>{{ activeTool === 'canvas' ? '在画布上组织你的创作' : '生成、编辑和整理图片' }}</h2>
      <p>使用本站余额。选择图片分组后，系统会配置只用于图片生成的专用密钥，无需复制或填写。</p>
      <label for="image-tool-group">图片分组</label>
      <select id="image-tool-group" v-model="selectedGroupId" :disabled="!authStore.isAuthenticated || loading || connecting || groups.length === 0">
        <option :value="0" disabled>请选择分组</option>
        <option v-for="group in groups" :key="group.id" :value="group.id">{{ group.name }}</option>
      </select>
      <button type="button" class="tools-button primary" :disabled="!authStore.isAuthenticated || !selectedGroupId || loading || connecting" @click="connect">
        {{ connecting ? '正在连接…' : '进入创作' }}
      </button>
      <p v-if="loading" role="status">正在读取可用分组…</p>
      <p v-else-if="groups.length === 0 && !error" role="status">当前账户没有已启用图片生成的 OpenAI 分组，请联系站点客服。</p>
      <div v-if="error" role="alert" class="tools-error">
        <p>{{ error }}</p>
        <button v-if="groups.length === 0" type="button" class="tools-button" @click="loadGroups">重新读取</button>
        <RouterLink to="/app/keys">查看我的密钥</RouterLink>
      </div>
    </div>

    <template v-else>
      <div class="tools-connection">
        <span>{{ profile.name }} · {{ profile.model }}</span>
        <button type="button" class="tools-button" @click="showDisconnect = true">更换分组</button>
      </div>
      <div v-for="tool in tools" v-show="activeTool === tool.id" :id="`panel-${tool.id}`" :key="tool.id"
        role="tabpanel" :aria-labelledby="`tab-${tool.id}`" class="tools-panel">
        <div v-if="visited.has(tool.id) && !connected[tool.id]" class="tools-wait" role="status">
          正在连接{{ tool.name }}。如果浏览器拦截了嵌入页面，请使用下方原工作台入口。
        </div>
        <iframe v-if="visited.has(tool.id)" :ref="element => setFrame(tool.id, element)"
          :key="`${sessionId}-${tool.id}`" :src="tool.url" :title="tool.name" class="tools-frame"
          sandbox="allow-scripts allow-same-origin allow-downloads allow-forms"
          allow="clipboard-read; clipboard-write; fullscreen" referrerpolicy="no-referrer" />
      </div>
    </template>

    <footer class="tools-footer">
      <span>作品保存在当前浏览器，重要作品请及时导出；两套工具的历史独立保存。</span>
      <a href="https://image.ssxzapi.com/" target="_blank" rel="noopener noreferrer">打开原工作台 / 找回旧作品 ↗</a>
    </footer>

    <BaseDialog :show="showDisconnect" title="更换图片分组？" width="narrow" :z-index="110" @close="showDisconnect = false">
        <p>这会重新打开两套工具。请先等待图片生成完成，并保存当前编辑；切换不会取消已经提交的扣费请求。</p>
      <template #footer>
        <button type="button" class="tools-button" @click="showDisconnect = false">继续当前创作</button>
        <button type="button" class="tools-button primary" @click="disconnect">已保存，更换分组</button>
      </template>
    </BaseDialog>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import type { ComponentPublicInstance } from 'vue'
import { RouterLink } from 'vue-router'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAuthStore } from '@/stores/auth'
import * as keysAPI from '@/api/keys'
import { userGroupsAPI } from '@/api/groups'
import type { ApiKey, Group } from '@/types'
import { imageToolInit, imageToolKeyName, isImageToolMessage, isEligibleImageToolKey, isPlainImageKey, isUsableImageToolKey, supportsStationImages, IMAGE_TOOL_MODEL } from '@/utils/imageTools'
import type { ImageToolProfile } from '@/utils/imageTools'

type ToolId = 'workbench' | 'canvas'
// 工具独立发布，保持旧工作台的域名；配置只传给当前 iframe 的精确来源。
const tools: { id: ToolId; name: string; url: string }[] = [
  { id: 'workbench', name: '图片工作台', url: import.meta.env.VITE_IMAGE_WORKBENCH_EMBED_URL || 'https://image.ssxzapi.com/?ssxz=1' },
  { id: 'canvas', name: '无限画布', url: import.meta.env.VITE_IMAGE_CANVAS_EMBED_URL || 'https://canvas.ssxzapi.com/canvas?ssxz=1' },
]
const authStore = useAuthStore()
const workspace = ref<HTMLElement | null>(null)
const activeTool = ref<ToolId>('workbench')
const visited = reactive(new Set<ToolId>(['workbench']))
const frames: Partial<Record<ToolId, HTMLIFrameElement>> = {}
const connected = reactive({ workbench: false, canvas: false })
const groups = ref<Group[]>([])
const selectedGroupId = ref(0)
const profile = ref<ImageToolProfile | null>(null)
const loading = ref(false)
const connecting = ref(false)
const error = ref('')
const sessionId = ref('')
const expanded = ref(false)
const showDisconnect = ref(false)
let sessionOwner: number | null = null
let requestVersion = 0
let modelRequest: AbortController | null = null
let pendingCreation: { owner: number; group: number; idempotencyKey: string } | null = null

function setFrame(id: ToolId, element: Element | ComponentPublicInstance | null) {
  if (element instanceof HTMLIFrameElement) frames[id] = element
  else delete frames[id]
}

function selectTool(id: ToolId) {
  activeTool.value = id
  visited.add(id)
}

function isCurrent(owner: number, version: number) {
  return authStore.isAuthenticated && authStore.user?.id === owner && requestVersion === version
}

async function loadGroups() {
  const owner = authStore.user?.id
  if (!owner || !authStore.isAuthenticated) return
  const version = requestVersion
  loading.value = true
  error.value = ''
  try {
    const available = await userGroupsAPI.getAvailable()
    if (!isCurrent(owner, version)) return
    groups.value = available.filter(supportsStationImages)
    if (groups.value.length === 1) selectedGroupId.value = groups.value[0].id
  } catch {
    if (isCurrent(owner, version)) error.value = '图片分组读取失败，请重新读取。'
  } finally {
    if (isCurrent(owner, version)) loading.value = false
  }
}

async function connect() {
  const owner = authStore.user?.id
  const group = groups.value.find(item => item.id === selectedGroupId.value)
  if (!owner || !group || connecting.value || !authStore.isAuthenticated) return
  const version = requestVersion
  connecting.value = true
  error.value = ''
  try {
    const existing: ApiKey[] = []
    for (let page = 1; ; page++) {
      const result = await keysAPI.list(page, 100, { search: imageToolKeyName(group.id) })
      if (!isCurrent(owner, version)) return
      existing.push(...result.items.filter(key => key.name === imageToolKeyName(group.id)))
      if (page * result.page_size >= result.total || result.items.length === 0) break
    }
    let key = existing.find(item => isEligibleImageToolKey(item, owner, group.id))
    if (!key && existing.length) {
      error.value = '图片专用密钥已停用、到期或配置不匹配，请在“我的密钥”中检查。系统不会绕过已有的限制。'
      return
    }
    if (!key) {
      // One key per creation attempt/retry, not a permanent account+group key.
      // A deliberately deleted key must not be replayed by the native idempotency cache.
      if (pendingCreation?.owner !== owner || pendingCreation.group !== group.id) {
        pendingCreation = { owner, group: group.id, idempotencyKey: crypto.randomUUID() }
      }
      key = await keysAPI.create(imageToolKeyName(group.id), group.id, [group.id], [IMAGE_TOOL_MODEL], undefined, undefined, undefined, undefined, undefined, undefined,
        { idempotencyKey: pendingCreation.idempotencyKey })
    }
    if (!isCurrent(owner, version)) return
    pendingCreation = null
    if (!isPlainImageKey(key.key)) {
      const revealed = await keysAPI.reveal(key.id)
      if (!isCurrent(owner, version)) return
      key = { ...key, key: revealed.key }
    }
    if (!isUsableImageToolKey(key, owner, group.id)) {
      error.value = '图片专用密钥未通过校验，请检查分组与模型权限。'
      return
    }
    modelRequest = new AbortController()
    const response = await fetch('/v1/models', { headers: { Authorization: `Bearer ${key.key}` }, signal: modelRequest.signal, cache: 'no-store' })
    if (!isCurrent(owner, version)) return
    if (!response.ok) {
      error.value = '分组连接未通过，请检查账户余额、密钥限制或稍后重试。尚未提交图片生成。'
      return
    }
    const catalog = await response.json()
    if (!isCurrent(owner, version)) return
    if (!Array.isArray(catalog?.data) || !catalog.data.some((model: { id?: string }) => model?.id === IMAGE_TOOL_MODEL)) {
      error.value = '这个分组未提供当前图片模型，请选择其他图片分组。尚未提交图片生成。'
      return
    }
    sessionOwner = owner
    sessionId.value = crypto.randomUUID()
    profile.value = { name: group.name, baseUrl: 'https://api.ssxzapi.com/v1', apiKey: key.key, model: IMAGE_TOOL_MODEL }
  } catch {
    if (isCurrent(owner, version)) error.value = '图片工具连接未完成，请稍后重试。若已创建专用密钥，再次连接会复用，不会重复开通。'
  } finally {
    if (isCurrent(owner, version)) connecting.value = false
  }
}

function receiveMessage(event: MessageEvent) {
  if (!profile.value || !sessionOwner || !authStore.isAuthenticated || authStore.user?.id !== sessionOwner) return
  for (const tool of tools) {
    const source = frames[tool.id]?.contentWindow || null
    const origin = new URL(tool.url).origin
    if (isImageToolMessage(event, source, origin, 'ready')) {
      source?.postMessage(imageToolInit(sessionOwner, sessionId.value, profile.value), origin)
    } else if (isImageToolMessage(event, source, origin, 'connected') && event.data.sessionId === sessionId.value) {
      connected[tool.id] = true
    }
  }
}

function disconnect() {
  requestVersion++
  modelRequest?.abort()
  modelRequest = null
  profile.value = null
  sessionOwner = null
  sessionId.value = ''
  pendingCreation = null
  connected.workbench = connected.canvas = false
  connecting.value = loading.value = showDisconnect.value = false
}

function onStorage(event: StorageEvent) {
  // 同一客户刷新令牌或余额不打断创作；退出/换号则立即卸载临时凭据。
  if (event.key === 'auth_user' && event.newValue) {
    try { if (JSON.parse(event.newValue)?.id === authStore.user?.id) return } catch { /* 无效身份视为失效 */ }
  } else if (event.key === 'auth_token' && event.newValue) return
  else if (event.key !== 'auth_token' && event.key !== 'auth_user' && event.key !== null) return
  disconnect()
  groups.value = []
  selectedGroupId.value = 0
  error.value = '登录信息已在其他标签页变更，请刷新页面后继续。'
}

watch(() => authStore.user?.id, () => {
  disconnect()
  groups.value = []
  selectedGroupId.value = 0
  void loadGroups()
})
watch(() => authStore.isAuthenticated, authenticated => { if (!authenticated) disconnect() })

onMounted(() => {
  window.addEventListener('message', receiveMessage)
  window.addEventListener('storage', onStorage)
  void loadGroups()
})
onBeforeUnmount(() => {
  disconnect()
  window.removeEventListener('message', receiveMessage)
  window.removeEventListener('storage', onStorage)
})
</script>

<style scoped>
.image-tools { display: flex; flex-direction: column; min-height: 38rem; border: 1px solid var(--ssxz-border); border-radius: 12px; background: var(--ssxz-bg); overflow: hidden; }
.image-tools.is-expanded { position: fixed; inset: 0; z-index: 100; height: 100dvh; border-radius: 0; }
.tools-header, .tools-connection, .tools-footer { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: .75rem; padding: .75rem 1rem; }
.tools-header { border-bottom: 1px solid var(--ssxz-border); }
.tools-tabs { display: flex; gap: .25rem; }
.tools-tabs button { padding: .6rem 1rem; border-radius: 7px; color: var(--ssxz-text-muted); }
.tools-tabs button.selected { background: var(--ssxz-surface-raised); color: var(--ssxz-text); font-weight: 650; box-shadow: inset 0 -2px var(--ssxz-text); }
.tools-button { border: 1px solid var(--ssxz-border); border-radius: 7px; padding: .5rem .85rem; font-size: .875rem; }
.tools-button.primary { color: var(--ssxz-bg); background: var(--ssxz-text); }
button:disabled { opacity: .5; cursor: not-allowed; }
button:focus-visible, select:focus-visible, a:focus-visible { outline: 2px solid var(--ssxz-text); outline-offset: 3px; }
.tools-connect { display: grid; align-content: center; gap: 1rem; width: min(100%, 36rem); padding: 3rem 1.5rem; margin: auto; }
.tools-connect h2 { font-size: 1.4rem; font-weight: 650; }
.tools-connect p, .tools-footer, .tools-wait { color: var(--ssxz-text-muted); font-size: .875rem; line-height: 1.7; }
.tools-connect select { max-width: 100%; padding: .75rem; color: var(--ssxz-text); background: var(--ssxz-surface-raised); border: 1px solid var(--ssxz-border); border-radius: 7px; }
.tools-error a, .tools-footer a { text-decoration: underline; text-underline-offset: 3px; }
.tools-error { display: grid; gap: .6rem; }
.tools-panel { position: relative; flex: 1; min-height: 38rem; }
.tools-frame { display: block; width: 100%; height: 75dvh; min-height: 38rem; border: 0; background: var(--ssxz-bg); }
.is-expanded .tools-panel { min-height: 0; }
.is-expanded .tools-frame { position: absolute; inset: 0; height: 100%; min-height: 0; }
.tools-wait { padding: .5rem 1rem; }
.tools-connection { font-size: .875rem; }
.tools-footer { border-top: 1px solid var(--ssxz-border); }
@media (max-width: 640px) { .tools-header { padding: .5rem; } .tools-tabs button { padding: .55rem .7rem; } .tools-frame { height: 75dvh; min-height: 28rem; } .tools-panel { min-height: 28rem; } .tools-footer { font-size: .75rem; } }
</style>
