-- 試験用の追加ダミーデータ（顧客2・案件・応募・請求・見込みを増やす）
INSERT INTO customers (name, description, contract_tier, contract_start, contract_end, status)
VALUES (
        'AAA企業',
        '契約 tier 2（掲載100件まで）のデモ顧客',
        2,
        CURRENT_DATE - 60,
        CURRENT_DATE + 180,
        'active'
    );

INSERT INTO job_admin_users (customer_id, email, password_hash)
VALUES (
        2,
        'jobadmin2@example.com',
        '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'
    );

INSERT INTO job_postings (customer_id, summary, requirements, publish_start, publish_end, publication_status)
VALUES
    (
        1,
        'すぐ表示チェック用 published',
        '求人サイト一覧に即表示されます。React の動作確認向け。',
        CURRENT_DATE,
        CURRENT_DATE + 60,
        'published'
    ),
    (
        1,
        '掲載終了サンプル ended',
        '期間外のため一覧には出ません。',
        CURRENT_DATE - 90,
        CURRENT_DATE - 1,
        'ended'
    ),
    (
        1,
        '近日開始の下書き draft',
        '掲載開始が未来日。バッチ後に枠が空ければ published 候補。',
        CURRENT_DATE + 3,
        CURRENT_DATE + 45,
        'draft'
    ),
    (
        2,
        'AAA企業インフラエンジニア募集',
        'インフラと業務アプリの保守開発',
        CURRENT_DATE,
        CURRENT_DATE + 14,
        'draft'
    );

INSERT INTO applications (job_posting_id, applicant_name, career_summary, contact)
VALUES
    (1, '応募太郎', 'SIerとして3年。Java と SQL が主務。', 'oubo-taro@example.com'),
    (1, '応募花子', 'フロントエンド2年 TypeScript Vue', 'oubo-hanako@example.com'),
    (1, '応募次郎', '新卒。インターンで Spring Boot', 'oubo-jiro@example.com');

INSERT INTO invoices (customer_id, issued_at, amount_cents, status, notes)
VALUES
    (1, CURRENT_DATE, 120000, 'confirmed', '2026年4月分'),
    (1, CURRENT_DATE - 30, 98000, 'confirmed', '前月分'),
    (2, CURRENT_DATE - 7, 55000, 'draft', '見積ベースのドラフト');

INSERT INTO prospects (company_name, contact_info, notes)
VALUES
    ('BBB企業', 'bbb-sales@example.co.jp', '来週デモ予定'),
    ('CCC企業', '050-1234-5678', '資料送付済み'),
    ('DDD企業', NULL, 'Web フォームからの問い合わせのみ');
