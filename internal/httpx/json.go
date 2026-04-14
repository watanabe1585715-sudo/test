// Package httpx は JSON を HTTP で返したり、リクエストボディから JSON を読んだりする補助だけを行います。
//
// 初心者向けメモ:
//   - ブラウザは多くの場合 Content-Type: application/json の本文でデータを送ります。
//   - ReadJSON はその本文を Go の構造体に入れ替え、JSON はレスポンスとしてクライアントへ返します。
package httpx

import (
	"encoding/json"
	"net/http"
)

// JSON は Content-Type を付与して JSON エンコードする。
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// ReadJSON はリクエストボディを JSON デコードする（未知フィールドは拒否）。
func ReadJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
