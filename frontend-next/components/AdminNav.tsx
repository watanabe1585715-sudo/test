"use client";

import Link from "next/link";

type Props = {
  onLogout?: () => void;
};

/** 顧客管理サイト共通ナビ（各ページから再利用）。お知らせ表示はログイン後トップ（/customers）のみ。 */
export function AdminNav({ onLogout }: Props) {
  return (
    <nav className="nav">
      <Link href="/customers">顧客一覧</Link>
      <Link href="/applications">応募一覧</Link>
      <Link href="/prospects">見込み顧客</Link>
      <Link href="/emails">メールキュー</Link>
      <Link href="/announcements">お知らせ管理</Link>
      <Link href="/customer-admins">管理者アカウント</Link>
      {onLogout && (
        <button className="secondary" type="button" style={{ marginLeft: "auto" }} onClick={onLogout}>
          ログアウト
        </button>
      )}
    </nav>
  );
}
