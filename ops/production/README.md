# Guarded systemd release

`preflight-systemd-release.sh` is for a systemd unit whose `WorkingDirectory`
and `ExecStart` resolve through `/opt/sub2api/current`. It does not retrofit the
legacy direct `/opt/sub2api/sub2api` unit automatically.

## Required layout

```text
/opt/sub2api/
  current -> /opt/sub2api/releases/<active-version>
  releases/
    <active-version>/sub2api
    <candidate-version>/sub2api
```

The production unit should use:

```ini
WorkingDirectory=/opt/sub2api/current
ExecStart=/opt/sub2api/current/sub2api
EnvironmentFile=/etc/sub2api/sub2api.env
```

Create a separate `/etc/sub2api/preflight.env` backed by an isolated database
and Redis. It must not contain production provider, payment, SMTP, storage,
OAuth, monitoring, or customer credentials. The script forces loopback, a
temporary port, and `BACKGROUND_JOBS_ENABLED=false`, but isolation of the
preflight database/Redis is still mandatory.

Every release also requires a recent PostgreSQL custom-format backup that has
been restored successfully into a uniquely named disposable database. The
restore verifier never reuses a database, refuses production/staging names,
checks critical tables and migrations, prints sanitized row counts, and drops
only the disposable database it created. If the database user cannot read the
root-owned archive, it stages a temporary read-only copy and removes that copy
on exit without changing the original backup.

Create the backup without changing the source database:

```bash
sudo install -d -m 0700 /opt/sub2api/backups/<release-backup>
cd /tmp
sudo -u postgres pg_dump --format=custom --no-owner sub2api \
  | sudo tee /opt/sub2api/backups/<release-backup>/sub2api.dump >/dev/null
sudo chmod 0600 /opt/sub2api/backups/<release-backup>/sub2api.dump
```

Verify it independently before release:

```bash
sudo RESTORE_OS_USER=postgres \
  ops/production/verify-backup-restore.sh \
  /opt/sub2api/backups/<release-backup>/sub2api.dump
```

## Run

```bash
sudo RELEASE_ROOT=/opt/sub2api/releases \
  CURRENT_LINK=/opt/sub2api/current \
  PRODUCTION_ENV_FILE=/etc/sub2api/sub2api.env \
  PREFLIGHT_ENV_FILE=/etc/sub2api/preflight.env \
  BACKUP_FILE=/opt/sub2api/backups/<release-backup>/sub2api.dump \
  PRODUCTION_HEALTH_URL=http://127.0.0.1:8080/health \
  ops/production/preflight-systemd-release.sh \
  /opt/sub2api/releases/<candidate-version>
```

The candidate binary and both environment files are checked as the service
user. The candidate first starts on a temporary loopback port. Only after that
health check passes is the `current` symlink replaced atomically and systemd
restarted. A failed restart or production health check restores the prior link,
restarts it, and verifies rollback health.

Run the self-contained tests before using the script:

```bash
bash -n ops/production/preflight-systemd-release.sh
bash -n ops/production/test-preflight-systemd-release.sh
bash -n ops/production/verify-backup-restore.sh
bash -n ops/production/test-verify-backup-restore.sh
bash ops/production/test-preflight-systemd-release.sh
bash ops/production/test-verify-backup-restore.sh
```

The test suite uses temporary directories and fake `runuser`, `curl`, and
`systemctl`; it never touches a real service.

## Commercial regression gate

`run-commercial-regression-gate.sh` is the mandatory quality gate before a
candidate can enter the guarded systemd release flow. It runs each stage
sequentially while bounding the internal Go and Vitest worker counts. A failure
stops the gate immediately.

Source mode verifies shell release tooling, all normal/unit/integration Go
tests, Go vet, frontend lint/typecheck/full Vitest, the production frontend
build, and a Go production build with the frontend embedded. The explicit
embedded build prevents a healthy API binary from shipping with every browser
route returning 404:

```bash
GATE_MODE=source GATE_PARALLELISM=4 \
  bash ops/production/run-commercial-regression-gate.sh
```

Release mode adds the live read-only staging isolation check and a real restore
of the selected backup into a disposable database. Both files are mandatory;
omitting either one is a hard failure:

```bash
sudo env \
  GATE_MODE=release \
  GATE_PARALLELISM=4 \
  STAGING_ENV_FILE=/etc/sub2api/staging.env \
  PRODUCTION_DATABASE_NAME=sub2api \
  BACKUP_FILE=/opt/sub2api/backups/<release-backup>/sub2api.dump \
  bash ops/production/run-commercial-regression-gate.sh
```

Each run creates a private timestamp-and-process-specific directory under
`.codex-evidence/` by default. `metadata.txt` records only the mode, commit,
time, and worker bounds; `summary.tsv` points to one mode-`0600` log per check.
Use `GATE_EVIDENCE_ROOT` to archive evidence outside the checkout. The gate does
not activate a release or restart a service.

Test the gate's own fail-closed orchestration with:

```bash
bash -n ops/production/run-commercial-regression-gate.sh
bash -n ops/production/test-commercial-regression-gate.sh
bash ops/production/test-commercial-regression-gate.sh
```
