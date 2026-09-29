"""Build failure tests use command doubles; no mounts or chroots are performed."""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class BuildCleanupTests(unittest.TestCase):
    def run_failure(self, fail_mount=""):
        with tempfile.TemporaryDirectory() as directory:
            base = Path(directory)
            rootfs = base / "rootfs"
            rootfs.mkdir()
            commands = base / "bin"
            commands.mkdir()
            scripts = {
                "debootstrap": 'mkdir -p "$3"/{dev,proc,run,sys,tmp,usr/sbin,etc/apt}',
                "mount": '''target="${!#}"
if [[ -n "$FAIL_MOUNT" && "$target" = */"$FAIL_MOUNT" ]]; then exit 42; fi
printf 'mount %s\n' "${target#"$TEST_ROOT"}" >> "$TEST_LOG"''',
                "umount": 'printf "umount %s\\n" "${1#"$TEST_ROOT"}" >> "$TEST_LOG"',
                "rsync": 'cp "${@: -2:1}" "${@: -1}"',
                "chroot": 'exit 42',
            }
            for name, body in scripts.items():
                path = commands / name
                path.write_text("#!/bin/bash\nset -e\n" + body + "\n")
                path.chmod(0o755)
            log = base / "log"
            result = subprocess.run(
                ["bash", str(ROOT / "liimstrap"), str(rootfs)],
                env={**os.environ, "PATH": f"{commands}:{os.environ['PATH']}",
                     "ROOT_PASSWORD": "test-only", "TEST_LOG": str(log),
                     "TEST_ROOT": str(rootfs), "FAIL_MOUNT": fail_mount},
                text=True, capture_output=True,
            )
            self.assertEqual(result.returncode, 42, result.stderr)
            return log.read_text().splitlines()

    def test_failed_chroot_unmounts_children_before_parents(self):
        events = self.run_failure()
        mounted = [line.removeprefix("mount ") for line in events if line.startswith("mount ")]
        unmounted = [line.removeprefix("umount ") for line in events if line.startswith("umount ")]
        self.assertEqual(unmounted, mounted[::-1])
        self.assertLess(unmounted.index("/dev/zero"), unmounted.index("/dev"))
        self.assertEqual(unmounted[-1], "/dev")

    def test_failed_mount_is_not_unmounted(self):
        events = self.run_failure("sys")
        self.assertNotIn("umount /sys", events)
        self.assertNotIn("mount /tmp", events)
        self.assertEqual(events[-1], "umount /dev")


if __name__ == "__main__":
    unittest.main()
