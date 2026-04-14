#!/usr/bin/env bash
# ホストで API を起動（DB は Docker の postgres を想定: make up / scripts/up.sh）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if ! command -v go >/dev/null 2>&1; then
  echo "Go が見つかりません。bootstrap します..."
  bash "$ROOT/scripts/bootstrap-go.sh"
  export PATH="$ROOT/.tools/go/bin:$PATH"
fi

export DATABASE_URL="${DATABASE_URL:-postgres://recruit:recruit@127.0.0.1:5432/recruit?sslmode=disable}"
export JWT_SECRET="${JWT_SECRET:-dev-insecure-change-me}"
export API_ADDR="${API_ADDR:-:8080}"
export CORS_ORIGINS="${CORS_ORIGINS:-http://localhost:5173,http://localhost:5174,http://localhost:3000}"

echo "DATABASE_URL=$DATABASE_URL"
echo "==> go run ./cmd/api"
exec go run ./cmd/api
