// Package mail は SMTP 送信の具象実装です（infrastructure）。
// SMTP が未設定のときは Noop でワーカーが「送らない」理由をログに出せます。
package mail

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// Sender はメール1通の送信を抽象化する。
type Sender interface {
	// Configured は実際にネットワーク送信が可能か（環境変数が揃っているか）。
	Configured() bool
	Send(ctx context.Context, from, to, subject, body string) error
}

// SMTPSender は net/smtp を使った送信（TLS は STARTTLS をサーバ任せ）。
type SMTPSender struct {
	Host     string
	Port     string
	User     string
	Password string
}

// NewSMTPSender はホスト必須。Port 空なら 587。
func NewSMTPSender(host, port, user, password string) *SMTPSender {
	if port == "" {
		port = "587"
	}
	return &SMTPSender{Host: host, Port: port, User: user, Password: password}
}

func (s *SMTPSender) Configured() bool {
	return strings.TrimSpace(s.Host) != ""
}

func (s *SMTPSender) Send(ctx context.Context, from, to, subject, body string) error {
	if !s.Configured() {
		return fmt.Errorf("SMTP host is empty")
	}
	// キャンセルだけ先に見る（送信本体はブロッキングのまま）。
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	addr := s.Host + ":" + s.Port
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Password, s.Host)
	}
	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n", from, to, subject)
	msg := []byte(headers + body)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}
