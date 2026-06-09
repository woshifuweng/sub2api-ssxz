import { getCurrentInstance, onBeforeUnmount, ref } from 'vue'
import type { WorkspaceAttachment } from './useWorkspaceConversation'

export interface WorkspaceAssetPreview {
  id: string
  file: File
  name: string
  url: string
  size: number
  sizeLabel: string
}

export interface RejectedWorkspaceFile {
  name: string
  reason: string
}

const MAX_IMAGES = 4

export function useWorkspaceAssets() {
  const previews = ref<WorkspaceAssetPreview[]>([])
  const rejectedFiles = ref<RejectedWorkspaceFile[]>([])
  const registering = ref(false)

  async function addFiles(files: File[]) {
    rejectedFiles.value = []
    const slots = Math.max(0, MAX_IMAGES - previews.value.length)
    for (const file of files.slice(0, slots)) {
      if (!file.type.startsWith('image/')) {
        rejectedFiles.value.push({ name: file.name, reason: '当前仅支持图片上传' })
        continue
      }
      previews.value.push({
        id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}-${file.name}`,
        file,
        name: file.name,
        url: URL.createObjectURL(file),
        size: file.size,
        sizeLabel: formatFileSize(file.size)
      })
    }
    for (const file of files.slice(slots)) {
      rejectedFiles.value.push({ name: file.name, reason: `最多添加 ${MAX_IMAGES} 张图片` })
    }
  }

  function removePreview(id: string) {
    const next = previews.value.filter((preview) => {
      if (preview.id !== id) return true
      URL.revokeObjectURL(preview.url)
      return false
    })
    previews.value = next
  }

  function clearPreviews() {
    for (const preview of previews.value) {
      URL.revokeObjectURL(preview.url)
    }
    previews.value = []
    rejectedFiles.value = []
  }

  function getLocalAttachments(): WorkspaceAttachment[] {
    return previews.value.map((preview) => ({
      id: preview.id,
      name: preview.name,
      url: preview.url,
      type: 'image'
    }))
  }

  if (getCurrentInstance()) {
    onBeforeUnmount(clearPreviews)
  }

  return {
    previews,
    registering,
    rejectedFiles,
    addFiles,
    clearPreviews,
    getLocalAttachments,
    removePreview
  }
}

function formatFileSize(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 KB'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  const precision = value >= 10 || unitIndex === 0 ? 0 : 1
  return `${value.toFixed(precision)} ${units[unitIndex]}`
}
