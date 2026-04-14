// Package auth は「ログインしたと言えるか」の判定に使う JWT とパスワードハッシュを扱います。
//
// 初心者向けメモ:
//   - JWT は「改ざんされにくいトークン」で、ログイン成功後にブラウザが保持し、以降の API に添えます。
//   - パスワードは DB に平文で保存せず、bcrypt でハッシュ化した文字列だけを保存します（CheckPassword で照合）。
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// RoleJobAdmin は案件管理サイト用ユーザー。
	RoleJobAdmin = "job_admin"
	// RoleCustomerAdmin は顧客管理サイト用ユーザー。
	RoleCustomerAdmin = "customer_admin"
	// RoleJobSeeker は求人サイト（求職者）ログイン用。
	RoleJobSeeker = "job_seeker"
	// TokenTTL はアクセストークンの有効期限。
	TokenTTL = 24 * time.Hour
)

// Claims は JWT ペイロード。UserID はロールごとの主キー（例: job_admin_users.id / customer_admin_users.id / job_seeker_accounts.id）。
// CustomerID は案件管理（job_admin）のときのみ顧客 ID が入り、それ以外は 0。
type Claims struct {
	UserID     int64  `json:"uid"`
	Role       string `json:"role"`
	CustomerID int64  `json:"cid,omitempty"`
	jwt.RegisteredClaims
}

// SignJWT は HS256 で署名したトークン文字列を返す。
func SignJWT(secret []byte, userID int64, role string, customerID int64) (string, error) {
	now := time.Now()
	c := Claims{
		UserID:     userID,
		Role:       role,
		CustomerID: customerID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	return t.SignedString(secret)
}

// ParseJWT は検証に成功した Claims を返す。
func ParseJWT(secret []byte, tokenStr string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	c, ok := t.Claims.(*Claims)
	if !ok || !t.Valid {
		return nil, errors.New("invalid token")
	}
	return c, nil
}
