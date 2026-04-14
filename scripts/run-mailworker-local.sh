#!/usr/bin/env bash
# ホストでメールワーカーを 1 回実行（pending → SMTP 送信）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1; then
  bash "$ROOT/scripts/bootstrap-go.sh"
  export PATH="$ROOT/.tools/go/bin:$PATH"
fi

export DATABASE_URL="${DATABASE_URL:-postgres://recruit:recruit@127.0.0.1:5432/recruit?sslmode=disable}"
exec go run ./cmd/mailworker
