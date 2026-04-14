"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { AdminNav } from "@/components/AdminNav";
import { apiOrigin, authHeaders } from "@/lib/api";

type Ann = {
  id: number;
  title: string;
  body: string;
  channel: string;
  active: boolean;
  valid_from?: string | null;
  valid_to?: string | null;
  sort_order: number;
  created_at: string;
  updated_at: string;
};

const channels = [
  { v: "public", l: "求人サイト（公開）" },
  { v: "job_admin", l: "案件管理サイト" },
  { v: "customer_admin", l: "顧客管理サイト" },
  { v: "all", l: "すべてのサイト" },
];

function toISO(s: string) {
  if (!s.trim()) return "";
  const d = new Date(s);
  return Number.isNaN(d.getTime()) ? "" : d.toISOString();
}

function isoToLocalInput(iso: string | null | undefined) {
  if (!iso) return "";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}

function channelLabel(channel: string) {
  return channels.find((c) => c.v === channel)?.l ?? channel;
}

function formatAnnouncePeriod(from?: string | null, to?: string | null) {
  if (!from && !to) return "制限なし";
  const fmt = (iso: string) =>
    new Date(iso).toLocaleString("ja-JP", { dateStyle: "short", timeStyle: "short" });
  if (from && to) return `${fmt(from)} 〜 ${fmt(to)}`;
  if (from) return `${fmt(from)} 〜`;
  return `〜 ${fmt(to!)}`;
}

function bodyPreview(body: string, max = 140) {
  const oneLine = body.replace(/\s+/g, " ").trim();
  if (oneLine.length <= max) return oneLine;
  return `${oneLine.slice(0, max)}…`;
}

