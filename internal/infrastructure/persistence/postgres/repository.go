// Package postgres は domain.StaffingRepository の PostgreSQL 実装です（DDD の infrastructure 層）。
//
// 初心者向け:
//   - domain パッケージは「契約（インターフェース）」だけを決め、このパッケージが実際に DB に問い合わせます。
//   - ループ（for rows.Next）では「1行ずつ Scan してスライスに積む」典型パターンです。Scan が失敗したら即 return します。
//   - 条件分岐（if errors.Is(err, pgx.ErrNoRows)）は「行が無かった＝見つからない」ときの特別扱いです。
package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"recruitment/internal/domain"
)

// Repository は pgx プールを保持し、domain.StaffingRepository を満たす。
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository は domain から参照される具象リポジトリを生成する。
func NewRepository(pool *pgxpool.Pool) domain.StaffingRepository {
	return &Repository{pool: pool}
}

func (r *Repository) ListPublicJobs(ctx context.Context, q string) ([]domain.PublicJob, error) {
	rows, err := r.pool.Query(ctx, `
SELECT j.id, j.summary, j.requirements, j.publish_start, j.publish_end
FROM job_postings j
JOIN customers c ON c.id = j.customer_id
WHERE j.publication_status = 'published'
  AND c.status = 'active'
  AND c.approval_status = 'approved'
  AND CURRENT_DATE BETWEEN j.publish_start AND j.publish_end
  AND CURRENT_DATE >= c.contract_start
  AND (c.contract_end IS NULL OR CURRENT_DATE <= c.contract_end)
  AND ($1 = '' OR j.summary ILIKE '%' || $1 || '%' OR j.requirements ILIKE '%' || $1 || '%')
ORDER BY j.publish_start ASC, j.id ASC
`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.PublicJob
	// rows.Next は「まだ読んでいない行があるか」を返す。false になったらループ終了。
	for rows.Next() {
		var j domain.PublicJob
		// 現在行の列を構造体フィールドへ順に流し込む。列数や型が合わないとここでエラー。
		if err := rows.Scan(&j.ID, &j.Summary, &j.Requirements, &j.PublishStart, &j.PublishEnd); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	// イテレーション中に起きたエラー（接続切断など）は Next 終了後に拾う。
	return out, rows.Err()
}

func (r *Repository) GetPublicJob(ctx context.Context, id int64) (*domain.PublicJob, error) {
	var j domain.PublicJob
	err := r.pool.QueryRow(ctx, `
SELECT j.id, j.summary, j.requirements, j.publish_start, j.publish_end
FROM job_postings j
JOIN customers c ON c.id = j.customer_id
WHERE j.id = $1
  AND j.publication_status = 'published'
  AND c.status = 'active'
  AND c.approval_status = 'approved'
  AND CURRENT_DATE BETWEEN j.publish_start AND j.publish_end
  AND CURRENT_DATE >= c.contract_start
  AND (c.contract_end IS NULL OR CURRENT_DATE <= c.contract_end)
`, id).Scan(&j.ID, &j.Summary, &j.Requirements, &j.PublishStart, &j.PublishEnd)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// CreateApplication は応募 INSERT。公開条件を満たす案件のみ受け付ける。
// 連絡先がメールアドレスらしければ、同一トランザクションで thank-you を email_outbox に積む。
func (r *Repository) CreateApplication(ctx context.Context, jobID int64, name, career, contact string) error {
	var ok bool
	err := r.pool.QueryRow(ctx, `
SELECT true FROM job_postings j
JOIN customers c ON c.id = j.customer_id
WHERE j.id = $1
  AND j.publication_status = 'published'
  AND c.status = 'active'
  AND c.approval_status = 'approved'
  AND CURRENT_DATE BETWEEN j.publish_start AND j.publish_end
`, jobID).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgx.ErrNoRows
	}
	if err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var appID int64
	err = tx.QueryRow(ctx, `
INSERT INTO applications (job_posting_id, applicant_name, career_summary, contact)
VALUES ($1, $2, $3, $4)
RETURNING id
`, jobID, name, career, contact).Scan(&appID)
	if err != nil {
		return err
	}

	// 簡易チェック: @ を含むならメールキューに積む（厳密な RFC 検証はしない）。
	if strings.Contains(contact, "@") {
		subject := "【求人サイト】応募を受け付けました"
		body := fmt.Sprintf("%s 様\n\nこの度はご応募ありがとうございます。内容を確認のうえ、担当よりご連絡いたします。\n\n（このメールは自動送信です）", name)
		_, err = tx.Exec(ctx, `
INSERT INTO email_outbox (kind, to_email, subject, body, status, related_application_id)
VALUES ('application_thankyou', $1, $2, $3, 'pending', $4)
`, contact, subject, body, appID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetJobAdminByEmail(ctx context.Context, email string) (id int64, hash string, customerID int64, err error) {
	err = r.pool.QueryRow(ctx, `
SELECT id, password_hash, customer_id FROM job_admin_users WHERE email = $1 AND active`, email,
	).Scan(&id, &hash, &customerID)
	return
}

func (r *Repository) GetCustomerAdminByEmail(ctx context.Context, email string) (id int64, hash string, registrationStatus string, err error) {
	err = r.pool.QueryRow(ctx, `
SELECT id, password_hash, registration_status FROM customer_admin_users WHERE email = $1 AND active`, email,
	).Scan(&id, &hash, &registrationStatus)
	return
}

func (r *Repository) ListJobsForCustomer(ctx context.Context, customerID int64, q string) ([]domain.JobRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, customer_id, summary, requirements, publish_start, publish_end, publication_status, created_at, updated_at
FROM job_postings
WHERE customer_id = $1
  AND ($2 = '' OR summary ILIKE '%' || $2 || '%' OR requirements ILIKE '%' || $2 || '%')
ORDER BY publish_start DESC, id DESC
`, customerID, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.JobRow
	for rows.Next() {
		var j domain.JobRow
		if err := rows.Scan(&j.ID, &j.CustomerID, &j.Summary, &j.Requirements, &j.PublishStart, &j.PublishEnd, &j.PublicationStatus, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

func (r *Repository) CreateJob(ctx context.Context, customerID int64, summary, requirements string, start, end time.Time) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO job_postings (customer_id, summary, requirements, publish_start, publish_end, publication_status)
VALUES ($1, $2, $3, $4::date, $5::date, 'draft')
RETURNING id
`, customerID, summary, requirements, start, end).Scan(&id)
	return id, err
}

func (r *Repository) GetJobForCustomer(ctx context.Context, customerID, jobID int64) (*domain.JobRow, error) {
	var j domain.JobRow
	err := r.pool.QueryRow(ctx, `
SELECT id, customer_id, summary, requirements, publish_start, publish_end, publication_status, created_at, updated_at
FROM job_postings WHERE id = $1 AND customer_id = $2`, jobID, customerID,
	).Scan(&j.ID, &j.CustomerID, &j.Summary, &j.Requirements, &j.PublishStart, &j.PublishEnd, &j.PublicationStatus, &j.CreatedAt, &j.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

func (r *Repository) UpdateJob(ctx context.Context, customerID, jobID int64, summary, requirements string, start, end time.Time) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE job_postings SET summary=$1, requirements=$2, publish_start=$3::date, publish_end=$4::date, updated_at=NOW()
WHERE id=$5 AND customer_id=$6
`, summary, requirements, start, end, jobID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteJob(ctx context.Context, customerID, jobID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM job_postings WHERE id=$1 AND customer_id=$2`, jobID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListApplicationsForJob(ctx context.Context, customerID, jobID int64) ([]domain.ApplicationRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT a.id, a.job_posting_id, a.applicant_name, a.career_summary, a.contact, a.created_at
FROM applications a
JOIN job_postings j ON j.id = a.job_posting_id
WHERE a.job_posting_id = $1 AND j.customer_id = $2
ORDER BY a.created_at DESC
`, jobID, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ApplicationRow
	for rows.Next() {
		var a domain.ApplicationRow
		if err := rows.Scan(&a.ID, &a.JobPostingID, &a.ApplicantName, &a.CareerSummary, &a.Contact, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) ListCustomers(ctx context.Context, q string) ([]domain.CustomerRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, name, description, contract_tier, contract_start, contract_end, status, approval_status, created_at, updated_at
FROM customers
WHERE $1 = '' OR name ILIKE '%' || $1 || '%' OR COALESCE(description,'') ILIKE '%' || $1 || '%'
ORDER BY id DESC
`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CustomerRow
	for rows.Next() {
		var c domain.CustomerRow
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.ContractTier, &c.ContractStart, &c.ContractEnd, &c.Status, &c.ApprovalStatus, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) GetCustomer(ctx context.Context, id int64) (*domain.CustomerRow, error) {
	var c domain.CustomerRow
	err := r.pool.QueryRow(ctx, `
SELECT id, name, description, contract_tier, contract_start, contract_end, status, approval_status, created_at, updated_at
FROM customers WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Description, &c.ContractTier, &c.ContractStart, &c.ContractEnd, &c.Status, &c.ApprovalStatus, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *Repository) CreateCustomer(ctx context.Context, name, description string, tier int, start time.Time, end *time.Time) (int64, error) {
	var id int64
	var endArg any
	if end != nil {
		endArg = *end
	}
	err := r.pool.QueryRow(ctx, `
INSERT INTO customers (name, description, contract_tier, contract_start, contract_end, status, approval_status)
VALUES ($1, NULLIF($2,''), $3, $4::date, $5::date, 'active', 'pending')
RETURNING id
`, name, description, tier, start, endArg).Scan(&id)
	return id, err
}

func (r *Repository) UpdateCustomer(ctx context.Context, id int64, name, description string, tier int, start time.Time, end *time.Time, approvalStatus *string) error {
	var endArg any
	if end != nil {
		endArg = *end
	}
	var appr any
	if approvalStatus != nil {
		appr = *approvalStatus
	}
	tag, err := r.pool.Exec(ctx, `
UPDATE customers SET name=$1, description=NULLIF($2,''), contract_tier=$3, contract_start=$4::date, contract_end=$5::date,
  approval_status = COALESCE($6, approval_status), updated_at=NOW()
WHERE id=$7
`, name, description, tier, start, endArg, appr, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// EndCustomerContract は顧客を ended にし、その顧客の掲載中案件を掲載終了へ寄せる。
func (r *Repository) EndCustomerContract(ctx context.Context, id int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE customers SET status='ended', updated_at=NOW() WHERE id=$1`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
UPDATE job_postings SET publication_status='ended', updated_at=NOW()
WHERE customer_id=$1 AND publication_status='published'
`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) ListJobAdminUsers(ctx context.Context, customerID int64) ([]domain.JobAdminUser, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, customer_id, email, active FROM job_admin_users WHERE customer_id=$1 ORDER BY id`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.JobAdminUser
	for rows.Next() {
		var u domain.JobAdminUser
		if err := rows.Scan(&u.ID, &u.CustomerID, &u.Email, &u.Active); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) CreateJobAdminUser(ctx context.Context, customerID int64, email, passwordHash string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO job_admin_users (customer_id, email, password_hash, active)
VALUES ($1, $2, $3, true) RETURNING id
`, customerID, email, passwordHash).Scan(&id)
	return id, err
}

func (r *Repository) UpdateJobAdminUser(ctx context.Context, customerID, userID int64, email string, active bool, passwordHash *string) error {
	if passwordHash != nil {
		tag, err := r.pool.Exec(ctx, `
UPDATE job_admin_users SET email=$1, active=$2, password_hash=$3 WHERE id=$4 AND customer_id=$5
`, email, active, *passwordHash, userID, customerID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	}
	tag, err := r.pool.Exec(ctx, `
UPDATE job_admin_users SET email=$1, active=$2 WHERE id=$3 AND customer_id=$4
`, email, active, userID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteJobAdminUser(ctx context.Context, customerID, userID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM job_admin_users WHERE id=$1 AND customer_id=$2`, userID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListInvoices(ctx context.Context, customerID *int64) ([]domain.InvoiceRow, error) {
	var rows pgx.Rows
	var err error
	if customerID == nil {
		rows, err = r.pool.Query(ctx, `
SELECT id, customer_id, issued_at, amount_cents, status, notes, created_at FROM invoices ORDER BY issued_at DESC, id DESC`)
	} else {
		rows, err = r.pool.Query(ctx, `
SELECT id, customer_id, issued_at, amount_cents, status, notes, created_at FROM invoices
WHERE customer_id=$1 ORDER BY issued_at DESC, id DESC`, *customerID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.InvoiceRow
	for rows.Next() {
		var inv domain.InvoiceRow
		if err := rows.Scan(&inv.ID, &inv.CustomerID, &inv.IssuedAt, &inv.AmountCents, &inv.Status, &inv.Notes, &inv.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *Repository) CreateInvoice(ctx context.Context, customerID int64, issuedAt time.Time, amountCents int64, status, notes string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO invoices (customer_id, issued_at, amount_cents, status, notes)
VALUES ($1, $2::date, $3, $4, NULLIF($5,''))
RETURNING id
`, customerID, issuedAt, amountCents, status, notes).Scan(&id)
	return id, err
}

func (r *Repository) GetInvoice(ctx context.Context, id int64) (*domain.InvoiceRow, error) {
	var inv domain.InvoiceRow
	err := r.pool.QueryRow(ctx, `
SELECT id, customer_id, issued_at, amount_cents, status, notes, created_at FROM invoices WHERE id=$1`, id,
	).Scan(&inv.ID, &inv.CustomerID, &inv.IssuedAt, &inv.AmountCents, &inv.Status, &inv.Notes, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *Repository) ListProspects(ctx context.Context, q string) ([]domain.ProspectRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, company_name, contact_info, notes, created_at FROM prospects
WHERE $1 = '' OR company_name ILIKE '%' || $1 || '%' OR COALESCE(contact_info,'') ILIKE '%' || $1 || '%'
ORDER BY id DESC`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ProspectRow
	for rows.Next() {
		var p domain.ProspectRow
		if err := rows.Scan(&p.ID, &p.CompanyName, &p.ContactInfo, &p.Notes, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) GetProspect(ctx context.Context, id int64) (*domain.ProspectRow, error) {
	var p domain.ProspectRow
	err := r.pool.QueryRow(ctx, `
SELECT id, company_name, contact_info, notes, created_at FROM prospects WHERE id=$1`, id,
	).Scan(&p.ID, &p.CompanyName, &p.ContactInfo, &p.Notes, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) CreateProspect(ctx context.Context, companyName, contactInfo, notes string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO prospects (company_name, contact_info, notes)
VALUES ($1, NULLIF($2, ''), NULLIF($3, ''))
RETURNING id
`, companyName, contactInfo, notes).Scan(&id)
	return id, err
}

func (r *Repository) UpdateProspect(ctx context.Context, id int64, companyName, contactInfo, notes string) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE prospects SET company_name=$1, contact_info=NULLIF($2, ''), notes=NULLIF($3, '') WHERE id=$4
`, companyName, contactInfo, notes, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteProspect(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM prospects WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListApplicationsAll(ctx context.Context, q string, limit int) ([]domain.ApplicationAdminRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := r.pool.Query(ctx, `
SELECT a.id, a.job_posting_id, j.summary, j.customer_id, c.name,
       a.applicant_name, a.career_summary, a.contact, a.created_at
FROM applications a
JOIN job_postings j ON j.id = a.job_posting_id
JOIN customers c ON c.id = j.customer_id
WHERE $1 = '' OR j.summary ILIKE '%' || $1 || '%' OR a.applicant_name ILIKE '%' || $1 || '%'
   OR a.contact ILIKE '%' || $1 || '%' OR c.name ILIKE '%' || $1 || '%'
ORDER BY a.created_at DESC
LIMIT $2
`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ApplicationAdminRow
	for rows.Next() {
		var row domain.ApplicationAdminRow
		if err := rows.Scan(&row.ID, &row.JobPostingID, &row.JobSummary, &row.CustomerID, &row.CustomerName,
			&row.ApplicantName, &row.CareerSummary, &row.Contact, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) EnqueueEmail(ctx context.Context, kind, toEmail, subject, body string, relatedApplicationID *int64) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO email_outbox (kind, to_email, subject, body, status, related_application_id)
VALUES ($1, $2, $3, $4, 'pending', $5)
RETURNING id
`, kind, toEmail, subject, body, relatedApplicationID).Scan(&id)
	return id, err
}

func (r *Repository) ListEmailQueue(ctx context.Context, status string, limit int) ([]domain.EmailOutboxRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = r.pool.Query(ctx, `
SELECT id, kind, to_email, subject, body, status, error_detail, related_application_id, created_at, sent_at
FROM email_outbox
ORDER BY id DESC
LIMIT $1
`, limit)
	} else {
		rows, err = r.pool.Query(ctx, `
SELECT id, kind, to_email, subject, body, status, error_detail, related_application_id, created_at, sent_at
FROM email_outbox
WHERE status = $1
ORDER BY id DESC
LIMIT $2
`, status, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEmailRows(rows)
}

func (r *Repository) ListPendingEmails(ctx context.Context, limit int) ([]domain.EmailOutboxRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, kind, to_email, subject, body, status, error_detail, related_application_id, created_at, sent_at
FROM email_outbox
WHERE status = 'pending'
ORDER BY id ASC
LIMIT $1
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEmailRows(rows)
}

func scanEmailRows(rows pgx.Rows) ([]domain.EmailOutboxRow, error) {
	var out []domain.EmailOutboxRow
	for rows.Next() {
		var e domain.EmailOutboxRow
		if err := rows.Scan(&e.ID, &e.Kind, &e.ToEmail, &e.Subject, &e.Body, &e.Status, &e.ErrorDetail, &e.RelatedApplicationID, &e.CreatedAt, &e.SentAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) MarkEmailSent(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE email_outbox SET status = 'sent', sent_at = NOW(), error_detail = NULL
WHERE id = $1 AND status = 'pending'
`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) MarkEmailFailed(ctx context.Context, id int64, errDetail string) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE email_outbox SET status = 'failed', error_detail = $2
WHERE id = $1 AND status = 'pending'
`, id, errDetail)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ResetEmailToPending(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE email_outbox SET status = 'pending', error_detail = NULL, sent_at = NULL
WHERE id = $1 AND status = 'failed'
`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListAnnouncementsActive(ctx context.Context, channel string) ([]domain.AnnouncementRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, title, body, channel, active, valid_from, valid_to, sort_order, created_at, updated_at
FROM announcements
WHERE active = TRUE
  AND (channel = $1 OR channel = 'all')
  AND (valid_from IS NULL OR valid_from <= NOW())
  AND (valid_to IS NULL OR valid_to >= NOW())
ORDER BY sort_order ASC, id DESC
`, channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAnnouncementRows(rows)
}

func (r *Repository) ListAnnouncementsManage(ctx context.Context) ([]domain.AnnouncementRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, title, body, channel, active, valid_from, valid_to, sort_order, created_at, updated_at
FROM announcements
ORDER BY sort_order ASC, id DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAnnouncementRows(rows)
}

func scanAnnouncementRows(rows pgx.Rows) ([]domain.AnnouncementRow, error) {
	var out []domain.AnnouncementRow
	for rows.Next() {
		var a domain.AnnouncementRow
		if err := rows.Scan(&a.ID, &a.Title, &a.Body, &a.Channel, &a.Active, &a.ValidFrom, &a.ValidTo, &a.SortOrder, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repository) GetAnnouncement(ctx context.Context, id int64) (*domain.AnnouncementRow, error) {
	var a domain.AnnouncementRow
	err := r.pool.QueryRow(ctx, `
SELECT id, title, body, channel, active, valid_from, valid_to, sort_order, created_at, updated_at
FROM announcements WHERE id = $1`, id,
	).Scan(&a.ID, &a.Title, &a.Body, &a.Channel, &a.Active, &a.ValidFrom, &a.ValidTo, &a.SortOrder, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *Repository) CreateAnnouncement(ctx context.Context, title, body, channel string, active bool, validFrom, validTo *time.Time, sortOrder int) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO announcements (title, body, channel, active, valid_from, valid_to, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id
`, title, body, channel, active, validFrom, validTo, sortOrder).Scan(&id)
	return id, err
}

func (r *Repository) UpdateAnnouncement(ctx context.Context, id int64, title, body, channel string, active bool, validFrom, validTo *time.Time, sortOrder int) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE announcements SET title=$1, body=$2, channel=$3, active=$4, valid_from=$5, valid_to=$6, sort_order=$7, updated_at=NOW()
WHERE id=$8
`, title, body, channel, active, validFrom, validTo, sortOrder, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteAnnouncement(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM announcements WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListCustomerAdminUsers(ctx context.Context) ([]domain.CustomerAdminUserRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, email, active, registration_status, created_at FROM customer_admin_users ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CustomerAdminUserRow
	for rows.Next() {
		var u domain.CustomerAdminUserRow
		if err := rows.Scan(&u.ID, &u.Email, &u.Active, &u.RegistrationStatus, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) GetCustomerAdminUser(ctx context.Context, id int64) (*domain.CustomerAdminUserRow, error) {
	var u domain.CustomerAdminUserRow
	err := r.pool.QueryRow(ctx, `
SELECT id, email, active, registration_status, created_at FROM customer_admin_users WHERE id=$1`, id,
	).Scan(&u.ID, &u.Email, &u.Active, &u.RegistrationStatus, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CountActiveCustomerAdmins(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM customer_admin_users WHERE active = TRUE`).Scan(&n)
	return n, err
}

func (r *Repository) CountCustomerAdminUsers(ctx context.Context) (int64, error) {
	var n int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM customer_admin_users`).Scan(&n)
	return n, err
}

func (r *Repository) CreateCustomerAdminUser(ctx context.Context, email, passwordHash string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO customer_admin_users (email, password_hash, active, registration_status)
VALUES ($1, $2, TRUE, 'pending')
RETURNING id
`, email, passwordHash).Scan(&id)
	return id, err
}

func (r *Repository) UpdateCustomerAdminUser(ctx context.Context, id int64, email string, active bool, passwordHash *string, registrationStatus *string) error {
	if passwordHash != nil && registrationStatus != nil {
		tag, err := r.pool.Exec(ctx, `
UPDATE customer_admin_users SET email=$1, active=$2, password_hash=$3, registration_status=$4 WHERE id=$5
`, email, active, *passwordHash, *registrationStatus, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	}
	if passwordHash != nil {
		tag, err := r.pool.Exec(ctx, `
UPDATE customer_admin_users SET email=$1, active=$2, password_hash=$3 WHERE id=$4
`, email, active, *passwordHash, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	}
	if registrationStatus != nil {
		tag, err := r.pool.Exec(ctx, `
UPDATE customer_admin_users SET email=$1, active=$2, registration_status=$3 WHERE id=$4
`, email, active, *registrationStatus, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return pgx.ErrNoRows
		}
		return nil
	}
	tag, err := r.pool.Exec(ctx, `
UPDATE customer_admin_users SET email=$1, active=$2 WHERE id=$3
`, email, active, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteCustomerAdminUser(ctx context.Context, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM customer_admin_users WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// CreateJobSeekerAccount はメール重複時にユニーク制約エラー。アカウントと空のプロフィール行を同一トランザクションで作る。
// 将来の「お気に入り」「閲覧履歴」も account_id を起点に関連付ける想定。
func (r *Repository) CreateJobSeekerAccount(ctx context.Context, email, passwordHash string) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id int64
	if err := tx.QueryRow(ctx, `
INSERT INTO job_seeker_accounts (email, password_hash, active)
VALUES (LOWER(TRIM($1)), $2, TRUE)
RETURNING id
`, email, passwordHash).Scan(&id); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO job_seeker_profiles (account_id, display_name, phone, career_summary, notes)
VALUES ($1, '', '', '', '')
`, id); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Repository) GetJobSeekerByEmail(ctx context.Context, email string) (id int64, hash string, err error) {
	err = r.pool.QueryRow(ctx, `
SELECT id, password_hash FROM job_seeker_accounts WHERE email = LOWER(TRIM($1)) AND active`, email,
	).Scan(&id, &hash)
	return
}

func (r *Repository) GetJobSeekerProfile(ctx context.Context, accountID int64) (*domain.JobSeekerProfileRow, error) {
	var p domain.JobSeekerProfileRow
	err := r.pool.QueryRow(ctx, `
SELECT p.account_id, a.email, p.display_name, p.phone, p.career_summary, p.notes, p.updated_at
FROM job_seeker_profiles p
JOIN job_seeker_accounts a ON a.id = p.account_id
WHERE p.account_id = $1 AND a.active`, accountID,
	).Scan(&p.AccountID, &p.Email, &p.DisplayName, &p.Phone, &p.CareerSummary, &p.Notes, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) UpdateJobSeekerProfile(ctx context.Context, accountID int64, displayName, phone, careerSummary, notes string) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE job_seeker_profiles
SET display_name=$1, phone=$2, career_summary=$3, notes=$4, updated_at=NOW()
WHERE account_id=$5
`, displayName, phone, careerSummary, notes, accountID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListCustomerEvents(ctx context.Context, customerID int64) ([]domain.CustomerEventRow, error) {
	// occurred_at 降順にして、最新の打ち合わせ・リスク情報を先頭に表示しやすくする。
	rows, err := r.pool.Query(ctx, `
SELECT id, customer_id, event_kind, occurred_at, title, body, is_risk_related, created_at
FROM customer_events
WHERE customer_id = $1
ORDER BY occurred_at DESC, id DESC
`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CustomerEventRow
	for rows.Next() {
		var e domain.CustomerEventRow
		if err := rows.Scan(&e.ID, &e.CustomerID, &e.EventKind, &e.OccurredAt, &e.Title, &e.Body, &e.IsRiskRelated, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCustomerEvent(ctx context.Context, customerID int64, eventKind string, occurredAt time.Time, title, body string, isRiskRelated bool) (int64, error) {
	var id int64
	var bodyArg any
	if strings.TrimSpace(body) != "" {
		bodyArg = body
	}
	err := r.pool.QueryRow(ctx, `
INSERT INTO customer_events (customer_id, event_kind, occurred_at, title, body, is_risk_related)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id
`, customerID, eventKind, occurredAt, title, bodyArg, isRiskRelated).Scan(&id)
	return id, err
}

func (r *Repository) UpdateCustomerEvent(ctx context.Context, customerID, eventID int64, eventKind string, occurredAt time.Time, title, body string, isRiskRelated bool) error {
	var bodyArg any
	if strings.TrimSpace(body) != "" {
		bodyArg = body
	}
	tag, err := r.pool.Exec(ctx, `
UPDATE customer_events
SET event_kind=$1, occurred_at=$2, title=$3, body=$4, is_risk_related=$5
WHERE id=$6 AND customer_id=$7
`, eventKind, occurredAt, title, bodyArg, isRiskRelated, eventID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteCustomerEvent(ctx context.Context, customerID, eventID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM customer_events WHERE id=$1 AND customer_id=$2`, eventID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListJobSeekerFavorites(ctx context.Context, accountID int64) ([]domain.JobSeekerFavoriteRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT job_posting_id, created_at
FROM job_seeker_favorites
WHERE account_id = $1
ORDER BY created_at DESC
`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.JobSeekerFavoriteRow
	for rows.Next() {
		var row domain.JobSeekerFavoriteRow
		if err := rows.Scan(&row.JobPostingID, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) AddJobSeekerFavorite(ctx context.Context, accountID, jobPostingID int64) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO job_seeker_favorites (account_id, job_posting_id)
VALUES ($1, $2)
ON CONFLICT (account_id, job_posting_id) DO NOTHING
`, accountID, jobPostingID)
	return err
}

func (r *Repository) DeleteJobSeekerFavorite(ctx context.Context, accountID, jobPostingID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM job_seeker_favorites WHERE account_id=$1 AND job_posting_id=$2`, accountID, jobPostingID)
	return err
}

func (r *Repository) ListJobSeekerViewHistory(ctx context.Context, accountID int64, limit int) ([]domain.JobViewHistoryRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, job_posting_id, viewed_at
FROM job_view_history
WHERE account_id=$1
ORDER BY viewed_at DESC
LIMIT $2
`, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.JobViewHistoryRow
	for rows.Next() {
		var row domain.JobViewHistoryRow
		if err := rows.Scan(&row.ID, &row.JobPostingID, &row.ViewedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) AddJobSeekerViewHistory(ctx context.Context, accountID, jobPostingID int64) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO job_view_history (account_id, job_posting_id)
VALUES ($1, $2)
`, accountID, jobPostingID)
	return err
}

func (r *Repository) GetCompanyProfile(ctx context.Context, customerID int64) (*domain.CompanyProfileRow, error) {
	var row domain.CompanyProfileRow
	err := r.pool.QueryRow(ctx, `
SELECT customer_id, company_name, description, address, google_map_url, website_url, youtube_embed_url, accept_foreigners, languages, updated_at
FROM company_profiles
WHERE customer_id=$1
`, customerID).Scan(&row.CustomerID, &row.CompanyName, &row.Description, &row.Address, &row.GoogleMapURL, &row.WebsiteURL, &row.YoutubeEmbedURL, &row.AcceptForeigners, &row.Languages, &row.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) UpsertCompanyProfile(ctx context.Context, row domain.CompanyProfileRow) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO company_profiles (customer_id, company_name, description, address, google_map_url, website_url, youtube_embed_url, accept_foreigners, languages, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
ON CONFLICT (customer_id) DO UPDATE SET
  company_name=EXCLUDED.company_name,
  description=EXCLUDED.description,
  address=EXCLUDED.address,
  google_map_url=EXCLUDED.google_map_url,
  website_url=EXCLUDED.website_url,
  youtube_embed_url=EXCLUDED.youtube_embed_url,
  accept_foreigners=EXCLUDED.accept_foreigners,
  languages=EXCLUDED.languages,
  updated_at=NOW()
`, row.CustomerID, row.CompanyName, row.Description, row.Address, row.GoogleMapURL, row.WebsiteURL, row.YoutubeEmbedURL, row.AcceptForeigners, row.Languages)
	return err
}

func (r *Repository) ListCompanyReviews(ctx context.Context, customerID int64) ([]domain.CompanyReviewRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, customer_id, reviewer, rating, comment, created_at
FROM company_reviews
WHERE customer_id=$1
ORDER BY created_at DESC
`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CompanyReviewRow
	for rows.Next() {
		var row domain.CompanyReviewRow
		if err := rows.Scan(&row.ID, &row.CustomerID, &row.Reviewer, &row.Rating, &row.Comment, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) ListCompanyVideos(ctx context.Context, customerID int64) ([]domain.CompanyVideoRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, customer_id, title, youtube_url, created_at
FROM company_videos
WHERE customer_id=$1
ORDER BY created_at DESC
`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.CompanyVideoRow
	for rows.Next() {
		var row domain.CompanyVideoRow
		if err := rows.Scan(&row.ID, &row.CustomerID, &row.Title, &row.YoutubeURL, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) CreateCompanyVideo(ctx context.Context, customerID int64, title, youtubeURL string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO company_videos (customer_id, title, youtube_url)
VALUES ($1,$2,$3)
RETURNING id
`, customerID, title, youtubeURL).Scan(&id)
	return id, err
}

func (r *Repository) DeleteCompanyVideo(ctx context.Context, customerID, videoID int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM company_videos WHERE id=$1 AND customer_id=$2`, videoID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) CreateSalarySimulation(ctx context.Context, customerID *int64, jobCategory, region string, yearsExp int) (low, median, high int64, err error) {
	err = r.pool.QueryRow(ctx, `
SELECT low_monthly_salary, median_monthly_salary, high_monthly_salary
FROM salary_market_rates
WHERE job_category=$1 AND region=$2 AND $3 BETWEEN years_exp_min AND years_exp_max
ORDER BY years_exp_min DESC
LIMIT 1
`, jobCategory, region, yearsExp).Scan(&low, &median, &high)
	if errors.Is(err, pgx.ErrNoRows) {
		err = r.pool.QueryRow(ctx, `
SELECT low_monthly_salary, median_monthly_salary, high_monthly_salary
FROM salary_market_rates
WHERE job_category=$1
ORDER BY updated_at DESC
LIMIT 1
`, jobCategory).Scan(&low, &median, &high)
	}
	if err != nil {
		return 0, 0, 0, err
	}
	_, err = r.pool.Exec(ctx, `
INSERT INTO salary_simulations (customer_id, job_category, region, years_exp, low_monthly_salary, median_monthly_salary, high_monthly_salary)
VALUES ($1,$2,$3,$4,$5,$6,$7)
`, customerID, jobCategory, region, yearsExp, low, median, high)
	return low, median, high, err
}

func (r *Repository) CreateAIAssistLog(ctx context.Context, customerID int64, jobTitle, prompt, suggestion string) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO ai_job_assist_logs (customer_id, job_title, prompt, suggestion)
VALUES ($1,$2,$3,$4)
RETURNING id
`, customerID, jobTitle, prompt, suggestion).Scan(&id)
	return id, err
}

func (r *Repository) ListMediaConnections(ctx context.Context, customerID int64) ([]domain.MediaConnectionRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, customer_id, media_name, external_account, status, settings_json::text, updated_at
FROM media_connections
WHERE customer_id=$1
ORDER BY id DESC
`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.MediaConnectionRow
	for rows.Next() {
		var row domain.MediaConnectionRow
		if err := rows.Scan(&row.ID, &row.CustomerID, &row.MediaName, &row.ExternalAccount, &row.Status, &row.SettingsJSON, &row.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) CreateMediaConnection(ctx context.Context, customerID int64, mediaName, externalAccount, status, settingsJSON string) (int64, error) {
	var id int64
	var ext any
	if strings.TrimSpace(externalAccount) != "" {
		ext = externalAccount
	}
	err := r.pool.QueryRow(ctx, `
INSERT INTO media_connections (customer_id, media_name, external_account, status, settings_json)
VALUES ($1,$2,$3,$4,$5::jsonb)
RETURNING id
`, customerID, mediaName, ext, status, settingsJSON).Scan(&id)
	return id, err
}

func (r *Repository) UpdateMediaConnection(ctx context.Context, customerID, id int64, mediaName, externalAccount, status, settingsJSON string) error {
	var ext any
	if strings.TrimSpace(externalAccount) != "" {
		ext = externalAccount
	}
	tag, err := r.pool.Exec(ctx, `
UPDATE media_connections
SET media_name=$1, external_account=$2, status=$3, settings_json=$4::jsonb, updated_at=NOW()
WHERE id=$5 AND customer_id=$6
`, mediaName, ext, status, settingsJSON, id, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) ListInflowAnalytics(ctx context.Context, customerID int64) ([]domain.InflowAnalyticsRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT media_name, SUM(views), SUM(clicks), SUM(applications), SUM(hires)
FROM media_inflow_stats
WHERE customer_id=$1
GROUP BY media_name
ORDER BY media_name
`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.InflowAnalyticsRow
	for rows.Next() {
		var row domain.InflowAnalyticsRow
		if err := rows.Scan(&row.MediaName, &row.Views, &row.Clicks, &row.Applications, &row.Hires); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) CreateInterviewLink(ctx context.Context, row domain.InterviewLinkRow) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO interview_links (customer_id, job_posting_id, provider, meeting_url, scheduled_at)
VALUES ($1,$2,$3,$4,$5)
RETURNING id
`, row.CustomerID, row.JobPostingID, row.Provider, row.MeetingURL, row.ScheduledAt).Scan(&id)
	return id, err
}

func (r *Repository) ListInterviewLinks(ctx context.Context, customerID int64) ([]domain.InterviewLinkRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, customer_id, job_posting_id, provider, meeting_url, scheduled_at, created_at
FROM interview_links
WHERE customer_id=$1
ORDER BY created_at DESC
`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.InterviewLinkRow
	for rows.Next() {
		var row domain.InterviewLinkRow
		if err := rows.Scan(&row.ID, &row.CustomerID, &row.JobPostingID, &row.Provider, &row.MeetingURL, &row.ScheduledAt, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) ListScouts(ctx context.Context, customerID int64) ([]domain.ScoutRow, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id, customer_id, job_posting_id, candidate_name, contact, message, status, created_at, updated_at
FROM scouts
WHERE customer_id=$1
ORDER BY created_at DESC
`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ScoutRow
	for rows.Next() {
		var row domain.ScoutRow
		if err := rows.Scan(&row.ID, &row.CustomerID, &row.JobPostingID, &row.CandidateName, &row.Contact, &row.Message, &row.Status, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) CreateScout(ctx context.Context, row domain.ScoutRow) (int64, error) {
	var id int64
	err := r.pool.QueryRow(ctx, `
INSERT INTO scouts (customer_id, job_posting_id, candidate_name, contact, message, status)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id
`, row.CustomerID, row.JobPostingID, row.CandidateName, row.Contact, row.Message, row.Status).Scan(&id)
	return id, err
}

func (r *Repository) UpdateScout(ctx context.Context, customerID, scoutID int64, status, message string) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE scouts
SET status=$1, message=$2, updated_at=NOW()
WHERE id=$3 AND customer_id=$4
`, status, message, scoutID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) GetFollowUpPolicy(ctx context.Context, customerID int64) (*domain.FollowUpPolicyRow, error) {
	var row domain.FollowUpPolicyRow
	err := r.pool.QueryRow(ctx, `
SELECT customer_id, enabled, max_follow_up_days, available_by_contract, notes, updated_at
FROM follow_up_policies
WHERE customer_id=$1
`, customerID).Scan(&row.CustomerID, &row.Enabled, &row.MaxFollowUpDays, &row.AvailableByContract, &row.Notes, &row.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *Repository) UpsertFollowUpPolicy(ctx context.Context, row domain.FollowUpPolicyRow) error {
	_, err := r.pool.Exec(ctx, `
INSERT INTO follow_up_policies (customer_id, enabled, max_follow_up_days, available_by_contract, notes, updated_at)
VALUES ($1,$2,$3,$4,$5,NOW())
ON CONFLICT (customer_id) DO UPDATE SET
  enabled=EXCLUDED.enabled,
  max_follow_up_days=EXCLUDED.max_follow_up_days,
  available_by_contract=EXCLUDED.available_by_contract,
  notes=EXCLUDED.notes,
  updated_at=NOW()
`, row.CustomerID, row.Enabled, row.MaxFollowUpDays, row.AvailableByContract, row.Notes)
	return err
}

func (r *Repository) GetFollowUpCapabilities(ctx context.Context, customerID int64) (enabled bool, reason string, maxDays int, err error) {
	var tier int
	var policyEnabled bool
	var availByContract bool
	var pDays int
	err = r.pool.QueryRow(ctx, `
SELECT c.contract_tier, COALESCE(f.enabled, false), COALESCE(f.available_by_contract, true), COALESCE(f.max_follow_up_days, 0)
FROM customers c
LEFT JOIN follow_up_policies f ON f.customer_id = c.id
WHERE c.id=$1
`, customerID).Scan(&tier, &policyEnabled, &availByContract, &pDays)
	if err != nil {
		return false, "", 0, err
	}
	if !policyEnabled {
		return false, "policy_disabled", 0, nil
	}
	if !availByContract {
		return false, "contract_not_allowed", 0, nil
	}
	limit := pDays
	if tier == 1 && limit > 30 {
		limit = 30
	}
	if tier == 2 && limit > 60 {
		limit = 60
	}
	if tier == 3 && limit == 0 {
		limit = 90
	}
	return true, "ok", limit, nil
}

func (r *Repository) UpdateJobGlobalOptions(ctx context.Context, customerID, jobID int64, acceptForeigners bool, languages string) error {
	tag, err := r.pool.Exec(ctx, `
UPDATE job_postings
SET accept_foreigners=$1, supported_languages=$2, updated_at=NOW()
WHERE id=$3 AND customer_id=$4
`, acceptForeigners, languages, jobID, customerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *Repository) GetDashboardSummary(ctx context.Context, customerID int64) (map[string]int64, error) {
	out := map[string]int64{}
	var jobs int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM job_postings WHERE customer_id=$1`, customerID).Scan(&jobs); err != nil {
		return nil, err
	}
	out["jobs"] = jobs
	var apps int64
	if err := r.pool.QueryRow(ctx, `
SELECT COUNT(*)
FROM applications a
JOIN job_postings j ON j.id = a.job_posting_id
WHERE j.customer_id=$1
`, customerID).Scan(&apps); err != nil {
		return nil, err
	}
	out["applications"] = apps
	var scouts int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM scouts WHERE customer_id=$1`, customerID).Scan(&scouts); err != nil {
		return nil, err
	}
	out["scouts"] = scouts
	var interviews int64
	if err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM interview_links WHERE customer_id=$1`, customerID).Scan(&interviews); err != nil {
		return nil, err
	}
	out["interviews"] = interviews
	return out, nil
}

func (r *Repository) ListReportSnapshots(ctx context.Context, customerID int64, limit int) ([]domain.ReportSnapshotRow, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, customer_id, report_kind, period_start, period_end, payload_json::text, created_at
FROM report_snapshots
WHERE customer_id=$1
ORDER BY created_at DESC
LIMIT $2
`, customerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ReportSnapshotRow
	for rows.Next() {
		var row domain.ReportSnapshotRow
		if err := rows.Scan(&row.ID, &row.CustomerID, &row.ReportKind, &row.PeriodStart, &row.PeriodEnd, &row.PayloadJSON, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
