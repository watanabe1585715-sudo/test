// 求人の「掲載中／下書き／終了」をまとめて更新するバッチ（メインプログラム）です。
//
// API サーバとは別に、cron や手動で1日1回など実行する想定です。
// タイムゾーンは Asia/Tokyo、日付だけ変えたいときは環境変数 BATCH_DATE=2006-01-02 を指定します。
//
// 実行例: DATABASE_URL=... go run ./cmd/batch
package main

import (
	"context"
	"log"
	"os"
	"time"

	"recruitment/internal/batch"
	"recruitment/internal/config"
	"recruitment/internal/db"
	"recruitment/internal/migrate"
)

func main() {
	cfg := config.Load()
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

	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		log.Fatalf("timezone: %v", err)
	}
	day := time.Now().In(loc).Format("2006-01-02")
	if v := os.Getenv("BATCH_DATE"); v != "" {
		t, err := time.ParseInLocation("2006-01-02", v, loc)
		if err != nil {
			log.Fatalf("BATCH_DATE: %v", err)
		}
		day = t.Format("2006-01-02")
	}

	pub, dr, end, err := batch.Run(ctx, pool, day)
	if err != nil {
		log.Fatalf("batch: %v", err)
	}
	log.Printf("batch date=%s published=%d draft=%d ended=%d", day, pub, dr, end)
}
