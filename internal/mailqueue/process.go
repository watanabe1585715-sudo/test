// Package mailqueue は email_outbox の pending を SMTP で送る処理です。
package mailqueue

import (
	"context"
	"fmt"
	"log"

	"recruitment/internal/domain"
	"recruitment/internal/infrastructure/mail"
)

// Run は最大 limit 件の pending を送信し、成功なら sent・失敗なら failed に更新する。
// sender が未設定（Configured=false）のときは何も更新せず 0, nil を返す（呼び出し側でログ推奨）。
func Run(ctx context.Context, repo domain.StaffingRepository, sender mail.Sender, from string, limit int) (sent int, err error) {
	if !sender.Configured() {
		return 0, nil
	}
	if from == "" {
		return 0, fmt.Errorf("MAIL_FROM is required when SMTP is configured")
	}
	list, err := repo.ListPendingEmails(ctx, limit)
	if err != nil {
		return 0, err
	}
	for _, row := range list {
		sendErr := sender.Send(ctx, from, row.ToEmail, row.Subject, row.Body)
		if sendErr != nil {
			if mErr := repo.MarkEmailFailed(ctx, row.ID, sendErr.Error()); mErr != nil {
				log.Printf("mailqueue: mark failed id=%d: %v", row.ID, mErr)
			}
			continue
		}
		if mErr := repo.MarkEmailSent(ctx, row.ID); mErr != nil {
			log.Printf("mailqueue: mark sent id=%d: %v", row.ID, mErr)
			continue
		}
		sent++
	}
	return sent, nil
}
