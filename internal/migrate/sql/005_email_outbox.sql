-- メール送信キュー（SMTP ワーカーが pending を処理して sent / failed に更新する）
CREATE TABLE email_outbox (
    id BIGSERIAL PRIMARY KEY,
    kind TEXT NOT NULL DEFAULT 'manual',
    to_email TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed', 'skipped')),
    error_detail TEXT,
    related_application_id BIGINT REFERENCES applications (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at TIMESTAMPTZ
);

CREATE INDEX idx_email_outbox_status_created ON email_outbox (status, created_at);
CREATE INDEX idx_email_outbox_pending ON email_outbox (id) WHERE status = 'pending';
