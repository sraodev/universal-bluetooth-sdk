#!/bin/sh

set -eu

project_url="https://github.com/sraodev/universal-bluetooth-sdk"
version=""
install_dir=""
dry_run=0
download_dir=""
target_stage=""
rollback_active=0

usage() {
  cat <<'EOF'
Install Universal Bluetooth SDK release binaries.

Usage:
  install.sh [--version vX.Y.Z] [--install-dir DIR] [--dry-run]
  install.sh --help

Options:
  --version VERSION  Install an exact release tag. Defaults to the latest release.
  --install-dir DIR  Install into DIR. Defaults to $HOME/.local/bin.
  --dry-run          Print the selected release and destination without changing files.
  -h, --help         Show this help.

The installer downloads ubt, ubtctl, and ubtd as one matching set. It never
uses sudo, starts the daemon, or edits shell profiles.
EOF
}

fail() {
  printf 'install.sh: %s\n' "$*" >&2
  exit 1
}

is_release_version() {
  case "$1" in
    *'
'*) return 1 ;;
  esac
  printf '%s\n' "$1" | LC_ALL=C grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-(alpha|beta|rc)\.(0|[1-9][0-9]*))?$'
}

restore_previous_installation() {
  restore_failed=0
  for name in ubt ubtctl ubtd; do
    destination="$install_dir/$name"
    if [ -f "$target_stage/had-$name" ]; then
      if [ -e "$target_stage/backup-$name" ] || [ -L "$target_stage/backup-$name" ]; then
        if ! rm -f "$destination"; then
          restore_failed=1
          continue
        fi
        if ! mv "$target_stage/backup-$name" "$destination"; then
          restore_failed=1
        fi
      fi
    else
      if ! rm -f "$destination"; then
        restore_failed=1
      fi
    fi
  done
  [ "$restore_failed" -eq 0 ]
}

cleanup() {
  status=$?
  trap - 0 HUP INT TERM

  keep_target_stage=0
  if [ "$rollback_active" -eq 1 ] && [ -n "$target_stage" ]; then
    if ! restore_previous_installation; then
      keep_target_stage=1
      printf 'install.sh: rollback incomplete; backups remain in %s\n' "$target_stage" >&2
    fi
  fi

  if [ -n "$download_dir" ]; then
    rm -rf "$download_dir" || true
  fi
  if [ -n "$target_stage" ] && [ "$keep_target_stage" -eq 0 ]; then
    rm -rf "$target_stage" || true
  fi
  exit "$status"
}

trap cleanup 0
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || fail "--version requires a value"
      version=$2
      shift 2
      ;;
    --install-dir)
      [ "$#" -ge 2 ] || fail "--install-dir requires a value"
      install_dir=$2
      shift 2
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    --)
      shift
      [ "$#" -eq 0 ] || fail "unexpected positional arguments: $*"
      break
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

if [ -n "$version" ]; then
  if ! is_release_version "$version"; then
    fail "invalid version: $version"
  fi
fi

if [ -z "$install_dir" ]; then
  [ -n "${HOME:-}" ] || fail "HOME is not set; pass --install-dir"
  install_dir="$HOME/.local/bin"
fi
[ -n "$install_dir" ] || fail "install directory must not be empty"

system_name=$(uname -s)
machine_name=$(uname -m)

case "$system_name" in
  Darwin)
    os=darwin
    if [ "$machine_name" = "x86_64" ] && command -v sysctl >/dev/null 2>&1; then
      translated=$(sysctl -in sysctl.proc_translated 2>/dev/null || true)
      if [ "$translated" = "1" ]; then
        machine_name=arm64
      fi
    fi
    ;;
  Linux)
    os=linux
    ;;
  *)
    fail "unsupported operating system: $system_name"
    ;;
esac

case "$machine_name" in
  x86_64 | amd64)
    arch=amd64
    ;;
  arm64 | aarch64)
    arch=arm64
    ;;
  *)
    fail "unsupported architecture: $machine_name"
    ;;
esac

archive_name="universal-bluetooth-sdk_${os}_${arch}.tar.gz"
if [ -n "$version" ]; then
  release_url="$project_url/releases/download/$version"
  release_label=$version
else
  release_url="$project_url/releases/latest/download"
  release_label=latest
fi
archive_url="$release_url/$archive_name"
checksums_url="$release_url/checksums.txt"

if [ "$dry_run" -eq 1 ]; then
  printf 'Release: %s\n' "$release_label"
  printf 'Archive: %s\n' "$archive_url"
  printf 'Checksums: %s\n' "$checksums_url"
  printf 'Install directory: %s\n' "$install_dir"
  printf 'No files changed.\n'
  exit 0
fi

for command_name in curl tar mktemp awk sort grep chmod mkdir cp mv rm; do
  command -v "$command_name" >/dev/null 2>&1 || fail "required command not found: $command_name"
done

download_dir=$(mktemp -d "${TMPDIR:-/tmp}/ubt-install.XXXXXX") || fail "create temporary directory"
archive_path="$download_dir/$archive_name"
checksums_path="$download_dir/checksums.txt"

