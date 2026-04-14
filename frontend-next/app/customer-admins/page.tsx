"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { AdminNav } from "@/components/AdminNav";
import { apiOrigin, authHeaders } from "@/lib/api";

type User = { id: number; email: string; active: boolean; registration_status: string; created_at: string };

export default function CustomerAdminsPage() {
  const router = useRouter();
  const qc = useQueryClient();
  const [hasToken, setHasToken] = useState(false);
  const [modal, setModal] = useState<"new" | User | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [active, setActive] = useState(true);
  const [registrationStatus, setRegistrationStatus] = useState("pending");
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (!localStorage.getItem("cust_token")) router.replace("/");
    else setHasToken(true);
  }, [router]);

  const { data: users = [], isPending } = useQuery<User[]>({
    queryKey: ["customer-admins"],
    enabled: hasToken,
    queryFn: async () => {
      const t = localStorage.getItem("cust_token")!;
      const res = await fetch(`${apiOrigin}/admin/customers/customer-admins`, { headers: authHeaders(t) });
      if (res.status === 401) {
        router.replace("/");
        return [];
      }
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      return (data.users || []) as User[];
    },
  });

  const saveMut = useMutation({
    mutationFn: async () => {
      if (modal === "new" && !password.trim()) {
        throw new Error("パスワードを入力してください");
      }
      const t = localStorage.getItem("cust_token")!;
      if (modal === "new") {
        const res = await fetch(`${apiOrigin}/admin/customers/customer-admins`, {
          method: "POST",
          headers: { "Content-Type": "application/json", ...authHeaders(t) },
          body: JSON.stringify({ email, password }),
        });
        if (!res.ok) throw new Error(await res.text());
        return;
      }
      const u = modal as User;
      const body: { email: string; active: boolean; password?: string; registration_status?: string } = {
        email,
        active,
        registration_status: registrationStatus,
      };
      if (password.trim()) body.password = password;
      const res = await fetch(`${apiOrigin}/admin/customers/customer-admins/${u.id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json", ...authHeaders(t) },
        body: JSON.stringify(body),
      });
      if (!res.ok) throw new Error(await res.text());
    },
    onSuccess: async () => {
      setModal(null);
      setPassword("");
      setErr(null);
      await qc.invalidateQueries({ queryKey: ["customer-admins"] });
    },
    onError: (e: Error) => setErr(e.message),
  });

  function logout() {
    localStorage.removeItem("cust_token");
    router.replace("/");
  }

  return (
    <>
      <AdminNav onLogout={logout} />
      <main>
        <div className="row" style={{ justifyContent: "space-between", alignItems: "center" }}>
          <h1 style={{ marginTop: 0 }}>顧客管理サイト・ログインアカウント</h1>
          <button
            type="button"
            onClick={() => {
              setEmail("");
              setPassword("");
              setActive(true);
              setRegistrationStatus("pending");
              setErr(null);
              setModal("new");
            }}
          >
            新規作成
          </button>
        </div>
        <p style={{ color: "#737373", fontSize: "0.9rem" }}>
          この一覧のユーザーが顧客管理サイト（本サイト）にログインできます。新規作成したアカウントは「承認待ち」で、既存管理者が承認するとログイン可能になります。
        </p>
        {isPending && <p>読み込み中…</p>}
        {users.map((u) => (
          <button key={u.id} type="button" className="list-btn"           onClick={() => {
            setModal(u);
            setEmail(u.email);
            setPassword("");
            setActive(u.active);
            setRegistrationStatus(u.registration_status || "pending");
            setErr(null);
          }}>
            <strong>{u.email}</strong>
            <span style={{ color: "#737373", marginLeft: "0.5rem", fontSize: "0.85rem" }}>
              {u.active ? "有効" : "無効"} ·{" "}
              {u.registration_status === "approved"
                ? "登録承認済"
                : u.registration_status === "pending"
                  ? "承認待ち"
                  : "却下"}
            </span>
          </button>
        ))}

        {modal && (
          <div className="modal" onClick={() => setModal(null)} role="presentation">
            <div className="modal-inner" onClick={(e) => e.stopPropagation()}>
              <h2>{modal === "new" ? "アカウント作成" : "アカウント編集"}</h2>
              <label>メール</label>
              <input style={{ width: "100%" }} value={email} onChange={(e) => setEmail(e.target.value)} />
              <label>{modal === "new" ? "パスワード" : "パスワード（変更する場合のみ入力）"}</label>
              <input type="password" style={{ width: "100%" }} value={password} onChange={(e) => setPassword(e.target.value)} />
              {modal !== "new" && (
                <>
                  <label>
                    <input type="checkbox" checked={active} onChange={(e) => setActive(e.target.checked)} /> 有効
                  </label>
                  <label>登録承認</label>
                  <select value={registrationStatus} onChange={(e) => setRegistrationStatus(e.target.value)} style={{ width: "100%" }}>
                    <option value="pending">承認待ち</option>
                    <option value="approved">承認済</option>
                    <option value="rejected">却下</option>
                  </select>
                </>
              )}
              {err && <p className="err">{err}</p>}
              <div className="row" style={{ justifyContent: "flex-end", marginTop: "0.75rem", gap: "0.35rem" }}>
                <button className="secondary" type="button" onClick={() => setModal(null)}>
                  キャンセル
                </button>
                <button type="button" onClick={() => saveMut.mutate()} disabled={saveMut.isPending}>
                  保存
                </button>
              </div>
            </div>
          </div>
        )}
      </main>
    </>
  );
}
