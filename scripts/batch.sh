#!/usr/bin/env bash
# 掲載バッチを 1 回 Docker で実行
# permission denied のときは: DOCKER="sudo docker" ./scripts/batch.sh
set -euo pipefail
cd "$(dirname "$0")/.."
DOCKER="${DOCKER:-docker}"
$DOCKER compose --profile tools run --rm batch
