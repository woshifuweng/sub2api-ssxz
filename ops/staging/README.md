# Isolated staging runbook

These files create a passive staging environment on a host that also runs
production.

## Required order

1. Stop staging and create a verified backup.
2. Install and start `redis-staging.service` on loopback port 6380.
3. Install `sub2api-staging-isolation.conf` as a systemd drop-in.
4. Copy this directory to `/opt/sub2api/ops/staging` with root ownership. Give
   the directory mode `0750`, group `postgres`, and give
   `sanitize-staging.sql` mode `0640`, group `postgres`.
5. Run `prepare-staging-env.sh`.
6. Run `apply-staging-sanitization.sh`.
7. Run `verify-staging-isolation.sh`; a non-zero exit blocks startup.
8. Start staging and inspect health, sockets, and logs.

## Non-negotiable checks

- Never point `DATABASE_DBNAME` at the production database.
- Never use Redis 6379 for staging.
- Never remove the systemd egress deny rule to make a provider test pass.
- Never copy a production credential back into staging.
- A failed verifier is a failed deployment even if the web page opens.

## Rollback

Stop staging first. Restore only the staging database, binary, environment file,
unit, and drop-ins from the pre-change bundle. Flush only Redis 6380. Production
must not be restarted.
