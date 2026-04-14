// Package usecase はアプリケーションサービス（ユースケース）層です。
//
// 初心者向け:
//   - domain は「何ができるか」の契約だけ、infrastructure は「DB への具体的な書き方」です。
//   - このパッケージはその間に立ち、「ログインしてトークンを返す」など複数ステップの流れをまとめます。
//   - 今はリポジトリへの委譲が中心ですが、ビジネスルールが増えたらここに書くと整理しやすいです。
package usecase

import (
	"errors"

	"recruitment/internal/domain"
)

// ErrCustomerAdminPending / ErrCustomerAdminRejected は顧客管理ログインが承認フローで弾かれたときの識別子（HTTP 403 用）。
var (
	ErrCustomerAdminPending  = errors.New("customer_admin_pending")
	ErrCustomerAdminRejected = errors.New("customer_admin_rejected")
)

// StaffingApp は求人・顧客まわりのユースケースに必要な依存を束ねる。
type StaffingApp struct {
	Repo      domain.StaffingRepository
	JWTSecret []byte
}

// NewStaffingApp は HTTP 層や cmd から渡されたリポジトリ実装と JWT 秘密鍵で App を作る。
func NewStaffingApp(repo domain.StaffingRepository, jwtSecret []byte) *StaffingApp {
	return &StaffingApp{Repo: repo, JWTSecret: jwtSecret}
}
