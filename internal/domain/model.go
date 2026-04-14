// Package domain は DDD の「ドメイン層」です。エンティティの形と、永続化のための契約（リポジトリIF）を置きます。
//
// 初心者向け:
//   - ここには「ビジネスの言葉で何を扱うか」（求人・顧客・応募など）のデータ形をまとめます。
//   - データベースの具体的な SQL は domain では書きません（infrastructure 層の仕事です）。
//   - フレームワーク（Gin や PostgreSQL）に依存しない純粋な Go の型だけにします。
package domain

import "time"

// PublicJob は求職者向けに公開する案件の読み取りモデル。
type PublicJob struct {
	ID           int64     `json:"id"`
	Summary      string    `json:"summary"`
	Requirements string    `json:"requirements"`
	PublishStart time.Time `json:"publish_start"`
	PublishEnd   time.Time `json:"publish_end"`
}

// JobRow は管理画面向けの案件1行。
type JobRow struct {
	ID                int64     `json:"id"`
	CustomerID        int64     `json:"customer_id"`
	Summary           string    `json:"summary"`
	Requirements      string    `json:"requirements"`
	PublishStart      time.Time `json:"publish_start"`
	PublishEnd        time.Time `json:"publish_end"`
	PublicationStatus string    `json:"publication_status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ApplicationRow は応募1件。
type ApplicationRow struct {
	ID            int64     `json:"id"`
	JobPostingID  int64     `json:"job_posting_id"`
	ApplicantName string    `json:"applicant_name"`
	CareerSummary string    `json:"career_summary"`
	Contact       string    `json:"contact"`
	CreatedAt     time.Time `json:"created_at"`
}

// CustomerRow は顧客1件。
type CustomerRow struct {
	ID             int64      `json:"id"`
	Name           string     `json:"name"`
	Description    *string    `json:"description,omitempty"`
	ContractTier   int        `json:"contract_tier"`
	ContractStart  time.Time  `json:"contract_start"`
	ContractEnd    *time.Time `json:"contract_end,omitempty"`
	Status         string     `json:"status"`
	ApprovalStatus string     `json:"approval_status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// JobAdminUser は案件管理サイト用ユーザー。
type JobAdminUser struct {
	ID         int64  `json:"id"`
	CustomerID int64  `json:"customer_id"`
	Email      string `json:"email"`
	Active     bool   `json:"active"`
}

// InvoiceRow は請求1件。
type InvoiceRow struct {
	ID          int64     `json:"id"`
	CustomerID  int64     `json:"customer_id"`
	IssuedAt    time.Time `json:"issued_at"`
	AmountCents int64     `json:"amount_cents"`
	Status      string    `json:"status"`
	Notes       *string   `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// ProspectRow は見込み顧客1件。
type ProspectRow struct {
	ID          int64     `json:"id"`
	CompanyName string    `json:"company_name"`
	ContactInfo *string   `json:"contact_info,omitempty"`
	Notes       *string   `json:"notes,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// EmailOutboxRow は送信待ち／送信済みメール1件（管理画面の一覧用）。
type EmailOutboxRow struct {
	ID                   int64      `json:"id"`
	Kind                 string     `json:"kind"`
	ToEmail              string     `json:"to_email"`
	Subject              string     `json:"subject"`
	Body                 string     `json:"body"`
	Status               string     `json:"status"`
	ErrorDetail          *string    `json:"error_detail,omitempty"`
	RelatedApplicationID *int64     `json:"related_application_id,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	SentAt               *time.Time `json:"sent_at,omitempty"`
}

// ApplicationAdminRow は顧客管理側で見る「全応募」1行（案件名・顧客名付き）。
type ApplicationAdminRow struct {
	ID            int64     `json:"id"`
	JobPostingID  int64     `json:"job_posting_id"`
	JobSummary    string    `json:"job_summary"`
	CustomerID    int64     `json:"customer_id"`
	CustomerName  string    `json:"customer_name"`
	ApplicantName string    `json:"applicant_name"`
	CareerSummary string    `json:"career_summary"`
	Contact       string    `json:"contact"`
	CreatedAt     time.Time `json:"created_at"`
}

// AnnouncementRow はお知らせ1件（管理画面・各サイトのトップ表示用）。
type AnnouncementRow struct {
	ID        int64      `json:"id"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	Channel   string     `json:"channel"`
	Active    bool       `json:"active"`
	ValidFrom *time.Time `json:"valid_from,omitempty"`
	ValidTo   *time.Time `json:"valid_to,omitempty"`
	SortOrder int        `json:"sort_order"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CustomerAdminUserRow は顧客管理サイト用ログインユーザ（一覧表示。パスワードは含めない）。
type CustomerAdminUserRow struct {
	ID                 int64     `json:"id"`
	Email              string    `json:"email"`
	Active             bool      `json:"active"`
	RegistrationStatus string    `json:"registration_status"`
	CreatedAt          time.Time `json:"created_at"`
}

// JobSeekerProfileRow は求人サイト求職者のマイページ用プロフィール（アカウントと JOIN）。
type JobSeekerProfileRow struct {
	AccountID     int64     `json:"account_id"`
	Email         string    `json:"email"`
	DisplayName   string    `json:"display_name"`
	Phone         string    `json:"phone"`
	CareerSummary string    `json:"career_summary"`
	Notes         string    `json:"notes"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CustomerEventRow は顧客に紐づく打ち合わせ・契約開始・特記・リスク関連などの履歴1件。
type CustomerEventRow struct {
	ID            int64     `json:"id"`
	CustomerID    int64     `json:"customer_id"`
	EventKind     string    `json:"event_kind"`
	OccurredAt    time.Time `json:"occurred_at"`
	Title         string    `json:"title"`
	Body          *string   `json:"body,omitempty"`
	IsRiskRelated bool      `json:"is_risk_related"`
	CreatedAt     time.Time `json:"created_at"`
}

// JobSeekerFavoriteRow は求職者のお気に入り求人。
type JobSeekerFavoriteRow struct {
	JobPostingID int64     `json:"job_posting_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// JobViewHistoryRow は求職者の求人閲覧履歴。
type JobViewHistoryRow struct {
	ID           int64     `json:"id"`
	JobPostingID int64     `json:"job_posting_id"`
	ViewedAt     time.Time `json:"viewed_at"`
}

// CompanyProfileRow は企業詳細ページで表示する会社プロフィール。
type CompanyProfileRow struct {
	CustomerID       int64     `json:"customer_id"`
	CompanyName      string    `json:"company_name"`
	Description      string    `json:"description"`
	Address          string    `json:"address"`
	GoogleMapURL     *string   `json:"google_map_url,omitempty"`
	WebsiteURL       *string   `json:"website_url,omitempty"`
	YoutubeEmbedURL  *string   `json:"youtube_embed_url,omitempty"`
	AcceptForeigners bool      `json:"accept_foreigners"`
	Languages        string    `json:"languages"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// CompanyReviewRow は企業口コミ。
type CompanyReviewRow struct {
	ID         int64     `json:"id"`
	CustomerID int64     `json:"customer_id"`
	Reviewer   string    `json:"reviewer"`
	Rating     int       `json:"rating"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

// CompanyVideoRow は企業の YouTube 動画。
type CompanyVideoRow struct {
	ID         int64     `json:"id"`
	CustomerID int64     `json:"customer_id"`
	Title      string    `json:"title"`
	YoutubeURL string    `json:"youtube_url"`
	CreatedAt  time.Time `json:"created_at"`
}

// MediaConnectionRow は求人媒体（Indeed 等）連携設定。
type MediaConnectionRow struct {
	ID              int64     `json:"id"`
	CustomerID      int64     `json:"customer_id"`
	MediaName       string    `json:"media_name"`
	ExternalAccount *string   `json:"external_account,omitempty"`
	Status          string    `json:"status"`
	SettingsJSON    string    `json:"settings_json"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// InflowAnalyticsRow は媒体別流入集計1行。
type InflowAnalyticsRow struct {
	MediaName    string `json:"media_name"`
	Views        int64  `json:"views"`
	Clicks       int64  `json:"clicks"`
	Applications int64  `json:"applications"`
	Hires        int64  `json:"hires"`
}

// InterviewLinkRow は Web 面談リンク情報。
type InterviewLinkRow struct {
	ID           int64      `json:"id"`
	CustomerID   int64      `json:"customer_id"`
	JobPostingID *int64     `json:"job_posting_id,omitempty"`
	Provider     string     `json:"provider"`
	MeetingURL   string     `json:"meeting_url"`
	ScheduledAt  *time.Time `json:"scheduled_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

// ScoutRow はスカウト送信情報。
type ScoutRow struct {
	ID            int64     `json:"id"`
	CustomerID    int64     `json:"customer_id"`
	JobPostingID  *int64    `json:"job_posting_id,omitempty"`
	CandidateName string    `json:"candidate_name"`
	Contact       string    `json:"contact"`
	Message       string    `json:"message"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// FollowUpPolicyRow は入社後フォローアップ設定（契約レベル依存）。
type FollowUpPolicyRow struct {
	CustomerID          int64     `json:"customer_id"`
	Enabled             bool      `json:"enabled"`
	MaxFollowUpDays     int       `json:"max_follow_up_days"`
	AvailableByContract bool      `json:"available_by_contract"`
	Notes               string    `json:"notes"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ReportSnapshotRow はダッシュボード/レポート用スナップショット。
type ReportSnapshotRow struct {
	ID          int64     `json:"id"`
	CustomerID  int64     `json:"customer_id"`
	ReportKind  string    `json:"report_kind"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	PayloadJSON string    `json:"payload_json"`
	CreatedAt   time.Time `json:"created_at"`
}
