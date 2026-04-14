"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { AdminNav } from "@/components/AdminNav";
import { apiOrigin, authHeaders } from "@/lib/api";

type Prospect = { id: number; company_name: string; contact_info?: string | null; notes?: string | null; created_at?: string };

export default function ProspectsPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [hasToken, setHasToken] = useState(false);
  const [q, setQ] = useState("");
  const [detail, setDetail] = useState<Prospect | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [editForm, setEditForm] = useState({ company_name: "", contact_info: "", notes: "" });
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!localStorage.getItem("cust_token")) router.replace("/");
    else setHasToken(true);
  }, [router]);

  const { data: list = [], error, isPending, refetch } = useQuery<Prospect[], Error>({
    queryKey: ["prospects", q],
    enabled: hasToken,
    queryFn: async () => {
      const t = localStorage.getItem("cust_token")!;
      const u = new URL(`${apiOrigin}/admin/customers/prospects`);
      if (q) u.searchParams.set("q", q);
      const res = await fetch(u.toString(), { headers: authHeaders(t) });
      if (res.status === 401) {
        localStorage.removeItem("cust_token");
        router.replace("/");
        throw new Error("セッション切れ");
      }
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      return (data.prospects || []) as Prospect[];
    },
  });

  const createMut = useMutation({
    mutationFn: async (body: typeof editForm) => {
      const t = localStorage.getItem("cust_token")!;
      const res = await fetch(`${apiOrigin}/admin/customers/prospects`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders(t) },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(await res.text());
    },
    onSuccess: async () => {
      setCreateOpen(false);
      setEditForm({ company_name: "", contact_info: "", notes: "" });
      setErr(null);
      await queryClient.invalidateQueries({ queryKey: ["prospects"] });
    },
    onError: (e: Error) => setErr(e.message),
  });

  const updateMut = useMutation({
    mutationFn: async ({ id, body }: { id: number; body: typeof editForm }) => {
      const t = localStorage.getItem("cust_token")!;
      const res = await fetch(`${apiOrigin}/admin/customers/prospects/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", ...authHeaders(t) },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(await res.text());
    },
    onSuccess: async () => {
      setDetail(null);
      setErr(null);
      await queryClient.invalidateQueries({ queryKey: ["prospects"] });
    },
    onError: (e: Error) => setErr(e.message),
  });

  const deleteMut = useMutation({
    mutationFn: async (id: number) => {
      const t = localStorage.getItem("cust_token")!;
      const res = await fetch(`${apiOrigin}/admin/customers/prospects/${id}`, {
        method: "DELETE",
        headers: authHeaders(t),
      });
      if (!res.ok) throw new Error(await res.text());
    },
    onSuccess: async () => {
      setDetail(null);
      setErr(null);
      await queryClient.invalidateQueries({ queryKey: ["prospects"] });
    },
    onError: (e: Error) => setErr(e.message),
  });

  async function open(id: number) {
    setErr(null);
    const t = localStorage.getItem("cust_token")!;
    const res = await fetch(`${apiOrigin}/admin/customers/prospects/${id}`, { headers: authHeaders(t) });
    if (!res.ok) return;
    const p = (await res.json()) as Prospect;
    setDetail(p);
    setEditForm({
      company_name: p.company_name,
      contact_info: p.contact_info || "",
      notes: p.notes || "",
    });
  }

  function logout() {
    localStorage.removeItem("cust_token");
    router.replace("/");
  }

  const qErr = error instanceof Error ? error.message : error ? String(error) : null;

  return (
    <>
      <AdminNav onLogout={logout} />
      <main>
        <div className="row" style={{ justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
          <h1 style={{ margin: 0 }}>見込み顧客</h1>
          <button
            type="button"
            onClick={() => {
              setEditForm({ company_name: "", contact_info: "", notes: "" });
              setErr(null);
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
        {(err || qErr) && <p className="err">{err || qErr}</p>}
        {list.map((p) => (
          <button key={p.id} type="button" className="list-btn" onClick={() => void open(p.id)}>
            <strong>{p.company_name}</strong>
            <div style={{ fontSize: "0.85rem", color: "#737373" }}>{p.contact_info || "—"}</div>
          </button>
        ))}

        {createOpen && (
          <div className="modal" onClick={() => setCreateOpen(false)} role="presentation">
            <div className="modal-inner" onClick={(e) => e.stopPropagation()}>
              <h2>見込み顧客を追加</h2>
              <label>会社名</label>
              <input style={{ width: "100%" }} value={editForm.company_name} onChange={(e) => setEditForm({ ...editForm, company_name: e.target.value })} />
              <label>連絡先</label>
              <input style={{ width: "100%" }} value={editForm.contact_info} onChange={(e) => setEditForm({ ...editForm, contact_info: e.target.value })} />
              <label>メモ</label>
              <textarea rows={4} style={{ width: "100%" }} value={editForm.notes} onChange={(e) => setEditForm({ ...editForm, notes: e.target.value })} />
              {err && <p className="err">{err}</p>}
              <div className="row" style={{ justifyContent: "flex-end", marginTop: "0.75rem", gap: "0.35rem" }}>
                <button className="secondary" type="button" onClick={() => setCreateOpen(false)}>
                  キャンセル
                </button>
                <button type="button" onClick={() => createMut.mutate(editForm)} disabled={createMut.isPending}>
                  作成
                </button>
              </div>
            </div>
          </div>
        )}

        {detail && (
          <div className="modal" onClick={() => setDetail(null)} role="presentation">
            <div className="modal-inner" onClick={(e) => e.stopPropagation()}>
              <h2>編集 #{detail.id}</h2>
              <label>会社名</label>
              <input style={{ width: "100%" }} value={editForm.company_name} onChange={(e) => setEditForm({ ...editForm, company_name: e.target.value })} />
              <label>連絡先</label>
              <input style={{ width: "100%" }} value={editForm.contact_info} onChange={(e) => setEditForm({ ...editForm, contact_info: e.target.value })} />
              <label>メモ</label>
              <textarea rows={4} style={{ width: "100%" }} value={editForm.notes} onChange={(e) => setEditForm({ ...editForm, notes: e.target.value })} />
              {err && <p className="err">{err}</p>}
              <div className="row" style={{ justifyContent: "space-between", marginTop: "0.75rem", flexWrap: "wrap", gap: "0.35rem" }}>
                <button
                  className="danger"
                  type="button"
                  onClick={() => {
                    if (confirm("削除しますか？")) deleteMut.mutate(detail.id);
                  }}
                  disabled={deleteMut.isPending}
                >
                  削除
                </button>
                <div className="row" style={{ gap: "0.35rem" }}>
                  <button className="secondary" type="button" onClick={() => setDetail(null)}>
                    閉じる
                  </button>
                  <button type="button" onClick={() => updateMut.mutate({ id: detail.id, body: editForm })} disabled={updateMut.isPending}>
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
