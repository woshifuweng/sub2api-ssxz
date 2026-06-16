<template>
  <form class="composer-card" @submit.prevent="handleSubmit">
    <div v-if="assetPreviews.length" class="attachment-preview-list" data-testid="workspace-asset-previews">
      <article v-for="asset in assetPreviews" :key="asset.id" class="attachment-preview-card">
        <img :src="asset.url" :alt="asset.name" />
        <span>{{ asset.name }}</span>
        <small>{{ asset.sizeLabel }}</small>
        <button
          type="button"
          class="remove-preview"
          :aria-label="`移除 ${asset.name}`"
          @click="emit('remove-asset', asset.id)"
        >
          <Icon name="x" size="xs" />
        </button>
      </article>
    </div>

    <textarea
      :value="modelValue"
      rows="2"
      :placeholder="placeholder"
      @input="handleInput"
      @keydown="handleKeydown"
    />

    <WorkspaceAssetPanel
      v-if="assetPanelOpen && imageCapabilityAvailable"
      :rejected-files="rejectedFiles"
      :image-upload-available="imageCapabilityAvailable"
      @files="emit('files', $event)"
    />

    <div class="composer-tool-row">
      <WorkspaceModelPicker
        :models="models"
        :selected-model="selectedModel"
        @update:selected-model="emit('update:selectedModel', $event)"
      />

      <button
        v-if="imageCapabilityAvailable"
        type="button"
        class="toolbar-tool"
        data-testid="workspace-add-content"
        aria-controls="workspace-asset-panel"
        :aria-expanded="assetPanelOpen"
        title="添加内容"
        @click="assetPanelOpen = !assetPanelOpen"
      >
        <Icon name="plus" size="sm" />
        <span>添加</span>
      </button>

      <button
        v-for="tool in capabilityTools"
        :key="tool.key"
        type="button"
        class="toolbar-tool"
        :class="{
          'is-unavailable': !tool.available,
          'is-active': tool.key === 'web-search' && tool.available && webSearchEnabled
        }"
        :disabled="!tool.available"
        :title="tool.description"
        :aria-label="`${tool.label}：${tool.description}`"
        :aria-disabled="!tool.available"
        :aria-pressed="tool.key === 'web-search' ? webSearchEnabled : undefined"
        :data-testid="`workspace-capability-${tool.key}`"
        @click="handleCapabilityToolClick(tool.key, tool.available)"
      >
        <Icon :name="tool.icon" size="sm" />
        <span>{{ tool.label }}</span>
        <small v-if="tool.key === 'web-search' && tool.available && webSearchEnabled">已启用</small>
        <small v-else-if="!tool.available">暂未接入</small>
      </button>

      <button
        class="send-button"
        type="submit"
        data-testid="workspace-send"
        :disabled="!canSubmit"
        :title="backendEnabled ? '发送' : '统一工作台后端正在接入，暂不可发送'"
        :aria-label="backendEnabled ? '发送' : '统一工作台后端正在接入，暂不可发送'"
      >
        <Icon v-if="sending" name="sync" size="sm" />
        <Icon v-else name="arrowUp" size="sm" />
      </button>
    </div>
  </form>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ChatModelOption } from '@/composables/useUserCapabilities'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import WorkspaceAssetPanel from './WorkspaceAssetPanel.vue'
import WorkspaceModelPicker from './WorkspaceModelPicker.vue'
import type { RejectedWorkspaceFile, WorkspaceAssetPreview } from './useWorkspaceAssets'
import type { WorkspaceIntent } from './useWorkspaceConversation'

type IconName = InstanceType<typeof Icon>['$props']['name']
type CapabilityToolKey = 'web-search' | 'memory' | 'toolbox'

const props = defineProps<{
  modelValue: string
  selectedModel: string
  models: ChatModelOption[]
  intent: WorkspaceIntent
  imageCapabilityAvailable?: boolean
  backendEnabled?: boolean
  sending: boolean
  assetPreviews: WorkspaceAssetPreview[]
  rejectedFiles: RejectedWorkspaceFile[]
  webSearchEnabled?: boolean
}>()

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
  (event: 'update:selectedModel', value: string): void
  (event: 'files', files: File[]): void
  (event: 'remove-asset', id: string): void
  (event: 'submit'): void
  (event: 'toggle-web-search'): void
}>()

const appStore = useAppStore()
const assetPanelOpen = ref(false)

const placeholders: Record<WorkspaceIntent, string> = {
  home: '直接输入问题，开始对话。',
  chat: '直接输入问题，开始对话。',
  image: '描述你想生成或修改的图片，也可以先上传参考图'
}

const capabilityTools = computed<Array<{
  key: CapabilityToolKey
  label: string
  description: string
  icon: IconName
  available: boolean
}>>(() => {
  const webSearchAvailable = appStore.cachedPublicSettings?.web_search?.available === true
  const tools: Array<{
    key: CapabilityToolKey
    label: string
    description: string
    icon: IconName
    available: boolean
  }> = [
    {
      key: 'memory',
      label: '记忆',
      description: '长期记忆暂未接入。',
      icon: 'brain',
      available: false
    },
    {
      key: 'toolbox',
      label: '工具箱',
      description: '文档、表格、代码工具暂未接入。',
      icon: 'grid',
      available: false
    }
  ]

  if (webSearchAvailable) {
    tools.push({
      key: 'web-search',
      label: '联网',
      description: '联网检索已接入，可为本次回答补充实时来源。',
      icon: 'globe',
      available: true
    })
  }

  return tools
})

const placeholder = computed(() =>
  props.imageCapabilityAvailable ? placeholders[props.intent] : placeholders.chat
)
const canSubmit = computed(() =>
  props.backendEnabled === true &&
  !props.sending &&
  (props.intent !== 'image' || props.imageCapabilityAvailable === true) &&
  Boolean(props.selectedModel) &&
  props.assetPreviews.length === 0 &&
  props.modelValue.trim().length > 0
)

function handleInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key !== 'Enter' || event.shiftKey || event.isComposing) return
  event.preventDefault()
  handleSubmit()
}

function handleSubmit() {
  if (canSubmit.value) emit('submit')
}

function handleCapabilityToolClick(key: CapabilityToolKey, available: boolean) {
  if (!available) return
  if (key === 'web-search') emit('toggle-web-search')
}
</script>
