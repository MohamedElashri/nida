#!/bin/sh
set -eu

repo="MohamedElashri/nida"
version="${NIDA_VERSION:-latest}"
system_install="${NIDA_INSTALL_SYSTEM:-0}"

default_install_dir() {
  if [ "$system_install" = "1" ] || [ "$system_install" = "true" ]; then
    printf '%s\n' "/usr/local/bin"
    return
  fi
  if [ -n "${XDG_BIN_HOME:-}" ]; then
    printf '%s\n' "$XDG_BIN_HOME"
    return
  fi
  if [ -n "${HOME:-}" ]; then
    printf '%s\n' "$HOME/.local/bin"
    return
  fi
  echo "nida install: HOME is not set; set NIDA_INSTALL_DIR to a writable directory" >&2
  exit 1
}

install_dir="${NIDA_INSTALL_DIR:-$(default_install_dir)}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "nida install: missing required command: $1" >&2
    exit 1
  fi
}

need curl
need sed
need tar
need uname
need awk
need install

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *)
    echo "nida install: unsupported operating system: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) arch="x86_64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "nida install: unsupported CPU architecture: $arch" >&2
    exit 1
    ;;
esac

if [ "$version" = "latest" ]; then
  version="$(curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" | sed -n 's/.*"tag_name":[[:space:]]*"\(v[^"]*\)".*/\1/p' | head -n 1)"
  if [ -z "$version" ]; then
    echo "nida install: could not resolve latest release" >&2
    exit 1
  fi
fi

case "$version" in
  v*) tag="$version" ;;
  *) tag="v$version" ;;
esac

plain_version="${tag#v}"
archive="nida_${plain_version}_${os}_${arch}.tar.gz"
base_url="https://github.com/${repo}/releases/download/${tag}"
tmp="${TMPDIR:-/tmp}/nida-install.$$"

cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT INT TERM

mkdir -p "$tmp/extract"

echo "nida install: downloading ${archive}"
curl -fL "$base_url/$archive" -o "$tmp/$archive"

echo "nida install: verifying checksum"
curl -fsSL "$base_url/checksums.txt" -o "$tmp/checksums.txt"
expected="$(awk -v archive="$archive" '$2 == archive { print $1 }' "$tmp/checksums.txt")"
if [ -z "$expected" ]; then
  echo "nida install: checksum for ${archive} not found" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$tmp" && printf '%s  %s\n' "$expected" "$archive" | sha256sum -c - >/dev/null)
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$tmp/$archive" | awk '{ print $1 }')"
  if [ "$actual" != "$expected" ]; then
    echo "nida install: checksum mismatch" >&2
    exit 1
  fi
else
  echo "nida install: sha256sum or shasum not found; skipping checksum verification" >&2
fi

tar -xzf "$tmp/$archive" -C "$tmp/extract"
if [ ! -f "$tmp/extract/nida" ]; then
  echo "nida install: archive did not contain a nida binary" >&2
  exit 1
fi
chmod +x "$tmp/extract/nida"

if [ "$system_install" = "1" ] || [ "$system_install" = "true" ]; then
  if [ -w "$install_dir" ]; then
    mkdir -p "$install_dir"
    install -m 0755 "$tmp/extract/nida" "$install_dir/nida"
  elif command -v sudo >/dev/null 2>&1; then
    sudo mkdir -p "$install_dir"
    sudo install -m 0755 "$tmp/extract/nida" "$install_dir/nida"
  else
    echo "nida install: ${install_dir} is not writable and sudo is not available" >&2
    exit 1
  fi
else
  mkdir -p "$install_dir"
  install -m 0755 "$tmp/extract/nida" "$install_dir/nida"
fi

echo "nida install: installed to ${install_dir}/nida"
case ":${PATH:-}:" in
  *:"$install_dir":*) ;;
  *)
    echo "nida install: add ${install_dir} to PATH to run nida from any shell" >&2
    ;;
esac
"$install_dir/nida" version
