# Secure installer review scope

This document defines the review boundary for the secure installer change. It adds a POSIX-shell installer for the release asset contract established by pull request #45. It does not publish a release or advertise a one-line command before compatible binary assets exist.

## Included

- `scripts/install.sh` for macOS and Linux on AMD64 and ARM64.
- Apple Silicon detection when the installer runs under Rosetta.
- Latest-release selection and exact semantic version selection with `--version`.
- A configurable destination through `--install-dir`, defaulting to `$HOME/.local/bin`.
- A read-only `--dry-run` and command help.
- Download of the matching archive and `checksums.txt` from fixed GitHub HTTPS release URLs.
- Curl configuration isolation and HTTPS-only initial and redirected requests.
- SHA-256 verification before archive inspection or extraction.
- Exact archive-member validation for `LICENSE.txt`, `README.md`, `ubt`, `ubtctl`, and `ubtd`.
- Staged execution checks for both CLI names and the daemon before the install directory is changed.
- Same-filesystem staging, backup, replacement, and full-set rollback for the three executables.
- Offline fixture tests on Linux and macOS CI, plus POSIX shell, Bash, and ShellCheck validation.

## Excluded

- Publishing or approving a compatible GitHub Release.
- Advertising the curl command in the README before that release exists.
- The stable public installer endpoint tracked by issue #44.
- The mise and Homebrew integrations tracked by issues #42 and #43.
- Sudo, root escalation, shell-profile edits, PATH changes, daemon startup, or service installation.
- Windows and PowerShell support.
- Code signing, notarization, provenance attestations, or SBOM publication.
- Bluetooth hardware or daemon runtime qualification.

## Design references

- The [curl command-line manual](https://curl.se/docs/manpage.html) documents that `--disable` must be the first option to prevent curl from reading its default configuration, and that `--proto` and `--proto-redir` constrain initial and redirected protocols.
- [GitHub's release-link documentation](https://docs.github.com/en/repositories/releasing-projects-on-github/linking-to-releases) defines the `/releases/latest/download/<asset>` route used for latest-release installation.

## Reviewer roadmap

Review the change in this order:

1. [`scripts/install.sh`](../../scripts/install.sh) defines option parsing, platform selection, the network trust boundary, artifact validation, staged verification, and the replacement transaction.
2. [`scripts/test_install.py`](../../scripts/test_install.py) supplies network-free fixtures and verifies success, default/custom destinations, invalid versions, checksum and archive rejection, binary-version agreement, Rosetta selection, and rollback.
3. [`.github/workflows/ci.yml`](../../.github/workflows/ci.yml) runs syntax and behavior tests on Linux and macOS and ShellCheck on the Linux runner.
4. Compare the [sequence](installer/sequence.svg), [architecture](installer/architecture.svg), and [reviewer roadmap](installer/reviewer-roadmap.svg) diagrams with those files.

## Security invariants

- Production downloads use only `https://github.com/sraodev/universal-bluetooth-sdk/releases/...`.
- `curl --disable` is the first curl option, so implicit curl configuration is ignored.
- Both the initial request and every redirect are restricted to HTTPS with TLS 1.2 or newer.
- An exact version must match the release workflow's semantic tag grammar before any download.
- The selected archive must have exactly one valid SHA-256 entry, and its bytes must match that digest.
- The archive must contain exactly the five release-contract members; paths, directories, links, and extra files are rejected before extraction.
- `ubt` and `ubtctl` must execute and report the same version; an explicitly requested version must match their reported version.
- `ubtd --help` must execute successfully before the install directory is changed.
- The target directory receives all three verified binaries or the previous three paths are restored.
- The installer never invokes sudo, starts `ubtd`, or modifies shell configuration.

## Failure behavior

- Unsupported systems, architectures, versions, or arguments fail before network or filesystem mutation.
- Download, checksum, archive, or executable verification failures leave existing installed binaries untouched.
- Replacement failures trigger rollback for all three executable names, including names that did not previously exist.
- If the operating system cannot complete rollback, the installer reports the retained backup directory instead of deleting the remaining backups.
- Temporary downloads and completed transaction staging are removed on normal exit and handled signals.

## Test evidence

The offline suite replaces only `curl`, `uname`, `sysctl`, and, for fault injection, `mv`. The installer still uses the host shell, tar, checksum implementation, filesystem, and process execution behavior. Tests cover:

- exact-version and default-directory installation;
- latest-release dry-run without download or target-directory creation;
- invalid version rejection before curl;
- SHA-256 mismatch;
- unexpected, parent-path, and symbolic-link archive members;
- executable verification failure;
- mismatched CLI versions;
- backup failure before replacement, plus replacement failure after a partial install, with restoration of all previous binaries; and
- Rosetta selecting the Darwin ARM64 archive.

## Rollout gates

1. Merge this PR only after Linux and macOS checks pass for its exact head commit.
2. Publish a compatible release from the reviewed release workflow and inspect all assets.
3. Run the installer against an exact released version on native macOS ARM64, macOS AMD64, Linux ARM64, and Linux AMD64 hosts.
4. Advertise the raw GitHub fallback only after released-asset installation passes.
5. Implement and verify the stable HTTPS endpoint in issue #44 before making it the canonical command.

## Evidence limits

Offline tests prove installer control flow and rollback for ordinary command failures and handled signals without trusting public network state. They do not prove recovery after `SIGKILL`, power loss, or concurrent installer processes. They also do not prove that a compatible release currently exists, that GitHub serves the expected public bytes, that macOS accepts unsigned binaries without user action, or that installed daemon and Bluetooth hardware behavior work on every supported host.
