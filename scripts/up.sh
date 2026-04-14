#!/usr/bin/env bash
# DB と API を Docker で起動する（リポジトリルートから実行）
# permission denied のときは: DOCKER="sudo docker" ./scripts/up.sh
set -euo pipefail
cd "$(dirname "$0")/.."

DOCKER="${DOCKER:-docker}"

if ! command -v "${DOCKER%% *}" >/dev/null 2>&1; then
  echo "docker が見つかりません。Docker Desktop 等をインストールしてください。" >&2
  exit 1
fi

echo "==> ${DOCKER} compose up (db, api)"
$DOCKER compose up -d --build db api

echo ""
echo "API: http://localhost:8080/health"
echo "掲載バッチ（求人サイトに案件を出す）:"
echo "  DOCKER=\"sudo docker\" ./scripts/batch.sh"
echo "  または: make DOCKER=\"sudo docker\" batch"
echo ""
echo "フロント（ホストに Node が必要）:"
echo "  cd frontend-react && npm i && npm run dev"
echo "  cd frontend-vue   && npm i && npm run dev"
echo "  cd frontend-next  && npm i && npm run dev"
