#!/usr/bin/env bash
# システムに Go が無い場合、プロジェクト配下 .tools/go に公式バイナリを展開する。
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEST="$ROOT/.tools/go"
GO_VERSION="${GO_VERSION:-1.22.10}"

if [ -x "$DEST/bin/go" ]; then
  echo "Go は既に $DEST にあります。"
  exit 0
fi

ARCH=linux-amd64
URL="https://go.dev/dl/go${GO_VERSION}.${ARCH}.tar.gz"
TMP="$(mktemp -d)"
echo "==> Download $URL"
curl -fsSL "$URL" -o "$TMP/go.tgz"
mkdir -p "$ROOT/.tools"
tar -C "$ROOT/.tools" -xzf "$TMP/go.tgz"
rm -rf "$TMP"
# アーカイブ展開後のディレクトリ名は常に go
if [ -d "$ROOT/.tools/go" ]; then
  echo "Go を $DEST に配置しました。"
else
  echo "展開に失敗しました: $ROOT/.tools" >&2
  exit 1
fi
