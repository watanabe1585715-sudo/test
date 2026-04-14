// Package config は「OS の環境変数」から接続先や秘密鍵などを読むだけの小さなパッケージです。
//
// 初心者向けメモ:
//   - DATABASE_URL や JWT_SECRET はソースコードに直書きせず、環境変数で渡すのが一般的です。
//   - .env ファイルを使う場合は、シェルで export するか docker-compose の environment に書きます。
package config

import (
	"os"
	"strings"
)

// Config は API サーバが起動時に参照する設定。
type Config struct {
	DatabaseURL string
	JWTSecret   string
	Addr        string
	CORSOrigins []string
}

// Load は環境変数を読み、未設定の CORS / Listen アドレスにデフォルトを補う。
func Load() Config {
	cors := os.Getenv("CORS_ORIGINS")
	if cors == "" {
		cors = "http://localhost:5173,http://localhost:5174,http://localhost:3000"
	}
	return Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Addr:        envDefault("API_ADDR", ":8080"),
		CORSOrigins: splitCSV(cors),
	}
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Mail は SMTP メールワーカー用。API 本体は参照しない場合もある。
type Mail struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	From         string
}

// LoadMail はメール送信ワーカー向けに SMTP 関連の環境変数を読む。
func LoadMail() Mail {
	return Mail{
		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     envDefault("SMTP_PORT", "587"),
		SMTPUser:     os.Getenv("SMTP_USER"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		From:         os.Getenv("MAIL_FROM"),
	}
}
