#!/usr/bin/env bash
# Deliberately single-release scoped. Requires independently recorded server QA.
set -euo pipefail
cd /
candidate=/opt/sub2api/releases/0.2.7-ssxz.20260921.ac2923c5e
previous=/opt/sub2api/releases/0.1.183-ssxz.20260904.1
backup=/opt/sub2api/backups/v027-release-20260921
expected=29df07a4cd03482a3058ebd485211e6f6ba9945f9f1b0bda5b3185181d689794
flag=/var/lib/ssxz-v027-maintenance
runtime=/run/systemd/system/sub2api.service.d/v027-acceptance.conf
switched=0
stopped=0
success=0
rollback_allowed=1

die() { echo "RELEASE BLOCKED: $*" >&2; exit 1; }
health() {
  for _ in {1..45}; do
    if curl --fail --silent --max-time 2 http://127.0.0.1:8080/health >/dev/null; then return 0; fi
    sleep 1
  done
  return 1
}
restore_nginx() {
  install -m 644 "$backup/nginx-sub2api.conf" /etc/nginx/sites-available/sub2api || return 1
  install -m 644 "$backup/nginx-default.conf" /etc/nginx/sites-available/default || return 1
  nginx -t && systemctl reload nginx
}
verify_preserved_quotas() {
  sudo -u postgres psql -X -d sub2api -At -v ON_ERROR_STOP=1 -c \
    "SELECT json_build_object('rows',count(*),'digest',md5(string_agg(row_data::text,',' ORDER BY quota_id))) FROM ssxz_v027_unlimited_quota_archive" > "$backup/quotas-after.json" || return 1
  cmp -s "$backup/quotas-before.json" "$backup/quotas-after.json"
}
set_link() {
  ln -s "$1" /opt/sub2api/current.v027-next
  mv -Tf /opt/sub2api/current.v027-next /opt/sub2api/current
}
cleanup() {
  local result=$?
  if (( success == 0 )); then
    echo "Release did not complete; preserving evidence and recovering old service."
    if (( rollback_allowed == 0 )); then
      install -m 644 /dev/null "$flag"
      install -m 644 "$candidate/server-nginx-sub2api.conf" /etc/nginx/sites-available/sub2api
      install -m 644 "$candidate/server-nginx-default.conf" /etc/nginx/sites-available/default
      if ! nginx -t || ! systemctl reload nginx; then
        echo "URGENT: could not verify external maintenance gate." >&2
      fi
      echo "New-runtime writes may have occurred; automatic downgrade prohibited. Maintenance retained." >&2
      exit 1
    fi
    if (( switched == 1 )); then
      systemctl stop sub2api || true
      if ! sudo -u postgres psql -X -d sub2api -v ON_ERROR_STOP=1 -f "$candidate/v027-rollback-check.sql" > "$backup/failed-rollback-gate.log"; then
        echo "Full rollback guard rejected downgrade; maintenance retained." >&2
        exit 1
      fi
      if ! sudo -u postgres psql -X -d sub2api -At -v ON_ERROR_STOP=1 -f "$candidate/v027-invariants.sql" > "$backup/invariants-at-failure.json" \
         || ! cmp -s "$backup/invariants-before.json" "$backup/invariants-at-failure.json" \
         || ! verify_preserved_quotas; then
        echo "Customer-data invariants not verified; maintenance retained for operator recovery." >&2
        exit 1
      fi
      set_link "$previous"
    fi
    if [[ -f "$runtime" ]]; then rm -- "$runtime"; systemctl daemon-reload; fi
    if (( stopped == 1 )); then systemctl start sub2api || true; fi
    if health; then
      if ! restore_nginx; then
        echo "Could not restore ingress configuration; maintenance retained." >&2
        exit 1
      fi
      rm -f -- "$flag"
      echo "Previous service available; database was NOT rewound."
    else
      echo "Service unhealthy; maintenance retained. Manual recovery required." >&2
    fi
  fi
  exit "$result"
}

