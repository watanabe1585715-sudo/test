-- 顧客管理: 管理者の登録承認、顧客の利用承認、顧客に紐づくイベント履歴
-- 求人サイト: 求職者アカウントとプロフィール

ALTER TABLE customer_admin_users
    ADD COLUMN IF NOT EXISTS registration_status TEXT NOT NULL DEFAULT 'approved'
        CHECK (registration_status IN ('pending', 'approved', 'rejected'));

ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS approval_status TEXT NOT NULL DEFAULT 'approved'
        CHECK (approval_status IN ('pending', 'approved', 'rejected'));

CREATE TABLE job_seeker_accounts (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_seeker_profiles (
    account_id BIGINT PRIMARY KEY REFERENCES job_seeker_accounts (id) ON DELETE CASCADE,
    display_name TEXT NOT NULL DEFAULT '',
    phone TEXT NOT NULL DEFAULT '',
    career_summary TEXT NOT NULL DEFAULT '',
    notes TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE customer_events (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    event_kind TEXT NOT NULL CHECK (event_kind IN ('meeting', 'contract_start', 'risk_flag', 'note', 'other')),
    occurred_at TIMESTAMPTZ NOT NULL,
    title TEXT NOT NULL,
    body TEXT,
    is_risk_related BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customer_events_customer_time ON customer_events (customer_id, occurred_at DESC);
