#!/usr/bin/env python3
"""Bounded frontend-only recovery; preserve the running backend and all data."""
from pathlib import Path
import subprocess
import hashlib

base=Path('/opt/sub2api/backups/v027-catalog-patch-20260921-v2')
files=[]
for name in ('sub2api','default'):
    path=Path('/etc/nginx/sites-enabled')/name
    original=path.read_bytes()
    assert original==(base/('nginx-'+name+'.conf')).read_bytes(), 'Enabled config changed'
    text=original.decode()
    marker='        sub_filter_once on;\n'
    assert text.count(marker)==1
    extra=("        sub_filter '/assets/index-DZ6Jp7RU.js' '/assets/index-BCzwrAYs.js';\n"
           "        sub_filter '/assets/index-DOJWOAln.css' '/assets/index-80zox4qL.css';\n")
    assert Path('/var/www/ssxz-ui-20260921/assets/index-BCzwrAYs.js').is_file()
    assert Path('/var/www/ssxz-ui-20260921/assets/index-80zox4qL.css').is_file()
    files.append((path,original,text.replace(marker,marker+extra,1)))
try:
    for path,original,replacement in files:
        path.write_text(replacement)
    subprocess.run(['nginx','-t'],check=True)
    subprocess.run(['systemctl','reload','nginx'],check=True)
except BaseException:
    for path,original,_ in files:
        path.write_bytes(original)
    subprocess.run(['nginx','-t'],check=True)
    subprocess.run(['systemctl','reload','nginx'],check=True)
    raise
print('frontend_only_recovery_active; backend_and_database_unchanged')
