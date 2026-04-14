// このファイルはパスワードのハッシュ化と照合だけを担当します（JWT は jwt.go）。
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword は bcrypt でハッシュ化した文字列を返す（保存用）。
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword は平文と保存ハッシュの一致を検証する。
func CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
