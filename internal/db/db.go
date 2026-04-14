// Package db は PostgreSQL へ接続するための「プール」を1つ開きます。
//
// 初心者向けメモ:
//   - プールとは「接続を何度も作り直さず使い回す」仕組みで、Web サーバではよく使います。
//   - 接続文字列（DATABASE_URL）の形式は postgres://ユーザー:パス@ホスト:ポート/DB名?sslmode=disable のような形です。
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect は databaseURL でプールを開き、Ping まで成功したものを返す。
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}
