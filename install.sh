#!/bin/sh
# Install hp from GitHub Releases.
set -eu

REPO="chenhg5/harness-projects"
DIR="${HP_INSTALL_DIR:-$HOME/.local/bin}"
version="${HP_VERSION:-}"
command -v curl >/dev/null 2>&1 || { echo "error: curl is required" >&2; exit 1; }
command -v tar >/dev/null 2>&1 || { echo "error: tar is required" >&2; exit 1; }
case "$(uname -s)" in Darwin) os=darwin ;; Linux) os=linux ;; *) echo "error: unsupported OS" >&2; exit 1 ;; esac
case "$(uname -m)" in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; *) echo "error: unsupported architecture" >&2; exit 1 ;; esac
if [ -z "$version" ]; then
  version=$(curl -fsSL "https://api.github.com/repos/$REPO/releases?per_page=1" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1) || true
fi
[ -n "$version" ] || { echo "error: set HP_VERSION=v0.0.6 when the release API is unavailable" >&2; exit 1; }
name="hp-$version-$os-$arch"
base="https://github.com/$REPO/releases/download/$version"
tmp=$(mktemp -d); trap 'rm -rf "$tmp"' EXIT
curl -fsSL -o "$tmp/$name.tar.gz" "$base/$name.tar.gz"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt"
expected=$(grep " $name.tar.gz$" "$tmp/checksums.txt" | cut -d' ' -f1)
[ -n "$expected" ] || { echo "error: checksum missing" >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$tmp/$name.tar.gz" | cut -d' ' -f1); else actual=$(shasum -a 256 "$tmp/$name.tar.gz" | cut -d' ' -f1); fi
[ "$expected" = "$actual" ] || { echo "error: checksum mismatch" >&2; exit 1; }
tar -xzf "$tmp/$name.tar.gz" -C "$tmp"
mkdir -p "$DIR"
install -m 0755 "$tmp/$name/hp" "$DIR/hp"
echo "Installed hp $version to $DIR/hp"
