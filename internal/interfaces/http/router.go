package api

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"recruitment/internal/auth"
	"recruitment/internal/usecase"
)

// NewEngine は CORS・ミドルウェア・全ルートを登録した *gin.Engine を返す。
// corsOrigins が空のときは Default() の挙動に任せる（開発時は * に近い設定のため本番では必ず指定すること）。
func NewEngine(app *usecase.StaffingApp, corsOrigins []string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	cfg := cors.Config{
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300 * time.Second,
	}
	if len(corsOrigins) == 0 {
		cfg.AllowAllOrigins = true
	} else {
		cfg.AllowOrigins = corsOrigins
	}
	r.Use(cors.New(cfg))

	h := &Handlers{App: app}

	// GET /health → health（プロセス生存確認）
	r.GET("/health", h.health)

	// --- 公開 API（匿名）: 求人サイトの案件一覧・応募・お知らせ ---
	pub := r.Group("/public")
	{
		pub.GET("/announcements", h.listAnnouncementsPublic)          // API名: 公開お知らせ一覧取得 / handler: listAnnouncementsPublic
		pub.GET("/jobs", h.listPublicJobs)                            // API名: 公開求人一覧取得 / handler: listPublicJobs
		pub.GET("/jobs/:id", h.getPublicJob)                          // API名: 公開求人詳細取得 / handler: getPublicJob
		pub.POST("/applications", h.createApplication)                // API名: 求人応募作成 / handler: createApplication
		pub.GET("/companies/:id", h.publicGetCompany)                 // API名: 企業詳細取得 / handler: publicGetCompany
		pub.GET("/companies/:id/reviews", h.publicListCompanyReviews) // API名: 企業口コミ一覧取得 / handler: publicListCompanyReviews
		pub.GET("/companies/:id/videos", h.publicListCompanyVideos)   // API名: 企業紹介動画一覧取得 / handler: publicListCompanyVideos
		pub.POST("/salary-simulations", h.publicSalarySimulation)     // API名: 給与相場試算 / handler: publicSalarySimulation
	}

	// --- 求職者（求人サイト React）: 登録・ログインは匿名。マイページは JWT（role=job_seeker）---
	r.POST("/job-seeker/register", h.jobSeekerRegister) // API名: 求職者アカウント登録 / handler: jobSeekerRegister
	r.POST("/job-seeker/login", h.jobSeekerLogin)       // API名: 求職者ログイン / handler: jobSeekerLogin
	jobSeeker := r.Group("/job-seeker")
	jobSeeker.Use(requireRole(app, auth.RoleJobSeeker))
	{
		jobSeeker.GET("/profile", h.jobSeekerGetProfile)                 // API名: 求職者プロフィール取得 / handler: jobSeekerGetProfile
		jobSeeker.PATCH("/profile", h.jobSeekerUpdateProfile)            // API名: 求職者プロフィール更新 / handler: jobSeekerUpdateProfile
		jobSeeker.GET("/favorites", h.jobSeekerListFavorites)            // API名: 求人お気に入り一覧取得 / handler: jobSeekerListFavorites
		jobSeeker.POST("/favorites", h.jobSeekerAddFavorite)             // API名: 求人お気に入り追加 / handler: jobSeekerAddFavorite
		jobSeeker.DELETE("/favorites/:jobId", h.jobSeekerDeleteFavorite) // API名: 求人お気に入り削除 / handler: jobSeekerDeleteFavorite
		jobSeeker.GET("/history", h.jobSeekerListHistory)                // API名: 求人閲覧履歴一覧取得 / handler: jobSeekerListHistory
		jobSeeker.POST("/history", h.jobSeekerAddHistory)                // API名: 求人閲覧履歴追加 / handler: jobSeekerAddHistory
	}

	// --- 案件管理サイト（Vue）: JWT role=job_admin、cid に顧客 ID ---
	jobAdmin := r.Group("/admin/jobs")
	{
		jobAdmin.POST("/login", h.jobAdminLogin) // API名: 案件管理ログイン / handler: jobAdminLogin
		ja := jobAdmin.Group("")
		ja.Use(requireRole(app, auth.RoleJobAdmin))
		{
			ja.GET("/jobs", h.jobAdminListJobs)                                    // API名: 自社案件一覧取得 / handler: jobAdminListJobs
			ja.POST("/jobs", h.jobAdminCreateJob)                                  // API名: 自社案件作成 / handler: jobAdminCreateJob
			ja.GET("/jobs/:id", h.jobAdminGetJob)                                  // API名: 自社案件詳細取得 / handler: jobAdminGetJob
			ja.PATCH("/jobs/:id", h.jobAdminUpdateJob)                             // API名: 自社案件更新 / handler: jobAdminUpdateJob
			ja.DELETE("/jobs/:id", h.jobAdminDeleteJob)                            // API名: 自社案件削除 / handler: jobAdminDeleteJob
			ja.GET("/jobs/:id/applications", h.jobAdminListApplications)           // API名: 自社案件応募一覧取得 / handler: jobAdminListApplications
			ja.GET("/announcements", h.listAnnouncementsJobAdminFeed)              // API名: 案件管理お知らせ一覧取得 / handler: listAnnouncementsJobAdminFeed
			ja.GET("/company-profile", h.jobAdminGetCompanyProfile)                // API名: 自社プロフィール取得 / handler: jobAdminGetCompanyProfile
			ja.PATCH("/company-profile", h.jobAdminUpdateCompanyProfile)           // API名: 自社プロフィール更新 / handler: jobAdminUpdateCompanyProfile
			ja.POST("/ai/job-assist", h.jobAdminAIAssist)                          // API名: 職種説明AI補助生成 / handler: jobAdminAIAssist
			ja.GET("/media-connections", h.jobAdminListMediaConnections)           // API名: 媒体連携一覧取得 / handler: jobAdminListMediaConnections
			ja.POST("/media-connections", h.jobAdminCreateMediaConnection)         // API名: 媒体連携作成 / handler: jobAdminCreateMediaConnection
			ja.PATCH("/media-connections/:id", h.jobAdminUpdateMediaConnection)    // API名: 媒体連携更新 / handler: jobAdminUpdateMediaConnection
			ja.GET("/dashboard", h.jobAdminDashboard)                              // API名: ダッシュボード集計取得 / handler: jobAdminDashboard
			ja.GET("/reports", h.jobAdminReports)                                  // API名: レポート一覧取得 / handler: jobAdminReports
			ja.GET("/analytics/inflow", h.jobAdminInflowAnalytics)                 // API名: 媒体別流入集計取得 / handler: jobAdminInflowAnalytics
			ja.POST("/interviews/links", h.jobAdminCreateInterviewLink)            // API名: 面談URL発行 / handler: jobAdminCreateInterviewLink
			ja.GET("/interviews", h.jobAdminListInterviews)                        // API名: 面談予定一覧取得 / handler: jobAdminListInterviews
			ja.GET("/scouts", h.jobAdminListScouts)                                // API名: スカウト一覧取得 / handler: jobAdminListScouts
			ja.POST("/scouts", h.jobAdminCreateScout)                              // API名: スカウト作成 / handler: jobAdminCreateScout
			ja.PATCH("/scouts/:id", h.jobAdminUpdateScout)                         // API名: スカウト更新 / handler: jobAdminUpdateScout
			ja.POST("/company-videos", h.jobAdminCreateCompanyVideo)               // API名: 企業動画登録 / handler: jobAdminCreateCompanyVideo
			ja.DELETE("/company-videos/:id", h.jobAdminDeleteCompanyVideo)         // API名: 企業動画削除 / handler: jobAdminDeleteCompanyVideo
			ja.GET("/follow-up-policy", h.jobAdminGetFollowUpPolicy)               // API名: フォローアップ設定取得 / handler: jobAdminGetFollowUpPolicy
			ja.PATCH("/follow-up-policy", h.jobAdminUpdateFollowUpPolicy)          // API名: フォローアップ設定更新 / handler: jobAdminUpdateFollowUpPolicy
			ja.PATCH("/jobs/:id/global-options", h.jobAdminUpdateJobGlobalOptions) // API名: 求人多言語条件更新 / handler: jobAdminUpdateJobGlobalOptions
			ja.POST("/salary-simulations", h.jobAdminSalarySimulation)             // API名: 企業向け給与相場試算 / handler: jobAdminSalarySimulation
		}
	}

	// --- 顧客管理サイト（Next）: JWT role=customer_admin。顧客・請求・承認・イベント履歴など ---
	cust := r.Group("/admin/customers")
	{
		cust.POST("/login", h.customerAdminLogin) // API名: 顧客管理ログイン / handler: customerAdminLogin
		ca := cust.Group("")
		ca.Use(requireRole(app, auth.RoleCustomerAdmin))
		{
			ca.GET("/announcements/feed", h.announcementsFeedCustomer)        // API名: 顧客管理お知らせフィード取得 / handler: announcementsFeedCustomer
			ca.GET("/announcements", h.listAnnouncementsManage)               // API名: 顧客管理お知らせ一覧取得 / handler: listAnnouncementsManage
			ca.POST("/announcements", h.createAnnouncement)                   // API名: 顧客管理お知らせ作成 / handler: createAnnouncement
			ca.GET("/announcements/:id", h.getAnnouncement)                   // API名: 顧客管理お知らせ詳細取得 / handler: getAnnouncement
			ca.PATCH("/announcements/:id", h.updateAnnouncement)              // API名: 顧客管理お知らせ更新 / handler: updateAnnouncement
			ca.DELETE("/announcements/:id", h.deleteAnnouncement)             // API名: 顧客管理お知らせ削除 / handler: deleteAnnouncement
			ca.GET("/customer-admins", h.listCustomerAdmins)                  // API名: 顧客管理管理者一覧取得 / handler: listCustomerAdmins
			ca.POST("/customer-admins", h.createCustomerAdmin)                // API名: 顧客管理管理者作成 / handler: createCustomerAdmin（新規は registration_status=pending）
			ca.PATCH("/customer-admins/:adminUserId", h.updateCustomerAdmin)  // API名: 顧客管理管理者更新 / handler: updateCustomerAdmin（承認は registration_status）
			ca.DELETE("/customer-admins/:adminUserId", h.deleteCustomerAdmin) // API名: 顧客管理管理者削除 / handler: deleteCustomerAdmin
			ca.GET("/customers", h.customerList)                              // API名: 顧客一覧取得 / handler: customerList
			ca.POST("/customers", h.customerCreate)                           // API名: 顧客作成 / handler: customerCreate（新規顧客は approval_status=pending）
			// 顧客に紐づく打ち合わせ・契約開始・特記・リスク等（:id は顧客 ID）
			ca.GET("/customers/:id/events", h.listCustomerEvents)                              // API名: 顧客イベント一覧取得 / handler: listCustomerEvents
			ca.POST("/customers/:id/events", h.createCustomerEvent)                            // API名: 顧客イベント作成 / handler: createCustomerEvent
			ca.PATCH("/customers/:id/events/:eventId", h.updateCustomerEvent)                  // API名: 顧客イベント更新 / handler: updateCustomerEvent
			ca.DELETE("/customers/:id/events/:eventId", h.deleteCustomerEvent)                 // API名: 顧客イベント削除 / handler: deleteCustomerEvent
			ca.GET("/customers/:id", h.customerGet)                                            // API名: 顧客詳細取得 / handler: customerGet
			ca.PATCH("/customers/:id", h.customerUpdate)                                       // API名: 顧客更新 / handler: customerUpdate（任意で approval_status）
			ca.GET("/customers/:id/follow-up-capabilities", h.customerGetFollowUpCapabilities) // API名: 顧客フォローアップ利用可否判定取得 / handler: customerGetFollowUpCapabilities
			ca.POST("/customers/:id/end-contract", h.customerEndContract)                      // API名: 顧客契約終了 / handler: customerEndContract
			ca.GET("/customers/:id/job-users", h.listJobUsers)                                 // API名: 案件管理ユーザー一覧取得 / handler: listJobUsers
			ca.POST("/customers/:id/job-users", h.createJobUser)                               // API名: 案件管理ユーザー作成 / handler: createJobUser
			ca.PATCH("/job-users/:userId", h.updateJobUser)                                    // API名: 案件管理ユーザー更新 / handler: updateJobUser（クエリ customer_id）
			ca.DELETE("/job-users/:userId", h.deleteJobUser)                                   // API名: 案件管理ユーザー削除 / handler: deleteJobUser（クエリ customer_id）
			ca.GET("/invoices", h.listInvoices)                                                // API名: 請求一覧取得 / handler: listInvoices
			ca.POST("/invoices", h.createInvoice)                                              // API名: 請求作成 / handler: createInvoice
			ca.GET("/invoices/:id", h.getInvoice)                                              // API名: 請求詳細取得 / handler: getInvoice
			ca.GET("/applications", h.adminListApplications)                                   // API名: 応募一覧取得（全顧客） / handler: adminListApplications
			ca.GET("/email-queue", h.listEmailQueue)                                           // API名: メールキュー一覧取得 / handler: listEmailQueue
			ca.POST("/email-queue", h.enqueueManualEmail)                                      // API名: メールキュー作成 / handler: enqueueManualEmail
			ca.POST("/email-queue/:id/retry", h.retryEmailOutbox)                              // API名: メール再送待ち戻し / handler: retryEmailOutbox
			ca.POST("/prospects", h.createProspect)                                            // API名: 見込み顧客作成 / handler: createProspect
			ca.PATCH("/prospects/:id", h.updateProspect)                                       // API名: 見込み顧客更新 / handler: updateProspect
			ca.DELETE("/prospects/:id", h.deleteProspect)                                      // API名: 見込み顧客削除 / handler: deleteProspect
			ca.GET("/prospects", h.listProspects)                                              // API名: 見込み顧客一覧取得 / handler: listProspects
			ca.GET("/prospects/:id", h.getProspect)                                            // API名: 見込み顧客詳細取得 / handler: getProspect
		}
	}

	return r
}
