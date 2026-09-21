#!/usr/bin/env python3
"""Derive a temporary gate from exact backed-up, enabled nginx configs."""
from pathlib import Path
import os
import sys

base = Path(sys.argv[1]).resolve()
assert os.geteuid() == 0
assert base == Path('/opt/sub2api/backups/v027-catalog-patch-20260921-v4')
marker = '    location / {\n'
gate = '        if (-f /var/lib/ssxz-v027-maintenance) { return 503; }\n'
for name in ('sub2api', 'default'):
    original = (base / ('nginx-'+name+'.conf')).read_text()
    assert original.count(marker) == 1 and 'ssxz-v027-maintenance' not in original
    assert 'proxy_pass http://127.0.0.1:8080;' in original
    output = base / ('maintenance-'+name+'.conf')
    with output.open('x') as stream:
        stream.write(original.replace(marker, marker+gate, 1))
    output.chmod(0o600)
print('maintenance_configs_derived_from_enabled_snapshots=true')
