<template>
  <div id="workspace-asset-panel" class="asset-panel" aria-label="上传能力面板">
    <label class="asset-option is-ready" title="上传图片">
      <Icon name="upload" size="sm" />
      <span>
        <strong>图片</strong>
        <small>上传参考图</small>
      </span>
      <input
        class="asset-file-input"
        type="file"
        accept="image/*"
        multiple
        aria-label="上传图片"
        @change="handleFileChange"
      />
    </label>

    <button
      v-for="item in futureCapabilities"
      :key="item.label"
      type="button"
      class="asset-option is-informational"
      :aria-label="`${item.label}，${item.description}`"
      @click="activeInfo = activeInfo === item.label ? '' : item.label"
    >
      <Icon :name="item.icon" size="sm" />
      <span>
        <strong>{{ item.label }}</strong>
        <small>{{ item.short }}</small>
      </span>
    </button>

    <p v-if="activeInfo" class="asset-panel-note">
      {{ activeDescription }}
    </p>
    <p v-for="file in rejectedFiles" :key="file.name" class="asset-panel-error">
      {{ file.name }}：{{ file.reason }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import type { RejectedWorkspaceFile } from './useWorkspaceAssets'

type IconName = InstanceType<typeof Icon>['$props']['name']

defineProps<{
  rejectedFiles: RejectedWorkspaceFile[]
}>()

const emit = defineEmits<{
  (event: 'files', files: File[]): void
}>()

const activeInfo = ref('')

const futureCapabilities: Array<{ label: string; short: string; description: string; icon: IconName }> = [
  {
    label: '文档',
    short: '即将接入',
    description: '即将接入文档分析，当前请先上传图片或输入文字。',
    icon: 'document'
  },
  {
    label: '表格',
    short: '即将接入',
    description: '即将接入表格分析，当前不会读取 xlsx/csv 文件。',
    icon: 'chart'
  },
  {
    label: '代码',
    short: '即将接入',
    description: '即将接入代码文件分析，当前请直接粘贴关键代码。',
    icon: 'terminal'
  }
]

const activeDescription = computed(() => (
  futureCapabilities.find((item) => item.label === activeInfo.value)?.description || ''
))

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  emit('files', Array.from(input.files || []))
  input.value = ''
}
</script>
