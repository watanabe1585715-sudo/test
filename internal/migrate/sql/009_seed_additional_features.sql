-- 追加機能のダミーデータ

INSERT INTO company_profiles (customer_id, company_name, description, address, google_map_url, website_url, youtube_embed_url, accept_foreigners, languages)
VALUES
    (1, 'AAA企業', 'IT 人材採用を強化中のデモ企業。', '東京都千代田区1-1-1',
     'https://maps.google.com/?q=Tokyo+Station', 'https://example.com',
     'https://www.youtube.com/embed/dQw4w9WgXcQ', TRUE, 'ja,en'),
    (2, 'BBB企業', 'インフラ運用と業務改善に強い企業。', '大阪府大阪市1-2-3',
     'https://maps.google.com/?q=Osaka+Station', 'https://example.org',
     'https://www.youtube.com/embed/aqz-KE-bpKQ', FALSE, 'ja')
ON CONFLICT (customer_id) DO NOTHING;

INSERT INTO company_reviews (customer_id, reviewer, rating, comment)
VALUES
    (1, '口コミ太郎', 4, '裁量が大きく、技術投資も前向き。'),
    (1, '口コミ花子', 5, '面接が丁寧で雰囲気がよかった。'),
    (2, '口コミ次郎', 3, '繁忙期は忙しいが、学べる機会は多い。');

INSERT INTO company_videos (customer_id, title, youtube_url)
VALUES
    (1, '会社紹介動画', 'https://www.youtube.com/watch?v=dQw4w9WgXcQ'),
    (1, 'エンジニア採用メッセージ', 'https://www.youtube.com/watch?v=aqz-KE-bpKQ'),
    (2, 'オフィス紹介', 'https://www.youtube.com/watch?v=oHg5SJYRHA0');

INSERT INTO salary_market_rates (job_category, region, years_exp_min, years_exp_max, low_monthly_salary, median_monthly_salary, high_monthly_salary, source)
VALUES
    ('backend_engineer', 'KANTO', 0, 2, 300000, 360000, 450000, 'demo_seed'),
    ('backend_engineer', 'KANSAI', 0, 2, 290000, 340000, 430000, 'demo_seed'),
    ('backend_engineer', 'JP', 0, 2, 280000, 330000, 420000, 'demo_seed'),
    ('backend_engineer', 'JP', 3, 5, 400000, 520000, 700000, 'demo_seed'),
    ('frontend_engineer', 'KANTO', 0, 2, 290000, 350000, 440000, 'demo_seed'),
    ('frontend_engineer', 'KANSAI', 0, 2, 280000, 330000, 420000, 'demo_seed'),
    ('frontend_engineer', 'JP', 0, 2, 270000, 320000, 410000, 'demo_seed'),
    ('frontend_engineer', 'JP', 3, 5, 390000, 500000, 680000, 'demo_seed'),
    ('sales', 'KANTO', 0, 2, 260000, 320000, 400000, 'demo_seed'),
    ('sales', 'KANSAI', 0, 2, 250000, 300000, 380000, 'demo_seed'),
    ('sales', 'JP', 0, 2, 250000, 300000, 390000, 'demo_seed'),
    ('sales', 'JP', 3, 5, 330000, 430000, 560000, 'demo_seed');

INSERT INTO media_connections (customer_id, media_name, external_account, status, settings_json)
VALUES
    (1, 'Indeed', 'demo-account-indeed', 'active', '{"budget":"500000","auto_post":true}'),
    (1, '求人ボックス', 'demo-account-kyujinbox', 'active', '{"budget":"200000","auto_post":false}'),
    (2, 'Indeed', 'test-account-indeed', 'paused', '{"budget":"120000","auto_post":true}');

INSERT INTO media_inflow_stats (customer_id, media_name, measured_date, views, clicks, applications, hires)
VALUES
    (1, 'Indeed', CURRENT_DATE - 2, 2200, 380, 54, 3),
    (1, '求人ボックス', CURRENT_DATE - 2, 1200, 160, 20, 1),
    (2, 'Indeed', CURRENT_DATE - 2, 900, 90, 8, 0);

INSERT INTO interview_links (customer_id, job_posting_id, provider, meeting_url, scheduled_at)
VALUES
    (1, 1, 'google_meet', 'https://meet.google.com/demo-abc', NOW() + INTERVAL '1 day'),
    (1, 1, 'zoom', 'https://zoom.us/j/123456789', NOW() + INTERVAL '3 days');

INSERT INTO scouts (customer_id, job_posting_id, candidate_name, contact, message, status)
VALUES
    (1, 1, '応募太郎', 'oubo-taro@example.com', 'ご経験に興味があり、ご連絡しました。', 'sent'),
    (1, 1, '応募花子', 'oubo-hanako@example.com', 'ぜひ一度お話させてください。', 'draft');

INSERT INTO follow_up_policies (customer_id, enabled, max_follow_up_days, available_by_contract, notes)
VALUES
    (1, TRUE, 90, TRUE, 'tier3 のため長期フォローを許可'),
    (2, TRUE, 30, TRUE, 'tier2 のため 30 日まで')
ON CONFLICT (customer_id) DO NOTHING;

INSERT INTO report_snapshots (customer_id, report_kind, period_start, period_end, payload_json)
VALUES
    (1, 'monthly_inflow', CURRENT_DATE - 30, CURRENT_DATE, '{"indeed":{"views":2200,"applications":54},"kyujinbox":{"views":1200,"applications":20}}'),
    (1, 'hiring_funnel', CURRENT_DATE - 30, CURRENT_DATE, '{"apply":82,"interview":20,"offer":6,"join":3}');
