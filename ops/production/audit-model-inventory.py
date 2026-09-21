#!/usr/bin/env python3
"""Read-only inventory: distinguish advertised models from configured routes.

No upstream generation, credentials, customer data, or configuration writes.
Provider evidence is the timestamped read-only catalog probe, not a smoke test.
"""
import datetime
import fnmatch
import json
import subprocess
from pathlib import Path

root = Path('/opt/sub2api/backups/v027-release-20260921')
catalogs = json.loads((root / 'all-active-model-catalogs.json').read_text())
sql = """BEGIN READ ONLY;
SELECT json_build_object(
 'accounts',(SELECT json_agg(x) FROM (SELECT a.id,a.platform,a.status,a.schedulable,
    a.credentials->'model_mapping' AS mapping,
    (SELECT json_agg(ag.group_id) FROM account_groups ag WHERE ag.account_id=a.id) AS groups
    FROM accounts a WHERE a.deleted_at IS NULL) x),
 'groups',(SELECT json_agg(x) FROM (SELECT id,name,platform,status,rate_multiplier,model_pricing
    FROM groups WHERE deleted_at IS NULL) x),
 'channels',(SELECT json_agg(x) FROM (SELECT c.id,c.status,c.restrict_models,c.billing_model_source,c.model_mapping,
    (SELECT json_agg(cg.group_id) FROM channel_groups cg WHERE cg.channel_id=c.id) AS groups,
    (SELECT json_agg(json_build_object('platform',p.platform,'models',p.models))
     FROM channel_model_pricing p WHERE p.channel_id=c.id) AS prices FROM channels c) x));
COMMIT;"""
data = json.loads(subprocess.run(['sudo','-u','postgres','psql','-X','-qAt','-d','sub2api','-v','ON_ERROR_STOP=1'],
    input=sql, check=True, capture_output=True, text=True).stdout.strip())
accounts = {a['id']: a for a in data['accounts']}
groups = {g['id']: g for g in data['groups']}
rows = []
for catalog in catalogs:
    a = accounts.get(catalog['account_id'])
    if not a:
        continue
    mapping = a['mapping'] or {}
    for model in catalog.get('models', []):
        accepted = not mapping or any(fnmatch.fnmatchcase(model, pattern) for pattern in mapping)
        for gid in a['groups'] or []:
            g = groups.get(gid)
            if not g:
                continue
            ch = next((c for c in data['channels'] if gid in (c['groups'] or [])), None)
            restricted = False
            if ch and ch['status'] == 'active' and ch['restrict_models']:
                mapped = (ch['model_mapping'] or {}).get(g['platform'], {}).get(model, model)
                billed = model if ch['billing_model_source'] == 'requested' else mapped
                restricted = not any(p['platform'] == g['platform'] and
                    any(fnmatch.fnmatchcase(billed.lower(), n.lower()) for n in p['models']) for p in (ch['prices'] or []))
            reason = ('account_disabled' if a['status'] != 'active' or not a['schedulable'] else
                      'group_disabled' if g['status'] != 'active' else
                      'account_model_not_enabled' if not accepted else
                      'channel_price_restriction' if restricted else 'configured_not_individually_smoke_tested')
            rows.append({'account_id': a['id'], 'platform': a['platform'], 'group_id': gid,
                         'group': g['name'], 'model': model, 'state': reason})
report = {'generated_utc':datetime.datetime.now(datetime.timezone.utc).isoformat(),
          'catalog_accounts':len(catalogs), 'note':'Provider catalogs are not proof of successful inference. Key-level grants are not widened.',
          'rows': rows}
output = root / 'model-readiness-inventory.json'
output.write_text(json.dumps(report, ensure_ascii=False, indent=2))
counts = {}
for row in rows:
    counts[row['state']] = counts.get(row['state'],0)+1
print(json.dumps({'catalog_accounts':len(catalogs),'unique_advertised_models':len({r['model'] for r in rows}),
                  'account_group_model_combinations':len(rows),'counts':counts},ensure_ascii=False))
