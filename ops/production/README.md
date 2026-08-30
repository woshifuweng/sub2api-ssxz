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

The candidate binary and both environment files are checked as the service
user. The candidate first starts on a temporary loopback port. Only after that
health check passes is the `current` symlink replaced atomically and systemd
restarted. A failed restart or production health check restores the prior link,
restarts it, and verifies rollback health.

Run the self-contained tests before using the script:

```bash
bash -n ops/production/preflight-systemd-release.sh
bash -n ops/production/test-preflight-systemd-release.sh
bash ops/production/test-preflight-systemd-release.sh
```

The test suite uses temporary directories and fake `runuser`, `curl`, and
`systemctl`; it never touches a real service.
