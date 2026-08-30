# Staging isolation implementation plan

1. Capture and checksum a staging-only recovery bundle, then stop staging.
2. Install `ops/staging/redis-staging.conf` and
   `ops/staging/redis-staging.service`; verify production Redis remains on 6379.
3. Install `ops/staging/sub2api-staging-isolation.conf` to select Redis 6380 and
   deny non-loopback network access.
4. Run `prepare-staging-env.sh` to rotate staging signing secrets and enforce
   passive provider switches.
5. Dry-run `sanitize-staging.sql` in a rollback transaction, then apply it once
   all schema and constraint checks pass.
6. Run `verify-staging-isolation.sh`; do not start staging unless every overlap
   and credential counter is zero.
7. Add the application-level background-job gate and passive pricing loader.
8. Run focused service tests and server compilation with Go parallelism limited
   to two workers.
9. Build and install the current source as the staging binary, start staging,
   and verify login, health, sockets, logs, Redis separation, and production
   process continuity.
10. Commit only source, runbook, and isolation assets; preserve unrelated local
    binaries and worktree changes.

Rollback: stop staging, restore files and database from the checksummed bundle,
flush only Redis 6380, and start staging. Do not restart production.
