#!/usr/bin/env bash
# Same-schema v0.2.7 patch only. Never downgrade to 0.1.183 or rewind data.
set -euo pipefail
cd /
candidate=${1:?candidate directory required}
expected=${2:?SHA256 required}
previous=/opt/sub2api/releases/0.2.7-ssxz.20260921.ac2923c5e
backup=/opt/sub2api/backups/v027-catalog-patch-20260921-v4
flag=/var/lib/ssxz-v027-maintenance
next=/opt/sub2api/current.catalog-next
switched=0
stopped=0
success=0
maintenance=0
die() { echo "PATCH BLOCKED: $*" >&2; exit 1; }
health() {
  for _ in {1..30}; do
    if curl -fsS --max-time 2 http://127.0.0.1:8080/health >/dev/null; then return 0; fi
    sleep 1
  done
  return 1
}
set_link() { ln -s "$1" "$next"; mv -Tf "$next" /opt/sub2api/current; }
restore_ingress() {
  install -m 644 "$backup/nginx-sub2api.conf" /etc/nginx/sites-enabled/sub2api
  install -m 644 "$backup/nginx-default.conf" /etc/nginx/sites-enabled/default
  nginx -t && systemctl reload nginx
}
cleanup() {
  local code=$?
  if ((success==0 && maintenance==1)); then
    if ((switched==1)); then
      systemctl stop sub2api || true
      set_link "$previous"
    fi
    if ((stopped==1)); then systemctl start sub2api || true; fi
    if health && restore_ingress; then
      rm -f -- "$flag"
      echo 'Patch aborted; previous v0.2.7 restored. No database rewind.'
    else
      echo 'URGENT: recovery not verified; maintenance retained.' >&2
    fi
  fi
  exit "$code"
}
[[ "$EUID" == 0 ]] || die 'root required'
[[ "$candidate" == /opt/sub2api/releases/0.2.7-ssxz.2.20260921.659ea5e23 && "$(realpath "$candidate")" == "$candidate" ]] || die 'candidate path'
[[ "$expected" =~ ^[a-f0-9]{64}$ ]] || die 'hash format'
[[ "$(readlink -f /opt/sub2api/current)" == "$previous" ]] || die 'current release changed'
[[ ! -e "$flag" && ! -e "$next" && ! -e "$backup" ]] || die 'existing operation or receipt'
[[ "$(sha256sum "$candidate/sub2api" | cut -d' ' -f1)" == "$expected" ]] || die 'candidate hash'
[[ "$(tr -d '\r\n' < "$candidate/PATCH_ACCEPTED.sha256")" == "$expected" ]] || die 'server acceptance missing'
[[ -L "$candidate/config.yaml" && "$(readlink -f "$candidate/config.yaml")" == /opt/sub2api/config.yaml ]] || die 'production config link missing'
[[ -L "$candidate/data" && "$(readlink -f "$candidate/data")" == "$(readlink -f "$previous/data")" ]] || die 'production data link missing'
[[ -f "$candidate/config.yaml" && -d "$candidate/data" ]] || die 'runtime path target missing'
[[ -f /etc/nginx/sites-enabled/sub2api && ! -L /etc/nginx/sites-enabled/sub2api ]] || die 'nginx api layout changed'
[[ -f /etc/nginx/sites-enabled/default && ! -L /etc/nginx/sites-enabled/default ]] || die 'nginx root layout changed'
install -d -m 700 "$backup"
install -m 600 /etc/nginx/sites-enabled/sub2api "$backup/nginx-sub2api.conf"
install -m 600 /etc/nginx/sites-enabled/default "$backup/nginx-default.conf"
python3 /opt/ssxz-acceptance-20260921/prepare-patch-ingress.py "$backup"
trap cleanup EXIT
maintenance=1
install -m 644 /dev/null "$flag"
install -m 644 "$backup/maintenance-sub2api.conf" /etc/nginx/sites-enabled/sub2api
install -m 644 "$backup/maintenance-default.conf" /etc/nginx/sites-enabled/default
nginx -t
systemctl reload nginx
# Confirm the actual loaded virtual hosts reject new requests before draining.
for host in api.ssxzapi.com ssxzapi.com www.ssxzapi.com; do
  blocked=0
  for _ in {1..10}; do
    code=$(curl -s --max-time 3 -o /dev/null -w '%{http_code}' --resolve "$host:443:127.0.0.1" "https://$host/health")
    if [[ "$code" == 503 ]]; then blocked=1; break; fi
    sleep 1
  done
  ((blocked==1)) || die "maintenance gate ineffective: $host"
done
for _ in {1..36}; do
  [[ "$(ss -Htn state established '( sport = :8080 )' | wc -l)" -eq 0 ]] && break
  sleep 5
done
[[ "$(ss -Htn state established '( sport = :8080 )' | wc -l)" -eq 0 ]] || die 'requests did not drain'
systemctl stop sub2api
stopped=1
sudo -u postgres pg_dump -Fc -d sub2api > "$backup/database.dump"
pg_restore --list "$backup/database.dump" > "$backup/database.contents"
sha256sum "$backup/database.dump" > "$backup/database.sha256"
set_link "$candidate"
switched=1
systemctl start sub2api
health || die 'candidate health'
for page in / /admin/dashboard /app/available-channels /app/usage /app/keys; do
  html=$(curl -fsS --max-time 10 "http://127.0.0.1:8080$page")
  [[ "$html" == *'/assets/index-D2XpKt6e.js'* && "$html" == *'ssxz-startup-status'* ]] || die "page $page"
done
pid=$(systemctl show sub2api -p MainPID --value)
[[ "$(sha256sum "/proc/$pid/exe" | cut -d' ' -f1)" == "$expected" ]] || die 'running binary mismatch'
restore_ingress
rm -- "$flag"
success=1
echo "PATCH ACTIVE: $candidate"
systemctl show sub2api -p ActiveState -p MainPID -p NRestarts -p ExecMainStartTimestamp
