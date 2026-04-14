// Package migrate は「はじめて DB を使うときにテーブルを作る」ための SQL を実行します。
//
// 初心者向けメモ:
//   - アプリを初めて起動するとき、空の PostgreSQL にテーブルが無いとエラーになります。
//   - このパッケージは sql フォルダ内の .sql を順番に流し、schema_migrations テーブルに
//     「もう実行したファイル名」を記録します。二回目以降は同じファイルは流しません。
//   - 001 がテーブル作成、002 以降が初期データ（シード）や追加テーブル（例: 005 メールキュー、007 求職者・顧客承認・イベント）です。
package migrate

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/*.sql
var files embed.FS

// Up は未適用の migration ファイルをトランザクション単位で実行する。
func Up(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("schema_migrations: %w", err)
	}
	order := []string{"001_init.sql", "002_seed.sql", "003_seed_demo.sql", "004_seed_bulk.sql", "005_email_outbox.sql", "006_announcements.sql", "007_jobseeker_customer_events.sql", "008_additional_features.sql", "009_seed_additional_features.sql", "010_customer_events_demo.sql"}
	for _, base := range order {
		name := "sql/" + base
		var dummy int
		err := pool.QueryRow(ctx, `SELECT 1 FROM schema_migrations WHERE name=$1`, base).Scan(&dummy)
		if err == nil {
			continue
		}
		if err != pgx.ErrNoRows {
			return err
		}
		b, err := files.ReadFile(name)
		if err != nil {
			return err
		}
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		for _, stmt := range splitSQL(string(b)) {
			if _, err := tx.Exec(ctx, stmt); err != nil {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("%s: %w", base, err)
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, base); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

// splitSQL はセミコロン区切りの素朴な分割（コメント行は stripLineComments で除去）。
func splitSQL(s string) []string {
	var parts []string
	for _, block := range strings.Split(s, ";") {
		q := stripLineComments(block)
		q = strings.TrimSpace(q)
		if q == "" {
			continue
		}
		parts = append(parts, q)
	}
	return parts
}

func stripLineComments(block string) string {
	lines := strings.Split(block, "\n")
	var out []string
	for _, ln := range lines {
		t := strings.TrimSpace(ln)
		if strings.HasPrefix(t, "--") {
			continue
		}
		out = append(out, ln)
	}
	return strings.Join(out, "\n")
}
