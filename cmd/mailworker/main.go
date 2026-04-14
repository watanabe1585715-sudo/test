// メール送信ワーカー: email_outbox の pending を SMTP で送り、sent / failed に更新する。
//
// SMTP が未設定（SMTP_HOST 空）のときは DB を変更せず終了コード 0（キューはそのまま残る）。
//
// 実行例:
//
//	DATABASE_URL=... SMTP_HOST=smtp.example.com MAIL_FROM=noreply@example.com go run ./cmd/mailworker
package main

import (
	"context"
	"log"
	"os"
	"strconv"

	"recruitment/internal/config"
	"recruitment/internal/db"
	"recruitment/internal/infrastructure/mail"
	"recruitment/internal/infrastructure/persistence/postgres"
	"recruitment/internal/mailqueue"
	"recruitment/internal/migrate"
)

func main() {
	cfg := config.Load()
	mailCfg := config.LoadMail()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := migrate.Up(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	repo := postgres.NewRepository(pool)
	sender := mail.NewSMTPSender(mailCfg.SMTPHost, mailCfg.SMTPPort, mailCfg.SMTPUser, mailCfg.SMTPPassword)

	if !sender.Configured() {
		log.Print("mailworker: SMTP_HOST is empty; not sending (pending rows unchanged)")
		return
	}
	if mailCfg.From == "" {
		log.Fatal("mailworker: MAIL_FROM is required when SMTP_HOST is set")
	}

	limit := 50
	if v := os.Getenv("MAIL_BATCH_LIMIT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	n, err := mailqueue.Run(ctx, repo, sender, mailCfg.From, limit)
	if err != nil {
		log.Fatalf("mailqueue: %v", err)
	}
	log.Printf("mailworker: sent %d messages (limit=%d)", n, limit)
}
