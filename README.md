# 求人広告サイト群（モノレポ）

[実装計画書.md](実装計画書.md) と [test](test) をもとに実装したリポジトリです。主要な Go コードには日本語コメントを付与しています。

## まず読む資料

- 全体像（構成図・フロー・API 一覧）: [docs/資料.md](docs/資料.md)
- 画面一覧: [docs/画面一覧.md](docs/画面一覧.md)
- スクリーンショット運用: [docs/screenshots/README.md](docs/screenshots/README.md)
- DB 接続手順: [docs/データベース接続.md](docs/データベース接続.md)

`docs/資料.md` の Mermaid 図を表示するには、Markdown プレビュー（`Ctrl+Shift+V`）を使用してください。図が表示されない場合は拡張機能 `bierner.markdown-mermaid` をインストールしてください。

## 構成

- **Go API**（`cmd/api`）: REST `/public/*`, `/admin/jobs/*`, `/admin/customers/*`（**Gin**、レイヤは **domain / usecase / infrastructure / interfaces** の DDD 風構成）
- **Go バッチ**（`cmd/batch`）: 掲載期間・契約枠に応じて `publication_status` を更新
- **PostgreSQL**: [docker-compose.yml](docker-compose.yml)
- **frontend-react**（5173）: 求人サイト（**TanStack Query**）
- **frontend-vue**（5174）: 案件管理サイト（**Pinia** + **TanStack Vue Query**）
- **frontend-next**（3000）: 顧客管理サイト（**TanStack Query** + DevTools）。**お知らせ**の登録・更新・削除、**顧客管理ログインアカウント**の一覧・作成は当サイトから。

## すぐ動かす（推奨: Docker + Makefile）

1. **Docker** が使えることを確認します。  
   WSL2 では Docker Desktop の **Settings → Resources → WSL integration** で、利用中ディストリビューションを有効化してください。

2. DB と API を起動します。

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

3. 動作確認を行います。

```bash
make health
# または: curl http://localhost:8080/health
```

4. **掲載バッチ**を実行します（シードのデモ求人を公開サイトに出すために必須）。

```bash
make batch
# 同等: ./scripts/batch.sh
# または: docker compose --profile tools run --rm batch
```

5. **メールワーカー**を実行します（`email_outbox` の `pending` を SMTP で送信）。

```bash
make mailworker
# または: docker compose --profile tools run --rm mailworker
```

6. **フロント**を起動します（Node 20+ / npm が必要）。

```bash
cd frontend-react && npm install && npm run dev
cd frontend-vue   && npm install && npm run dev
cd frontend-next  && npm install && npm run dev
```

API を別ホストで動かす場合は、`VITE_API_ORIGIN` / `NEXT_PUBLIC_API_ORIGIN` を設定してください。

## DB だけ Docker、API はホストの Go で動かす

Docker の API（ポート `8080`）とホストの API は同時起動しないでください。

```bash
make db
```

システムに Go がない場合、`scripts/run-api-local.sh` は [scripts/bootstrap-go.sh](scripts/bootstrap-go.sh) を使って、**リポジトリ内 `.tools/go`** に公式 Go を展開します。

```bash
./scripts/run-api-local.sh
```

別ターミナルでバッチを起動します。

```bash
./scripts/run-batch-local.sh
```

`Makefile` の `make api-local` / `make batch-local` / `make mailworker-local` でも同じ処理を実行できます。

接続文字列の既定値は `postgres://recruit:recruit@127.0.0.1:5432/recruit?sslmode=disable` です。

## 従来どおりホストに Go がある場合

```bash
cp .env.example .env
# .env の JWT_SECRET は必ず変更
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

シードには **顧客 2 社・複数案件・応募 3 件・請求・見込み** などが含まれます（`002_seed.sql` + `003_seed_demo.sql`）。  
求人サイトに **初期表示される** `published` 案件も含まれます。その他の案件は **バッチ実行後** に `published` になる場合があります。

GUI クライアント（A5:SQL Mk-2）で DB を確認する手順は [docs/データベース接続.md](docs/データベース接続.md) を参照してください。

## API 概要

- `GET /public/jobs?q=` … 公開中求人一覧
- `GET /public/jobs/{id}` … 詳細
- `POST /public/applications` … 応募
- `POST /admin/jobs/login` … 案件管理 JWT
- `GET/POST/PATCH/DELETE /admin/jobs/jobs...` … 案件 CRUD（Bearer 必須）
- `POST /admin/customers/login` … 顧客管理 JWT
- `GET/POST/PATCH /admin/customers/customers...` … 顧客 CRUD
- `POST /admin/customers/customers/{id}/end-contract` … 契約終了
- `GET/POST /admin/customers/customers/{id}/job-users` … 案件管理ユーザー
- `PATCH/DELETE /admin/customers/job-users/{id}?customer_id=` … 更新・削除
- `GET/POST /admin/customers/invoices` … 請求書
- `GET /admin/customers/prospects` … 見込み一覧

## 利用上の注意

本リポジトリはデモ・学習用途です。本番運用時は、シークレット管理・HTTPS・権限設計を必ず見直してください。