if ! curl --disable --proto '=https' --proto-redir '=https' --tlsv1.2 --fail --silent --show-error --location --output "$archive_path" "$archive_url"; then
  fail "download $archive_url"
fi
if ! curl --disable --proto '=https' --proto-redir '=https' --tlsv1.2 --fail --silent --show-error --location --output "$checksums_path" "$checksums_url"; then
  fail "download $checksums_url"
fi

checksum_matches=$(awk -v name="$archive_name" '$2 == name { count++ } END { print count + 0 }' "$checksums_path")
[ "$checksum_matches" -eq 1 ] || fail "checksums.txt must contain exactly one entry for $archive_name"
expected_checksum=$(awk -v name="$archive_name" '$2 == name { print $1 }' "$checksums_path")
[ "${#expected_checksum}" -eq 64 ] || fail "invalid SHA-256 for $archive_name"
case "$expected_checksum" in
  *[!0123456789abcdefABCDEF]*) fail "invalid SHA-256 for $archive_name" ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
  actual_checksum=$(sha256sum "$archive_path" | awk '{ print $1 }')
elif command -v shasum >/dev/null 2>&1; then
  actual_checksum=$(shasum -a 256 "$archive_path" | awk '{ print $1 }')
else
  fail "required SHA-256 tool not found: install sha256sum or shasum"
fi

expected_checksum=$(printf '%s\n' "$expected_checksum" | awk '{ print tolower($0) }')
[ "$actual_checksum" = "$expected_checksum" ] || fail "SHA-256 mismatch for $archive_name"

members_path="$download_dir/archive-members.txt"
if ! tar -tzf "$archive_path" >"$members_path"; then
  fail "cannot read $archive_name"
fi
actual_members=$(LC_ALL=C sort "$members_path")
expected_members='LICENSE.txt
README.md
ubt
ubtctl
ubtd'
[ "$actual_members" = "$expected_members" ] || fail "unexpected archive members in $archive_name"

member_details_path="$download_dir/archive-member-details.txt"
if ! LC_ALL=C tar -tvzf "$archive_path" >"$member_details_path"; then
  fail "cannot inspect $archive_name"
fi
if ! awk 'substr($1, 1, 1) != "-" { exit 1 }' "$member_details_path"; then
  fail "non-regular archive member in $archive_name"
fi

extracted_dir="$download_dir/extracted"
mkdir "$extracted_dir"
if ! tar -xzf "$archive_path" -C "$extracted_dir"; then
  fail "cannot extract $archive_name"
fi

for name in ubt ubtctl ubtd; do
  [ -f "$extracted_dir/$name" ] || fail "archive is missing $name"
  chmod 755 "$extracted_dir/$name"
done

if ! ubt_version_output=$("$extracted_dir/ubt" version --client-only 2>/dev/null); then
  fail "ubt verification failed"
fi
if ! ubtctl_version_output=$("$extracted_dir/ubtctl" version --client-only 2>/dev/null); then
  fail "ubtctl verification failed"
fi
if ! "$extracted_dir/ubtd" --help >/dev/null 2>&1; then
  fail "ubtd verification failed"
fi

ubt_version=$(printf '%s\n' "$ubt_version_output" | awk 'NR == 1 && $1 == "ubt" { print $2 }')
ubtctl_version=$(printf '%s\n' "$ubtctl_version_output" | awk 'NR == 1 && $1 == "ubtctl" { print $2 }')
[ -n "$ubt_version" ] || fail "ubt did not report a version"
[ "$ubt_version" = "$ubtctl_version" ] || fail "ubt and ubtctl report different versions"
is_release_version "$ubt_version" || fail "release binaries report an invalid version: $ubt_version"
if [ -n "$version" ] && [ "$ubt_version" != "$version" ]; then
  fail "release binaries report $ubt_version, expected $version"
fi

mkdir -p "$install_dir"
install_dir=$(cd "$install_dir" && pwd -P) || fail "resolve install directory"
target_stage=$(mktemp -d "$install_dir/.ubt-install.XXXXXX") || fail "create installation staging directory"

for name in ubt ubtctl ubtd; do
  destination="$install_dir/$name"
  if { [ -e "$destination" ] || [ -L "$destination" ]; } && [ ! -f "$destination" ]; then
    fail "refusing to replace non-file path: $destination"
  fi
  if [ -e "$destination" ] || [ -L "$destination" ]; then
    : >"$target_stage/had-$name"
  fi
  cp "$extracted_dir/$name" "$target_stage/new-$name"
  chmod 755 "$target_stage/new-$name"
done

rollback_active=1
for name in ubt ubtctl ubtd; do
  destination="$install_dir/$name"
  if [ -f "$target_stage/had-$name" ]; then
    mv "$destination" "$target_stage/backup-$name"
  fi
done

for name in ubt ubtctl ubtd; do
  mv "$target_stage/new-$name" "$install_dir/$name"
done
rollback_active=0

printf 'Installed ubt, ubtctl, and ubtd in %s\n' "$install_dir"
