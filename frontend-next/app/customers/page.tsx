"use client";

/**
 * 顧客管理の中心画面（顧客 CRUD・契約終了・案件管理ユーザー・請求の入口）。
 * API: Bearer 付き /admin/customers/*
 */
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { AdminNav } from "@/components/AdminNav";
import { AnnouncementBanner } from "@/components/AnnouncementBanner";
import { apiOrigin, authHeaders } from "@/lib/api";

type Customer = {
  id: number;
  name: string;
  description?: string | null;
  contract_tier: number;
  contract_start: string;
  contract_end?: string | null;
  status: string;
  approval_status: string;
};

type JobUser = { id: number; email: string; active: boolean };

type InvoiceRow = {
  id: number;
  customer_id: number;
  issued_at: string;
  amount_cents: number;
  status: string;
  notes?: string | null;
  created_at: string;
};
type FollowUpCapability = { enabled: boolean; reason: string; max_follow_up_days: number };

export default function CustomersPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [q, setQ] = useState("");
  const [hasToken, setHasToken] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [modal, setModal] = useState<Customer | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState({
    name: "",
    description: "",
    contract_tier: 3,
    contract_start: "",
    contract_end: "",
    approval_status: "pending",
  });
  const [jobUsers, setJobUsers] = useState<JobUser[]>([]);
  const [newUser, setNewUser] = useState({ email: "", password: "" });
  const [invoices, setInvoices] = useState<InvoiceRow[]>([]);
  const [followUpCapability, setFollowUpCapability] = useState<FollowUpCapability | null>(null);
  const [invForm, setInvForm] = useState({ customer_id: 0, issued_at: "", amount_cents: 10000, status: "draft", notes: "" });

  useEffect(() => {
    const t = localStorage.getItem("cust_token");
    if (!t) router.replace("/");
    else setHasToken(true);
  }, [router]);

  const {
    data: list = [],
    error: qErr,
    isPending,
    refetch,
  } = useQuery<Customer[], Error>({
    queryKey: ["customers", q],
    enabled: hasToken,
    queryFn: async () => {
      const t = localStorage.getItem("cust_token");
      if (!t) return [];
      const u = new URL(`${apiOrigin}/admin/customers/customers`);
      if (q) u.searchParams.set("q", q);
      const res = await fetch(u.toString(), { headers: authHeaders(t) });
      if (res.status === 401) {
        localStorage.removeItem("cust_token");
        router.replace("/");
        throw new Error("セッションが切れました");
      }
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      return (data.customers || []) as Customer[];
    },
  });

  async function loadJobUsers(cid: number) {
    const t = localStorage.getItem("cust_token")!;
    const res = await fetch(`${apiOrigin}/admin/customers/customers/${cid}/job-users`, { headers: authHeaders(t) });
    if (!res.ok) return;
    const data = await res.json();
    setJobUsers(data.users || []);
  }

  async function loadInvoices() {
    const t = localStorage.getItem("cust_token")!;
    const res = await fetch(`${apiOrigin}/admin/customers/invoices`, { headers: authHeaders(t) });
    if (!res.ok) return;
    const data = await res.json();
    setInvoices(data.invoices || []);
  }
  async function loadFollowUpCapability(cid: number) {
    const t = localStorage.getItem("cust_token")!;
    const res = await fetch(`${apiOrigin}/admin/customers/customers/${cid}/follow-up-capabilities`, { headers: authHeaders(t) });
    if (!res.ok) return;
    setFollowUpCapability(await res.json());
  }

  async function quickSetCustomerApproval(id: number, approval_status: "approved" | "rejected") {
    const t = localStorage.getItem("cust_token")!;
    const getRes = await fetch(`${apiOrigin}/admin/customers/customers/${id}`, { headers: authHeaders(t) });
    if (!getRes.ok) {
      setErr(await getRes.text());
      return;
    }
    const cust = (await getRes.json()) as Customer;
    const body: Record<string, unknown> = {
      name: cust.name,
      description: cust.description ?? "",
      contract_tier: cust.contract_tier,
      contract_start: String(cust.contract_start).slice(0, 10),
      approval_status,
    };
    if (cust.contract_end) body.contract_end = String(cust.contract_end).slice(0, 10);
    const res = await fetch(`${apiOrigin}/admin/customers/customers/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json", ...authHeaders(t) },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      setErr(await res.text());
      return;
    }
    await queryClient.invalidateQueries({ queryKey: ["customers"] });
  }

  function openEdit(c: Customer) {
    setModal(c);
    setForm({
      name: c.name,
      description: c.description || "",
      contract_tier: c.contract_tier,
      contract_start: String(c.contract_start).slice(0, 10),
      contract_end: c.contract_end ? String(c.contract_end).slice(0, 10) : "",
      approval_status: c.approval_status || "approved",
    });
    void loadJobUsers(c.id);
    setInvForm((f) => ({ ...f, customer_id: c.id }));
    void loadInvoices();
    void loadFollowUpCapability(c.id);
  }

  async function saveCustomer(id?: number) {
    const t = localStorage.getItem("cust_token")!;
    const body: Record<string, unknown> = {
      name: form.name,
      description: form.description,
      contract_tier: form.contract_tier,
      contract_start: form.contract_start,
    };
    if (form.contract_end) body.contract_end = form.contract_end;
    if (id) body.approval_status = form.approval_status;
    const url = id ? `${apiOrigin}/admin/customers/customers/${id}` : `${apiOrigin}/admin/customers/customers`;
    const res = await fetch(url, {
      method: id ? "PATCH" : "POST",
      headers: { "Content-Type": "application/json", ...authHeaders(t) },
      body: JSON.stringify(body),
    });
    if (!res.ok) {
      setErr(await res.text());
      return;
    }
    setCreateOpen(false);
    setModal(null);
    await queryClient.invalidateQueries({ queryKey: ["customers"] });
  }

  async function endContract(id: number) {
    if (!confirm("契約終了しますか？")) return;
    const t = localStorage.getItem("cust_token")!;
    const res = await fetch(`${apiOrigin}/admin/customers/customers/${id}/end-contract`, {
      method: "POST",
      headers: authHeaders(t),
    });
    if (!res.ok) {
      setErr(await res.text());
      return;
    }
    setModal(null);
    await queryClient.invalidateQueries({ queryKey: ["customers"] });
  }

  async function addJobUser(cid: number) {
    const t = localStorage.getItem("cust_token")!;
    const res = await fetch(`${apiOrigin}/admin/customers/customers/${cid}/job-users`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders(t) },
      body: JSON.stringify(newUser),
    });
    if (!res.ok) {
      setErr(await res.text());
      return;
    }
    setNewUser({ email: "", password: "" });
    await loadJobUsers(cid);
  }

  async function deleteJobUser(cid: number, uid: number) {
    if (!confirm("削除しますか？")) return;
    const t = localStorage.getItem("cust_token")!;
    const res = await fetch(`${apiOrigin}/admin/customers/job-users/${uid}?customer_id=${cid}`, {
      method: "DELETE",
      headers: authHeaders(t),
    });
    if (!res.ok) {
      setErr(await res.text());
      return;
    }
    await loadJobUsers(cid);
  }

  async function createInvoice() {
    const t = localStorage.getItem("cust_token")!;
    const res = await fetch(`${apiOrigin}/admin/customers/invoices`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders(t) },
      body: JSON.stringify(invForm),
    });
    if (!res.ok) {
      setErr(await res.text());
      return;
    }
    await loadInvoices();
  }

  function logout() {
    localStorage.removeItem("cust_token");
    router.replace("/");
  }

  return (
    <>
      <AdminNav onLogout={logout} />
      <div style={{ maxWidth: 960, margin: "0 auto", padding: "0.75rem 1.25rem 0" }}>
        <AnnouncementBanner variant="customer_feed" />
      </div>
      <main>
        <div className="row" style={{ justifyContent: "space-between", marginBottom: "1rem" }}>
          <h1 style={{ margin: 0 }}>顧客一覧</h1>
          <button
            type="button"
            onClick={() => {
              setForm({
                name: "",
                description: "",
                contract_tier: 3,
                contract_start: "",
                contract_end: "",
                approval_status: "pending",
              });
              setCreateOpen(true);
            }}
          >
            新規作成
          </button>
        </div>
        <div className="row" style={{ marginBottom: "1rem" }}>
          <input placeholder="検索" value={q} onChange={(e) => setQ(e.target.value)} style={{ flex: 1, minWidth: 220 }} />
          <button type="button" onClick={() => void refetch()}>
            検索
          </button>
        </div>
        {isPending && <p>読み込み中…</p>}
        {(() => {
          const qMsg = qErr instanceof Error ? qErr.message : qErr ? String(qErr) : null;
          const msg = err || qMsg;
          return msg ? <p className="err">{msg}</p> : null;
        })()}
        {list.map((c) => (
          <div key={c.id} className="row" style={{ alignItems: "stretch", gap: "0.5rem", marginBottom: "0.5rem" }}>
            <button type="button" className="list-btn" style={{ flex: 1, textAlign: "left" }} onClick={() => openEdit(c)}>
              <strong>{c.name}</strong>
              <span style={{ color: "#737373", marginLeft: "0.5rem" }}>tier {c.contract_tier}</span>
              <span
                style={{
                  marginLeft: "0.5rem",
                  fontSize: "0.8rem",
                  color: c.approval_status === "approved" ? "#15803d" : c.approval_status === "pending" ? "#a16207" : "#b91c1c",
                }}
              >
                [{c.approval_status === "approved" ? "利用承認済" : c.approval_status === "pending" ? "利用承認待ち" : "却下"}]
              </span>
              <div style={{ fontSize: "0.85rem", color: "#737373" }}>
                {String(c.contract_start).slice(0, 10)} 〜 {c.contract_end ? String(c.contract_end).slice(0, 10) : "—"} ({c.status})
              </div>
            </button>
            <div style={{ display: "flex", flexDirection: "column", gap: "0.25rem", justifyContent: "center", minWidth: "7rem" }}>
              <Link href={`/customers/${c.id}/events`} style={{ fontSize: "0.85rem" }}>
                履歴・予定
              </Link>
              {c.approval_status === "pending" && (
                <>
                  <button type="button" className="secondary" style={{ fontSize: "0.8rem" }} onClick={() => void quickSetCustomerApproval(c.id, "approved")}>
                    利用承認
                  </button>
                  <button type="button" className="danger" style={{ fontSize: "0.8rem" }} onClick={() => void quickSetCustomerApproval(c.id, "rejected")}>
                    却下
                  </button>
                </>
              )}
            </div>
          </div>
        ))}

        {createOpen && (
          <div className="modal" onClick={() => setCreateOpen(false)} role="presentation">
            <div className="modal-inner" onClick={(e) => e.stopPropagation()}>
              <h2>顧客作成</h2>
              <p style={{ fontSize: "0.85rem", color: "#737373" }}>新規顧客は自動で「利用承認待ち」になります。承認後に公開求人へ反映されます。</p>
              <label>名前</label>
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} style={{ width: "100%" }} />
              <label>概要</label>
              <textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} rows={3} style={{ width: "100%" }} />
              <label>契約レベル (1=10件,2=100件,3=無制限)</label>
              <input
                type="number"
                min={1}
                max={3}
                value={form.contract_tier}
                onChange={(e) => setForm({ ...form, contract_tier: Number(e.target.value) })}
                style={{ width: "100%" }}
              />
              <label>契約開始</label>
              <input type="date" value={form.contract_start} onChange={(e) => setForm({ ...form, contract_start: e.target.value })} style={{ width: "100%" }} />
              <label>契約終了（任意）</label>
              <input type="date" value={form.contract_end} onChange={(e) => setForm({ ...form, contract_end: e.target.value })} style={{ width: "100%" }} />
              <div className="row" style={{ marginTop: "0.75rem", justifyContent: "flex-end" }}>
                <button className="secondary" type="button" onClick={() => setCreateOpen(false)}>
                  キャンセル
                </button>
                <button type="button" onClick={() => void saveCustomer()}>
                  登録
                </button>
              </div>
            </div>
          </div>
        )}

        {modal && (
          <div className="modal" onClick={() => setModal(null)} role="presentation">
            <div className="modal-inner" onClick={(e) => e.stopPropagation()}>
              <h2>顧客更新</h2>
              <p style={{ marginTop: 0 }}>
                <Link href={`/customers/${modal.id}/events`}>スケジュール・履歴を開く</Link>
              </p>
              <label>名前</label>
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} style={{ width: "100%" }} />
              <label>概要</label>
              <textarea value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} rows={3} style={{ width: "100%" }} />
              <label>契約レベル</label>
              <input
                type="number"
                min={1}
                max={3}
                value={form.contract_tier}
                onChange={(e) => setForm({ ...form, contract_tier: Number(e.target.value) })}
                style={{ width: "100%" }}
              />
              <label>契約開始</label>
              <input type="date" value={form.contract_start} onChange={(e) => setForm({ ...form, contract_start: e.target.value })} style={{ width: "100%" }} />
              <label>契約終了（任意）</label>
              <input type="date" value={form.contract_end} onChange={(e) => setForm({ ...form, contract_end: e.target.value })} style={{ width: "100%" }} />
              <label>利用承認（公開求人・応募の可否）</label>
              <select value={form.approval_status} onChange={(e) => setForm({ ...form, approval_status: e.target.value })} style={{ width: "100%" }}>
                <option value="pending">承認待ち</option>
                <option value="approved">承認済</option>
                <option value="rejected">却下</option>
              </select>
              <h3 style={{ marginTop: "1rem" }}>案件管理ユーザー</h3>
              {jobUsers.map((u) => (
                <div key={u.id} className="row" style={{ justifyContent: "space-between" }}>
                  <span>
                    {u.email} {u.active ? "" : "(無効)"}
                  </span>
                  <button className="danger" type="button" onClick={() => void deleteJobUser(modal.id, u.id)}>
                    削除
                  </button>
                </div>
              ))}
              <label>新規メール</label>
              <input value={newUser.email} onChange={(e) => setNewUser({ ...newUser, email: e.target.value })} style={{ width: "100%" }} />
              <label>新規パスワード</label>
              <input type="password" value={newUser.password} onChange={(e) => setNewUser({ ...newUser, password: e.target.value })} style={{ width: "100%" }} />
              <button type="button" style={{ marginTop: "0.35rem" }} onClick={() => void addJobUser(modal.id)}>
                ユーザ追加
              </button>

              <h3 style={{ marginTop: "1rem" }}>請求書</h3>
              {followUpCapability && (
                <div style={{ background: "#f8fafc", border: "1px solid #e2e8f0", borderRadius: 8, padding: "0.5rem 0.75rem", marginBottom: "0.75rem" }}>
                  <strong>入社後フォローアップ可否:</strong>{" "}
                  {followUpCapability.enabled ? `利用可（最大 ${followUpCapability.max_follow_up_days} 日）` : `利用不可（${followUpCapability.reason}）`}
                </div>
              )}
              <div style={{ fontSize: "0.9rem", maxHeight: 140, overflow: "auto" }}>
                {invoices
                  .filter((i) => i.customer_id === modal.id)
                  .map((i) => (
                    <div key={i.id} style={{ borderBottom: "1px solid #eee", padding: "0.25rem 0" }}>
                      {i.issued_at} — {i.amount_cents} 円 ({i.status})
                    </div>
                  ))}
              </div>
              <label>請求日</label>
              <input type="date" value={invForm.issued_at} onChange={(e) => setInvForm({ ...invForm, issued_at: e.target.value })} style={{ width: "100%" }} />
              <label>金額（円・整数）</label>
              <input
                type="number"
                value={invForm.amount_cents}
                onChange={(e) => setInvForm({ ...invForm, amount_cents: Number(e.target.value) })}
                style={{ width: "100%" }}
              />
              <label>備考</label>
              <input value={invForm.notes} onChange={(e) => setInvForm({ ...invForm, notes: e.target.value })} style={{ width: "100%" }} />
              <button type="button" style={{ marginTop: "0.35rem" }} onClick={() => void createInvoice()}>
                請求書作成
              </button>

              <div className="row" style={{ marginTop: "0.85rem", justifyContent: "flex-end", gap: "0.35rem" }}>
                <button className="danger" type="button" onClick={() => void endContract(modal.id)}>
                  契約終了
                </button>
                <button className="secondary" type="button" onClick={() => setModal(null)}>
                  閉じる
                </button>
                <button type="button" onClick={() => void saveCustomer(modal.id)}>
                  更新
                </button>
              </div>
            </div>
          </div>
        )}
      </main>
    </>
  );
}
