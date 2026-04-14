-- bcrypt: password (開発用)
INSERT INTO customer_admin_users (email, password_hash)
VALUES (
        'admin@example.com',
        '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'
    );

INSERT INTO customers (name, description, contract_tier, contract_start, contract_end, status)
VALUES (
        'デモ広告主株式会社',
        '契約中のデモ顧客',
        3,
        CURRENT_DATE - 30,
        CURRENT_DATE + 365,
        'active'
    );

INSERT INTO job_admin_users (customer_id, email, password_hash)
VALUES (
        1,
        'jobadmin@example.com',
        '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'
    );

INSERT INTO prospects (company_name, contact_info, notes)
VALUES ('見込み商事', 'tel: 03-0000-0000', '来期契約予定');

INSERT INTO job_postings (customer_id, summary, requirements, publish_start, publish_end, publication_status)
VALUES (
        1,
        'デモ求人（バッチ実行後に公開サイトへ表示）',
        'Go / React 経験者歓迎',
        CURRENT_DATE,
        CURRENT_DATE + 30,
        'draft'
    );
