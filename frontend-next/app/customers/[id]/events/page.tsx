"use client";

/**
 * 顧客のスケジュール・履歴（打ち合わせ・契約開始・特記・リスク関連など）。
 * API: GET/POST/PATCH/DELETE /admin/customers/customers/:id/events
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { AdminNav } from "@/components/AdminNav";
import { apiOrigin, authHeaders } from "@/lib/api";

type CustomerEvent = {
  id: number;
  customer_id: number;
  event_kind: string;
  occurred_at: string;
  title: string;
  body?: string | null;
  is_risk_related: boolean;
  created_at: string;
};

const KINDS = [
  { v: "meeting", label: "打ち合わせ" },
  { v: "contract_start", label: "利用開始" },
  { v: "risk_flag", label: "リスク・要注意" },
  { v: "note", label: "特記・メモ" },
  { v: "other", label: "その他" },
];

export default function CustomerEventsPage() {
  const router = useRouter();
  const params = useParams();
  const customerId = Number(params.id);
  const qc = useQueryClient();
  const [hasToken, setHasToken] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [modal, setModal] = useState<CustomerEvent | "new" | null>(null);
  const [form, setForm] = useState({
    event_kind: "meeting",
    occurred_at: "",
    title: "",
    body: "",
    is_risk_related: false,
  });

  useEffect(() => {
    if (!localStorage.getItem("cust_token")) router.replace("/");
    else setHasToken(true);
  }, [router]);

  const { data: events = [], isPending } = useQuery<CustomerEvent[]>({
    queryKey: ["customer-events", customerId],
    enabled: hasToken && Number.isFinite(customerId),
    queryFn: async () => {
      const t = localStorage.getItem("cust_token")!;
      const res = await fetch(`${apiOrigin}/admin/customers/customers/${customerId}/events`, { headers: authHeaders(t) });
      if (res.status === 401) {
        router.replace("/");
        return [];
      }
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      return (data.events || []) as CustomerEvent[];
    },
  });

  const saveMut = useMutation({
    mutationFn: async () => {
      const t = localStorage.getItem("cust_token")!;
      const occurredISO = new Date(form.occurred_at).toISOString();
      const payload = {
        event_kind: form.event_kind,
        occurred_at: occurredISO,
        title: form.title,
        body: form.body,
        is_risk_related: form.is_risk_related,
      };
      if (modal === "new") {
        const res = await fetch(`${apiOrigin}/admin/customers/customers/${customerId}/events`, {
          method: "POST",
          headers: { "Content-Type": "application/json", ...authHeaders(t) },
          body: JSON.stringify(payload),
        });
        if (!res.ok) throw new Error(await res.text());
        return;
      }
      const ev = modal as CustomerEvent;
      const res = await fetch(`${apiOrigin}/admin/customers/customers/${customerId}/events/${ev.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", ...authHeaders(t) },
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(await res.text());
    },
    onSuccess: async () => {
      setModal(null);
      setErr(null);
      await qc.invalidateQueries({ queryKey: ["customer-events", customerId] });
    },
    onError: (e: Error) => setErr(e.message),
  });

  async function deleteEvent(id: number) {
    if (!confirm("削除しますか？")) return;
    const t = localStorage.getItem("cust_token")!;
    const res = await fetch(`${apiOrigin}/admin/customers/customers/${customerId}/events/${id}`, {
      method: "DELETE",
      headers: authHeaders(t),
    });
    if (!res.ok) {
      setErr(await res.text());
      return;
    }
    setModal(null);
    await qc.invalidateQueries({ queryKey: ["customer-events", customerId] });
  }

  function logout() {
    localStorage.removeItem("cust_token");
    router.replace("/");
  }

  function openNew() {
    const now = new Date();
    const local = new Date(now.getTime() - now.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
    setForm({ event_kind: "meeting", occurred_at: local, title: "", body: "", is_risk_related: false });
    setErr(null);
    setModal("new");
  }

  function openEdit(e: CustomerEvent) {
    const d = new Date(e.occurred_at);
    const local = new Date(d.getTime() - d.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
    setForm({
      event_kind: e.event_kind,
      occurred_at: local,
      title: e.title,
      body: e.body || "",
      is_risk_related: e.is_risk_related,
    });
    setErr(null);
    setModal(e);
  }

  return (
    <>
      <AdminNav onLogout={logout} />
      <main>
        <div className="row" style={{ justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
          <h1 style={{ margin: 0 }}>顧客 #{customerId} — スケジュール・履歴</h1>
          <div className="row" style={{ gap: "0.5rem" }}>
            <Link href="/customers">← 顧客一覧</Link>
            <button type="button" onClick={() => openNew()}>
              イベント追加
            </button>
          </div>
        </div>
        <p style={{ color: "#737373", fontSize: "0.9rem" }}>
          打ち合わせ日時・利用開始・危険顧客フラグ・特記事項などを時系列で記録します。
        </p>
        {isPending && <p>読み込み中…</p>}
        {err && <p className="err">{err}</p>}
        {events.map((e) => (
          <button key={e.id} type="button" className="list-btn" style={{ textAlign: "left", marginBottom: "0.35rem" }} onClick={() => openEdit(e)}>
            <strong>{e.title}</strong>
            <span style={{ color: "#737373", marginLeft: "0.5rem", fontSize: "0.85rem" }}>
              {KINDS.find((k) => k.v === e.event_kind)?.label || e.event_kind}
            </span>
            {e.is_risk_related && (
              <span style={{ marginLeft: "0.35rem", color: "#b91c1c", fontSize: "0.8rem" }}>要注意</span>
            )}
            <div style={{ fontSize: "0.85rem", color: "#737373" }}>{new Date(e.occurred_at).toLocaleString("ja-JP")}</div>
            {e.body && <div style={{ fontSize: "0.9rem", marginTop: "0.25rem", whiteSpace: "pre-wrap" }}>{e.body}</div>}
          </button>
        ))}

        {modal && (
          <div className="modal" onClick={() => setModal(null)} role="presentation">
            <div className="modal-inner" onClick={(ev) => ev.stopPropagation()}>
              <h2>{modal === "new" ? "イベント追加" : "イベント編集"}</h2>
              <label>種別</label>
              <select value={form.event_kind} onChange={(e) => setForm({ ...form, event_kind: e.target.value })} style={{ width: "100%" }}>
                {KINDS.map((k) => (
                  <option key={k.v} value={k.v}>
                    {k.label}
                  </option>
                ))}
              </select>
              <label>日時（ローカル）</label>
              <input type="datetime-local" value={form.occurred_at} onChange={(e) => setForm({ ...form, occurred_at: e.target.value })} style={{ width: "100%" }} />
              <label>タイトル</label>
              <input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} style={{ width: "100%" }} />
              <label>本文</label>
              <textarea rows={4} value={form.body} onChange={(e) => setForm({ ...form, body: e.target.value })} style={{ width: "100%" }} />
              <label>
                <input type="checkbox" checked={form.is_risk_related} onChange={(e) => setForm({ ...form, is_risk_related: e.target.checked })} />{" "}
                リスク・要注意に関連
              </label>
              {err && <p className="err">{err}</p>}
              <div className="row" style={{ justifyContent: "space-between", marginTop: "0.75rem" }}>
                {modal !== "new" && (
                  <button className="danger" type="button" onClick={() => void deleteEvent((modal as CustomerEvent).id)}>
                    削除
                  </button>
                )}
                <div className="row" style={{ gap: "0.35rem", marginLeft: "auto" }}>
                  <button className="secondary" type="button" onClick={() => setModal(null)}>
                    キャンセル
                  </button>
                  <button type="button" onClick={() => saveMut.mutate()} disabled={saveMut.isPending}>
                    保存
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}
      </main>
    </>
  );
}
