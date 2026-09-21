#!/usr/bin/env python3
"""Single-release server evidence check; temporary staging tables only."""
import json
import os
from pathlib import Path
import re
import subprocess
import sys

sys.dont_write_bytecode = True
sys.path.insert(0, '/opt/ssxz-acceptance-20260921')
from acceptance import NEW, OLD, RELEASE, schema, http_checks
from prepare import BASE, CONF, DB, UNIT, command, digest, env_read, isolation_counts, prod_identity, require, private_write

require(os.geteuid() == 0, 'Root required')
manifest = json.loads((CONF / 'prepared.json').read_text())
cycle = json.loads((CONF / 'cycle-passed.json').read_text())
require(cycle == dict(candidate_sha256=NEW, old_sha256=OLD, invariants_unchanged=True, production_unchanged=True), 'Cycle evidence mismatch')
require(prod_identity() == manifest['production'], 'Production changed')
pid = command(['systemctl','show',UNIT,'-p','MainPID','--value'])
require(digest('/proc/'+pid+'/exe') == NEW and digest(RELEASE/'sub2api') == NEW, 'Candidate mismatch')
runtime = dict(x.split('=',1) for x in Path('/proc/'+pid+'/environ').read_text().split('\0') if '=' in x)
require(runtime.get('DATABASE_DBNAME') == DB and runtime.get('REDIS_PORT') == '6380' and runtime.get('BACKGROUND_JOBS_ENABLED') == 'false', 'Runtime isolation failed')
masked = Path('/proc/'+pid+'/root/opt/sub2api')
require(not list(masked.iterdir()), 'Production directory exposed')
require('0.0.0.0/0' in command(['systemctl','show',UNIT,'-p','IPAddressDeny','--value']), 'Egress gate missing')
isolation_counts(env_read('/etc/sub2api/staging.env'))
backup = Path(manifest['backup'])
hashes = json.loads((backup/'checksums.json').read_text())
require(all(digest(backup/name)==value for name,value in hashes.items()), 'Original staging backup hash changed')
dump = command(['pg_restore','--file=-','--data-only','--table=groups','--table=user_platform_quotas',str(backup/'staging-before.dump')])
copies = {}
for match in re.finditer(r'COPY public\.(groups|user_platform_quotas) (\([^\n]+\) FROM stdin;\n)(.*?)\n\\\.', dump, re.S):
    table, header, data = match.groups()
    require(table not in copies, 'Duplicate dump table')
    copies[table] = 'COPY snapshot_'+table+' '+header+data+'\n\\.\n'
require(set(copies)=={'groups','user_platform_quotas'}, 'Expected snapshot tables missing')
sql = "BEGIN; SET LOCAL statement_timeout='30s';\n"
for table, copy in copies.items():
    sql += 'CREATE TEMP TABLE snapshot_'+table+' AS SELECT * FROM public.'+table+' WITH NO DATA;\n'+copy
sql += """
SELECT 'groups|' || count(*) FROM snapshot_groups g FULL JOIN ssxz_v027_legacy_model_lists a ON a.group_id=g.id WHERE g.models_list_config IS DISTINCT FROM a.models_list_config OR g.id IS NULL OR a.group_id IS NULL;
SELECT 'quotas|' || count(*) FROM (SELECT id,to_jsonb(q) AS original FROM snapshot_user_platform_quotas q WHERE daily_limit_usd IS NULL AND weekly_limit_usd IS NULL AND monthly_limit_usd IS NULL) q FULL JOIN ssxz_v027_unlimited_quota_archive a ON a.quota_id=q.id WHERE q.original IS DISTINCT FROM a.row_data;
SELECT 'limited|' || count(*) FROM (SELECT to_jsonb(q) AS original FROM snapshot_user_platform_quotas q WHERE daily_limit_usd IS NOT NULL OR weekly_limit_usd IS NOT NULL OR monthly_limit_usd IS NOT NULL EXCEPT SELECT to_jsonb(q) FROM user_platform_quotas q) missing;
ROLLBACK;
"""
check = subprocess.run(['sudo','-n','-u','postgres','psql','-X','-w','-qAt','-v','ON_ERROR_STOP=1','-d',DB],input=sql,capture_output=True,text=True)
require(check.returncode==0, 'Snapshot comparison SQL: '+next((line for line in check.stderr.splitlines() if line.startswith('ERROR:')), 'failed; inspect privately'))
result = check.stdout.strip()
require(result.splitlines()==['groups|0','quotas|0','limited|0'], 'Pre-migration archive equivalence failed: '+result)
print('original_snapshot_archive_equivalence=True',flush=True)
schema()
http_checks('final_candidate')
started = command(['systemctl','show',UNIT,'-p','ExecMainStartTimestamp','--value'])
logs = command(['journalctl','-u',UNIT,'--since',started,'--no-pager','-o','cat'])
require(not re.search(r'\t(?:ERROR|FATAL|PANIC)\t|panic:',logs), 'Startup errors require review')
known_warnings = ['TOTP encryption key auto-generated','configured JWT secret mismatches persisted value','payment encryption/signing key is not explicitly configured','server.trusted_proxies is not configured','CORS allowed_origins not configured']
unknown = [line for line in logs.splitlines() if '\tWARN\t' in line and not any(message in line for message in known_warnings)]
require(not unknown, 'Unreviewed startup warning count='+str(len(unknown)))
require(prod_identity()==manifest['production'], 'Production changed during final review')
private_write(BASE/'ACCEPTED.sha256',NEW+'\n')
print('server_acceptance_receipt_created=True',flush=True)
print('production_unchanged=True',flush=True)
