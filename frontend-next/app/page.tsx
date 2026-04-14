"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";
import { apiOrigin } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState("admin@example.com");
  const [password, setPassword] = useState("password");
  const [err, setErr] = useState<string | null>(null);

  async function login(e: FormEvent) {
    e.preventDefault();
    setErr(null);
    const res = await fetch(`${apiOrigin}/admin/customers/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (res.status === 403) {
      try {
        const j = (await res.json()) as { error?: string; message?: string };
        if (j.error === "registration_pending") {
          setErr(j.message || "管理者による承認待ちです");
          return;
        }
        if (j.error === "registration_rejected") {
          setErr(j.message || "このアカウントは承認されませんでした");
          return;
        }
      } catch {
        /* fallthrough */
      }
      setErr("ログインできません（承認状態を確認してください）");
      return;
    }
    if (!res.ok) {
      setErr("ログインに失敗しました");
      return;
    }
    const data = await res.json();
    localStorage.setItem("cust_token", data.token);
    router.push("/customers");
  }

  return (
    <main style={{ maxWidth: 420, margin: "3rem auto" }}>
      <div className="card">
        <h1 style={{ marginTop: 0 }}>顧客管理ログイン</h1>
        <form onSubmit={login}>
          <label>メール</label>
          <input value={email} onChange={(e) => setEmail(e.target.value)} style={{ width: "100%", marginBottom: "0.5rem" }} />
          <label>パスワード</label>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            style={{ width: "100%", marginBottom: "0.5rem" }}
          />
          {err && <p className="err">{err}</p>}
          <button type="submit">ログイン</button>
        </form>
        <p style={{ fontSize: "0.85rem", color: "#737373", marginTop: "0.75rem" }}>初期: admin@example.com / password</p>
      </div>
    </main>
  );
}
