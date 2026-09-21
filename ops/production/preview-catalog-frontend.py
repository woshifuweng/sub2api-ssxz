#!/usr/bin/env python3
"""Same-origin opt-in web preview, with production authentication unchanged."""
import hashlib
from pathlib import Path
import shutil
import subprocess
import tarfile

archive=Path('/opt/ssxz-acceptance-20260921/catalog-web-payment-isolated.tgz')
assert hashlib.sha256(archive.read_bytes()).hexdigest()=='777ba8e9166bd8aaebeceaf8b313d569855d8610e9a4612b557255973ebbb501'
old=Path('/var/www/ssxz-ui-20260921')
new=Path('/var/www/ssxz-ui-20260921-catalog')
backup=Path('/opt/sub2api/backups/v027-catalog-preview-20260921')
mapping=Path('/etc/nginx/conf.d/ssxz-catalog-preview.conf')
assert not new.exists() and not backup.exists() and not mapping.exists()
sources=[]
for name in ('sub2api','default'):
    path=Path('/etc/nginx/sites-enabled')/name
    content=path.read_bytes()
    assert content==(Path('/opt/sub2api/backups/v027-catalog-patch-20260921-v2')/('nginx-'+name+'.conf')).read_bytes()
    sources.append((path,content))
backup.mkdir(mode=0o700)
for path,content in sources:
    (backup/path.name).write_bytes(content)
shutil.copytree(old,new)
with tarfile.open(archive) as bundle:
    for item in bundle.getmembers():
        parts=Path(item.name).parts
        assert not item.name.startswith('/') and '..' not in parts and not item.issym() and not item.islnk()
        assert parts[0] in ('assets','index.html')
        target=new.joinpath(*parts)
        if item.isdir():
            target.mkdir(exist_ok=True,parents=True)
            continue
        assert item.isfile()
        data=bundle.extractfile(item).read()
        if target.exists() and parts[0]=='assets':
            assert target.read_bytes()==data, 'Immutable asset collision: '+item.name
        target.write_bytes(data)
        target.chmod(0o644)
mapping.write_text('''map $arg_ssxz_catalog_preview $ssxz_catalog_entry {
    default /assets/index-CLzwlvSf.js;
    payment-isolated /assets/index-CuZMQdEF.js;
}
map $arg_ssxz_catalog_preview $ssxz_catalog_vendor {
    default /assets/vendor-misc-BUu-QUbp.js;
    payment-isolated /assets/vendor-misc-Czm8xlOJ.js;
}
''')
mapping.chmod(0o644)
try:
    for path,content in sources:
        text=content.decode().replace(str(old)+';',str(new)+';')
        marker='        sub_filter_once on;\n'
        assert text.count(marker)==1
        extra=('        sub_filter \'/assets/index-CLzwlvSf.js\' \'$ssxz_catalog_entry\';\n'
               '        sub_filter \'/assets/index-DZ6Jp7RU.js\' \'$ssxz_catalog_entry\';\n'
               '        sub_filter \'/assets/vendor-misc-BUu-QUbp.js\' \'$ssxz_catalog_vendor\';\n')
        path.write_text(text.replace(marker,marker+extra,1))
    subprocess.run(['nginx','-t'],check=True)
    subprocess.run(['systemctl','reload','nginx'],check=True)
except BaseException:
    for path,content in sources:path.write_bytes(content)
    mapping.unlink()
    subprocess.run(['nginx','-t'],check=True)
    subprocess.run(['systemctl','reload','nginx'],check=True)
    raise
print('opt_in_preview_ready; default_entry_unchanged; backend_unchanged')
