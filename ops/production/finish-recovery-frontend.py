#!/usr/bin/env python3
"""Remove temporary entry rewrites after the cohesive embedded release is active."""
from pathlib import Path
import shutil
import subprocess
import urllib.request

release = Path('/opt/sub2api/releases/0.2.7-ssxz.2.20260921.659ea5e23')
assert Path('/opt/sub2api/current').resolve() == release
with urllib.request.urlopen('http://127.0.0.1:8080/') as response:
    html = response.read().decode()
assert '/assets/index-D2XpKt6e.js' in html and 'ssxz-startup-status' in html
backup = Path('/opt/sub2api/backups/v027-preview-retirement-20260921')
assert not backup.exists()
mapping = Path('/etc/nginx/conf.d/ssxz-catalog-preview.conf')
configs = [Path('/etc/nginx/sites-enabled') / name for name in ('sub2api', 'default')]
snapshots = {path: path.read_bytes() for path in configs + [mapping]}
backup.mkdir(mode=0o700)
for path, data in snapshots.items(): (backup / path.name).write_bytes(data)
try:
    for path in configs:
        text = snapshots[path].decode()
        assert 'root /var/www/ssxz-ui-20260921-catalog;' in text
        assert text.count('sub_filter_once on;') == 1
        path.write_text(''.join(line for line in text.splitlines(keepends=True)
                               if not line.lstrip().startswith(('sub_filter ', 'sub_filter_once '))))
    mapping.unlink()
    subprocess.run(['nginx', '-t'], check=True)
    subprocess.run(['systemctl', 'reload', 'nginx'], check=True)
except BaseException:
    for path, data in snapshots.items(): path.write_bytes(data)
    subprocess.run(['nginx', '-t'], check=True)
    subprocess.run(['systemctl', 'reload', 'nginx'], check=True)
    raise
diagnostic = Path('/var/www/ssxz-ui-20260921-catalog/assets/entry-diagnostic-20260921.js')
if diagnostic.is_file(): shutil.move(str(diagnostic), str(backup / diagnostic.name))
subprocess.run(['systemctl', 'stop', 'ssxz-acceptance-20260921'], check=True)
print('embedded_frontend_is_authoritative; preview_overrides_retired_with_backups; isolated_test_unit_stopped')
