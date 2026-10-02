# SSXZ: independent upgrades and two in-site image tools

## Approved result

Update Sub2API and the existing image workbench from their official upstreams; add the native infinite canvas as a second independent tool. Keep both complete experiences instead of merging their editors. Reuse station authentication and paid gateway. Do not expose upstream credentials or require customers to configure them. Preserve historical works and existing billing/failover/UI fixes. Do not modify the concurrent documentation task.

## Verified baseline (2026-10-02)

- Production backend: `0.2.8-ssxz.20261002.image-failover2`, SHA256 `f3b6929d4a2352de992a264463540e5d6b00ef87ed0fd95c72508aaa3fa5070e`, health OK.
- Backend source candidate: `2fd7744259d79090f1a872fbef0f7cecd6d52e56`; includes affordable-cost balance preflight and image-provider failover fixes.
- Production frontend is independent of the binary: nginx uses `/var/www/ssxz-ui-20260927-brandfix`. A concurrent docs-only update must survive this release.
- `/app/image` currently redirects to `https://image.ssxzapi.com/`. Existing browser works are visible there; preserve its origin and IndexedDB schemas.
- Official targets pinned: Sub2API v0.2.12 (`5106065716e494204fc0e8db16f68f6e9d576be0`); CookSleep/gpt_image_playground v0.7.14 (`6decf3c1ee06d2149fa8a67724aeba8a5de37e32`); basketikun/infinite-canvas v0.19.0 (`e856c878e0a34651bb828e28f0af20d71016a7d4`).
- Live workbench Settings says v0.7.12; Cloudflare active version `15d38520-a08b-4dcd-be99-a1790b184106`, assets `index-BAfySrA0.js` / `index-D7JT8FM1.css`. Preserve this rollback target, not obsolete v0.7.8 deployment notes.
- Live frontend provenance includes dirty source in `.codex-work/usage-export-account-billing-fix-v028`, not only backend base commit. Frontend owner is reconciling supplier overflow, scrolling selects, profile, billing banner, auth, and current concurrent docs patches.

## Delivery slices / gates

1. **Provenance and backups**: inventory exact stable tags, deployed customization commits, live frontend entry/assets, database migration delta and independent rollback targets. No blind replacement with an upstream binary.
2. **Sub2 upgrade**: merge the official stable release into the deployed custom baseline; preserve newer live frontend fixes/docs. Run backend targeted billing/image tests, frontend tests/typecheck/build. If already latest stable, explicitly record that rather than manufacture an update.
3. **Workbench update**: use deployed fork lineage, merge official stable release, preserve station adaptations, history and settings. Test existing-task display/export and native generation path.
4. **Canvas standalone**: pin upstream commit, keep native editor/storage. Implement only minimal station-specific connection and safe configuration. Disable arbitrary executable provider scripts in the station-connected build if upstream permits them.
5. **In-site entry**: two clear tool choices in station image area; preserve workbench origin. Embedded mode only when upload, download, editing, storage and keyboard interaction work. Full-page native fallback must remain reachable. No credential in URLs, postMessage wildcards, arbitrary proxy target or admin credential in child app.
6. **Acceptance**: real browser click-through for logged-out/in, both tools, upload/reference image, text generation, edit, download, reload history, responsive layout and account isolation; verify actual gateway accounting for paid test. Record pass/fail with evidence; do not equate build success with online acceptance.
7. **Independent release**: back up exact current versions, release one app at a time, verify online after each. Roll back only the changed artifact if regression occurs. Never overwrite another task's live docs release.

## Completion ledger

- [x] Isolated Sub2 worktree created; production backend/version/health inspected.
- [x] Existing workbench origin and visible historical work checked in browser.
- [x] Upstream inventories and safe baselines recorded.
- [x] Sub2 candidate merged/tested (`7e45bd449`; final frontend 2881/2881 passing).
- [x] Workbench candidate merged/tested (v0.7.14; 615/615 passing; production dependency audit zero).
- [x] Canvas candidate integrated/tested (v0.19.0; final production build, web 13/13, SSXZ 10/10 and production audit zero).
- [x] Two-tool station entry tested with the actual Vue host and synthetic accounts (not live provider/billing acceptance).
- [ ] Independent production releases and customer-journey acceptance passed.

No production modification has been made by this task at plan creation.

## Verified progress (not production acceptance)

