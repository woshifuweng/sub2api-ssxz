#!/usr/bin/env python3
"""Read only; print non-secret effective infrastructure settings and privileges."""
import json
import os
from pathlib import Path
import shlex
import subprocess
import yaml


def env_file(path):
    result = {}
    for line in Path(path).read_text().splitlines():
        parts = shlex.split(line, comments=True)
        if not parts:
            continue
        if parts[0] == 'export':
            parts.pop(0)
        if len(parts) != 1 or '=' not in parts[0]:
            raise ValueError('unsupported environment syntax')
        key, value = parts[0].split('=', 1)
        result[key] = value
    return result


def snapshot(label, env_path, config_path):
    env = env_file(env_path)
    config = yaml.safe_load(Path(config_path).read_text()) or {}
    def setting(section, key, default=None):
        return env.get(f'{section}_{key}'.upper(), config.get(section, {}).get(key, default))
    db = {key: setting('database', key) for key in ('host', 'port', 'dbname', 'user')}
    redis = {key: setting('redis', key) for key in ('host', 'port', 'db')}
    print(json.dumps({'runtime': label, 'database': db, 'redis': redis,
                      'data_dir': env.get('DATA_DIR'), 'role': env.get('APP_RUNTIME_ROLE')}, ensure_ascii=False))
    pgenv = {**os.environ, 'PGPASSWORD': str(setting('database', 'password', '')),
             'PGCONNECT_TIMEOUT': '5'}
    sql = "SELECT current_database(), current_user, has_table_privilege(current_user, 'public.schema_migrations', 'SELECT'), has_schema_privilege(current_user, 'public', 'CREATE');"
    proc = subprocess.run(['psql', '-h', str(db['host']), '-p', str(db['port']),
                           '-U', str(db['user']), '-d', str(db['dbname']), '-Atqc', sql],
                          env=pgenv, capture_output=True, text=True, timeout=10)
    print(json.dumps({'runtime': label, 'permission_check_exit': proc.returncode,
                      'result': proc.stdout.strip() if proc.returncode == 0 else 'database access failed'}))


if __name__ == '__main__':
    snapshot('production', '/etc/sub2api/production-payment-resume.env', '/opt/sub2api/config.yaml')
    snapshot('preflight', '/etc/sub2api/preflight.env', '/etc/sub2api/preflight-data/config.yaml')
