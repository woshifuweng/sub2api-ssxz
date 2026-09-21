import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('Composite channel platform options', () => {
  it('includes the CN concrete providers for pricing and model mapping', () => {
    const source = readFileSync(resolve('src/views/admin/ChannelsView.vue'), 'utf8')
    const declaration = source.match(/const compositePlatforms:[^=]+=[^\n]+/)?.[0]

    expect(declaration).toContain("'kimi'")
    expect(declaration).toContain("'zhipu'")
    expect(declaration).toContain("'deepseek'")
    expect(declaration).toContain("'minimax'")
    expect(declaration).toContain("'opencode_go'")
  })

  it('keeps the model and billing health check visible on the channel page', () => {
    const source = readFileSync(resolve('src/views/admin/ChannelsView.vue'), 'utf8')

    expect(source).toContain('模型与计费体检')
    expect(source).toContain('const modelHealth = computed')
    expect(source).toContain('entry.billing_mode === "image"')
    expect(source).toContain('图片模型，但没有图片输出价或按次价格')
  })
})
