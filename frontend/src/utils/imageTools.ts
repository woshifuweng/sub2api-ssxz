import type { ApiKey, Group } from '@/types'

export const IMAGE_TOOL_MODEL = 'gpt-image-2'
export const IMAGE_TOOL_PROTOCOL = 1

export type ImageToolProfile = {
  name: string
  baseUrl: string
  apiKey: string
  model: typeof IMAGE_TOOL_MODEL
}

export function imageToolKeyName(groupId: number): string {
  return `SSXZ 图片工具 · ${groupId}`
}

export function supportsStationImages(group: Pick<Group, 'status' | 'platform' | 'allow_image_generation'>): boolean {
  return group.status === 'active' && group.platform === 'openai' && group.allow_image_generation === true
}

type ImageKey = Pick<ApiKey, 'name' | 'user_id' | 'status' | 'group_id' | 'group_ids' | 'allowed_models' | 'expires_at' | 'quota' | 'quota_used' | 'key'>

// 只复用当前客户的图片专用 Key，绝不把通用 Key 或后台凭据交给图片应用。
export function isEligibleImageToolKey(key: ImageKey, userId: number, groupId: number, now = Date.now()): boolean {
  const groups = key.group_ids?.length ? key.group_ids : [key.group_id]
  const expiry = key.expires_at ? Date.parse(key.expires_at) : Infinity
  return key.name === imageToolKeyName(groupId)
    && key.user_id === userId
    && key.status === 'active'
    && groups.length === 1 && groups[0] === groupId
    && key.allowed_models?.length === 1 && key.allowed_models[0] === IMAGE_TOOL_MODEL
    && expiry > now
    && (key.quota === 0 || key.quota_used < key.quota)
}

export function isPlainImageKey(key?: string): boolean {
  return Boolean(key?.trim()) && !/\*|\.\.\.|\[redacted\]/i.test(key || '')
}

export function isUsableImageToolKey(key: ImageKey, userId: number, groupId: number, now = Date.now()): boolean {
  return isEligibleImageToolKey(key, userId, groupId, now) && isPlainImageKey(key.key)
}

export function isImageToolMessage(event: Pick<MessageEvent, 'source' | 'origin' | 'data'>, source: Window | null, origin: string, type: 'ready' | 'connected'): boolean {
  if (!source || event.source !== source || event.origin !== origin) return false
  const data = event.data
  return data !== null && typeof data === 'object'
    && data.type === `ssxz:image-tool:${type}` && data.version === IMAGE_TOOL_PROTOCOL
}

export function imageToolInit(accountId: number, sessionId: string, profile: ImageToolProfile) {
  return {
    type: 'ssxz:image-tool:init',
    version: IMAGE_TOOL_PROTOCOL,
    accountId: String(accountId),
    sessionId,
    // Vue ref/reactive inputs are proxies; postMessage only accepts cloneable data.
    profile: { name: profile.name, baseUrl: profile.baseUrl, apiKey: profile.apiKey, model: profile.model },
  }
}
