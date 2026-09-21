#!/usr/bin/env python3
"""Install an explicitly hashed candidate without switching production."""
import hashlib
from pathlib import Path
import tarfile

archive = Path('/opt/ssxz-acceptance-20260921/sub2api-recovery-binary.tgz')
target = Path('/opt/sub2api/releases/0.2.7-ssxz.2.20260921.659ea5e23')
previous = Path('/opt/sub2api/releases/0.2.7-ssxz.20260921.ac2923c5e')
assert not target.exists()
assert Path('/opt/sub2api/current').resolve() == previous
assert (previous / 'config.yaml').resolve() == Path('/opt/sub2api/config.yaml')
assert (previous / 'config.yaml').is_file() and (previous / 'data').is_dir()
assert hashlib.sha256(archive.read_bytes()).hexdigest() == 'fbcfbe913e7f2b43c3f33012f04454d6edf18c07f81e201913931464562cfcbc'
with tarfile.open(archive) as bundle:
    members = bundle.getmembers()
    assert len(members) == 1 and members[0].isfile() and members[0].name == 'sub2api-0.2.7-ssxz.2'
    data = bundle.extractfile(members[0]).read()
assert hashlib.sha256(data).hexdigest() == 'e49fdddc57efca68432c904492809dccca07fd1373357e239022f54548ac15ac'
target.mkdir(mode=0o755)
(target / 'sub2api').write_bytes(data)
(target / 'sub2api').chmod(0o755)
(target / 'config.yaml').symlink_to('/opt/sub2api/config.yaml')
(target / 'data').symlink_to(previous / 'data', target_is_directory=True)
print('recovery_candidate_installed; production_unchanged')
