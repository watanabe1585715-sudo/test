"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { AdminNav } from "@/components/AdminNav";
import { apiOrigin, authHeaders } from "@/lib/api";

type EmailRow = {
  id: number;
  kind: string;
  to_email: string;
  subject: string;
  body: string;
  status: string;
  error_detail?: string | null;
  related_application_id?: number | null;
  created_at: string;
  sent_at?: string | null;
};

export default function EmailsPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [hasToken, setHasToken] = useState(false);
  const [statusFilter, setStatusFilter] = useState("");
  const [composeOpen, setComposeOpen] = useState(false);
  const [detail, setDetail] = useState<EmailRow | null>(null);
  const [form, setForm] = useState({ to_email: "", subject: "", body: "" });
  const [formErr, setFormErr] = useState<string | null>(null);

  useEffect(() => {
    if (!localStorage.getItem("cust_token")) router.replace("/");
    else setHasToken(true);
  }, [router]);

  const { data: emails = [], error, isPending, refetch } = useQuery<EmailRow[], Error>({
    queryKey: ["email-queue", statusFilter],
    enabled: hasToken,
    queryFn: async () => {
      const t = localStorage.getItem("cust_token")!;
      const u = new URL(`${apiOrigin}/admin/customers/email-queue`);
      if (statusFilter) u.searchParams.set("status", statusFilter);
      u.searchParams.set("limit", "150");
      const res = await fetch(u.toString(), { headers: authHeaders(t) });
      if (res.status === 401) {
        localStorage.removeItem("cust_token");
        router.replace("/");
        throw new Error("セッション切れ");
      }
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      return (data.emails || []) as EmailRow[];
    },
  });

  const sendMutation = useMutation({
    mutationFn: async () => {
      const t = localStorage.getItem("cust_token")!;
      const res = await fetch(`${apiOrigin}/admin/customers/email-queue`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders(t) },
        body: JSON.stringify(form),
      });
      if (!res.ok) throw new Error(await res.text());
    },
    onSuccess: async () => {
      setComposeOpen(false);
      setForm({ to_email: "", subject: "", body: "" });
      setFormErr(null);
      await queryClient.invalidateQueries({ queryKey: ["email-queue"] });
    },
    onError: (e: Error) => setFormErr(e.message),
  });

  const retryMutation = useMutation({
    mutationFn: async (id: number) => {
      const t = localStorage.getItem("cust_token")!;
      const res = await fetch(`${apiOrigin}/admin/customers/email-queue/${id}/retry`, {
        method: "POST",
        headers: authHeaders(t),
      });
      if (!res.ok) throw new Error(await res.text());
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["email-queue"] });
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
        <div className="row" style={{ justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
          <h1 style={{ margin: 0 }}>メールキュー</h1>
          <button type="button" onClick={() => setComposeOpen(true)}>
            手動でキューに追加
          </button>
        </div>
        <p style={{ color: "#737373", fontSize: "0.9rem", marginTop: 0 }}>
          応募時に連絡先がメール形式のとき自動で積まれます。送信は <code>mailworker</code>（SMTP 設定時）を実行してください。
        </p>
        <div className="row" style={{ marginBottom: "1rem", flexWrap: "wrap", gap: "0.5rem" }}>
          <label>
            状態:{" "}
            <select value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
              <option value="">すべて</option>
              <option value="pending">pending</option>
              <option value="sent">sent</option>
              <option value="failed">failed</option>
              <option value="skipped">skipped</option>
            </select>
          </label>
          <button type="button" className="secondary" onClick={() => void refetch()}>
            再読込
          </button>
        </div>
        {isPending && <p>読み込み中…</p>}
        {errMsg && <p className="err">{errMsg}</p>}
        <div style={{ overflowX: "auto" }}>
          <table style={{ width: "100%", borderCollapse: "collapse", fontSize: "0.9rem" }}>
            <thead>
              <tr style={{ textAlign: "left", borderBottom: "1px solid #e5e5e5" }}>
                <th style={{ padding: "0.35rem" }}>ID</th>
                <th style={{ padding: "0.35rem" }}>状態</th>
                <th style={{ padding: "0.35rem" }}>種別</th>
                <th style={{ padding: "0.35rem" }}>宛先</th>
                <th style={{ padding: "0.35rem" }}>件名</th>
                <th style={{ padding: "0.35rem" }}>作成</th>
                <th style={{ padding: "0.35rem" }} />
              </tr>
            </thead>
            <tbody>
              {emails.map((e) => (
                <tr key={e.id} style={{ borderBottom: "1px solid #f5f5f5" }}>
                  <td style={{ padding: "0.35rem" }}>{e.id}</td>
                  <td style={{ padding: "0.35rem" }}>{e.status}</td>
                  <td style={{ padding: "0.35rem" }}>{e.kind}</td>
                  <td style={{ padding: "0.35rem" }}>{e.to_email}</td>
                  <td style={{ padding: "0.35rem" }}>{e.subject}</td>
                  <td style={{ padding: "0.35rem", whiteSpace: "nowrap" }}>{new Date(e.created_at).toLocaleString("ja-JP")}</td>
                  <td style={{ padding: "0.35rem" }}>
                    <button type="button" className="secondary" style={{ fontSize: "0.8rem" }} onClick={() => setDetail(e)}>
                      詳細
                    </button>
                    {e.status === "failed" && (
                      <button
                        type="button"
                        style={{ fontSize: "0.8rem", marginLeft: "0.25rem" }}
                        onClick={() => retryMutation.mutate(e.id)}
                        disabled={retryMutation.isPending}
                      >
                        再送予約
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {composeOpen && (
          <div className="modal" onClick={() => setComposeOpen(false)} role="presentation">
            <div className="modal-inner" onClick={(ev) => ev.stopPropagation()}>
              <h2>メールをキューに追加</h2>
              <p style={{ fontSize: "0.85rem", color: "#737373" }}>即時送信ではなく pending として積みます。</p>
              <label>宛先（メール）</label>
              <input style={{ width: "100%" }} value={form.to_email} onChange={(ev) => setForm({ ...form, to_email: ev.target.value })} />
              <label>件名</label>
              <input style={{ width: "100%" }} value={form.subject} onChange={(ev) => setForm({ ...form, subject: ev.target.value })} />
              <label>本文</label>
              <textarea rows={6} style={{ width: "100%" }} value={form.body} onChange={(ev) => setForm({ ...form, body: ev.target.value })} />
              {formErr && <p className="err">{formErr}</p>}
              <div className="row" style={{ justifyContent: "flex-end", marginTop: "0.75rem", gap: "0.35rem" }}>
                <button className="secondary" type="button" onClick={() => setComposeOpen(false)}>
                  キャンセル
                </button>
                <button type="button" onClick={() => sendMutation.mutate()} disabled={sendMutation.isPending}>
                  追加
                </button>
              </div>
            </div>
          </div>
        )}

        {detail && (
          <div className="modal" onClick={() => setDetail(null)} role="presentation">
            <div className="modal-inner" onClick={(ev) => ev.stopPropagation()}>
              <h2>メール #{detail.id}</h2>
              <p>
                <strong>宛先</strong> {detail.to_email}
              </p>
              <p>
                <strong>件名</strong> {detail.subject}
              </p>
              <p>
                <strong>状態</strong> {detail.status}
              </p>
              {detail.error_detail && (
                <p className="err" style={{ whiteSpace: "pre-wrap" }}>
                  {detail.error_detail}
                </p>
              )}
              <pre style={{ whiteSpace: "pre-wrap", background: "#fafafa", padding: "0.75rem", borderRadius: 6 }}>{detail.body}</pre>
              <div className="row" style={{ justifyContent: "flex-end" }}>
                <button className="secondary" type="button" onClick={() => setDetail(null)}>
                  閉じる
                </button>
              </div>
            </div>
          </div>
        )}
      </main>
    </>
  );
}
