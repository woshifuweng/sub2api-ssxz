import { ref } from 'vue'
import {
  CHAT_ASSET_KIND_IMAGE,
  CHAT_ASSET_ROLE_ATTACHMENT,
  CHAT_ASSET_SOURCE_USER_UPLOAD,
  registerAsset,
  type ChatAsset
} from '@/api/chatWorkspace'
import type { WorkspaceAttachment } from './useWorkspaceConversation'

export interface WorkspaceAssetPreview {
  id: string
  file: File
  name: string
  url: string
  size: number
  sizeLabel: string
  asset?: ChatAsset
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
        url: await readAsDataUrl(file),
        size: file.size,
        sizeLabel: formatFileSize(file.size)
      })
    }
    for (const file of files.slice(slots)) {
      rejectedFiles.value.push({ name: file.name, reason: `最多添加 ${MAX_IMAGES} 张图片` })
    }
  }

  function removePreview(id: string) {
    previews.value = previews.value.filter((preview) => preview.id !== id)
  }

  function clearPreviews() {
    previews.value = []
    rejectedFiles.value = []
  }

  async function registerPendingAssets(conversationId: number): Promise<WorkspaceAttachment[]> {
    registering.value = true
    try {
      const attachments: WorkspaceAttachment[] = []
      for (const preview of previews.value) {
        const asset = preview.asset || await registerAsset({
          conversation_id: conversationId,
          asset_kind: CHAT_ASSET_KIND_IMAGE,
          source_type: CHAT_ASSET_SOURCE_USER_UPLOAD,
          asset_role: CHAT_ASSET_ROLE_ATTACHMENT,
          storage_provider: 'pending',
          url: preview.url,
          preview_url: preview.url,
          original_name: preview.name,
          content_type: preview.file.type,
          byte_size: preview.file.size
        })
        preview.asset = asset
        attachments.push({
          id: preview.id,
          name: preview.name,
          url: preview.url,
          type: 'image',
          asset
        })
      }
      return attachments
    } finally {
      registering.value = false
    }
  }

  return {
    previews,
    registering,
    rejectedFiles,
    addFiles,
    clearPreviews,
    registerPendingAssets,
    removePreview
  }
}

function readAsDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '')
    reader.onerror = () => reject(reader.error || new Error('file read failed'))
    reader.readAsDataURL(file)
  })
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
