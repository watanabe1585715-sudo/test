package domain

import (
	"context"
	"time"
)

// StaffingRepository は求人・顧客・応募まわりの永続化の抽象（インターフェース）です。
// 実装は infrastructure/persistence/postgres にあります（依存性逆転）。
type StaffingRepository interface {
	// --- 公開求人（匿名） ---
	ListPublicJobs(ctx context.Context, q string) ([]PublicJob, error)
	GetPublicJob(ctx context.Context, id int64) (*PublicJob, error)
	CreateApplication(ctx context.Context, jobID int64, name, career, contact string) error

	// --- ログイン認証（JWT 発行前の資格情報照合） ---
	GetJobAdminByEmail(ctx context.Context, email string) (id int64, hash string, customerID int64, err error)
	// GetCustomerAdminByEmail はログイン用。registration_status も返し、承認前はパスワード検証の前に弾ける。
	GetCustomerAdminByEmail(ctx context.Context, email string) (id int64, hash string, registrationStatus string, err error)

	// --- 案件管理（job_admin） ---
	ListJobsForCustomer(ctx context.Context, customerID int64, q string) ([]JobRow, error)
	CreateJob(ctx context.Context, customerID int64, summary, requirements string, start, end time.Time) (int64, error)
	GetJobForCustomer(ctx context.Context, customerID, jobID int64) (*JobRow, error)
	UpdateJob(ctx context.Context, customerID, jobID int64, summary, requirements string, start, end time.Time) error
	DeleteJob(ctx context.Context, customerID, jobID int64) error
	ListApplicationsForJob(ctx context.Context, customerID, jobID int64) ([]ApplicationRow, error)

	// --- 顧客管理（customer_admin） ---
	ListCustomers(ctx context.Context, q string) ([]CustomerRow, error)
	GetCustomer(ctx context.Context, id int64) (*CustomerRow, error)
	CreateCustomer(ctx context.Context, name, description string, tier int, start time.Time, end *time.Time) (int64, error)
	UpdateCustomer(ctx context.Context, id int64, name, description string, tier int, start time.Time, end *time.Time, approvalStatus *string) error
	EndCustomerContract(ctx context.Context, id int64) error

	ListJobAdminUsers(ctx context.Context, customerID int64) ([]JobAdminUser, error)
	CreateJobAdminUser(ctx context.Context, customerID int64, email, passwordHash string) (int64, error)
	UpdateJobAdminUser(ctx context.Context, customerID, userID int64, email string, active bool, passwordHash *string) error
	DeleteJobAdminUser(ctx context.Context, customerID, userID int64) error

	ListInvoices(ctx context.Context, customerID *int64) ([]InvoiceRow, error)
	CreateInvoice(ctx context.Context, customerID int64, issuedAt time.Time, amountCents int64, status, notes string) (int64, error)
	GetInvoice(ctx context.Context, id int64) (*InvoiceRow, error)

	ListProspects(ctx context.Context, q string) ([]ProspectRow, error)
	GetProspect(ctx context.Context, id int64) (*ProspectRow, error)
	CreateProspect(ctx context.Context, companyName, contactInfo, notes string) (int64, error)
	UpdateProspect(ctx context.Context, id int64, companyName, contactInfo, notes string) error
	DeleteProspect(ctx context.Context, id int64) error

	ListApplicationsAll(ctx context.Context, q string, limit int) ([]ApplicationAdminRow, error)

	EnqueueEmail(ctx context.Context, kind, toEmail, subject, body string, relatedApplicationID *int64) (int64, error)
	ListEmailQueue(ctx context.Context, status string, limit int) ([]EmailOutboxRow, error)
	ListPendingEmails(ctx context.Context, limit int) ([]EmailOutboxRow, error)
	MarkEmailSent(ctx context.Context, id int64) error
	MarkEmailFailed(ctx context.Context, id int64, errDetail string) error
	ResetEmailToPending(ctx context.Context, id int64) error

	// お知らせ（channel: public | job_admin | customer_admin | all）
	ListAnnouncementsActive(ctx context.Context, channel string) ([]AnnouncementRow, error)
	ListAnnouncementsManage(ctx context.Context) ([]AnnouncementRow, error)
	GetAnnouncement(ctx context.Context, id int64) (*AnnouncementRow, error)
	CreateAnnouncement(ctx context.Context, title, body, channel string, active bool, validFrom, validTo *time.Time, sortOrder int) (int64, error)
	UpdateAnnouncement(ctx context.Context, id int64, title, body, channel string, active bool, validFrom, validTo *time.Time, sortOrder int) error
	DeleteAnnouncement(ctx context.Context, id int64) error

	ListCustomerAdminUsers(ctx context.Context) ([]CustomerAdminUserRow, error)
	GetCustomerAdminUser(ctx context.Context, id int64) (*CustomerAdminUserRow, error)
	CountActiveCustomerAdmins(ctx context.Context) (int64, error)
	CountCustomerAdminUsers(ctx context.Context) (int64, error)
	CreateCustomerAdminUser(ctx context.Context, email, passwordHash string) (int64, error)
	UpdateCustomerAdminUser(ctx context.Context, id int64, email string, active bool, passwordHash *string, registrationStatus *string) error
	DeleteCustomerAdminUser(ctx context.Context, id int64) error

	// 求職者（求人サイトログイン）
	CreateJobSeekerAccount(ctx context.Context, email, passwordHash string) (int64, error)
	GetJobSeekerByEmail(ctx context.Context, email string) (id int64, hash string, err error)
	GetJobSeekerProfile(ctx context.Context, accountID int64) (*JobSeekerProfileRow, error)
	UpdateJobSeekerProfile(ctx context.Context, accountID int64, displayName, phone, careerSummary, notes string) error
	ListJobSeekerFavorites(ctx context.Context, accountID int64) ([]JobSeekerFavoriteRow, error)
	AddJobSeekerFavorite(ctx context.Context, accountID, jobPostingID int64) error
	DeleteJobSeekerFavorite(ctx context.Context, accountID, jobPostingID int64) error
	ListJobSeekerViewHistory(ctx context.Context, accountID int64, limit int) ([]JobViewHistoryRow, error)
	AddJobSeekerViewHistory(ctx context.Context, accountID, jobPostingID int64) error

	// 顧客イベント（打ち合わせ・利用開始・特記・リスク等）
	ListCustomerEvents(ctx context.Context, customerID int64) ([]CustomerEventRow, error)
	CreateCustomerEvent(ctx context.Context, customerID int64, eventKind string, occurredAt time.Time, title, body string, isRiskRelated bool) (int64, error)
	UpdateCustomerEvent(ctx context.Context, customerID, eventID int64, eventKind string, occurredAt time.Time, title, body string, isRiskRelated bool) error
	DeleteCustomerEvent(ctx context.Context, customerID, eventID int64) error

	// 追加機能（企業詳細、媒体連携、分析、スカウト、フォローアップ等）
	GetCompanyProfile(ctx context.Context, customerID int64) (*CompanyProfileRow, error)
	UpsertCompanyProfile(ctx context.Context, row CompanyProfileRow) error
	ListCompanyReviews(ctx context.Context, customerID int64) ([]CompanyReviewRow, error)
	ListCompanyVideos(ctx context.Context, customerID int64) ([]CompanyVideoRow, error)
	CreateCompanyVideo(ctx context.Context, customerID int64, title, youtubeURL string) (int64, error)
	DeleteCompanyVideo(ctx context.Context, customerID, videoID int64) error

	CreateSalarySimulation(ctx context.Context, customerID *int64, jobCategory, region string, yearsExp int) (low, median, high int64, err error)
	CreateAIAssistLog(ctx context.Context, customerID int64, jobTitle, prompt, suggestion string) (int64, error)

	ListMediaConnections(ctx context.Context, customerID int64) ([]MediaConnectionRow, error)
	CreateMediaConnection(ctx context.Context, customerID int64, mediaName, externalAccount, status, settingsJSON string) (int64, error)
	UpdateMediaConnection(ctx context.Context, customerID, id int64, mediaName, externalAccount, status, settingsJSON string) error
	ListInflowAnalytics(ctx context.Context, customerID int64) ([]InflowAnalyticsRow, error)

	CreateInterviewLink(ctx context.Context, row InterviewLinkRow) (int64, error)
	ListInterviewLinks(ctx context.Context, customerID int64) ([]InterviewLinkRow, error)

	ListScouts(ctx context.Context, customerID int64) ([]ScoutRow, error)
	CreateScout(ctx context.Context, row ScoutRow) (int64, error)
	UpdateScout(ctx context.Context, customerID, scoutID int64, status, message string) error

	GetFollowUpPolicy(ctx context.Context, customerID int64) (*FollowUpPolicyRow, error)
	UpsertFollowUpPolicy(ctx context.Context, row FollowUpPolicyRow) error
	GetFollowUpCapabilities(ctx context.Context, customerID int64) (enabled bool, reason string, maxDays int, err error)

	UpdateJobGlobalOptions(ctx context.Context, customerID, jobID int64, acceptForeigners bool, languages string) error
	GetDashboardSummary(ctx context.Context, customerID int64) (map[string]int64, error)
	ListReportSnapshots(ctx context.Context, customerID int64, limit int) ([]ReportSnapshotRow, error)
}
