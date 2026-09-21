#!/usr/bin/env python3
"""Attach existing production runtime paths; never create or rewrite config/data."""
from pathlib import Path
import hashlib

previous = Path('/opt/sub2api/releases/0.2.7-ssxz.20260921.ac2923c5e')
candidate = Path('/opt/sub2api/releases/0.2.7-ssxz.2.20260921.659ea5e23')
assert Path('/opt/sub2api/current').resolve() == previous
assert hashlib.sha256((candidate / 'sub2api').read_bytes()).hexdigest() == 'e49fdddc57efca68432c904492809dccca07fd1373357e239022f54548ac15ac'
expected = {'config.yaml': Path('/opt/sub2api/config.yaml'), 'data': previous / 'data'}
for name, target in expected.items():
    assert (previous / name).resolve() == target.resolve()
    assert target.exists()
    link = candidate / name
    if link.is_symlink() or link.exists():
        assert link.is_symlink() and link.resolve() == target.resolve()
    else:
        link.symlink_to(target, target_is_directory=target.is_dir())
assert (candidate / 'data' / '.installed').is_file() or (candidate / 'config.yaml').is_file()
print('existing_config_and_data_linked; no_configuration_or_database_content_changed')
