-- 追加機能: お気に入り、閲覧履歴、企業詳細、動画、口コミ、給与相場、媒体連携、分析、面談、スカウト、フォローアップ

ALTER TABLE job_postings
    ADD COLUMN IF NOT EXISTS accept_foreigners BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS supported_languages TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS job_seeker_favorites (
    account_id BIGINT NOT NULL REFERENCES job_seeker_accounts (id) ON DELETE CASCADE,
    job_posting_id BIGINT NOT NULL REFERENCES job_postings (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (account_id, job_posting_id)
);

CREATE TABLE IF NOT EXISTS job_view_history (
    id BIGSERIAL PRIMARY KEY,
    account_id BIGINT NOT NULL REFERENCES job_seeker_accounts (id) ON DELETE CASCADE,
    job_posting_id BIGINT NOT NULL REFERENCES job_postings (id) ON DELETE CASCADE,
    viewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_job_view_history_account_time ON job_view_history (account_id, viewed_at DESC);

CREATE TABLE IF NOT EXISTS company_profiles (
    customer_id BIGINT PRIMARY KEY REFERENCES customers (id) ON DELETE CASCADE,
    company_name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    google_map_url TEXT,
    website_url TEXT,
    youtube_embed_url TEXT,
    accept_foreigners BOOLEAN NOT NULL DEFAULT FALSE,
    languages TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS company_reviews (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    reviewer TEXT NOT NULL,
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS company_videos (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    youtube_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS salary_market_rates (
    id BIGSERIAL PRIMARY KEY,
    job_category TEXT NOT NULL,
    region TEXT NOT NULL DEFAULT 'JP',
    years_exp_min INT NOT NULL DEFAULT 0,
    years_exp_max INT NOT NULL DEFAULT 100,
    low_monthly_salary BIGINT NOT NULL,
    median_monthly_salary BIGINT NOT NULL,
    high_monthly_salary BIGINT NOT NULL,
    source TEXT NOT NULL DEFAULT 'internal_estimation',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_salary_market_rates_job_region ON salary_market_rates (job_category, region, years_exp_min, years_exp_max);

CREATE TABLE IF NOT EXISTS salary_simulations (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT REFERENCES customers (id) ON DELETE SET NULL,
    job_category TEXT NOT NULL,
    region TEXT NOT NULL,
    years_exp INT NOT NULL,
    low_monthly_salary BIGINT NOT NULL,
    median_monthly_salary BIGINT NOT NULL,
    high_monthly_salary BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS ai_job_assist_logs (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    job_title TEXT NOT NULL,
    prompt TEXT NOT NULL,
    suggestion TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS media_connections (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    media_name TEXT NOT NULL,
    external_account TEXT,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'error')),
    settings_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS media_inflow_stats (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    media_name TEXT NOT NULL,
    measured_date DATE NOT NULL,
    views BIGINT NOT NULL DEFAULT 0,
    clicks BIGINT NOT NULL DEFAULT 0,
    applications BIGINT NOT NULL DEFAULT 0,
    hires BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS interview_links (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    job_posting_id BIGINT REFERENCES job_postings (id) ON DELETE SET NULL,
    provider TEXT NOT NULL DEFAULT 'google_meet',
    meeting_url TEXT NOT NULL,
    scheduled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS scouts (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    job_posting_id BIGINT REFERENCES job_postings (id) ON DELETE SET NULL,
    candidate_name TEXT NOT NULL,
    contact TEXT NOT NULL,
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'sent', 'replied', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS follow_up_policies (
    customer_id BIGINT PRIMARY KEY REFERENCES customers (id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    max_follow_up_days INT NOT NULL DEFAULT 0,
    available_by_contract BOOLEAN NOT NULL DEFAULT TRUE,
    notes TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS report_snapshots (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    report_kind TEXT NOT NULL,
    period_start DATE NOT NULL,
    period_end DATE NOT NULL,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
