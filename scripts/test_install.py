import hashlib
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest


REPOSITORY_ROOT = Path(__file__).resolve().parents[1]
INSTALLER = REPOSITORY_ROOT / "scripts" / "install.sh"
ARCHIVE_NAME = "universal-bluetooth-sdk_linux_amd64.tar.gz"
BINARY_NAMES = ("ubt", "ubtctl", "ubtd")


class InstallerTests(unittest.TestCase):
    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.root = Path(self.temporary_directory.name)
        self.fixtures = self.root / "fixtures"
        self.fake_bin = self.root / "bin"
        self.install_dir = self.root / "installed"
        self.home = self.root / "home"
        self.fixtures.mkdir()
        self.fake_bin.mkdir()
        self.home.mkdir()
        self.curl_log = self.root / "curl.log"
        self._write_tool(
            "uname",
            """#!/bin/sh
if [ "$1" = "-s" ]; then
  printf '%s\n' "${TEST_UNAME_S:-Linux}"
else
  printf '%s\n' "${TEST_UNAME_M:-x86_64}"
fi
""",
        )
        self._write_tool(
            "sysctl",
            """#!/bin/sh
printf '%s\n' "${TEST_SYSCTL_TRANSLATED:-0}"
""",
        )
        self._write_tool(
            "curl",
            """#!/bin/sh
printf '%s\n' "$*" >>"$TEST_CURL_LOG"
output=""
url=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output)
      output=$2
      shift 2
      ;;
    https://*)
      url=$1
      shift
      ;;
    *)
      shift
      ;;
  esac
done
[ -n "$output" ] && [ -n "$url" ]
cp "$TEST_FIXTURE_DIR/${url##*/}" "$output"
""",
        )

    def tearDown(self):
        self.temporary_directory.cleanup()

    def _write_tool(self, name, contents):
        path = self.fake_bin / name
        path.write_text(contents, encoding="utf-8")
        path.chmod(0o755)

    def _environment(self, **overrides):
        environment = os.environ.copy()
        environment.update(
            {
                "HOME": str(self.home),
                "PATH": f"{self.fake_bin}{os.pathsep}{environment['PATH']}",
                "TEST_CURL_LOG": str(self.curl_log),
                "TEST_FIXTURE_DIR": str(self.fixtures),
                "TEST_UNAME_S": "Linux",
                "TEST_UNAME_M": "x86_64",
                "TMPDIR": str(self.root),
            }
        )
        environment.update(overrides)
        return environment

    def _create_release(
        self,
        *,
        extra_member=None,
        failing_binary=None,
        mismatched_binary=None,
        symlink_binary=None,
    ):
        archive = self.fixtures / ARCHIVE_NAME
        with tarfile.open(archive, "w:gz", format=tarfile.USTAR_FORMAT) as bundle:
            self._add_member(bundle, "README.md", b"fixture release\n", 0o644)
            self._add_member(bundle, "LICENSE.txt", b"fixture license\n", 0o644)
            for name in BINARY_NAMES:
                if name == symlink_binary:
                    self._add_symlink(bundle, name, "/bin/true")
                    continue
                if name == failing_binary:
                    script = b"#!/bin/sh\nexit 1\n"
                elif name == "ubtd":
                    script = b'#!/bin/sh\n[ "${1:-}" = "--help" ]\n'
                else:
                    reported_version = "v9.9.9" if name == mismatched_binary else "v1.2.3"
                    script = (
                        f'#!/bin/sh\n[ "${{1:-}}" = "version" ] && '
                        f'[ "${{2:-}}" = "--client-only" ]\n'
                        f'printf "%s   {reported_version}\\n" "{name}"\n'
                    ).encode()
                self._add_member(bundle, name, script, 0o755)
            if extra_member is not None:
                self._add_member(bundle, extra_member, b"unexpected\n", 0o644)

        digest = hashlib.sha256(archive.read_bytes()).hexdigest()
        (self.fixtures / "checksums.txt").write_text(
            f"{digest}  {ARCHIVE_NAME}\n", encoding="utf-8"
        )

    @staticmethod
    def _add_member(bundle, name, contents, mode):
        member = tarfile.TarInfo(name)
        member.size = len(contents)
        member.mode = mode
        member.mtime = 0
        bundle.addfile(member, io.BytesIO(contents))

    @staticmethod
    def _add_symlink(bundle, name, target):
        member = tarfile.TarInfo(name)
        member.type = tarfile.SYMTYPE
        member.linkname = target
        member.mode = 0o755
        member.mtime = 0
        bundle.addfile(member)

    def _run(self, *arguments, environment=None):
        return subprocess.run(
            ["sh", str(INSTALLER), *arguments],
            cwd=REPOSITORY_ROOT,
            env=environment or self._environment(),
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )

    def _write_previous_installation(self):
        self.install_dir.mkdir(parents=True)
        for name in BINARY_NAMES:
            path = self.install_dir / name
            path.write_text(f"old-{name}\n", encoding="utf-8")
            path.chmod(0o755)

    def _assert_previous_installation(self):
        for name in BINARY_NAMES:
            self.assertEqual(
                (self.install_dir / name).read_text(encoding="utf-8"),
                f"old-{name}\n",
            )

    def test_exact_release_installs_matching_binary_set(self):
        self._create_release()

        result = self._run(
            "--version", "v1.2.3", "--install-dir", str(self.install_dir)
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("Installed ubt, ubtctl, and ubtd in ", result.stdout)
        for name in BINARY_NAMES:
            self.assertTrue((self.install_dir / name).is_file())
        self.assertIn(
            "ubt   v1.2.3",
            subprocess.check_output(
                [self.install_dir / "ubt", "version", "--client-only"], text=True
            ),
        )
        curl_calls = self.curl_log.read_text(encoding="utf-8").splitlines()
        self.assertEqual(len(curl_calls), 2)
        self.assertTrue(all(call.startswith("--disable --proto =https") for call in curl_calls))
        self.assertTrue(all("--proto-redir =https" in call for call in curl_calls))
        self.assertTrue(all("/releases/download/v1.2.3/" in call for call in curl_calls))

    def test_default_install_directory_is_under_home(self):
        self._create_release()

        result = self._run("--version", "v1.2.3")

        self.assertEqual(result.returncode, 0, result.stderr)
        for name in BINARY_NAMES:
            self.assertTrue((self.home / ".local" / "bin" / name).is_file())

    def test_latest_release_installs_matching_binary_set(self):
        self._create_release()

        result = self._run("--install-dir", str(self.install_dir))

        self.assertEqual(result.returncode, 0, result.stderr)
        for name in BINARY_NAMES:
            self.assertTrue((self.install_dir / name).is_file())
        curl_calls = self.curl_log.read_text(encoding="utf-8").splitlines()
        self.assertEqual(len(curl_calls), 2)
        self.assertTrue(all("/releases/latest/download/" in call for call in curl_calls))

    def test_bad_checksum_preserves_previous_installation(self):
        self._create_release()
        self._write_previous_installation()
        (self.fixtures / "checksums.txt").write_text(
            f"{'0' * 64}  {ARCHIVE_NAME}\n", encoding="utf-8"
        )

        result = self._run(
            "--version", "v1.2.3", "--install-dir", str(self.install_dir)
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("SHA-256 mismatch", result.stderr)
        self._assert_previous_installation()

    def test_unexpected_archive_member_preserves_previous_installation(self):
        self._create_release(extra_member="unexpected")
        self._write_previous_installation()

        result = self._run(
            "--version", "v1.2.3", "--install-dir", str(self.install_dir)
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unexpected archive members", result.stderr)
        self._assert_previous_installation()

    def test_parent_path_archive_member_is_rejected_before_extraction(self):
        self._create_release(extra_member="../escape")

        result = self._run(
            "--version", "v1.2.3", "--install-dir", str(self.install_dir)
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unexpected archive members", result.stderr)
        self.assertFalse((self.root / "escape").exists())
        self.assertFalse(self.install_dir.exists())

    def test_symlink_binary_is_rejected_before_extraction(self):
        self._create_release(symlink_binary="ubt")

        result = self._run(
            "--version", "v1.2.3", "--install-dir", str(self.install_dir)
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("non-regular archive member", result.stderr)
        self.assertFalse(self.install_dir.exists())

    def test_binary_verification_failure_preserves_previous_installation(self):
        self._create_release(failing_binary="ubtd")
        self._write_previous_installation()

        result = self._run(
            "--version", "v1.2.3", "--install-dir", str(self.install_dir)
        )

        self.assertNotEqual(result.returncode, 0)
        self._assert_previous_installation()

    def test_mismatched_cli_versions_preserve_previous_installation(self):
        self._create_release(mismatched_binary="ubtctl")
        self._write_previous_installation()

        result = self._run(
            "--version", "v1.2.3", "--install-dir", str(self.install_dir)
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ubt and ubtctl report different versions", result.stderr)
        self._assert_previous_installation()

    def test_replacement_failure_rolls_back_all_binaries(self):
        self._create_release()
        self._write_previous_installation()
        failure_marker = self.root / "mv-failed"
        self._write_tool(
            "mv",
            """#!/bin/sh
case "$1:$2" in
  */new-ubtctl:*/ubtctl)
    if [ ! -e "$TEST_MV_FAILURE_MARKER" ]; then
      : >"$TEST_MV_FAILURE_MARKER"
      exit 1
    fi
    ;;
esac
exec /bin/mv "$@"
""",
        )
        environment = self._environment(TEST_MV_FAILURE_MARKER=str(failure_marker))

        result = self._run(
            "--version",
            "v1.2.3",
            "--install-dir",
            str(self.install_dir),
            environment=environment,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(failure_marker.exists())
        self._assert_previous_installation()

    def test_backup_failure_preserves_unprocessed_existing_binaries(self):
        self._create_release()
        self._write_previous_installation()
        failure_marker = self.root / "mv-failed"
        self._write_tool(
            "mv",
            """#!/bin/sh
case "$1:$2" in
  */installed/ubtctl:*/backup-ubtctl)
    if [ ! -e "$TEST_MV_FAILURE_MARKER" ]; then
      : >"$TEST_MV_FAILURE_MARKER"
      exit 1
    fi
    ;;
esac
exec /bin/mv "$@"
""",
        )
        environment = self._environment(TEST_MV_FAILURE_MARKER=str(failure_marker))

        result = self._run(
            "--version",
            "v1.2.3",
            "--install-dir",
            str(self.install_dir),
            environment=environment,
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(failure_marker.exists())
        self._assert_previous_installation()

    def test_invalid_version_fails_before_download(self):
        result = self._run(
            "--version", "latest;echo unsafe", "--install-dir", str(self.install_dir)
        )

        self.assertNotEqual(result.returncode, 0)
        self.assertIn("invalid version", result.stderr)
        self.assertFalse(self.curl_log.exists())
        self.assertFalse(self.install_dir.exists())

    def test_dry_run_uses_latest_release_without_downloading(self):
        result = self._run("--dry-run", "--install-dir", str(self.install_dir))

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("/releases/latest/download/", result.stdout)
        self.assertIn(ARCHIVE_NAME, result.stdout)
        self.assertIn("No files changed.", result.stdout)
        self.assertFalse(self.curl_log.exists())
        self.assertFalse(self.install_dir.exists())

    def test_rosetta_selects_apple_silicon_archive(self):
        environment = self._environment(
            TEST_UNAME_S="Darwin",
            TEST_UNAME_M="x86_64",
            TEST_SYSCTL_TRANSLATED="1",
        )

        result = self._run(
            "--dry-run",
            "--install-dir",
            str(self.install_dir),
            environment=environment,
        )

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("universal-bluetooth-sdk_darwin_arm64.tar.gz", result.stdout)


if __name__ == "__main__":
    unittest.main()
