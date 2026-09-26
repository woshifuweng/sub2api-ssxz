#!/usr/bin/env python3
import json
import subprocess
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "ops" / "production"))
import audit_upstream_release  # noqa: E402


class ProvenanceAuditTest(unittest.TestCase):
    def test_offline_manifest_matches_merge_parents(self):
        manifest_path = ROOT / "docs" / "release-ledger" / "sub2api-v0.2.8.json"
        manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
        result = audit_upstream_release.audit(ROOT, manifest_path, offline=True)
        self.assertEqual(result["upstream_commit"], manifest["upstream_commit"])
        self.assertEqual(result["merge_commit"], manifest["merge_commit"])
        self.assertFalse(result["remote_verified"])

    def test_remote_tag_matches_official_commit(self):
        result = audit_upstream_release.audit(
            ROOT,
            ROOT / "docs" / "release-ledger" / "sub2api-v0.2.8.json",
            offline=False,
        )
        self.assertTrue(result["remote_verified"])


if __name__ == "__main__":
    unittest.main()
