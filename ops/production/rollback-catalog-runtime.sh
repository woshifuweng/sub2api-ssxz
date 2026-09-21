#!/usr/bin/env bash
# Emergency same-schema rollback, preserving every committed customer write.
set -euo pipefail
current=/opt/sub2api/releases/0.2.7-ssxz.1.20260921.d5147b2b2
previous=/opt/sub2api/releases/0.2.7-ssxz.20260921.ac2923c5e
backup=/opt/sub2api/backups/v027-catalog-patch-20260921-v2
flag=/var/lib/ssxz-v027-maintenance
next=/opt/sub2api/current.recovery-next
[[ "$(readlink -f /opt/sub2api/current)" == "$current" ]]
[[ ! -e "$flag" && ! -e "$next" ]]
[[ "$(sha256sum "$previous/sub2api" | cut -d' ' -f1)" == 29df07a4cd03482a3058ebd485211e6f6ba9945f9f1b0bda5b3185181d689794 ]]
restore_ingress() {
 install -m 644 "$backup/nginx-sub2api.conf" /etc/nginx/sites-enabled/sub2api
 install -m 644 "$backup/nginx-default.conf" /etc/nginx/sites-enabled/default
 nginx -t && systemctl reload nginx
 rm -f -- "$flag"
}
trap restore_ingress EXIT
install -m 644 /dev/null "$flag"
install -m 644 "$backup/maintenance-sub2api.conf" /etc/nginx/sites-enabled/sub2api
install -m 644 "$backup/maintenance-default.conf" /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx
sleep 1
[[ "$(curl -s -o /dev/null -w '%{http_code}' --resolve api.ssxzapi.com:443:127.0.0.1 https://api.ssxzapi.com/health)" == 503 ]]
for _ in {1..36}; do
 [[ "$(ss -Htn state established '( sport = :8080 )' | wc -l)" -eq 0 ]] && break
 sleep 5
done
[[ "$(ss -Htn state established '( sport = :8080 )' | wc -l)" -eq 0 ]]
systemctl stop sub2api
ln -s "$previous" "$next"
mv -Tf "$next" /opt/sub2api/current
systemctl start sub2api
for _ in {1..30}; do
 curl -fsS --max-time 2 http://127.0.0.1:8080/health >/dev/null && break
 sleep 1
done
curl -fsS --max-time 2 http://127.0.0.1:8080/health >/dev/null
echo 'previous_verified_v027_restored; database_not_rewound; new_model_configuration_preserved'
systemctl show sub2api -p ActiveState -p MainPID -p ExecMainStartTimestamp
