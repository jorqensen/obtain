#!/bin/sh
# Install obtain from GitHub releases into ~/.local/bin (Linux and macOS).
#
#   curl -fsSL https://raw.githubusercontent.com/jorqensen/obtain/main/install.sh | sh
#
# Environment:
#   OBTAIN_VERSION      version to install, e.g. v0.1.0 (default: latest)
#   OBTAIN_INSTALL_DIR  install directory (default: ~/.local/bin)
set -eu

repo="jorqensen/obtain"
dir="${OBTAIN_INSTALL_DIR:-$HOME/.local/bin}"

err() {
  echo "install: $*" >&2
  exit 1
}

command -v curl >/dev/null 2>&1 || err "curl is required"
command -v tar >/dev/null 2>&1 || err "tar is required"

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) err "unsupported OS: $(uname -s) (Windows users: download the zip from https://github.com/$repo/releases)" ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  aarch64 | arm64) arch=arm64 ;;
  *) err "unsupported architecture: $(uname -m)" ;;
esac

tag="${OBTAIN_VERSION:-}"
if [ -z "$tag" ]; then
  # /releases/latest redirects to /releases/tag/<tag>.
  url=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$repo/releases/latest") ||
    err "could not determine the latest version"
  tag="${url##*/}"
fi
case "$tag" in
  v*) ;;
  *) tag="v$tag" ;;
esac
version="${tag#v}"

archive="obtain_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/$tag"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "Downloading obtain $tag ($os/$arch)..."
curl -fsSL -o "$tmp/$archive" "$base/$archive" || err "download failed: $base/$archive"

# Verify the checksum when a sha256 tool is available.
if curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt"; then
  want=$(grep " $archive\$" "$tmp/checksums.txt" | cut -d ' ' -f 1)
  if command -v sha256sum >/dev/null 2>&1; then
    got=$(sha256sum "$tmp/$archive" | cut -d ' ' -f 1)
  elif command -v shasum >/dev/null 2>&1; then
    got=$(shasum -a 256 "$tmp/$archive" | cut -d ' ' -f 1)
  else
    got=""
    echo "install: no sha256sum or shasum found, skipping checksum verification" >&2
  fi
  if [ -n "$got" ] && [ "$got" != "$want" ]; then
    err "checksum mismatch for $archive"
  fi
else
  echo "install: could not fetch checksums.txt, skipping verification" >&2
fi

tar -xzf "$tmp/$archive" -C "$tmp" obtain || err "could not extract obtain from $archive"

mkdir -p "$dir"
mv "$tmp/obtain" "$dir/obtain"
chmod 755 "$dir/obtain"

echo "Installed obtain $tag to $dir/obtain"

case ":$PATH:" in
  *":$dir:"*) ;;
  *) echo "Note: $dir is not on your PATH. Add it, e.g. in your shell profile:  export PATH=\"$dir:\$PATH\"" ;;
esac