export default function AnnouncementsManagePage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [hasToken, setHasToken] = useState(false);
  const [modal, setModal] = useState<Ann | "new" | null>(null);
  const [form, setForm] = useState({
    title: "",
    body: "",
    channel: "customer_admin",
    active: true,
    valid_from: "",
    valid_to: "",
    sort_order: 0,
  });
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!localStorage.getItem("cust_token")) router.replace("/");
    else setHasToken(true);
  }, [router]);

  const { data: list = [], isPending } = useQuery<Ann[]>({
    queryKey: ["announcements-manage"],
    enabled: hasToken,
    queryFn: async () => {
      const t = localStorage.getItem("cust_token")!;
      const res = await fetch(`${apiOrigin}/admin/customers/announcements`, { headers: authHeaders(t) });
      if (res.status === 401) {
        router.replace("/");
        return [];
      }
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      return (data.announcements || []) as Ann[];
    },
  });

  const saveMut = useMutation({
    mutationFn: async () => {
      const t = localStorage.getItem("cust_token")!;
      const payload = {
        title: form.title,
        body: form.body,
        channel: form.channel,
        active: form.active,
        valid_from: toISO(form.valid_from),
        valid_to: toISO(form.valid_to),
        sort_order: form.sort_order,
      };
      const isNew = modal === "new";
      const url = isNew
        ? `${apiOrigin}/admin/customers/announcements`
        : `${apiOrigin}/admin/customers/announcements/${(modal as Ann).id}`;
      const res = await fetch(url, {
        method: isNew ? "POST" : "PATCH",
        headers: { "Content-Type": "application/json", ...authHeaders(t) },
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(await res.text());
    },
    onSuccess: async () => {
      setModal(null);
      setErr(null);
      await qc.invalidateQueries({ queryKey: ["announcements-manage"] });
      await qc.invalidateQueries({ queryKey: ["ann-feed"] });
    },
    onError: (e: Error) => setErr(e.message),
  });

  const delMut = useMutation({
    mutationFn: async (id: number) => {
      const t = localStorage.getItem("cust_token")!;
      const res = await fetch(`${apiOrigin}/admin/customers/announcements/${id}`, { method: "DELETE", headers: authHeaders(t) });
      if (!res.ok) throw new Error(await res.text());
    },
    onSuccess: async () => {
      setModal(null);
      await qc.invalidateQueries({ queryKey: ["announcements-manage"] });
      await qc.invalidateQueries({ queryKey: ["ann-feed"] });
    },
  });

  function openNew() {
    setForm({
      title: "",
      body: "",
      channel: "customer_admin",
      active: true,
      valid_from: "",
      valid_to: "",
      sort_order: 0,
    });
    setErr(null);
    setModal("new");
  }

  function openEdit(a: Ann) {
    setForm({
      title: a.title,
      body: a.body,
      channel: a.channel,
      active: a.active,
      valid_from: isoToLocalInput(a.valid_from),
      valid_to: isoToLocalInput(a.valid_to),
      sort_order: a.sort_order,
    });
    setErr(null);
    setModal(a);
  }

  function logout() {
    localStorage.removeItem("cust_token");
    router.replace("/");
  }

  return (
    <>
      <AdminNav onLogout={logout} />
      <main>
        <div className="row" style={{ justifyContent: "space-between", alignItems: "center" }}>
          <h1 style={{ marginTop: 0 }}>お知らせ管理</h1>
          <button type="button" onClick={openNew}>
            新規登録
          </button>
        </div>
        <p style={{ color: "#737373", fontSize: "0.9rem" }}>
          表示先チャネル・掲載期間（任意・ローカル日時）を設定できます。日時は ISO 形式で API に送ります。
        </p>
        {isPending && <p>読み込み中…</p>}
        {list.map((a) => (
          <button
            key={a.id}
            type="button"
            className="list-btn"
            onClick={() => openEdit(a)}
            style={{ display: "flex", flexDirection: "column", alignItems: "flex-start", gap: "0.35rem" }}
          >
            <div style={{ fontSize: "0.85rem", color: "#525252" }}>
              <strong style={{ color: "#171717" }}>表示先:</strong> {channelLabel(a.channel)}
              {!a.active ? "（無効）" : ""}
              <span style={{ margin: "0 0.35rem" }}>|</span>
              <strong style={{ color: "#171717" }}>表示期間:</strong> {formatAnnouncePeriod(a.valid_from, a.valid_to)}
            </div>
            <strong style={{ fontSize: "1rem" }}>{a.title}</strong>
            <span style={{ fontSize: "0.9rem", color: "#404040", lineHeight: 1.45 }}>
              <strong style={{ color: "#171717" }}>表示内容:</strong> {bodyPreview(a.body)}
            </span>
          </button>
        ))}

        {modal && (
          <div className="modal" onClick={() => setModal(null)} role="presentation">
            <div className="modal-inner" onClick={(e) => e.stopPropagation()}>
              <h2>{modal === "new" ? "お知らせを登録" : "お知らせを編集"}</h2>
              <label>タイトル</label>
              <input style={{ width: "100%" }} value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
              <label>本文</label>
              <textarea rows={5} style={{ width: "100%" }} value={form.body} onChange={(e) => setForm({ ...form, body: e.target.value })} />
              <label>表示チャネル</label>
              <select style={{ width: "100%" }} value={form.channel} onChange={(e) => setForm({ ...form, channel: e.target.value })}>
                {channels.map((c) => (
                  <option key={c.v} value={c.v}>
                    {c.l}
                  </option>
                ))}
              </select>
              <label>
                <input type="checkbox" checked={form.active} onChange={(e) => setForm({ ...form, active: e.target.checked })} /> 有効
              </label>
              <label>掲載開始（任意・ローカル）</label>
              <input type="datetime-local" style={{ width: "100%" }} value={form.valid_from} onChange={(e) => setForm({ ...form, valid_from: e.target.value })} />
              <label>掲載終了（任意）</label>
              <input type="datetime-local" style={{ width: "100%" }} value={form.valid_to} onChange={(e) => setForm({ ...form, valid_to: e.target.value })} />
              <label>並び順（小さいほど上）</label>
              <input type="number" style={{ width: "100%" }} value={form.sort_order} onChange={(e) => setForm({ ...form, sort_order: Number(e.target.value) })} />
              {err && <p className="err">{err}</p>}
              <div className="row" style={{ justifyContent: "space-between", marginTop: "0.75rem" }}>
                {modal !== "new" && (
                  <button
                    className="danger"
                    type="button"
                    onClick={() => {
                      if (confirm("削除しますか？")) delMut.mutate((modal as Ann).id);
                    }}
                    disabled={delMut.isPending}
                  >
                    削除
                  </button>
                )}
                <div className="row" style={{ marginLeft: "auto", gap: "0.35rem" }}>
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
