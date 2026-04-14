// Package api は Gin による REST インターフェース（DDD のインターフェース層）です。
//
// 初心者向け:
//   - requireRole が付いたルートは「Authorization: Bearer … の JWT が正しく、ロールが一致するか」を先に調べます。
//   - エディタで Ctrl+クリック（Mac は Cmd+クリック）してハンドラや domain の型へジャンプするには、
//     ワークスペース直下の .vscode/settings.json で gopls / TypeScript を有効にしておくと安定します。
package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"recruitment/internal/auth"
	"recruitment/internal/usecase"
)

const ginClaimsKey = "jwtClaims"

// claimsFromGin は認証ミドルウェアが載せた JWT を取り出す。無いときは nil。
func claimsFromGin(c *gin.Context) *auth.Claims {
	v, ok := c.Get(ginClaimsKey)
	if !ok {
		return nil
	}
	cl, _ := v.(*auth.Claims)
	return cl
}

// requireRole は Authorization: Bearer を検証し、ロールが一致するときだけ次のハンドラへ進む。
func requireRole(app *usecase.StaffingApp, want string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if raw == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			c.Abort()
			return
		}
		cl, err := auth.ParseJWT(app.JWTSecret, raw)
		// トークンが壊れている・期限切れ・ロール違いのいずれも 401 でまとめる（詳細はログに出さない運用が多い）。
		if err != nil || cl.Role != want {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}
		c.Set(ginClaimsKey, cl)
		c.Next()
	}
}
