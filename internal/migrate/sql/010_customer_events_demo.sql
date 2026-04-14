-- 顧客 #20（004_seed_bulk の最終行付近）のスケジュール・履歴用デモ行
INSERT INTO customer_events (customer_id, event_kind, occurred_at, title, body, is_risk_related)
SELECT
    20,
    'meeting',
    NOW() - INTERVAL '14 days',
    '初回打ち合わせ（AAA商事様）',
    'オンラインにて採用課題と希望条件をヒアリング。次回は媒体選定の擦り合わせ予定。',
    FALSE
WHERE EXISTS (SELECT 1 FROM customers WHERE id = 20);

INSERT INTO customer_events (customer_id, event_kind, occurred_at, title, body, is_risk_related)
SELECT
    20,
    'contract_start',
    NOW() - INTERVAL '9 days',
    '本サービス利用開始',
    '契約書締結済み。顧客管理サイトのアカウント発行を完了。',
    FALSE
WHERE EXISTS (SELECT 1 FROM customers WHERE id = 20);

INSERT INTO customer_events (customer_id, event_kind, occurred_at, title, body, is_risk_related)
SELECT
    20,
    'note',
    NOW() - INTERVAL '4 days',
    '応募太郎様 応募フォロー',
    '辞退理由のヒアリングを実施。待遇面の認識差あり。次回提案資料を準備。',
    FALSE
WHERE EXISTS (SELECT 1 FROM customers WHERE id = 20);

INSERT INTO customer_events (customer_id, event_kind, occurred_at, title, body, is_risk_related)
SELECT
    20,
    'risk_flag',
    NOW() - INTERVAL '1 days',
    '危険顧客フラグの棚卸し',
    '滞納履歴なし。担当交代に伴う定期レビュー。継続監視は不要と判断。',
    TRUE
WHERE EXISTS (SELECT 1 FROM customers WHERE id = 20);
