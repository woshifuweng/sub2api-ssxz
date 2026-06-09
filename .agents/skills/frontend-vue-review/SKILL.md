---
name: frontend-vue-review
description: Review Vue frontend changes, route guards, composables, object URL handling, disabled UI behavior, tests, and typecheck.
---

# Frontend Vue Review

Use this skill for Vue, TypeScript, router, composable, and frontend test changes.

## Checks

1. Run or verify:
   - frontend typecheck
   - relevant component/composable tests
   - git diff --check

2. Review:
   - route guards
   - auth redirects
   - component lifecycle cleanup
   - object URL revokeObjectURL
   - no long-lived base64 payload
   - no token/secret logging
   - no fake success state
   - disabled actions cannot call API
   - API wrapper errors are handled safely

3. Tests must not be weakened:
   - no .skip
   - no .only
   - no todo replacing required assertions
   - no deleted coverage just to pass CI

## Output

Return:
- typecheck status
- tests run
- changed components
- lifecycle risks
- API call risks
- disabled UI risks
- test integrity status
