"use client";

import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { AdminNav } from "@/components/AdminNav";
import { apiOrigin, authHeaders } from "@/lib/api";

type Row = {
  id: number;
  job_posting_id: number;
  job_summary: string;
  customer_id: number;
  customer_name: string;
  applicant_name: string;
  career_summary: string;
  contact: string;
  created_at: string;
};

export default function ApplicationsPage() {
  const router = useRouter();
  const [hasToken, setHasToken] = useState(false);
  const [q, setQ] = useState("");

  useEffect(() => {
    if (!localStorage.getItem("cust_token")) router.replace("/");
    else setHasToken(true);
  }, [router]);

  const { data: rows = [], error, isPending, refetch } = useQuery<Row[], Error>({
    queryKey: ["applications", q],
    enabled: hasToken,
    queryFn: async () => {
      const t = localStorage.getItem("cust_token")!;
      const u = new URL(`${apiOrigin}/admin/customers/applications`);
      if (q) u.searchParams.set("q", q);
      const res = await fetch(u.toString(), { headers: authHeaders(t) });
      if (res.status === 401) {
        localStorage.removeItem("cust_token");
        router.replace("/");
        throw new Error("セッション切れ");
      }
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      return (data.applications || []) as Row[];
    },
  });

  function logout() {
    localStorage.removeItem("cust_token");
    router.replace("/");
  }

  const errMsg = error instanceof Error ? error.message : error ? String(error) : null;

  return (
    <>
      <AdminNav onLogout={logout} />
      <main>
        <h1 style={{ marginTop: 0 }}>応募一覧（全顧客）</h1>
        <p style={{ color: "#737373", fontSize: "0.9rem" }}>案件名・応募者・連絡先から検索できます。</p>
        <div className="row" style={{ marginBottom: "1rem" }}>
          <input placeholder="検索" value={q} onChange={(e) => setQ(e.target.value)} style={{ flex: 1, minWidth: 220 }} />
          <button type="button" onClick={() => void refetch()}>
            検索
          </button>
        </div>
        {isPending && <p>読み込み中…</p>}
        {errMsg && <p className="err">{errMsg}</p>}
        <div style={{ display: "flex", flexDirection: "column", gap: "0.5rem" }}>
          {rows.map((r) => (
            <div key={r.id} className="card" style={{ padding: "0.75rem 1rem" }}>
              <div style={{ display: "flex", flexWrap: "wrap", gap: "0.5rem", alignItems: "baseline" }}>
                <strong>{r.applicant_name}</strong>
                <span style={{ color: "#737373", fontSize: "0.85rem" }}>
                  {r.customer_name} / {r.job_summary}
                </span>
              </div>
              <div style={{ fontSize: "0.85rem", marginTop: "0.35rem" }}>
                連絡先: {r.contact} — {new Date(r.created_at).toLocaleString("ja-JP")}
              </div>
              <pre style={{ margin: "0.5rem 0 0", fontSize: "0.8rem", whiteSpace: "pre-wrap", color: "#525252" }}>{r.career_summary}</pre>
            </div>
          ))}
        </div>
      </main>
    </>
  );
}
