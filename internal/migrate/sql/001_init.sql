-- customers: contract_tier 1 = 10件, 2 = 100件, 3 = 無制限
CREATE TABLE customers (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    contract_tier SMALLINT NOT NULL CHECK (contract_tier IN (1, 2, 3)),
    contract_start DATE NOT NULL,
    contract_end DATE,
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'ended')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE customer_admin_users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE job_admin_users (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- publication_status: draft | published | ended
CREATE TABLE job_postings (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    summary TEXT NOT NULL,
    requirements TEXT NOT NULL,
    publish_start DATE NOT NULL,
    publish_end DATE NOT NULL,
    publication_status TEXT NOT NULL DEFAULT 'draft' CHECK (publication_status IN ('draft', 'published', 'ended')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE applications (
    id BIGSERIAL PRIMARY KEY,
    job_posting_id BIGINT NOT NULL REFERENCES job_postings (id) ON DELETE CASCADE,
    applicant_name TEXT NOT NULL,
    career_summary TEXT NOT NULL,
    contact TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE invoices (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id) ON DELETE CASCADE,
    issued_at DATE NOT NULL,
    amount_cents BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'confirmed')),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE prospects (
    id BIGSERIAL PRIMARY KEY,
    company_name TEXT NOT NULL,
    contact_info TEXT,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_job_postings_customer ON job_postings (customer_id);
CREATE INDEX idx_job_postings_status ON job_postings (publication_status);
CREATE INDEX idx_applications_job ON applications (job_posting_id);
