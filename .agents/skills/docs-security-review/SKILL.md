---
name: docs-security-review
description: Review AGENTS.md and docs/security changes for project safety rules, launch checklist coverage, and documentation-only PR scope.
---

# Docs Security Review

Use this skill for documentation-only PRs involving AGENTS.md, docs/security, launch checklists, or development rules.

## Checks

1. Diff must only include documentation unless task says otherwise.
2. No business logic changes.
3. No lockfile changes.
4. No production config changes.
5. Security rules must cover:
   - provider keys
   - frontend not security boundary
   - user ownership
   - provider URL allowlist / SSRF
   - billing/ledger
   - assets
   - logs/privacy
   - prompt/tool safety
   - no demo bypass
   - no test cheating
6. Launch checklist must cover:
   - auth
   - API relay/provider
   - Unified Chat Workspace
   - image upload/generation
   - billing/ledger
   - rate limits/abuse
   - admin/audit
   - logs/privacy
   - CI/tests

## Output

Return:
- docs changed
- missing security topics
- non-doc changes found
- git diff --check
- PR recommendation
