---
name: pr-ci-gate
description: Review GitHub PR readiness, CI status, mergeability, diff scope, and forbidden changes before merge.
---

# PR CI Gate

Use this skill when the task mentions PR review, GitHub checks, merge readiness, CI status, or whether a PR can be merged.

## Required reads

Before reviewing:
1. Read AGENTS.md.
2. Read docs/security/launch-checklist.md if present.
3. Read docs/security/unified-chat-workspace-v1-security-strategy.md if present.

## Workflow

1. Confirm current branch.
2. Confirm target PR and base branch.
3. Run:
   - git status
   - git branch --show-current
   - git log -1 --oneline
   - git diff --check
4. Query GitHub PR checks if available.
5. Compare diff against origin/main.
6. Classify changed files by module.
7. Check forbidden scope.

## Forbidden scope unless explicitly requested

- payment
- database migrations
- provider production logic
- admin permissions
- production deployment config
- lockfile
- go.mod / go.sum
- AGENTS.md / docs/security unless task is documentation-only
- CI security baseline
- billing / ledger
- real image task backend logic

## Output

Return:
- current branch
- HEAD commit
- whether based on latest main
- git status
- git diff --check
- GitHub checks
- changed file list
- forbidden changes found
- blocker/high/medium/low findings
- merge recommendation
