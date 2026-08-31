# Staging isolation design

## Goal

Run a useful staging instance on the production host without sharing mutable
runtime state, credentials, customer identity data, or autonomous side effects
with production.

## Safety boundaries

- Production and staging use different PostgreSQL databases.
- Staging uses a dedicated Redis process on loopback port 6380; production stays
  on 6379.
- The staging systemd unit may connect only to loopback. SMTP, providers,
  webhooks, remote backups, probes, and update checks are blocked by the OS.
- `BACKGROUND_JOBS_ENABLED=false` prevents autonomous workers from starting in
  application code. The variable is production-compatible: an absent value
  preserves the current production default; an invalid explicit value fails
  closed.
- Request-serving local workers may continue to run. They cannot reach an
  external service because provider credentials are removed and OS egress is
  denied.

## Staging data policy

- Keep row shape and operational history needed for UI verification.
- Preserve only two dedicated staging login identities and re-hash their
  staging passwords.
- Disable and rotate every copied API key.
- Remove account, proxy, payment, OAuth, passkey, SMTP, monitor, plugin, image,
  and provider credentials.
- Redact customer messages, prompts, email addresses, IP addresses, media
  locations, and free-form audit content.
- Empty pending outboxes and sessions before first isolated start.

## Recovery

Before isolation changes, stop staging and save the binary, environment file,
systemd unit/drop-ins, shared config, and a PostgreSQL custom-format dump with
SHA-256 checksums. Rollback restores only staging assets; production is never
restarted as part of this procedure.

## Acceptance criteria

- Production PID and activation timestamp are unchanged; public health is 200.
- Production/staging API key and password-hash overlap are both zero.
- Staging has zero schedulable provider accounts, pending login sessions,
  passkeys, non-test identities, and enabled provider/payment/monitor
  configurations. A normal email identity created by either dedicated staging
  login is allowed.
- Staging login and local health return 200.
- Staging sockets connect only to loopback PostgreSQL and dedicated Redis.
- Staging starts without pricing, update, backup, probe, refresh, report, or
  monitor background attempts.