[[ "$EUID" == 0 ]] || die "root required"
[[ "$(readlink -f /opt/sub2api/current)" == "$previous" ]] || die "current release changed"
[[ ! -e "$flag" && ! -e "$runtime" && ! -e /opt/sub2api/current.v027-next ]] || die "existing maintenance operation"
[[ "$(sha256sum "$candidate/sub2api" | cut -d' ' -f1)" == "$expected" ]] || die "candidate hash"
[[ "$(tr -d '\r\n' < /opt/ssxz-acceptance-20260921/ACCEPTED.sha256)" == "$expected" ]] || die "server QA receipt absent or stale"
(cd "$backup" && sha256sum -c SHA256SUMS) || die "recovery bundle hash"
cmp -s /etc/nginx/sites-available/sub2api "$backup/nginx-sub2api.conf" || die "nginx changed since snapshot"
cmp -s /etc/nginx/sites-available/default "$backup/nginx-default.conf" || die "nginx changed since snapshot"
trap cleanup EXIT

install -m 644 "$candidate/server-nginx-sub2api.conf" /etc/nginx/sites-available/sub2api
install -m 644 "$candidate/server-nginx-default.conf" /etc/nginx/sites-available/default
nginx -t
systemctl reload nginx
install -m 644 /dev/null "$flag"
echo "Maintenance enabled; draining existing requests."
for _ in {1..36}; do
  [[ "$(ss -Htn state established '( sport = :8080 )' | wc -l)" -eq 0 ]] && break
  sleep 5
done
[[ "$(ss -Htn state established '( sport = :8080 )' | wc -l)" -eq 0 ]] || die "requests did not drain; no forced interruption"
systemctl stop sub2api
stopped=1
systemctl is-active --quiet sub2api && die "old service still active"

# Fresh closed-write snapshot; the earlier backup did not stop customer activity.
sudo -u postgres pg_dump -Fc sub2api > "$backup/closed-window.dump"
pg_restore --list "$backup/closed-window.dump" > "$backup/closed-window.contents"
sha256sum "$backup/closed-window.dump" > "$backup/closed-window.sha256"
sudo -u postgres psql -X -d sub2api -At -v ON_ERROR_STOP=1 -f "$candidate/v027-invariants.sql" > "$backup/invariants-before.json"
sudo -u postgres psql -X -d sub2api -At -v ON_ERROR_STOP=1 -c \
  "SELECT json_build_object('rows',count(*),'digest',md5(string_agg(to_jsonb(q)::text,',' ORDER BY id))) FROM user_platform_quotas q WHERE daily_limit_usd IS NULL AND weekly_limit_usd IS NULL AND monthly_limit_usd IS NULL" > "$backup/quotas-before.json"

install -d -m 755 /run/systemd/system/sub2api.service.d
install -m 644 "$candidate/v027-acceptance.conf" "$runtime"
systemctl daemon-reload
set_link "$candidate"
switched=1
systemctl start sub2api
health || die "candidate health"
for page in / /admin/dashboard /app/usage /app/keys; do
  curl --fail --silent --max-time 10 "http://127.0.0.1:8080$page" | grep -q '<html' || die "candidate page $page"
done
sudo -u postgres psql -X -d sub2api -At -v ON_ERROR_STOP=1 -f "$candidate/v027-invariants.sql" > "$backup/invariants-after.json"
cmp -s "$backup/invariants-before.json" "$backup/invariants-after.json" || die "customer money/permissions changed"
verify_preserved_quotas || die "quota preservation mismatch"
sudo -u postgres psql -X -d sub2api -v ON_ERROR_STOP=1 -f "$candidate/v027-rollback-check.sql" > "$backup/rollback-gate.log"
echo "Closed-window acceptance passed; customer balances, grants and quota history preserved."

# Restore normal jobs before reopening traffic; no customer request is replayed.
# From this point on any background write closes the automatic downgrade window.
rollback_allowed=0
systemctl stop sub2api
rm -- "$runtime"
systemctl daemon-reload
systemctl start sub2api
health || die "normal-runtime health"
restore_nginx
rm -- "$flag"
success=1
echo "RELEASE ACTIVE: $candidate"
systemctl show sub2api -p MainPID -p ActiveState -p NRestarts -p ExecMainStartTimestamp
