-- お知らせ（表示チャネルごと。顧客管理サイトから CRUD、各フロントは GET のみ）
CREATE TABLE announcements (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    channel TEXT NOT NULL CHECK (channel IN ('public', 'job_admin', 'customer_admin', 'all')),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    valid_from TIMESTAMPTZ,
    valid_to TIMESTAMPTZ,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_announcements_channel_active ON announcements (channel, active);

INSERT INTO announcements (title, body, channel, active, sort_order)
VALUES
('デモ: 求人サイト向けお知らせ', '顧客管理サイトの「お知らせ管理」から編集できます。', 'public', TRUE, 0),
('デモ: 案件管理サイト向け', '同じく顧客管理から登録・更新します。', 'job_admin', TRUE, 1),
('デモ: 顧客管理サイト向け', 'ログイン後トップに表示されます。', 'customer_admin', TRUE, 2);