- Backend targeted billing/payment/redeem/image tests and compile passed. Isolated PostgreSQL tests passed, preserving all 331 old SQL migration files; two official 241 migrations added. Non-embedded Linux candidate SHA256 `2fde0c2c6770944af760b8cef612482a09d27f2869ffc6f88b6d0c8d797328cc`. Do not release this alone without handling nginx asset fallback.
- Main station entry: 29 tests across imageTools, ImageToolsWorkspace, AppSidebar passed; frontend typecheck passed. Native masked-key reveal handled; same-user token/balance updates do not interrupt creation. No arbitrary key rotation or bypass of disabled/expired/quota-limited keys.
- Canvas typecheck/build passed. Original plus station tests passed (later retry and real multipart transport tests added in `0aed5f7`). UI: native create/text edit/reload persist; original ZIP export downloaded. Embedded A→B→A showed A-only canvas absent for B and restored for A.
- Workbench UI at 0.7.14: imported actual history-only v0.7.12 backup (1 task and image), displayed thumbnail, added output as reference, submitted edit to a LOCAL SYNTHETIC 422 fixture; correct failure UI, no real image-generation or billing claim. Header navigation escaping embedded mode discovered during browser testing and assigned for correction.
- Browser connection temporarily timed out, later recovered. Local fixture `http://127.0.0.1:52388/` connects child canvas52389/workbench52390 with synthetic identities and no production credentials; exclude fixtures from publication.
- Still required: complete reconciled frontend tests/build, review and commit merge; candidate Linux run and fresh rollback backups; production child builds (no localhost config); complete native generation/edit/download/charges via scoped test key; separate production release/acceptance without overwriting concurrent docs.

## Final review corrections and evidence (2026-10-02 evening)

- Merge committed as `7e45bd449`, retaining the final parallel docs task (`ec135d8c6`) and the previously published SSXZ billing/UI fixes. Guide text/tests match that task; the only guide component difference is the copied asset import path.
- Real Vue reactive profiles are copied into plain scalars before `postMessage`; `structuredClone` regression uses a Vue Proxy. A plain-JS QA harness alone did not prove the actual host was correct.
- Tool availability no longer depends on the optional public channel showcase: native active image-enabled OpenAI groups and `/v1/models` remain authoritative.
- Key-creation idempotency is per creation operation, retained for uncertain retries, cleared after success/logout. Deliberate deletion followed by reopening gets a new operation, rather than replaying a deleted key.
- Independent review found these three corrections covered, with no new definite P1/P2. Main frontend final suite: **2881 passed, zero failed/skipped**; typecheck passed.
- Workbench embedded header no longer navigates outside its account-isolated embedded mode. The actual old v0.7.12 image was downloaded from v0.7.14 and SHA256 matches the original history ZIP (`39b0b0029ebd442ce00e64f9da4e445888c0024ab3633daac889de3d2700588c`).
- Synthetic parent logout removed both child frames. Synthetic tests intentionally do not validate production login, provider success or billing. No successful paid generation is claimed.
- Workbench dependency patch `fb24c0f`: DOMPurify 3.4.16, production audit zero; nine dev/build advisories remain for separate risk review, not silently marked clean.
- No production switch has been made. Formal publication choice was requested while completing the isolated verification; do not infer an answer from a default UI selection.
- Public read-only preflight found an explicit release prerequisite: `Origin: https://canvas.ssxzapi.com` currently gets 403 for `/v1/images/generations`, while the existing image workbench origin gets 204. The new canvas origin must be added to the existing exact CORS allowlist at authorized release; never replace it with `*`. This is not an existing-customer outage or a completed fix.
- Real Vue-host Chrome acceptance now passed: both children connected and host wait banners disappeared; `DataCloneError=0`; same-account refresh/tool switching reused the connection; A→B→A kept each account's works separate; logout removed frames. Real BaseDialog confirmed group change and remained visible over fullscreen. 390×844 entrance/fullscreen canvas passed the inspected layout, then viewport reset.
- Logout controls are disabled (`84b2f8106`), with a regression assertion and browser recheck. Two image-tool suites pass 18/18.
- Main production dependency audit is now zero after exact colord/query-string/decode-uri-component fixes (`2ed44a513`). Updated full frontend suite 2881/2881 and final `pnpm run build` passed, including i18n, types and payment-entry isolation.
- Canvas web production dependency audit originally found 24 advisories. Minimal reviewed fixes now pass frozen install, production audit zero, typecheck/build, web 13/13 and SSXZ 10/10. One documented scoped Prism override has actual seven-language React rendering/escaping checks; no top-level major upgrade. Main restarted the guarded child and rechecked real Vue connection, old project navigation and retained image/text. This does not replace live generation/billing acceptance.
- Linux isolated rehearsal PASS (`F:/CodexHome/artifacts/sub2-v0212-7e45bd449-20261002/rehearsal-result.json`): protected server-only snapshot restored, original 333 migration/hash pairs retained, two migrations added, restart idempotent, old binary healthy against additive schema with no bonus/TypeSafe use. Sampled full account-balance and order financial fingerprints unchanged; no real provider/payment calls. Exact canvas/image CORS preflights 204 and stranger origins 403. Temporary units/processes stopped and cloned sensitive data sealed root-only. Production binary/PID/schema/watchdog witnesses unchanged.
- Final traced Linux non-embedded candidate SHA256 `cd5728b4fabb7c3d1043112e802d0302a0fd7b412117a5ba8e42dc070409fdb5` (backend commit `7e45bd449`). This supersedes the earlier pre-commit build fingerprint.
- Workbench minimal dependency follow-up `b388a408`: frozen install/build, 615/615 and complete audit zero; final browser reconnect and historical-image detail rechecked. No upstream major-version upgrade or audit-gate bypass.
