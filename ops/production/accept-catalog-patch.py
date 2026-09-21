#!/usr/bin/env python3
"""Same-schema patch acceptance in the already isolated server test unit."""
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time
import urllib.request

sys.dont_write_bytecode = True
sys.path.insert(0, '/opt/ssxz-acceptance-20260921')
from prepare import BASE, CONF, DB, UNIT, command, digest, env_read, prod_identity, require, private_write
from acceptance import http_checks

candidate = Path(sys.argv[1]).resolve()
expected = sys.argv[2]
require(os.geteuid() == 0, 'Root required')
require(str(candidate).startswith('/opt/sub2api/releases/0.2.7-ssxz.1.20260921.'), 'Candidate path')
require(re.fullmatch('[a-f0-9]{64}', expected), 'Hash format')
require(digest(candidate/'sub2api') == expected, 'Candidate hash')
before = prod_identity()
command(['systemctl','stop',UNIT])
old = BASE/'candidate.before-catalog-patch'
require(not old.exists(), 'Existing patch operation; inspect before resuming')
command(['cp','--preserve=mode,timestamps',str(BASE/'candidate'),str(old)])
command(['install','-m','755',str(candidate/'sub2api'),str(BASE/'candidate')])
command(['systemctl','start',UNIT])
for attempt in range(30):
    try:
        with urllib.request.urlopen('http://127.0.0.1:18080/health',timeout=2) as r:
            if r.status == 200: break
    except OSError:
        time.sleep(1)
else:
    raise SystemExit('Isolated patch did not become healthy; production not changed')
pid = command(['systemctl','show',UNIT,'-p','MainPID','--value'])
require(digest('/proc/'+pid+'/exe') == expected,'Isolated running binary mismatch')
runtime = dict(x.split('=',1) for x in Path('/proc/'+pid+'/environ').read_text().split('\0') if '=' in x)
require(runtime.get('DATABASE_DBNAME') == DB and runtime.get('REDIS_PORT') == '6380' and runtime.get('BACKGROUND_JOBS_ENABLED') == 'false','Isolation configuration mismatch')
require(not list(Path('/proc/'+pid+'/root/opt/sub2api').iterdir()),'Production directory not masked')
require('0.0.0.0/0' in command(['systemctl','show',UNIT,'-p','IPAddressDeny','--value']),'Egress gate missing')
http_checks('catalog_patch')
identities = env_read(CONF/'test-identities.env')
for who in ['ADMIN','BETA']:
    body = json.dumps({'email':identities['STAGING_'+who+'_EMAIL'], 'password':identities['STAGING_'+who+'_PASSWORD']}).encode()
    request = urllib.request.Request('http://127.0.0.1:18080/api/v1/auth/login',data=body,headers={'Content-Type':'application/json'})
    with urllib.request.urlopen(request,timeout=10) as r:
        payload = json.load(r)
    token = payload['data']['access_token']
    request = urllib.request.Request('http://127.0.0.1:18080/api/v1/channels/available',headers={'Authorization':'Bearer '+token})
    with urllib.request.urlopen(request,timeout=10) as r:
        data = json.load(r)['data']
    require(isinstance(data,list),'Catalog response contract')
    def safe_keys(value):
        if isinstance(value,dict):
            require(not ({'credentials','api_key','password','access_token'} & value.keys()),'Secret field in catalog')
            for child in value.values(): safe_keys(child)
        if isinstance(value,list):
            for child in value: safe_keys(child)
    safe_keys(data)
    print('authenticated_catalog_'+who.lower()+'=passed',flush=True)
started = command(['systemctl','show',UNIT,'-p','ExecMainStartTimestamp','--value'])
logs = command(['journalctl','-u',UNIT,'--since',started,'--no-pager','-o','cat'])
require(not re.search(r'\t(?:ERROR|FATAL|PANIC)\t|panic:',logs),'Isolated runtime error')
require(prod_identity() == before,'Production changed during isolated acceptance')
private_write(candidate/'PATCH_ACCEPTED.sha256',expected+'\n')
print('patch_acceptance=passed; production_unchanged=true',flush=True)
