export function getSafeLocalStorageItem(key: string): string | null {
  try {
    return window.localStorage.getItem(key)
  } catch {
    return null
  }
}

export function setSafeLocalStorageItem(key: string, value: string): void {
  try {
    window.localStorage.setItem(key, value)
  } catch {
    // Storage can be unavailable in restricted browsers or embedded QA contexts.
  }
}

export function removeSafeLocalStorageItem(key: string): void {
  try {
    window.localStorage.removeItem(key)
  } catch {
    // Ignore storage failures in restricted browsers.
  }
}

export function getSafeSessionStorageItem(key: string): string | null {
  try {
    return window.sessionStorage.getItem(key)
  } catch {
    return null
  }
}

export function setSafeSessionStorageItem(key: string, value: string): void {
  try {
    window.sessionStorage.setItem(key, value)
  } catch {
    // Storage failures should not blank the app or block route recovery.
  }
}

export function removeSafeSessionStorageItem(key: string): void {
  try {
    window.sessionStorage.removeItem(key)
  } catch {
    // Ignore storage failures in restricted browsers.
  }
}
