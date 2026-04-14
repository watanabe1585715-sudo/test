// 求人広告システムの HTTP API サーバ（メインプログラム）です。
//
// 起動の流れ（初心者向け）:
//  1. 環境変数を読む（internal/config）
//  2. PostgreSQL に接続する（internal/db）
//  3. テーブルが無ければ作る・初期データを入れる（internal/migrate）
//  4. URL ごとの処理を登録した Gin サーバを起動し、Ctrl+C まで待つ（internal/interfaces/http）
//
// 実行例: DATABASE_URL=... JWT_SECRET=... go run ./cmd/api
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"recruitment/internal/config"
	"recruitment/internal/db"
	"recruitment/internal/infrastructure/persistence/postgres"
	"recruitment/internal/interfaces/http"
	"recruitment/internal/migrate"
	"recruitment/internal/usecase"
)

func main() {
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	if err := migrate.Up(ctx, pool); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	repo := postgres.NewRepository(pool)
	app := usecase.NewStaffingApp(repo, []byte(cfg.JWTSecret))
	h := api.NewEngine(app, cfg.CORSOrigins)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("API listening on %s", cfg.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = httpSrv.Shutdown(shutdownCtx)
}
