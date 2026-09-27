# Guarded systemd release

`preflight-systemd-release.sh` is for a systemd unit whose `WorkingDirectory`
and `ExecStart` resolve through `/opt/sub2api/current`. It does not retrofit the
legacy direct `/opt/sub2api/sub2api` unit automatically.

Release binaries must be built with the `embed` build tag so the frontend is
included in the executable. The release gate checks the health endpoint and
the main SPA routes before switching the production symlink; a healthy API
with a missing frontend is not a valid release.

## Required layout

```text
/opt/sub2api/
  current -> /opt/sub2api/releases/<active-version>
  releases/
    <active-version>/sub2api
    <candidate-version>/sub2api
    <candidate-version>/release-provenance.json
    <candidate-version>/config.yaml -> /opt/sub2api/config.yaml
    <candidate-version>/data -> <isolated-or-production-data>
```

Every candidate must carry `release-provenance.json` with the official
upstream repository/tag, upstream commit, merge commit, and SSXZ custom parent
commit. This manifest records the artifact; it does not replace the read-only
`audit-upstream-release.py` check against GitHub.

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
preflight database/Redis is still mandatory. If `DATABASE_DBNAME` or `DATA_DIR`
is declared in both environment files, the release gate rejects a matching
value. Missing declarations are warned about rather than guessed; verify the
actual `DATABASE_URL`, Redis database/prefix, and file-backed state before
releasing.

## Run

```bash
sudo RELEASE_ROOT=/opt/sub2api/releases \
  CURRENT_LINK=/opt/sub2api/current \
  PRODUCTION_ENV_FILE=/etc/sub2api/sub2api.env \
  PREFLIGHT_ENV_FILE=/etc/sub2api/preflight.env \
  PRODUCTION_HEALTH_URL=http://127.0.0.1:8080/health \
  ops/production/preflight-systemd-release.sh \
  /opt/sub2api/releases/<candidate-version>
```

The candidate binary, provenance manifest, config/data links, and both
environment files are checked as the service user. The candidate first starts
on a temporary loopback port. Only after the
health and page checks pass is the `current` symlink replaced atomically and
systemd restarted. A failed restart or production health check restores the prior link,
restarts it, and verifies rollback health.

Run the self-contained tests before using the script:

```bash
bash -n ops/production/preflight-systemd-release.sh
bash -n ops/production/test-preflight-systemd-release.sh
bash ops/production/test-preflight-systemd-release.sh
```

The test suite uses temporary directories and fake `runuser`, `curl`, and
`systemctl`; it never touches a real service.
## Target model probe

`probe-target-models.sh` performs one minimal `/v1/chat/completions` request for each target model. It is intentionally opt-in because it can consume real quota. Use only an isolated QA API key, never a customer key:

```bash
MODEL_PROBE_CONFIRM=YES \
MODEL_PROBE_BASE_URL=https://api.example.com \
MODEL_PROBE_API_KEY='[redacted]' \
./ops/production/probe-target-models.sh
```

The script prints only model, HTTP status, latency and PASS/FAIL; it does not print the key or response body. `MODEL_PROBE_DRY_RUN=1` verifies the selected model list without sending requests. A PASS proves the endpoint returned HTTP 2xx for that key; it does not by itself replace the customer-facing PC/mobile acceptance test.
