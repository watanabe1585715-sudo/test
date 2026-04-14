// Package batch は、求人が「サイトに載るか／載らないか」を日次（など）で揃える処理です。
//
// 初心者向けメモ:
//   - 案件（job_postings）には publication_status という列があり、published だと求職者向け API に出ます。
//   - 管理画面で作っただけでは draft のままのこともあり、掲載期間と契約の「枠」が合って初めて published にします。
//   - このパッケージはその判定をまとめて DB を更新します。API サーバとは別プロセス（cmd/batch）で動かします。
//   - 公開 API（postgres 実装の ListPublicJobs）は published かつ今日が掲載期間内のものだけ返します。
package batch

import (
	"context"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5/pgxpool"
)

// tierLimit は契約 tier（1=10件, 2=100件, 3=無制限）ごとの掲載上限。
func tierLimit(tier int32) int {
	switch tier {
	case 1:
		return 10
	case 2:
		return 100
	default:
		return math.MaxInt32
	}
}

// Run は基準日 day（YYYY-MM-DD、DB の ::date 比較用）で次を行う:
//  1. 掲載期間外 → ended
//  2. 期間内だが顧客が契約無効・期間外 → draft
//  3. 期間内かつ有効顧客 → 顧客ごとに優先順位で published、枠超えは draft
func Run(ctx context.Context, pool *pgxpool.Pool, day string) (published, draft, ended int64, err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
UPDATE job_postings SET publication_status = 'ended', updated_at = NOW()
WHERE $1::date NOT BETWEEN publish_start AND publish_end
`, day)
	if err != nil {
		return 0, 0, 0, err
	}
	ended += tag.RowsAffected()

	if _, err := tx.Exec(ctx, `
UPDATE job_postings j SET publication_status = 'draft', updated_at = NOW()
FROM customers c
WHERE j.customer_id = c.id
  AND $1::date BETWEEN j.publish_start AND j.publish_end
  AND (
    c.status <> 'active'
    OR c.approval_status <> 'approved'
    OR $1::date < c.contract_start
    OR (c.contract_end IS NOT NULL AND $1::date > c.contract_end)
  )
`, day); err != nil {
		return 0, 0, 0, err
	}

	rows, err := tx.Query(ctx, `
SELECT j.id, j.customer_id, c.contract_tier
FROM job_postings j
JOIN customers c ON c.id = j.customer_id
WHERE $1::date BETWEEN j.publish_start AND j.publish_end
  AND c.status = 'active'
  AND c.approval_status = 'approved'
  AND $1::date >= c.contract_start
  AND (c.contract_end IS NULL OR $1::date <= c.contract_end)
ORDER BY j.customer_id, j.publish_start ASC, j.created_at ASC, j.id ASC
`, day)
	if err != nil {
		return 0, 0, 0, err
	}
	defer rows.Close()

	type row struct {
		id       int64
		customer int64
		tier     int32
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.customer, &r.tier); err != nil {
			return 0, 0, 0, err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, err
	}

	var curCustomer int64 = -1
	var limit = math.MaxInt32
	var used int

	for _, r := range list {
		if r.customer != curCustomer {
			curCustomer = r.customer
			limit = tierLimit(r.tier)
			used = 0
		}
		var status string
		if used < limit {
			status = "published"
			used++
			published++
		} else {
			status = "draft"
			draft++
		}
		tag, err := tx.Exec(ctx, `
UPDATE job_postings SET publication_status = $1, updated_at = NOW() WHERE id = $2
`, status, r.id)
		if err != nil {
			return 0, 0, 0, err
		}
		if tag.RowsAffected() != 1 {
			return 0, 0, 0, fmt.Errorf("unexpected rows for job %d", r.id)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, 0, 0, err
	}
	return published, draft, ended, nil
}
