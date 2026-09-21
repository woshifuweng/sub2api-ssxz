#!/usr/bin/env python3
"""Temporarily enable origin handshake diagnostics, with exact restore guard."""
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys

config = Path('/etc/nginx/sites-enabled/sub2api')
record = Path('/opt/ssxz-acceptance-20260921/tls-diagnostic-20260921.json')
log = Path('/var/log/nginx/ssxz-origin-handshake.log')
line = '    error_log /var/log/nginx/ssxz-origin-handshake.log info;\n'
assert os.geteuid() == 0 and config.is_file() and not config.is_symlink()
def run(*args):
    subprocess.run(args, check=True, capture_output=True)
if sys.argv[1] == 'begin':
    assert not record.exists(), 'Existing diagnostic: inspect and end it first'
    original = config.read_text()
    assert 'error_log ' not in original and original.count('server {') == 2
    changed = original.replace('server {\n', 'server {\n'+line, 1)
    record.write_text(json.dumps({'original':original, 'changed_sha':hashlib.sha256(changed.encode()).hexdigest()}))
    record.chmod(0o600)
    try:
        config.write_text(changed)
        run('nginx','-t')
        run('systemctl','reload','nginx')
    except BaseException:
        config.write_text(original)
        raise
    print('temporary_handshake_logging=enabled')
elif sys.argv[1] == 'end':
    state = json.loads(record.read_text())
    assert hashlib.sha256(config.read_bytes()).hexdigest() == state['changed_sha'], 'Concurrent nginx edit: refusing overwrite'
    config.write_text(state['original'])
    run('nginx','-t')
    run('systemctl','reload','nginx')
    if log.exists():
        lines = log.read_text(errors='replace').splitlines()
        # Only classify; no request URLs, client IPs, tokens or full headers.
        print(json.dumps({term:sum(term in row for row in lines) for term in ['SSL_do_handshake() failed','SSL handshaking','timed out','reset by peer','prematurely closed']}))
    record.rename(record.with_suffix('.completed.json'))
    print('original_nginx_config=restored')
else:
    raise SystemExit('begin or end required')
