#!/usr/bin/env bash
set -euo pipefail

# Guarded same-schema release for the handoff fixes. This is intentionally
# separate from the older .2/.3 scripts: it refuses an unexpected live version,
# keeps the real enabled nginx files, and restores the prior release on failure.
candidate=${1:?candidate directory required}
expected=${2:?candidate sha256 required}
previous=/opt/sub2api/releases/0.2.7-ssxz.2.20260921.659ea5e23
current=/opt/sub2api/current
next=/opt/sub2api/current.handoff-next
backup=/opt/sub2api/backups/v027-handoff-20260921-r6
flag=/var/lib/ssxz-v027-handoff-maintenance
success=0
switched=0
stopped=0
die(){ echo "HANDOFF BLOCKED: $*" >&2; exit 1; }
health(){ for _ in {1..30}; do curl -fsS --max-time 2 http://127.0.0.1:8080/health >/dev/null && return 0; sleep 1; done; return 1; }
set_current(){ ln -s "$1" "$next"; mv -Tf "$next" "$current"; }
restore(){
  install -m 644 "$backup/nginx-sub2api.conf" /etc/nginx/sites-enabled/sub2api
  install -m 644 "$backup/nginx-default.conf" /etc/nginx/sites-enabled/default
  nginx -t && systemctl reload nginx
}
cleanup(){
  code=$?
  if ((success==0)); then
    if ((switched==1)); then systemctl stop sub2api || true; set_current "$previous"; fi
    if ((stopped==1)); then systemctl start sub2api || true; fi
    if health; then restore || true; rm -f -- "$flag"; fi
  fi
  exit "$code"
}
[[ $EUID == 0 ]] || die root-required
[[ "$candidate" == /opt/sub2api/releases/0.2.7-ssxz.3.20260921.handoff && "$(realpath "$candidate")" == "$candidate" ]] || die candidate-path
[[ "$expected" =~ ^[a-f0-9]{64}$ ]] || die hash-format
[[ "$(readlink -f "$current")" == "$previous" ]] || die current-version-changed
[[ ! -e "$flag" && ! -e "$next" && ! -e "$backup" ]] || die existing-operation
[[ -f "$candidate/sub2api" && -L "$candidate/config.yaml" && -L "$candidate/data" ]] || die candidate-runtime-links
[[ "$(sha256sum "$candidate/sub2api" | cut -d' ' -f1)" == "$expected" ]] || die candidate-hash
[[ "$(readlink -f "$candidate/config.yaml")" == /opt/sub2api/config.yaml ]] || die config-link
[[ "$(readlink -f "$candidate/data")" == "$(readlink -f "$previous/data")" ]] || die data-link
[[ -f /etc/nginx/sites-enabled/default && ! -L /etc/nginx/sites-enabled/default ]] || die nginx-default-layout
[[ -f /etc/nginx/sites-enabled/sub2api && ! -L /etc/nginx/sites-enabled/sub2api ]] || die nginx-api-layout
install -d -m 700 "$backup"
install -m 600 /etc/nginx/sites-enabled/sub2api "$backup/nginx-sub2api.conf"
install -m 600 /etc/nginx/sites-enabled/default "$backup/nginx-default.conf"
sudo -u postgres pg_dump -Fc -d sub2api > "$backup/database.dump"
pg_restore --list "$backup/database.dump" > "$backup/database.contents"
sha256sum "$backup/database.dump" > "$backup/database.sha256"
trap cleanup EXIT
install -m 644 /dev/null "$flag"
python3 - "$backup" <<'PY'
from pathlib import Path
import sys
base=Path(sys.argv[1])
gate='    return 503;\n'
for name in ('sub2api','default'):
    src=(base/f'nginx-{name}.conf').read_text()
    marker='server {\n'
    assert src.count(marker)>=1 and 'ssxz-v027-handoff-maintenance' not in src
    out=src.replace(marker,marker+gate)
    (base/f'maintenance-{name}.conf').write_text(out)
    (base/f'maintenance-{name}.conf').chmod(0o600)
PY
install -m 644 "$backup/maintenance-sub2api.conf" /etc/nginx/sites-enabled/sub2api
install -m 644 "$backup/maintenance-default.conf" /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx
for host in api.ssxzapi.com ssxzapi.com www.ssxzapi.com; do
  blocked=0
  for _ in {1..10}; do
    code=$(curl --noproxy '*' -k -sS --max-time 5 -o /dev/null -w '%{http_code}' --resolve "$host:443:127.0.0.1" "https://$host/health" || true)
    if [[ "$code" == 503 ]]; then blocked=1; break; fi
    sleep 1
  done
  ((blocked==1)) || die "real ingress maintenance ineffective:$host:$code"
done
for _ in {1..36}; do [[ "$(ss -Htn state established '( sport = :8080 )' | wc -l)" == 0 ]] && break; sleep 5; done
[[ "$(ss -Htn state established '( sport = :8080 )' | wc -l)" == 0 ]] || die requests-not-drained
systemctl stop sub2api; stopped=1
set_current "$candidate"; switched=1
systemctl start sub2api
health || die candidate-health
pid=$(systemctl show sub2api -p MainPID --value)
[[ "$(sha256sum /proc/$pid/exe | cut -d' ' -f1)" == "$expected" ]] || die running-hash
[[ "$(/proc/$pid/exe -version 2>&1)" == *'0.2.7-ssxz.3'* ]] || die running-version
for page in / /admin/dashboard /admin/groups /admin/ops /admin/accounts /app/usage /app/keys; do
  html=$(curl -fsS --max-time 15 "http://127.0.0.1:8080$page")
  [[ "$html" == *'/assets/index-VEnQXzrv.js'* && "$html" == *'ssxz-startup-status'* ]] || die "stale-page:$page"
done
restore
rm -f -- "$flag"
success=1
trap - EXIT
echo 'HANDOFF ACTIVE'
systemctl show sub2api -p ActiveState -p MainPID -p NRestarts -p ExecMainStartTimestamp
