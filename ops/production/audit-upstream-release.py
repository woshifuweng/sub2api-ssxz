#!/usr/bin/env python3
"""Read-only provenance audit for an upstream Sub2API release."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

SHA = re.compile(r"^[0-9a-f]{40}$")


def run(repo: Path, *args: str) -> str:
    return subprocess.check_output(["git", "-C", str(repo), *args], text=True).strip()


def validate_manifest(manifest: dict) -> None:
    required = ("upstream_repo", "upstream_tag", "upstream_commit", "merge_commit", "custom_parent")
    missing = [key for key in required if not manifest.get(key)]
    if missing:
        raise ValueError(f"missing manifest fields: {', '.join(missing)}")
    for key in ("upstream_commit", "merge_commit", "custom_parent"):
        if not SHA.fullmatch(str(manifest[key])):
            raise ValueError(f"invalid {key}: {manifest[key]}")


def audit(repo: Path, manifest_path: Path, offline: bool = False) -> dict:
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    validate_manifest(manifest)
    merge = manifest["merge_commit"]
    parents = run(repo, "show", "-s", "--format=%P", merge).split()
    if manifest["upstream_commit"] not in parents:
        raise ValueError("official upstream commit is not a parent of the recorded merge commit")
    if manifest["custom_parent"] not in parents:
        raise ValueError("recorded SSXZ custom parent is not a parent of the merge commit")

    remote_commit = None
    if not offline:
        ref = f"refs/tags/{manifest['upstream_tag']}^{{}}"
        remote_lines = subprocess.check_output(
            ["git", "ls-remote", manifest["upstream_repo"], ref], text=True
        ).splitlines()
        if not remote_lines:
            raise ValueError(f"remote tag not found: {ref}")
        remote_commit = remote_lines[0].split()[0]
        if remote_commit != manifest["upstream_commit"]:
            raise ValueError(f"remote tag mismatch: {remote_commit} != {manifest['upstream_commit']}")

    return {
        "upstream_tag": manifest["upstream_tag"],
        "upstream_commit": manifest["upstream_commit"],
        "merge_commit": merge,
        "custom_parent": manifest["custom_parent"],
        "remote_verified": not offline,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, default=Path.cwd())
    parser.add_argument(
        "--manifest",
        type=Path,
        default=Path("docs/release-ledger/sub2api-v0.2.8.json"),
    )
    parser.add_argument("--offline", action="store_true", help="skip remote GitHub lookup")
    args = parser.parse_args()
    try:
        result = audit(args.repo.resolve(), args.manifest.resolve(), args.offline)
    except (OSError, ValueError, subprocess.CalledProcessError, json.JSONDecodeError) as exc:
        print(f"PROVENANCE AUDIT FAILED: {exc}", file=sys.stderr)
        return 1
    print(json.dumps(result, ensure_ascii=False, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
