import type { GroupPlatform } from '@/types'

/**
 * Resolve the platform used for a group's visual identity.
 *
 * Some Claude-branded groups use an OpenAI-compatible upstream for routing,
 * so the backend platform remains openai. That transport detail should not
 * make the user-facing group selector display the GPT logo.
 */
export function resolveGroupDisplayPlatform(name: string | null | undefined, platform: GroupPlatform): GroupPlatform
export function resolveGroupDisplayPlatform(
  name: string | null | undefined,
  platform: GroupPlatform | null | undefined
): GroupPlatform | undefined
export function resolveGroupDisplayPlatform(
  name: string | null | undefined,
  platform: GroupPlatform | null | undefined
): GroupPlatform | undefined {
  if (name?.trim().toLocaleLowerCase().startsWith('claude')) {
    return 'anthropic'
  }
  return platform ?? undefined
}
