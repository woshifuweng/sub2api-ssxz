import { describe, expect, it } from 'vitest'
import { renderRichContent, sanitizeHtml, sanitizeSvg } from '@/utils/sanitize'

describe('content sanitization', () => {
  it('removes executable HTML while preserving safe content', () => {
    const sanitized = sanitizeHtml(
      '<p>safe</p><script>alert(1)</script><img src="x" onerror="alert(2)">'
    )

    expect(sanitized).toContain('<p>safe</p>')
    expect(sanitized).not.toContain('<script')
    expect(sanitized).not.toContain('onerror')
  })

  it('removes unsafe links from rendered Markdown', () => {
    const sanitized = renderRichContent('[unsafe](javascript:alert(1))\n\n**safe**')

    expect(sanitized).not.toContain('javascript:')
    expect(sanitized).toContain('<strong>safe</strong>')
  })

  it('removes executable SVG content', () => {
    const sanitized = sanitizeSvg(
      '<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script><circle cx="5" cy="5" r="5" onload="alert(2)" /></svg>'
    )

    expect(sanitized).toContain('<circle')
    expect(sanitized).not.toContain('<script')
    expect(sanitized).not.toContain('onload')
  })
})
