import { reactive } from 'vue'

export interface AuthPortalDraft {
  email: string
  password: string
  affiliate_code: string
}

const authPortalDraft = reactive<AuthPortalDraft>({
  email: '',
  password: '',
  affiliate_code: ''
})

export function useAuthPortalDraft(): AuthPortalDraft {
  return authPortalDraft
}

export function clearAuthPortalDraft(): void {
  authPortalDraft.email = ''
  authPortalDraft.password = ''
  authPortalDraft.affiliate_code = ''
}
