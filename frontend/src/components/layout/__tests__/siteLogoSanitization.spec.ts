import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const sidebarSource = readFileSync(resolve(dir, '../AppSidebar.vue'), 'utf8')
const homeViewSource = readFileSync(resolve(dir, '../../../views/HomeView.vue'), 'utf8')
const keyUsageViewSource = readFileSync(resolve(dir, '../../../views/KeyUsageView.vue'), 'utf8')

describe('site_logo sanitization', () => {
  it('AppSidebar uses the static branded logo rather than a raw setting URL', () => {
    expect(sidebarSource).toContain('<BrandLogo')
    expect(sidebarSource).not.toContain(':src="appStore.siteLogo"')
  })

  it('HomeView applies sanitizeUrl to siteLogo', () => {
    expect(homeViewSource).toMatch(/sanitizeUrl\([\s\S]{0,180}cachedPublicSettings\?\.site_logo\s*\|\|\s*appStore\.siteLogo/)
  })

  it('KeyUsageView applies sanitizeUrl to siteLogo', () => {
    expect(keyUsageViewSource).toMatch(/sanitizeUrl\([\s\S]{0,220}cachedPublicSettings\?\.site_logo\s*\|\|\s*appStore\.siteLogo/)
  })

  it('dynamic logo consumers allow safe relative and data image URLs', () => {
    for (const src of [homeViewSource, keyUsageViewSource]) {
      expect(src).toContain('allowRelative: true')
      expect(src).toContain('allowDataUrl: true')
    }
  })
})
