#!/usr/bin/env python3
"""Read-only discovery through existing administrator test keys. No generation."""
import json
import subprocess
import urllib.request

for key_id, expected in [(76, ['gpt-6','gpt-6-astra']), (78, ['claude-fable-5-1','claude-opus-5'])]:
    sql = f"""SELECT k.key FROM api_keys k JOIN users u ON u.id=k.user_id
        WHERE k.id={key_id} AND k.status='active' AND k.deleted_at IS NULL
        AND u.role='admin' AND u.status='active' AND u.deleted_at IS NULL"""
    key = subprocess.run(['sudo','-u','postgres','psql','-X','-qAt','-d','sub2api','-v','ON_ERROR_STOP=1','-c',sql],
                         check=True,capture_output=True,text=True).stdout.strip()
    if not key or '\n' in key: raise SystemExit('Eligible owner test key unavailable')
    req = urllib.request.Request('https://api.ssxzapi.com/v1/models',headers={'Authorization':'Bearer '+key,
        'User-Agent':'SSXZ-approved-readonly-catalog-check/0.2.7', 'Accept':'application/json'})
    with urllib.request.urlopen(req,timeout=25) as response:
        data = json.load(response)
        status = response.status
    models = sorted({m['id'] for m in data.get('data',[]) if 'id' in m})
    print(json.dumps({'key_id':key_id,'status':status,'model_count':len(models),
                      'expected':{m:m in models for m in expected},'models':models}),flush=True)
