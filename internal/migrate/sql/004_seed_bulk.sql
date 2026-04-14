-- 試験用の一括ダミーデータ（約10倍相当を generate_series で投入）
INSERT INTO customers (name, description, contract_tier, contract_start, contract_end, status)
SELECT
    '自動生成顧客_' || LPAD(g::text, 3, '0'),
    '004_seed_bulk で投入',
    1 + ((g - 1) % 3),
    CURRENT_DATE - 40,
    CURRENT_DATE + 300,
    'active'
FROM generate_series(1, 18) AS g;

INSERT INTO job_postings (customer_id, summary, requirements, publish_start, publish_end, publication_status)
SELECT
    c.id,
    '自動案件_' || g::text || '_c' || c.id::text,
    '要件説明_' || g::text,
    CURRENT_DATE - (g % 20),
    CURRENT_DATE + (20 + (g % 80)),
    (ARRAY['draft', 'draft', 'published', 'ended']::text[])[1 + (g % 4)]
FROM generate_series(1, 50) AS g
CROSS JOIN LATERAL (
    SELECT id
    FROM customers
    ORDER BY id
    OFFSET ((g - 1) % (SELECT COUNT(*)::int FROM customers))
    LIMIT 1
) AS c;

INSERT INTO applications (job_posting_id, applicant_name, career_summary, contact)
SELECT
    jp.id,
    'バルク応募_' || g::text,
    '自動生成の職歴サマリ ' || g::text,
    'bulk_applicant_' || g::text || '@seed.local'
FROM generate_series(1, 35) AS g
CROSS JOIN LATERAL (
    SELECT id
    FROM job_postings
    ORDER BY id
    OFFSET ((g - 1) % GREATEST((SELECT COUNT(*)::int FROM job_postings), 1))
    LIMIT 1
) AS jp;

INSERT INTO invoices (customer_id, issued_at, amount_cents, status, notes)
SELECT
    c.id,
    CURRENT_DATE - (g % 45),
    10000 + (g * 137) % 500000,
    (ARRAY['draft', 'confirmed']::text[])[1 + (g % 2)],
    '一括請求_' || g::text
FROM generate_series(1, 25) AS g
CROSS JOIN LATERAL (
    SELECT id
    FROM customers
    ORDER BY id
    OFFSET ((g - 1) % (SELECT COUNT(*)::int FROM customers))
    LIMIT 1
) AS c;

INSERT INTO prospects (company_name, contact_info, notes)
SELECT
    '見込みバルク_' || g::text,
    'contact_' || g::text || '@bulk.example',
    '展示会フォロー ' || g::text
FROM generate_series(1, 28) AS g;
