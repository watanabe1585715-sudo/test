// Module recruitment は求人広告サイト群の「サーバ（API）」と「掲載バッチ」をまとめた単一の Go モジュールです。
//
// フォルダの読み方（上から順に追うと理解しやすいです）:
//   - cmd/api … ブラウザから届く HTTP を受け取るプログラムの入口（main）。
//   - cmd/batch … 夜間などに実行して掲載状態を直すプログラムの入口（main）。
//   - internal/config … 接続文字列や秘密鍵など、環境変数から設定を読むだけ。
//   - internal/db … PostgreSQL に接続する「プール」を開くだけ。
//   - internal/migrate … テーブル作成と初期データ投入の SQL を実行。
//   - internal/auth … ログイン時のパスワード照合と JWT の発行・検証。
//   - internal/domain … エンティティとリポジトリの契約（インターフェース）。
//   - internal/usecase … アプリケーションサービス（ログインの組み立てなど）。
//   - internal/infrastructure/persistence/postgres … StaffingRepository の実装。
//   - internal/interfaces/http … Gin で URL ごとに JSON を返す。
//   - internal/httpx … JSON を HTTP で返すための小さな補助（バッチ等で利用する場合あり）。
//   - internal/batch … バッチ本体のロジック（期間と契約枠で published を決める）。
module recruitment

go 1.22

// 一部の間接依存が go 1.25 を要求するため、Go 1.22 でビルドできる版へ寄せる。
replace (
	github.com/gin-contrib/sse => github.com/gin-contrib/sse v0.1.0
	golang.org/x/net => golang.org/x/net v0.21.0
	golang.org/x/sync => golang.org/x/sync v0.8.0
	golang.org/x/sys => golang.org/x/sys v0.26.0
	golang.org/x/text => golang.org/x/text v0.19.0
	google.golang.org/protobuf => google.golang.org/protobuf v1.34.2
)

require (
	github.com/gin-contrib/cors v1.4.0
	github.com/gin-gonic/gin v1.8.2
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/jackc/pgx/v5 v5.7.1
	golang.org/x/crypto v0.28.0
)

require (
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.23.0 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/ugorji/go/codec v1.3.1 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
