// このファイルは Gin 用の HTTP ハンドラです。URL ごとに JSON の形とステータスコードを整えます。
// 各ハンドラがどの URL に載るかは router.go のコメント（関数名併記）と対応させて読むと追いやすいです。
//
// 初心者向けメモ:
//   - 各メソッドは「1 種類の API」に対応します（一覧・ログイン・更新など）。
//   - 流れは (1) クエリや JSON を読む (2) usecase / domain のリポジトリで DB に問い合わせる (3) c.JSON で返す、です。
//   - 認証が必要なグループでは router.go のミドルウェアが先に JWT を検査し、claimsFromGin で中身を取れます。
package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"recruitment/internal/auth"
	"recruitment/internal/domain"
	"recruitment/internal/usecase"
)

// Handlers はユースケースへの参照を持ち、Gin のハンドラメソッドのレシーバになる。
type Handlers struct {
	App *usecase.StaffingApp
}

func (h *Handlers) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// listPublicJobs は GET /public/jobs?q= 。一覧が nil のときは空配列にしてフロントが扱いやすくする。
func (h *Handlers) listPublicJobs(c *gin.Context) {
	q := c.Query("q")
	list, err := h.App.Repo.ListPublicJobs(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []domain.PublicJob{}
	}
	c.JSON(http.StatusOK, gin.H{"jobs": list})
}

func (h *Handlers) getPublicJob(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	j, err := h.App.Repo.GetPublicJob(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if j == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, j)
}

type applicationBody struct {
	JobPostingID  int64  `json:"job_posting_id"`
	ApplicantName string `json:"applicant_name"`
	CareerSummary string `json:"career_summary"`
	Contact       string `json:"contact"`
}

func (h *Handlers) createApplication(c *gin.Context) {
	var b applicationBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	// 必須フィールドのどれかが欠けていれば 400（個別メッセージに分けてもよい）。
	if b.JobPostingID == 0 || b.ApplicantName == "" || b.CareerSummary == "" || b.Contact == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing fields"})
		return
	}
	err := h.App.Repo.CreateApplication(c.Request.Context(), b.JobPostingID, b.ApplicantName, b.CareerSummary, b.Contact)
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not accepting applications"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "ok"})
}

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// jobAdminLogin は POST /admin/jobs/login 。認証ロジック本体は usecase に寄せている。
func (h *Handlers) jobAdminLogin(c *gin.Context) {
	var b loginBody
	if err := c.ShouldBindJSON(&b); err != nil || b.Email == "" || b.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	tok, cid, err := h.App.JobAdminLogin(c.Request.Context(), b.Email, b.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok, "customer_id": cid})
}

func (h *Handlers) jobAdminListJobs(c *gin.Context) {
	cl := claimsFromGin(c)
	q := c.Query("q")
	list, err := h.App.Repo.ListJobsForCustomer(c.Request.Context(), cl.CustomerID, q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"jobs": list})
}

type jobBody struct {
	Summary      string `json:"summary"`
	Requirements string `json:"requirements"`
	PublishStart string `json:"publish_start"`
	PublishEnd   string `json:"publish_end"`
}

func parseDates(b jobBody) (time.Time, time.Time, error) {
	ps, err := time.Parse("2006-01-02", b.PublishStart)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	pe, err := time.Parse("2006-01-02", b.PublishEnd)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return ps, pe, nil
}

