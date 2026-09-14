#!/usr/bin/env python3
"""Exercise the release target guard without depending on a host SDK."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest


class MacOSTargetTests(unittest.TestCase):
    def check_target(self, output, goos="darwin"):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "output").write_text(output)
            tool = root / "otool"
            tool.write_text('#!/bin/sh\ncat "$TARGET_FIXTURE"\n')
            tool.chmod(0o755)
            return subprocess.run(
                ["./scripts/check-macos-target", "fixture", goos],
                env={**os.environ, "PATH": f"{root}:{os.environ['PATH']}",
                     "TARGET_FIXTURE": str(root / "output")},
                capture_output=True, text=True,
            ).returncode

    def test_exact_minimum(self):
        self.assertEqual(self.check_target("cmd LC_BUILD_VERSION\ncmdsize 24\nplatform 1\nminos 13.0\nsdk 26.2\n"), 0)

    def test_legacy_minimum(self):
        self.assertEqual(self.check_target("cmd LC_VERSION_MIN_MACOSX\ncmdsize 16\nversion 13.0.0\nsdk 26.2\n"), 0)

    def test_rejects_missing_or_changed_minimum(self):
        for version in ("12.0", "14.0", "invalid"):
            with self.subTest(version=version):
                self.assertNotEqual(self.check_target(f"cmd LC_BUILD_VERSION\nminos {version}\n"), 0)
        self.assertNotEqual(self.check_target(""), 0)

    def test_rejects_mixed_slices(self):
        self.assertNotEqual(self.check_target("cmd LC_BUILD_VERSION\nminos 13.0\nLoad command 1\ncmd LC_BUILD_VERSION\nminos 14.0\n"), 0)

    def test_linux_does_not_need_macho(self):
        self.assertEqual(self.check_target("", "linux"), 0)


if __name__ == "__main__":
    unittest.main()
