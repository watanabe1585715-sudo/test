package usecase

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"recruitment/internal/auth"
)

// このファイルは「ログイン認証ユースケース」をまとめる。
// HTTP 層はここを呼ぶだけでよく、パスワード照合や承認状態判定を集約できる。

// JobAdminLogin は案件管理ユーザのメール・パスワードを検証し、成功時に JWT と顧客 ID を返す。
// 失敗時は err が nil でなく、呼び出し側が HTTP ステータスを決める想定（ErrNoSuchUser 等）。
func (a *StaffingApp) JobAdminLogin(ctx context.Context, email, password string) (token string, customerID int64, err error) {
	// 空入力はここでは扱わず、HTTP 層で 400 にすることが多いが、二重防御として弾く。
	if email == "" || password == "" {
		return "", 0, errors.New("empty credentials")
	}
	id, hash, cid, err := a.Repo.GetJobAdminByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, pgx.ErrNoRows
		}
		return "", 0, err
	}
	// パスワード照合: ハッシュが一致しなければ認証失敗として同じ「行無し」と同等に扱える。
	if auth.CheckPassword(hash, password) != nil {
		return "", 0, pgx.ErrNoRows
	}
	tok, err := auth.SignJWT(a.JWTSecret, id, auth.RoleJobAdmin, cid)
	if err != nil {
		return "", 0, err
	}
	return tok, cid, nil
}

// CustomerAdminLogin は顧客管理ユーザのログイン。顧客スコープは JWT の customer_id=0 で表す。
func (a *StaffingApp) CustomerAdminLogin(ctx context.Context, email, password string) (token string, err error) {
	if email == "" || password == "" {
		return "", errors.New("empty credentials")
	}
	id, hash, reg, err := a.Repo.GetCustomerAdminByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", pgx.ErrNoRows
		}
		return "", err
	}
	switch reg {
	case "pending":
		return "", ErrCustomerAdminPending
	case "rejected":
		return "", ErrCustomerAdminRejected
	case "approved":
		// 続行
	default:
		return "", ErrCustomerAdminRejected
	}
	if auth.CheckPassword(hash, password) != nil {
		return "", pgx.ErrNoRows
	}
	// customer_admin ロールは全顧客を見られるため CustomerID は 0 を渡す。
	return auth.SignJWT(a.JWTSecret, id, auth.RoleCustomerAdmin, 0)
}
