# SSXZ v0.2.7 upgrade / rollback contract

Production release requires separate approval. Do not run the generic
`preflight-systemd-release.sh` for this schema-changing upgrade: it assumes a
binary-only rollback and is not a database migration orchestrator.

## Preparation

1. Pin the candidate commit, upstream `aea725f2ea644d5592d0bbb1d63b607efa7e200a`,
   source-clean build hash, and byte-verified old binary. Do not infer a version
   from its release-directory name. Preserve the existing DB pool 60/24 settings.
2. Verify a private database recovery bundle (including `schema_migrations`),
   configuration, old binary and static assets. Restore that bundle in an
   **isolated** rehearsal environment and prove it readable. Backup existence is
   not proof of recoverability. Never print credentials or customer row contents.
3. Record exact database/server identity and counts/hashes of legacy group JSON,
   quota rows, real authorization grants and financial tables. Recheck drift in
   the maintenance window. Retain reviewed evidence privately.
4. Stop all old/new writers and drain in-flight requests before actual migration:
   gateway replicas, admin mutations and background jobs. Do not use rolling
   overlap for this schema transition. The migration lock only serializes
   migrators; it does not stop old application writers.

## Candidate migration

The candidate embeds two additional **new**, immutable migrations. Original
upstream 235/236/238 SQL and checksums are unchanged.

- `234_ssxz_preserve_legacy_model_listing.sql`: only an unambiguously legacy
  schema is converted. Original listing JSON is archived and retained in the old
  column. The separate native admission policy starts disabled (as the previous
  gateway effectively behaved); existing API-key/account/group grants are
  untouched. Native installations keep their real policy. Ambiguous dual-column
  or already-migrated-but-renamed-back databases fail closed for investigation.
- `237_ssxz_archive_unlimited_quotas.sql`: complete unlimited quota rows,
  including usage/window/deleted state, are archived before native cleanup.
  Zero/positive limits are untouched. The archive has no public API and is not
  replayed into live accounting. It is evidence/recovery material, not revenue.

Start the candidate only after the isolated rehearsal succeeds. During production
acceptance keep external traffic/admin writes closed and jobs disabled. Verify
startup, static assets, dedicated login, group permission and billing fixtures;
never generate paid content as a smoke test. Compare archives with the private
pre-migration snapshot. Preserve migration records, do not forge/delete checksums.

## Rollback boundary

The preserved old column avoids the old-binary missing-column failure. This alone
does **not** make a downgrade safe after using new functionality.

- Before opening traffic: run `v027-rollback-check.sql` with `ON_ERROR_STOP=1`
  against the explicitly selected DB, verify the closed-write window and old
  artifact hash, then use the rehearsed old binary. Keep **both** columns, all
  additive schema and migration checksums. Never rename the new column back.
- The guard rejects native-only model restrictions, public-group restrictions,
  Fast/effort policies, pricing and provider data that the old binary cannot honor.
  Missing tables/columns also fail, never count as a pass.
- Unlimited quota history stays recoverable in its private archive; do not
  restore stale counters over new counters, or restore a whole old DB over new
  financial activity. A selective recovery needs its own reviewed transaction.
- Once customer/admin writes or new features are allowed: the immediate downgrade
  window is **closed**. Prefer a schema-compatible forward fix. A later downgrade
  requires a fresh compatibility and financial reconciliation review.
- Re-upgrading uses unchanged migration records and the retained native column;
  it must not overwrite deliberate native policy with old display preferences.

## Acceptance evidence required

- Real local PostgreSQL migration/rollback/refusal/replay tests (not skipped).
- Isolated Linux staging start with egress denial, separate DB/Redis, no provider
  credentials or schedulable accounts; production PID/start unchanged.
- Candidate and preserved old binary startup against that isolated migrated
  schema, then candidate again; archive and ledger invariants remain unchanged.
- Unit/contract suites, native frontend inventory, built-page/auth/browser checks.
- Independent review, source/binary hashes, remaining limitations and **explicit
  formal-publication approval**. A successful local build is not live acceptance.
