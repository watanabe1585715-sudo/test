# 求人広告サイト群（モノレポ）

[実装計画書.md](実装計画書.md) と [test](test) に基づく実装です。主要な Go コードには日本語コメントを付与しています。

**図・API 一覧・各システムの概要（言語・技術スタック）**: [docs/資料.md](docs/資料.md)（開いて **Ctrl+Shift+V** でプレビュー。図が出ない場合は拡張 `bierner.markdown-mermaid` をインストール）

## 構成

- **Go API**（`cmd/api`）: REST `/public/*`, `/admin/jobs/*`, `/admin/customers/*`（**Gin**、レイヤは **domain / usecase / infrastructure / interfaces** の DDD 風構成）
- **Go バッチ**（`cmd/batch`）: 掲載期間・契約枠に応じて `publication_status` を更新
- **PostgreSQL**: [docker-compose.yml](docker-compose.yml)
- **frontend-react**（5173）: 求人サイト（**TanStack Query**）
- **frontend-vue**（5174）: 案件管理サイト（**Pinia** + **TanStack Vue Query**）
- **frontend-next**（3000）: 顧客管理サイト（**TanStack Query** + DevTools）。**お知らせ**の登録・更新・削除、**顧客管理ログインアカウント**の一覧・作成は当サイトから。

## すぐ動かす（推奨: Docker + Makefile）

1. **Docker** が使えること（WSL2 では Docker Desktop の **Settings → Resources → WSL integration** で使用中のディストリビューションをオンにすると、`docker` がその WSL から使えます）。

2. DB と API を起動:

```bash
cd /path/to/watanabe_test
make up
# 同等: ./scripts/up.sh
```

### Docker で `permission denied`（`/var/run/docker.sock`）のとき

Linux / WSL でよくある「現在のユーザーが `docker` グループに入っていない」状態です。**推奨**は次のどちらかです。

**A. ユーザーを `docker` グループに追加（以降は `sudo` 不要）**

```bash
sudo usermod -aG docker "$USER"
# いったん WSL / ターミナルを終了して再ログイン（または newgrp docker）
newgrp docker
make up
```

**B. その場だけ `sudo` で Docker を叩く**

```bash
make DOCKER="sudo docker" up
make DOCKER="sudo docker" batch
# または
DOCKER="sudo docker" ./scripts/up.sh
DOCKER="sudo docker" ./scripts/batch.sh
```

Makefile では先頭の `DOCKER ?= docker` を上書きしているだけです。

3. 動作確認:

```bash
make health
# または: curl http://localhost:8080/health
```

4. **掲載バッチ**（シードのデモ求人を公開サイトに出すために必須）:

```bash
make batch
# 同等: ./scripts/batch.sh
# または: docker compose --profile tools run --rm batch
```

5. **メールワーカー**（`email_outbox` の pending を SMTP で送信。`.env.example` の `SMTP_HOST` 等を参照）:

```bash
make mailworker
# または: docker compose --profile tools run --rm mailworker
```

6. **フロント**（Node 20+ / npm が必要）:

```bash
cd frontend-react && npm install && npm run dev
cd frontend-vue   && npm install && npm run dev
cd frontend-next  && npm install && npm run dev
```

API を別ホストにする場合は `VITE_API_ORIGIN` / `NEXT_PUBLIC_API_ORIGIN` を設定してください。

## DB だけ Docker、API はホストの Go で動かす

Docker の API（ポート 8080）とホストの API を同時に起動しないでください。

```bash
make db
```

システムに Go が無い場合、`scripts/run-api-local.sh` が [scripts/bootstrap-go.sh](scripts/bootstrap-go.sh) で **リポジトリ内 `.tools/go`** に公式 Go を展開します。

```bash
./scripts/run-api-local.sh
```

別ターミナルでバッチ:

```bash
./scripts/run-batch-local.sh
```

`Makefile` の `make api-local` / `make batch-local` / `make mailworker-local` でも同じです。

接続文字列の既定は `postgres://recruit:recruit@127.0.0.1:5432/recruit?sslmode=disable` です。

## 従来どおりホストに Go がある場合

```bash
cp .env.example .env
# .env の JWT_SECRET を変更
export $(grep -v '^#' .env | xargs)
go run ./cmd/api
go run ./cmd/batch
```

## 初期ログイン（シード）

| サイト   | メール               | パスワード | 備考 |
| -------- | -------------------- | ---------- | ---- |
| 案件管理 | jobadmin@example.com | password   | 顧客「デモ広告主株式会社」 |
| 案件管理 | jobadmin2@example.com | password  | 顧客「テスト商事有限会社」（tier 2） |
| 顧客管理 | admin@example.com    | password   | 全顧客を操作可 |

シードには **顧客2社・複数案件・応募3件・請求・見込み増量** などが含まれます（`002_seed.sql` + `003_seed_demo.sql`）。  
求人サイトで **すぐ一覧に出る** `published` 案件も入っています。別案件は **バッチ実行後** に `published` 化されることがあります。

**DB を GUI で見る・A5:SQL Mk-2**: [docs/データベース接続.md](docs/データベース接続.md)

## API 概要

- `GET /public/jobs?q=` … 公開中求人一覧
- `GET /public/jobs/{id}` … 詳細
- `POST /public/applications` … 応募
- `POST /admin/jobs/login` … 案件管理 JWT
- `GET/POST/PATCH/DELETE /admin/jobs/jobs...` … 案件 CRUD（要 Bearer）
- `POST /admin/customers/login` … 顧客管理 JWT
- `GET/POST/PATCH /admin/customers/customers...` … 顧客
- `POST /admin/customers/customers/{id}/end-contract` … 契約終了
- `GET/POST /admin/customers/customers/{id}/job-users` … 案件管理ユーザー
- `PATCH/DELETE /admin/customers/job-users/{id}?customer_id=` … 更新・削除
- `GET/POST /admin/customers/invoices` … 請求書
- `GET /admin/customers/prospects` … 見込み一覧

## ライセンス

デモ・学習用。本番ではシークレット・HTTPS・権限設計を見直してください。
