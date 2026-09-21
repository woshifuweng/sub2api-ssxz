#!/usr/bin/env python3
"""Read-only provider model discovery. Never print credentials or base URLs."""
import json
import subprocess
import urllib.request
import urllib.error
from pathlib import Path
from urllib.parse import urlsplit
from concurrent.futures import ThreadPoolExecutor

query = """BEGIN READ ONLY; SELECT json_agg(t) FROM (
SELECT DISTINCT a.id,a.platform,a.credentials FROM accounts a JOIN account_groups ag ON ag.account_id=a.id
WHERE a.type='apikey' AND a.status='active' AND a.schedulable AND a.deleted_at IS NULL
) t; COMMIT;"""
raw = subprocess.run(['sudo','-u','postgres','psql','-X','-qAt','-v','ON_ERROR_STOP=1','-d','sub2api'],input=query,capture_output=True,text=True,check=True).stdout.strip()
accounts = json.loads(raw)

def probe(account):
    cred = account['credentials']
    base = cred.get('base_url','').rstrip('/')
    key = cred.get('api_key','')
    result = {'account_id':account['id'],'platform':account['platform']}
    if not base or not key or urlsplit(base).scheme != 'https':
        return dict(result,error='missing_https_base_or_key')
    if base.endswith('/models'):
        endpoint = base
    elif urlsplit(base).path.rstrip('/').endswith(('/v1','/v2','/v3','/v4','/v1beta')):
        endpoint = base+'/models'
    else:
        endpoint = base+('/v1beta/models' if account['platform']=='gemini' else '/v1/models')
    try:
        headers={'Authorization':'Bearer '+key,'User-Agent':'SSXZ-readonly-model-discovery/0.2.7'}
        if account['platform']=='anthropic':
            headers.update({'x-api-key':key,'anthropic-version':'2023-06-01'})
        if account['platform']=='gemini':
            headers={'x-goog-api-key':key,'User-Agent':'SSXZ-readonly-model-discovery/0.2.7'}
        req = urllib.request.Request(endpoint,headers=headers)
        with urllib.request.urlopen(req,timeout=15) as response:
            data=json.loads(response.read(4*1024*1024))
        ids=sorted({x.get('id',x.get('name','')).removeprefix('models/') for x in data.get('data',data.get('models',[])) if isinstance(x,dict)})
        result.update(status=200,total=len(ids),target_models=[x for x in ids if x in ['gpt-6','gpt-6-astra','gpt-5.1','claude-fable-5.1','claude-fable-5-1','claude-opus-5']],models=ids)
    except urllib.error.HTTPError as error:
        result.update(status=error.code)
    except Exception as error:
        result.update(error=type(error).__name__)
    return result

with ThreadPoolExecutor(max_workers=2) as pool:
    results=[]
    for result in pool.map(probe,accounts):
        results.append(result)
        print(json.dumps({key:value for key,value in result.items() if key!='models'}),flush=True)
    evidence=Path('/opt/sub2api/backups/v027-release-20260921/all-active-model-catalogs.json')
    with evidence.open('x') as output:
        json.dump(results,output,indent=2)
