#!/usr/bin/env python3
"""Read-only provider model discovery. Never print credentials or base URLs."""
import json
import subprocess
import urllib.request
import urllib.error
from urllib.parse import urlsplit
from concurrent.futures import ThreadPoolExecutor

query = """BEGIN READ ONLY; SELECT json_agg(t) FROM (
SELECT DISTINCT a.id,a.credentials FROM accounts a JOIN account_groups ag ON ag.account_id=a.id
WHERE ag.group_id IN (10,15,23) AND a.type='apikey' AND a.status='active' AND a.schedulable AND a.deleted_at IS NULL
) t; COMMIT;"""
raw = subprocess.run(['sudo','-u','postgres','psql','-X','-qAt','-v','ON_ERROR_STOP=1','-d','sub2api'],input=query,capture_output=True,text=True,check=True).stdout.strip()
accounts = json.loads(raw)

def probe(account):
    cred = account['credentials']
    base = cred.get('base_url','').rstrip('/')
    key = cred.get('api_key','')
    result = {'account_id':account['id']}
    if not base or not key or urlsplit(base).scheme != 'https':
        return dict(result,error='missing_https_base_or_key')
    endpoint = base+'/models' if base.endswith('/v1') else base+'/v1/models'
    try:
        req = urllib.request.Request(endpoint,headers={'Authorization':'Bearer '+key,'User-Agent':'SSXZ-readonly-model-discovery/0.2.7'})
        with urllib.request.urlopen(req,timeout=15) as response:
            data=json.loads(response.read(4*1024*1024))
        ids=sorted({x.get('id','') for x in data.get('data',[]) if isinstance(x,dict)})
        result.update(status=200,total=len(ids),target_models=[x for x in ids if x in ['gpt-6','gpt-6-astra','gpt-5.1']],models=ids)
    except urllib.error.HTTPError as error:
        result.update(status=error.code)
    except Exception as error:
        result.update(error=type(error).__name__)
    return result

with ThreadPoolExecutor(max_workers=2) as pool:
    for result in pool.map(probe,accounts):
        print(json.dumps(result),flush=True)
