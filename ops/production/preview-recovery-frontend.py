#!/usr/bin/env python3
"""Advance only the explicit preview; preserve immutable assets and default users."""
import hashlib
from pathlib import Path
import re
import subprocess
import tarfile

archive = Path('/opt/ssxz-acceptance-20260921/catalog-web-recovery.tgz')
assert hashlib.sha256(archive.read_bytes()).hexdigest() == 'b8dbf84b9060c0cd55636eb5fb32ef6160e2b8953f88b7c59232de0d7e361664'
root = Path('/var/www/ssxz-ui-20260921-catalog')
mapping = Path('/etc/nginx/conf.d/ssxz-catalog-preview.conf')
backup = Path('/opt/sub2api/backups/v027-recovery-preview-20260921')
assert root.is_dir() and not backup.exists()
before = mapping.read_text()
assert 'payment-isolated /assets/entry-diagnostic-20260921.js;' in before
paths = [mapping] + [Path('/etc/nginx/sites-enabled') / name for name in ('sub2api', 'default')]
snapshots = {path: path.read_bytes() for path in paths}
backup.mkdir(mode=0o700)
for path, data in snapshots.items():
    (backup / path.name).write_bytes(data)
with tarfile.open(archive) as bundle:
    members = bundle.getmembers()
    for item in members:
        parts = Path(item.name).parts
        assert not item.name.startswith('/') and '..' not in parts and parts[0] in ('assets', 'index.html')
        assert item.isdir() or item.isfile()
        if item.isfile() and parts[0] == 'assets':
            data = bundle.extractfile(item).read()
            target = root.joinpath(*parts)
            if target.exists(): assert target.read_bytes() == data, 'Immutable collision: ' + item.name
    for item in members:
        if not item.isfile() or not item.name.startswith('assets/'): continue
        target = root / item.name
        target.parent.mkdir(parents=True, exist_ok=True)
        if not target.exists(): target.write_bytes(bundle.extractfile(item).read())
        target.chmod(0o644)
    html = bundle.extractfile('index.html').read().decode()
entry = re.search(r'<script type="module" crossorigin src="([^"]+)"', html)[1]
assert entry == '/assets/index-D2XpKt6e.js'
shell = re.search(r'<div id="app">[\s\S]*?</div>', html)[0].replace('\n', ' ')
assert "'" not in shell and '$' not in shell
try:
    mapping.write_text(before.replace('/assets/entry-diagnostic-20260921.js', entry) +
        '\nmap $arg_ssxz_catalog_preview $ssxz_startup_shell {\n' +
        '    default \'<div id="app"></div>\';\n' +
        "    payment-isolated '" + shell + "';\n}\n")
    for path in paths[1:]:
        text = snapshots[path].decode()
        assert text.count('        sub_filter_once on;\n') == 1
        path.write_text(text.replace('        sub_filter_once on;\n',
            '        sub_filter_once on;\n        sub_filter \'<div id="app"></div>\' \'$ssxz_startup_shell\';\n'))
    subprocess.run(['nginx', '-t'], check=True)
    subprocess.run(['systemctl', 'reload', 'nginx'], check=True)
except BaseException:
    for path, data in snapshots.items(): path.write_bytes(data)
    subprocess.run(['nginx', '-t'], check=True)
    subprocess.run(['systemctl', 'reload', 'nginx'], check=True)
    raise
print('recovery_preview_ready; default_users_unchanged')
