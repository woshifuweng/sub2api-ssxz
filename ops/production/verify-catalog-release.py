#!/usr/bin/env python3
"""Read-only final runtime, public assets and post-start error evidence."""
import hashlib
import json
from pathlib import Path
import re
import subprocess
import urllib.request

def command(args):
    return subprocess.run(args, check=True, capture_output=True, text=True).stdout.strip()

def digest(path):
    value=hashlib.sha256()
    with open(path,'rb') as stream:
        for chunk in iter(lambda:stream.read(1024*1024), b''):
            value.update(chunk)
    return value.hexdigest()

expected = 'e49fdddc57efca68432c904492809dccca07fd1373357e239022f54548ac15ac'
pid = command(['systemctl','show','sub2api','-p','MainPID','--value'])
started = command(['systemctl','show','sub2api','-p','ExecMainStartTimestamp','--value'])
result = {'started':started, 'active':command(['systemctl','is-active','sub2api']),
          'restarts':int(command(['systemctl','show','sub2api','-p','NRestarts','--value'])),
          'binary_sha256':digest('/proc/'+pid+'/exe'),
          'maintenance_cleared':not Path('/var/lib/ssxz-v027-maintenance').exists(), 'pages':{}}
for path in ['/health','/','/admin/dashboard','/app/available-channels','/app/usage','/app/keys']:
    req = urllib.request.Request('https://api.ssxzapi.com'+path,
        headers={'User-Agent':'SSXZ-approved-readonly-release-check/0.2.7','Cache-Control':'no-cache'})
    with urllib.request.urlopen(req,timeout=15) as response:
        body=response.read().decode()
        result['pages'][path]={'http':response.status}
    if path != '/health':
        assert '<html' in body
        assert '/assets/index-D2XpKt6e.js' in body and 'ssxz-startup-status' in body, 'Stale frontend: '+path
asset_names = re.findall(r'(?:src|href)="/assets/([^"]+)"', body)
asset_hashes = {asset: digest(Path('/var/www/ssxz-ui-20260921-catalog/assets')/asset) for asset in asset_names}
for asset,expected_asset in asset_hashes.items():
    req=urllib.request.Request('https://api.ssxzapi.com/assets/'+asset,
        headers={'User-Agent':'SSXZ-approved-readonly-release-check/0.2.7'})
    with urllib.request.urlopen(req,timeout=20) as response:
        body=response.read()
        assert response.status == 200 and not body.lstrip().startswith(b'<!DOCTYPE')
        if expected_asset:
            assert hashlib.sha256(body).hexdigest()==expected_asset, 'Public asset hash mismatch: '+asset
        result['pages']['/assets/'+asset]={'http':response.status,'bytes':len(body)}
logs=command(['journalctl','-u','sub2api','--since',started,'--no-pager','-o','cat'])
result['redis_cleanup_errors']=logs.count('cleanup expired account slots failed')
result['fatal_or_panic']=len(re.findall(r'\t(?:FATAL|PANIC)\t|panic:',logs))
result['passed']=(result['active']=='active' and result['restarts']==0 and
                  result['binary_sha256']==expected and result['maintenance_cleared'] and
                  result['redis_cleanup_errors']==0 and result['fatal_or_panic']==0)
print(json.dumps(result,ensure_ascii=False,indent=2))
assert result['passed']
