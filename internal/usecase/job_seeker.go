package usecase

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"recruitment/internal/auth"
	"recruitment/internal/domain"
)

// ErrJobSeekerEmptyCredentials は求職者登録・ログインでメール／パスワードが空のとき。
var ErrJobSeekerEmptyCredentials = errors.New("empty credentials")

// JobSeekerRegister はメールとパスワードでアカウント＋空プロフィールを作る。
func (a *StaffingApp) JobSeekerRegister(ctx context.Context, email, password string) (int64, error) {
	if email == "" || password == "" {
		return 0, ErrJobSeekerEmptyCredentials
	}
	h, err := auth.HashPassword(password)
	if err != nil {
		return 0, err
	}
	return a.Repo.CreateJobSeekerAccount(ctx, email, h)
}

// JobSeekerLogin は検証成功時に JWT（role=job_seeker, cid=0）を返す。
func (a *StaffingApp) JobSeekerLogin(ctx context.Context, email, password string) (string, error) {
	if email == "" || password == "" {
		return "", ErrJobSeekerEmptyCredentials
	}
	id, hash, err := a.Repo.GetJobSeekerByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		return "", err
	}
	if auth.CheckPassword(hash, password) != nil {
		return "", pgx.ErrNoRows
	}
	return auth.SignJWT(a.JWTSecret, id, auth.RoleJobSeeker, 0)
}

// JobSeekerGetProfile は JWT の uid（求職者アカウント ID）に対応するプロフィールを返す。
func (a *StaffingApp) JobSeekerGetProfile(ctx context.Context, accountID int64) (*domain.JobSeekerProfileRow, error) {
	return a.Repo.GetJobSeekerProfile(ctx, accountID)
}

// JobSeekerUpdateProfile はマイページの表示名・連絡先・職歴要約・メモを更新する。
func (a *StaffingApp) JobSeekerUpdateProfile(ctx context.Context, accountID int64, displayName, phone, careerSummary, notes string) error {
	return a.Repo.UpdateJobSeekerProfile(ctx, accountID, displayName, phone, careerSummary, notes)
}