func (h *Handlers) jobAdminCreateJob(c *gin.Context) {
	var b jobBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	ps, pe, err := parseDates(b)
	if err != nil || b.Summary == "" || b.Requirements == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	cl := claimsFromGin(c)
	id, err := h.App.Repo.CreateJob(c.Request.Context(), cl.CustomerID, b.Summary, b.Requirements, ps, pe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) jobAdminGetJob(c *gin.Context) {
	cl := claimsFromGin(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	j, err := h.App.Repo.GetJobForCustomer(c.Request.Context(), cl.CustomerID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if j == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, j)
}

func (h *Handlers) jobAdminUpdateJob(c *gin.Context) {
	cl := claimsFromGin(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var b jobBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	ps, pe, err := parseDates(b)
	if err != nil || b.Summary == "" || b.Requirements == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	if err := h.App.Repo.UpdateJob(c.Request.Context(), cl.CustomerID, id, b.Summary, b.Requirements, ps, pe); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) jobAdminDeleteJob(c *gin.Context) {
	cl := claimsFromGin(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.App.Repo.DeleteJob(c.Request.Context(), cl.CustomerID, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) jobAdminListApplications(c *gin.Context) {
	cl := claimsFromGin(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	list, err := h.App.Repo.ListApplicationsForJob(c.Request.Context(), cl.CustomerID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"applications": list})
}

func (h *Handlers) customerAdminLogin(c *gin.Context) {
	var b loginBody
	if err := c.ShouldBindJSON(&b); err != nil || b.Email == "" || b.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	tok, err := h.App.CustomerAdminLogin(c.Request.Context(), b.Email, b.Password)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		if errors.Is(err, usecase.ErrCustomerAdminPending) {
			c.JSON(http.StatusForbidden, gin.H{"error": "registration_pending", "message": "管理者による承認待ちです"})
			return
		}
		if errors.Is(err, usecase.ErrCustomerAdminRejected) {
			c.JSON(http.StatusForbidden, gin.H{"error": "registration_rejected", "message": "このアカウントは承認されませんでした"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok})
}

func (h *Handlers) customerList(c *gin.Context) {
	q := c.Query("q")
	list, err := h.App.Repo.ListCustomers(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"customers": list})
}

type customerBody struct {
	Name           string  `json:"name"`
	Description    string  `json:"description"`
	ContractTier   int     `json:"contract_tier"`
	ContractStart  string  `json:"contract_start"`
	ContractEnd    *string `json:"contract_end"`
	ApprovalStatus *string `json:"approval_status"`
}

func parseCustomerDates(b customerBody) (time.Time, *time.Time, error) {
	st, err := time.Parse("2006-01-02", b.ContractStart)
	if err != nil {
		return time.Time{}, nil, err
	}
	var endPtr *time.Time
	// contract_end が JSON で null または空文字のときは「終了日なし」とみなす。
	if b.ContractEnd != nil && *b.ContractEnd != "" {
		t, err := time.Parse("2006-01-02", *b.ContractEnd)
		if err != nil {
			return time.Time{}, nil, err
		}
		endPtr = &t
	}
	return st, endPtr, nil
}

func (h *Handlers) customerCreate(c *gin.Context) {
	var b customerBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	st, endPtr, err := parseCustomerDates(b)
	if err != nil || b.Name == "" || b.ContractTier < 1 || b.ContractTier > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	id, err := h.App.Repo.CreateCustomer(c.Request.Context(), b.Name, b.Description, b.ContractTier, st, endPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) customerGet(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	cust, err := h.App.Repo.GetCustomer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if cust == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, cust)
}

func (h *Handlers) customerUpdate(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var b customerBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	st, endPtr, err := parseCustomerDates(b)
	if err != nil || b.Name == "" || b.ContractTier < 1 || b.ContractTier > 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	var appr *string
	if b.ApprovalStatus != nil && *b.ApprovalStatus != "" {
		switch *b.ApprovalStatus {
		case "pending", "approved", "rejected":
			appr = b.ApprovalStatus
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid approval_status"})
			return
		}
	}
	if err := h.App.Repo.UpdateCustomer(c.Request.Context(), id, b.Name, b.Description, b.ContractTier, st, endPtr, appr); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) customerEndContract(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.App.Repo.EndCustomerContract(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) listJobUsers(c *gin.Context) {
	cid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	list, err := h.App.Repo.ListJobAdminUsers(c.Request.Context(), cid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": list})
}

type jobUserBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handlers) createJobUser(c *gin.Context) {
	cid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var b jobUserBody
	if err := c.ShouldBindJSON(&b); err != nil || b.Email == "" || b.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	hash, err := auth.HashPassword(b.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, err := h.App.Repo.CreateJobAdminUser(c.Request.Context(), cid, b.Email, hash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

type jobUserPatchBody struct {
	Email    string `json:"email"`
	Active   bool   `json:"active"`
	Password string `json:"password"`
}

func (h *Handlers) updateJobUser(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	cidStr := c.Query("customer_id")
	if cidStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id query required"})
		return
	}
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id"})
		return
	}
	var b jobUserPatchBody
	if err := c.ShouldBindJSON(&b); err != nil || b.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var hashPtr *string
	if b.Password != "" {
		hh, err := auth.HashPassword(b.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		hashPtr = &hh
	}
	if err := h.App.Repo.UpdateJobAdminUser(c.Request.Context(), cid, uid, b.Email, b.Active, hashPtr); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) deleteJobUser(c *gin.Context) {
	uid, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	cidStr := c.Query("customer_id")
	cid, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id"})
		return
	}
	if err := h.App.Repo.DeleteJobAdminUser(c.Request.Context(), cid, uid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) listInvoices(c *gin.Context) {
	var cid *int64
	if v := c.Query("customer_id"); v != "" {
		x, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer_id"})
			return
		}
		cid = &x
	}
	list, err := h.App.Repo.ListInvoices(c.Request.Context(), cid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"invoices": list})
}

type invoiceBody struct {
	CustomerID  int64  `json:"customer_id"`
	IssuedAt    string `json:"issued_at"`
	AmountCents int64  `json:"amount_cents"`
	Status      string `json:"status"`
	Notes       string `json:"notes"`
}

func (h *Handlers) createInvoice(c *gin.Context) {
	var b invoiceBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	d, err := time.Parse("2006-01-02", b.IssuedAt)
	if err != nil || b.CustomerID == 0 || b.AmountCents <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	st := b.Status
	if st == "" {
		st = "draft"
	}
	id, err := h.App.Repo.CreateInvoice(c.Request.Context(), b.CustomerID, d, b.AmountCents, st, b.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) getInvoice(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	inv, err := h.App.Repo.GetInvoice(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if inv == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, inv)
}

func (h *Handlers) listProspects(c *gin.Context) {
	q := c.Query("q")
	list, err := h.App.Repo.ListProspects(c.Request.Context(), q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"prospects": list})
}

func (h *Handlers) getProspect(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	p, err := h.App.Repo.GetProspect(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

// adminListApplications は全顧客横断の応募一覧（顧客管理サイト用）。
func (h *Handlers) adminListApplications(c *gin.Context) {
	q := c.Query("q")
	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	list, err := h.App.Repo.ListApplicationsAll(c.Request.Context(), q, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"applications": list})
}

func (h *Handlers) listEmailQueue(c *gin.Context) {
	status := c.Query("status")
	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	list, err := h.App.Repo.ListEmailQueue(c.Request.Context(), status, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"emails": list})
}

type manualEmailBody struct {
	ToEmail string `json:"to_email"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// enqueueManualEmail は管理画面から任意メールをキューに積む（送信は mailworker）。
func (h *Handlers) enqueueManualEmail(c *gin.Context) {
	var b manualEmailBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if b.ToEmail == "" || !strings.Contains(b.ToEmail, "@") || b.Subject == "" || b.Body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	id, err := h.App.Repo.EnqueueEmail(c.Request.Context(), "manual", b.ToEmail, b.Subject, b.Body, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// retryEmailOutbox は failed を pending に戻し、次回ワーカーで再送できるようにする。
func (h *Handlers) retryEmailOutbox(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.App.Repo.ResetEmailToPending(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found or not failed"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type prospectWriteBody struct {
	CompanyName string `json:"company_name"`
	ContactInfo string `json:"contact_info"`
	Notes       string `json:"notes"`
}

func (h *Handlers) createProspect(c *gin.Context) {
	var b prospectWriteBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if b.CompanyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_name required"})
		return
	}
	id, err := h.App.Repo.CreateProspect(c.Request.Context(), b.CompanyName, b.ContactInfo, b.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) updateProspect(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var b prospectWriteBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if b.CompanyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company_name required"})
		return
	}
	if err := h.App.Repo.UpdateProspect(c.Request.Context(), id, b.CompanyName, b.ContactInfo, b.Notes); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) deleteProspect(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.App.Repo.DeleteProspect(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func validAnnouncementChannel(ch string) bool {
	switch ch {
	case "public", "job_admin", "customer_admin", "all":
		return true
	default:
		return false
	}
}

func parseRFC3339Ptr(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// listAnnouncementsPublic は GET /public/announcements?channel=public|job_admin|customer_admin
// （all チャネルの行はいずれの channel 指定でも返る）。
func (h *Handlers) listAnnouncementsPublic(c *gin.Context) {
	ch := c.Query("channel")
	if ch == "" || !validAnnouncementChannel(ch) || ch == "all" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "channel must be public, job_admin, or customer_admin"})
		return
	}
	list, err := h.App.Repo.ListAnnouncementsActive(c.Request.Context(), ch)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"announcements": list})
}

func (h *Handlers) listAnnouncementsJobAdminFeed(c *gin.Context) {
	list, err := h.App.Repo.ListAnnouncementsActive(c.Request.Context(), "job_admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"announcements": list})
}

func (h *Handlers) announcementsFeedCustomer(c *gin.Context) {
	list, err := h.App.Repo.ListAnnouncementsActive(c.Request.Context(), "customer_admin")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"announcements": list})
}

func (h *Handlers) listAnnouncementsManage(c *gin.Context) {
	list, err := h.App.Repo.ListAnnouncementsManage(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"announcements": list})
}

func (h *Handlers) getAnnouncement(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	a, err := h.App.Repo.GetAnnouncement(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if a == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, a)
}

type announcementBody struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	Channel   string `json:"channel"`
	Active    *bool  `json:"active"`
	ValidFrom string `json:"valid_from"`
	ValidTo   string `json:"valid_to"`
	SortOrder int    `json:"sort_order"`
}

func (h *Handlers) createAnnouncement(c *gin.Context) {
	var b announcementBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if b.Title == "" || b.Body == "" || !validAnnouncementChannel(b.Channel) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	active := true
	if b.Active != nil {
		active = *b.Active
	}
	vf, err := parseRFC3339Ptr(b.ValidFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid_from invalid"})
		return
	}
	vt, err := parseRFC3339Ptr(b.ValidTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid_to invalid"})
		return
	}
	id, err := h.App.Repo.CreateAnnouncement(c.Request.Context(), b.Title, b.Body, b.Channel, active, vf, vt, b.SortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) updateAnnouncement(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var b announcementBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if b.Title == "" || b.Body == "" || !validAnnouncementChannel(b.Channel) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	active := true
	if b.Active != nil {
		active = *b.Active
	}
	vf, err := parseRFC3339Ptr(b.ValidFrom)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid_from invalid"})
		return
	}
	vt, err := parseRFC3339Ptr(b.ValidTo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "valid_to invalid"})
		return
	}
	if err := h.App.Repo.UpdateAnnouncement(c.Request.Context(), id, b.Title, b.Body, b.Channel, active, vf, vt, b.SortOrder); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) deleteAnnouncement(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.App.Repo.DeleteAnnouncement(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) listCustomerAdmins(c *gin.Context) {
	list, err := h.App.Repo.ListCustomerAdminUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": list})
}

type customerAdminCreateBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handlers) createCustomerAdmin(c *gin.Context) {
	var b customerAdminCreateBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if b.Email == "" || !strings.Contains(b.Email, "@") || b.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid fields"})
		return
	}
	hash, err := auth.HashPassword(b.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, err := h.App.Repo.CreateCustomerAdminUser(c.Request.Context(), b.Email, hash)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

type customerAdminPatchBody struct {
	Email              string  `json:"email"`
	Active             *bool   `json:"active"`
	Password           string  `json:"password"`
	RegistrationStatus *string `json:"registration_status"`
}

func (h *Handlers) updateCustomerAdmin(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("adminUserId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var b customerAdminPatchBody
	if err := c.ShouldBindJSON(&b); err != nil || b.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	active := true
	if b.Active != nil {
		active = *b.Active
	}
	ctx := c.Request.Context()
	cur, err := h.App.Repo.GetCustomerAdminUser(ctx, id)
	if err != nil || cur == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	// 最後の有効アカウントを無効化しない（ロックアウト防止）。
	if cur.Active && !active {
		n, err := h.App.Repo.CountActiveCustomerAdmins(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if n <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot deactivate the last active admin"})
			return
		}
	}
	var hashPtr *string
	if b.Password != "" {
		hh, err := auth.HashPassword(b.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		hashPtr = &hh
	}
	var regPtr *string
	if b.RegistrationStatus != nil && *b.RegistrationStatus != "" {
		switch *b.RegistrationStatus {
		case "pending", "approved", "rejected":
			regPtr = b.RegistrationStatus
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid registration_status"})
			return
		}
	}
	if err := h.App.Repo.UpdateCustomerAdminUser(ctx, id, b.Email, active, hashPtr, regPtr); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) deleteCustomerAdmin(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("adminUserId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	ctx := c.Request.Context()
	total, err := h.App.Repo.CountCustomerAdminUsers(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if total <= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last admin user"})
		return
	}
	if err := h.App.Repo.DeleteCustomerAdminUser(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// --- 求職者（求人サイト React）---

type jobSeekerRegisterBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// jobSeekerRegister は POST /job-seeker/register 。成功時にアカウント ID を返す。
func (h *Handlers) jobSeekerRegister(c *gin.Context) {
	var b jobSeekerRegisterBody
	if err := c.ShouldBindJSON(&b); err != nil || b.Email == "" || b.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	id, err := h.App.JobSeekerRegister(c.Request.Context(), b.Email, b.Password)
	if err != nil {
		var pe *pgconn.PgError
		if errors.As(err, &pe) && pe.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}
		if errors.Is(err, usecase.ErrJobSeekerEmptyCredentials) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// jobSeekerLogin は POST /job-seeker/login 。
func (h *Handlers) jobSeekerLogin(c *gin.Context) {
	var b loginBody
	if err := c.ShouldBindJSON(&b); err != nil || b.Email == "" || b.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	tok, err := h.App.JobSeekerLogin(c.Request.Context(), b.Email, b.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrJobSeekerEmptyCredentials) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tok})
}

// jobSeekerGetProfile は GET /job-seeker/profile（JWT の uid = 求職者アカウント ID）。
func (h *Handlers) jobSeekerGetProfile(c *gin.Context) {
	cl := claimsFromGin(c)
	if cl == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing claims"})
		return
	}
	p, err := h.App.JobSeekerGetProfile(c.Request.Context(), cl.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if p == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

type jobSeekerProfilePatchBody struct {
	DisplayName   string `json:"display_name"`
	Phone         string `json:"phone"`
	CareerSummary string `json:"career_summary"`
	Notes         string `json:"notes"`
}

// jobSeekerUpdateProfile は PATCH /job-seeker/profile 。
func (h *Handlers) jobSeekerUpdateProfile(c *gin.Context) {
	cl := claimsFromGin(c)
	if cl == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing claims"})
		return
	}
	var b jobSeekerProfilePatchBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if err := h.App.JobSeekerUpdateProfile(c.Request.Context(), cl.UserID, b.DisplayName, b.Phone, b.CareerSummary, b.Notes); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// --- 顧客イベント（顧客管理サイト）---

var allowedCustomerEventKinds = map[string]bool{
	"meeting": true, "contract_start": true, "risk_flag": true, "note": true, "other": true,
}

func parseCustomerEventTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("empty time")
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("invalid occurred_at")
}

type customerEventBody struct {
	EventKind     string `json:"event_kind"`
	OccurredAt    string `json:"occurred_at"`
	Title         string `json:"title"`
	Body          string `json:"body"`
	IsRiskRelated bool   `json:"is_risk_related"`
}

// listCustomerEvents は GET /admin/customers/customers/:id/events 。
func (h *Handlers) listCustomerEvents(c *gin.Context) {
	cid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}
	list, err := h.App.Repo.ListCustomerEvents(c.Request.Context(), cid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if list == nil {
		list = []domain.CustomerEventRow{}
	}
	c.JSON(http.StatusOK, gin.H{"events": list})
}

// createCustomerEvent は POST /admin/customers/customers/:id/events 。
func (h *Handlers) createCustomerEvent(c *gin.Context) {
	cid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}
	var b customerEventBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if !allowedCustomerEventKinds[b.EventKind] || strings.TrimSpace(b.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_kind or title"})
		return
	}
	at, err := parseCustomerEventTime(b.OccurredAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid occurred_at"})
		return
	}
	id, err := h.App.Repo.CreateCustomerEvent(c.Request.Context(), cid, b.EventKind, at, b.Title, b.Body, b.IsRiskRelated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// updateCustomerEvent は PATCH /admin/customers/customers/:id/events/:eventId 。
func (h *Handlers) updateCustomerEvent(c *gin.Context) {
	cid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}
	eid, err := strconv.ParseInt(c.Param("eventId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}
	var b customerEventBody
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if !allowedCustomerEventKinds[b.EventKind] || strings.TrimSpace(b.Title) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event_kind or title"})
		return
	}
	at, err := parseCustomerEventTime(b.OccurredAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid occurred_at"})
		return
	}
	if err := h.App.Repo.UpdateCustomerEvent(c.Request.Context(), cid, eid, b.EventKind, at, b.Title, b.Body, b.IsRiskRelated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// deleteCustomerEvent は DELETE /admin/customers/customers/:id/events/:eventId 。
func (h *Handlers) deleteCustomerEvent(c *gin.Context) {
	cid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}
	eid, err := strconv.ParseInt(c.Param("eventId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event id"})
		return
	}
	if err := h.App.Repo.DeleteCustomerEvent(c.Request.Context(), cid, eid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// --- 追加機能（設計メモ5） ---

func (h *Handlers) jobSeekerListFavorites(c *gin.Context) {
	cl := claimsFromGin(c)
	list, err := h.App.Repo.ListJobSeekerFavorites(c.Request.Context(), cl.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"favorites": list})
}

type jobSeekerFavoriteBody struct {
	JobPostingID int64 `json:"job_posting_id"`
}

func (h *Handlers) jobSeekerAddFavorite(c *gin.Context) {
	cl := claimsFromGin(c)
	var b jobSeekerFavoriteBody
	if err := c.ShouldBindJSON(&b); err != nil || b.JobPostingID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.App.Repo.AddJobSeekerFavorite(c.Request.Context(), cl.UserID, b.JobPostingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "ok"})
}

func (h *Handlers) jobSeekerDeleteFavorite(c *gin.Context) {
	cl := claimsFromGin(c)
	jobID, err := strconv.ParseInt(c.Param("jobId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}
	if err := h.App.Repo.DeleteJobSeekerFavorite(c.Request.Context(), cl.UserID, jobID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) jobSeekerListHistory(c *gin.Context) {
	cl := claimsFromGin(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	list, err := h.App.Repo.ListJobSeekerViewHistory(c.Request.Context(), cl.UserID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"history": list})
}

func (h *Handlers) jobSeekerAddHistory(c *gin.Context) {
	cl := claimsFromGin(c)
	var b jobSeekerFavoriteBody
	if err := c.ShouldBindJSON(&b); err != nil || b.JobPostingID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.App.Repo.AddJobSeekerViewHistory(c.Request.Context(), cl.UserID, b.JobPostingID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "ok"})
}

func (h *Handlers) publicGetCompany(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	row, err := h.App.Repo.GetCompanyProfile(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if row == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *Handlers) publicListCompanyReviews(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	list, err := h.App.Repo.ListCompanyReviews(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reviews": list})
}

func (h *Handlers) publicListCompanyVideos(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	list, err := h.App.Repo.ListCompanyVideos(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"videos": list})
}

type salarySimulationBody struct {
	JobCategory string `json:"job_category"`
	Region      string `json:"region"`
	YearsExp    int    `json:"years_exp"`
}

func (h *Handlers) publicSalarySimulation(c *gin.Context) {
	var b salarySimulationBody
	if err := c.ShouldBindJSON(&b); err != nil || b.JobCategory == "" || b.YearsExp < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if b.Region == "" {
		b.Region = "JP"
	}
	low, median, high, err := h.App.Repo.CreateSalarySimulation(c.Request.Context(), nil, b.JobCategory, b.Region, b.YearsExp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"low": low, "median": median, "high": high})
}

func (h *Handlers) jobAdminGetCompanyProfile(c *gin.Context) {
	cl := claimsFromGin(c)
	row, err := h.App.Repo.GetCompanyProfile(c.Request.Context(), cl.CustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if row == nil {
		c.JSON(http.StatusOK, gin.H{"customer_id": cl.CustomerID})
		return
	}
	c.JSON(http.StatusOK, row)
}

type companyProfileBody struct {
	CompanyName      string  `json:"company_name"`
	Description      string  `json:"description"`
	Address          string  `json:"address"`
	GoogleMapURL     *string `json:"google_map_url"`
	WebsiteURL       *string `json:"website_url"`
	YoutubeEmbedURL  *string `json:"youtube_embed_url"`
	AcceptForeigners bool    `json:"accept_foreigners"`
	Languages        string  `json:"languages"`
}

func (h *Handlers) jobAdminUpdateCompanyProfile(c *gin.Context) {
	cl := claimsFromGin(c)
	var b companyProfileBody
	if err := c.ShouldBindJSON(&b); err != nil || b.CompanyName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	err := h.App.Repo.UpsertCompanyProfile(c.Request.Context(), domain.CompanyProfileRow{
		CustomerID:       cl.CustomerID,
		CompanyName:      b.CompanyName,
		Description:      b.Description,
		Address:          b.Address,
		GoogleMapURL:     b.GoogleMapURL,
		WebsiteURL:       b.WebsiteURL,
		YoutubeEmbedURL:  b.YoutubeEmbedURL,
		AcceptForeigners: b.AcceptForeigners,
		Languages:        b.Languages,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type aiAssistBody struct {
	JobTitle string `json:"job_title"`
	Prompt   string `json:"prompt"`
}

func japaneseJobTitleForAssist(code string) string {
	switch code {
	case "backend_engineer":
		return "バックエンドエンジニア"
	case "frontend_engineer":
		return "フロントエンドエンジニア"
	case "sales":
		return "営業"
	default:
		return code
	}
}

func (h *Handlers) jobAdminAIAssist(c *gin.Context) {
	cl := claimsFromGin(c)
	var b aiAssistBody
	if err := c.ShouldBindJSON(&b); err != nil || b.JobTitle == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	jt := japaneseJobTitleForAssist(b.JobTitle)
	suggestion := "【AIアシスト草案】\n職種: " + jt + "\n- 期待役割: 要件定義〜改善提案\n- 必須スキル: 実務経験、コミュニケーション\n- 歓迎: 自動化・分析経験\n- 訴求: 成長機会と裁量の大きさ"
	id, err := h.App.Repo.CreateAIAssistLog(c.Request.Context(), cl.CustomerID, b.JobTitle, b.Prompt, suggestion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "suggestion": suggestion})
}

type mediaConnectionBody struct {
	MediaName       string `json:"media_name"`
	ExternalAccount string `json:"external_account"`
	Status          string `json:"status"`
	SettingsJSON    string `json:"settings_json"`
}

func (h *Handlers) jobAdminListMediaConnections(c *gin.Context) {
	cl := claimsFromGin(c)
	list, err := h.App.Repo.ListMediaConnections(c.Request.Context(), cl.CustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"connections": list})
}

func (h *Handlers) jobAdminCreateMediaConnection(c *gin.Context) {
	cl := claimsFromGin(c)
	var b mediaConnectionBody
	if err := c.ShouldBindJSON(&b); err != nil || b.MediaName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if b.Status == "" {
		b.Status = "active"
	}
	if b.SettingsJSON == "" {
		b.SettingsJSON = "{}"
	}
	id, err := h.App.Repo.CreateMediaConnection(c.Request.Context(), cl.CustomerID, b.MediaName, b.ExternalAccount, b.Status, b.SettingsJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) jobAdminUpdateMediaConnection(c *gin.Context) {
	cl := claimsFromGin(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var b mediaConnectionBody
	if err := c.ShouldBindJSON(&b); err != nil || b.MediaName == "" || b.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if b.SettingsJSON == "" {
		b.SettingsJSON = "{}"
	}
	if err := h.App.Repo.UpdateMediaConnection(c.Request.Context(), cl.CustomerID, id, b.MediaName, b.ExternalAccount, b.Status, b.SettingsJSON); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) jobAdminDashboard(c *gin.Context) {
	cl := claimsFromGin(c)
	s, err := h.App.Repo.GetDashboardSummary(c.Request.Context(), cl.CustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"summary": s})
}

func (h *Handlers) jobAdminReports(c *gin.Context) {
	cl := claimsFromGin(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	list, err := h.App.Repo.ListReportSnapshots(c.Request.Context(), cl.CustomerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reports": list})
}

func (h *Handlers) jobAdminInflowAnalytics(c *gin.Context) {
	cl := claimsFromGin(c)
	list, err := h.App.Repo.ListInflowAnalytics(c.Request.Context(), cl.CustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"inflow": list})
}

type interviewBody struct {
	JobPostingID *int64  `json:"job_posting_id"`
	Provider     string  `json:"provider"`
	MeetingURL   string  `json:"meeting_url"`
	ScheduledAt  *string `json:"scheduled_at"`
}

func (h *Handlers) jobAdminCreateInterviewLink(c *gin.Context) {
	cl := claimsFromGin(c)
	var b interviewBody
	if err := c.ShouldBindJSON(&b); err != nil || b.MeetingURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	var scheduled *time.Time
	if b.ScheduledAt != nil && *b.ScheduledAt != "" {
		t, err := time.Parse(time.RFC3339, *b.ScheduledAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid scheduled_at"})
			return
		}
		scheduled = &t
	}
	if b.Provider == "" {
		b.Provider = "google_meet"
	}
	id, err := h.App.Repo.CreateInterviewLink(c.Request.Context(), domain.InterviewLinkRow{
		CustomerID:   cl.CustomerID,
		JobPostingID: b.JobPostingID,
		Provider:     b.Provider,
		MeetingURL:   b.MeetingURL,
		ScheduledAt:  scheduled,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) jobAdminListInterviews(c *gin.Context) {
	cl := claimsFromGin(c)
	list, err := h.App.Repo.ListInterviewLinks(c.Request.Context(), cl.CustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"interviews": list})
}

type scoutBody struct {
	JobPostingID  *int64 `json:"job_posting_id"`
	CandidateName string `json:"candidate_name"`
	Contact       string `json:"contact"`
	Message       string `json:"message"`
	Status        string `json:"status"`
}

func (h *Handlers) jobAdminListScouts(c *gin.Context) {
	cl := claimsFromGin(c)
	list, err := h.App.Repo.ListScouts(c.Request.Context(), cl.CustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"scouts": list})
}

func (h *Handlers) jobAdminCreateScout(c *gin.Context) {
	cl := claimsFromGin(c)
	var b scoutBody
	if err := c.ShouldBindJSON(&b); err != nil || b.CandidateName == "" || b.Contact == "" || b.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if b.Status == "" {
		b.Status = "draft"
	}
	id, err := h.App.Repo.CreateScout(c.Request.Context(), domain.ScoutRow{
		CustomerID:    cl.CustomerID,
		JobPostingID:  b.JobPostingID,
		CandidateName: b.CandidateName,
		Contact:       b.Contact,
		Message:       b.Message,
		Status:        b.Status,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) jobAdminUpdateScout(c *gin.Context) {
	cl := claimsFromGin(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var b scoutBody
	if err := c.ShouldBindJSON(&b); err != nil || b.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.App.Repo.UpdateScout(c.Request.Context(), cl.CustomerID, id, b.Status, b.Message); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) jobAdminCreateCompanyVideo(c *gin.Context) {
	cl := claimsFromGin(c)
	var b struct {
		Title      string `json:"title"`
		YoutubeURL string `json:"youtube_url"`
	}
	if err := c.ShouldBindJSON(&b); err != nil || b.Title == "" || b.YoutubeURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	id, err := h.App.Repo.CreateCompanyVideo(c.Request.Context(), cl.CustomerID, b.Title, b.YoutubeURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func (h *Handlers) jobAdminDeleteCompanyVideo(c *gin.Context) {
	cl := claimsFromGin(c)
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.App.Repo.DeleteCompanyVideo(c.Request.Context(), cl.CustomerID, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) jobAdminGetFollowUpPolicy(c *gin.Context) {
	cl := claimsFromGin(c)
	row, err := h.App.Repo.GetFollowUpPolicy(c.Request.Context(), cl.CustomerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if row == nil {
		c.JSON(http.StatusOK, gin.H{"customer_id": cl.CustomerID, "enabled": false, "max_follow_up_days": 0})
		return
	}
	c.JSON(http.StatusOK, row)
}

type followUpPolicyBody struct {
	Enabled             bool   `json:"enabled"`
	MaxFollowUpDays     int    `json:"max_follow_up_days"`
	AvailableByContract bool   `json:"available_by_contract"`
	Notes               string `json:"notes"`
}

func (h *Handlers) jobAdminUpdateFollowUpPolicy(c *gin.Context) {
	cl := claimsFromGin(c)
	var b followUpPolicyBody
	if err := c.ShouldBindJSON(&b); err != nil || b.MaxFollowUpDays < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	err := h.App.Repo.UpsertFollowUpPolicy(c.Request.Context(), domain.FollowUpPolicyRow{
		CustomerID:          cl.CustomerID,
		Enabled:             b.Enabled,
		MaxFollowUpDays:     b.MaxFollowUpDays,
		AvailableByContract: b.AvailableByContract,
		Notes:               b.Notes,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) jobAdminUpdateJobGlobalOptions(c *gin.Context) {
	cl := claimsFromGin(c)
	jobID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var b struct {
		AcceptForeigners bool   `json:"accept_foreigners"`
		Languages        string `json:"languages"`
	}
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if err := h.App.Repo.UpdateJobGlobalOptions(c.Request.Context(), cl.CustomerID, jobID, b.AcceptForeigners, b.Languages); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handlers) jobAdminSalarySimulation(c *gin.Context) {
	cl := claimsFromGin(c)
	var b salarySimulationBody
	if err := c.ShouldBindJSON(&b); err != nil || b.JobCategory == "" || b.YearsExp < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	if b.Region == "" {
		b.Region = "JP"
	}
	low, median, high, err := h.App.Repo.CreateSalarySimulation(c.Request.Context(), &cl.CustomerID, b.JobCategory, b.Region, b.YearsExp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"low": low, "median": median, "high": high})
}

func (h *Handlers) customerGetFollowUpCapabilities(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}
	enabled, reason, maxDays, err := h.App.Repo.GetFollowUpCapabilities(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": enabled, "reason": reason, "max_follow_up_days": maxDays})
}
