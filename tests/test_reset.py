"""Exercise fail-closed idle selection without contacting the host logind."""
import os
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[1]


def run_shell(body):
    return subprocess.run(
        ["bash", "-c", 'source "$RESET_SCRIPT"\n' + body],
        env={**os.environ, "RESET_SCRIPT": str(ROOT / "bin/liims-reset.sh")},
        text=True, capture_output=True,
    )


class ResetTests(unittest.TestCase):
    def selection(self, rows):
        # logind properties may arrive in any order.
        entries = "\n".join(
            f"{session}) printf '%s\\n' 'Seat={seat}' 'Remote={remote}' "
            f"'Service={service}' 'Name={name}' ;;"
            for session, name, service, remote, seat in rows
        )
        return run_shell("""
loginctl() {
  if [[ $1 = list-sessions ]]; then
    printf '%s\n' LIST
  else
    case "$2" in
      ENTRIES
      *) return 1 ;;
    esac
  fi
}
find_session
""".replace("LIST", " ".join("'" + r[0] + " 1000 liims'" for r in rows))
             .replace("ENTRIES", entries))

    def test_ssh_does_not_supply_idle_state(self):
        result = self.selection([
            ("c1", "liims", "sshd", "yes", ""),
            ("c2", "liims", "greetd", "no", "seat0"),
            ("c3", "root", "greetd", "no", "seat0"),
        ])
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), "c2")

    def test_ambiguous_or_missing_desktop_fails(self):
        for rows in [[], [("c1", "liims", "sshd", "yes", "")], [
            ("c1", "liims", "greetd", "no", "seat0"),
            ("c2", "liims", "greetd", "no", "seat0"),
        ]]:
            self.assertNotEqual(self.selection(rows).returncode, 0)

    def test_unavailable_monitor_does_not_reset(self):
        result = run_shell("""
find_session() { echo c1; }
check_monitor() { return 1; }
loginctl() { echo yes; }
wait_for_idle
""")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("idle monitor unavailable", result.stderr)

    def test_invalid_idle_state_does_not_reset(self):
        result = run_shell("""
find_session() { echo c1; }
check_monitor() { return 0; }
loginctl() { echo unknown; }
wait_for_idle
""")
        self.assertNotEqual(result.returncode, 0)

    def test_active_user_is_checked_again(self):
        result = run_shell("""
find_session() { echo c1; }
check_monitor() { return 0; }
loginctl() { if [[ ${resumed:-0} = 0 ]]; then echo no; else echo yes; fi; }
sleep() { resumed=1; }
wait_for_idle
""")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.strip(), "c1")

    def test_session_change_cancels_pending_reset(self):
        result = run_shell("""
find_session() { if [[ ${resumed:-0} = 0 ]]; then echo c1; else echo c2; fi; }
check_monitor() { return 0; }
loginctl() { echo no; }
sleep() { resumed=1; }
wait_for_idle
""")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("session changed", result.stderr)


if __name__ == "__main__":
    unittest.main()
