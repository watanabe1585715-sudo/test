/**
 * 求職者向け求人サイト（一覧・詳細・応募）＋求職者ログイン・登録・マイページ（プロフィール）。
 * 追加機能: お気に入り、閲覧履歴、企業詳細（口コミ/動画/地図）、給与相場シミュレータ。
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState, type FormEvent } from "react";

const apiOrigin = import.meta.env.VITE_API_ORIGIN || "http://localhost:8080";
const TOKEN_KEY = "job_seeker_token";

type Announcement = { id: number; title: string; body: string };
type Job = { id: number; summary: string; requirements: string; publish_start: string; publish_end: string };
type Favorite = { job_posting_id: number; created_at: string };
type History = { id: number; job_posting_id: number; viewed_at: string };
type CompanyProfile = {
  customer_id: number;
  company_name: string;
  description: string;
  address: string;
  google_map_url?: string | null;
  website_url?: string | null;
  youtube_embed_url?: string | null;
  accept_foreigners: boolean;
  languages: string;
};
type CompanyReview = { id: number; reviewer: string; rating: number; comment: string; created_at: string };
type CompanyVideo = { id: number; title: string; youtube_url: string; created_at: string };
const JOB_CATEGORY_OPTIONS = [
  { value: "backend_engineer", label: "バックエンドエンジニア" },
  { value: "frontend_engineer", label: "フロントエンドエンジニア" },
  { value: "sales", label: "営業" },
];
const REGION_OPTIONS = [
  { value: "KANTO", label: "関東" },
  { value: "KANSAI", label: "関西" },
  { value: "CHUBU", label: "中部" },
  { value: "KYUSHU", label: "九州" },
  { value: "TOHOKU", label: "東北" },
];

type Page = "jobs" | "login" | "register" | "profile";
type JobSeekerProfile = {
  account_id: number;
  email: string;
  display_name: string;
  phone: string;
  career_summary: string;
  notes: string;
  updated_at: string;
};

function authHeader(): HeadersInit {
  const t = localStorage.getItem(TOKEN_KEY);
  return t ? { Authorization: `Bearer ${t}` } : {};
}

export default function App() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState<Page>("jobs");
  const [loggedIn, setLoggedIn] = useState(false);

  useEffect(() => {
    setLoggedIn(!!localStorage.getItem(TOKEN_KEY));
  }, [page]);

  const [q, setQ] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [detail, setDetail] = useState<Job | null>(null);
  const [applyJob, setApplyJob] = useState<Job | null>(null);
  const [selectedCompanyID, setSelectedCompanyID] = useState<number | null>(null);
  const [form, setForm] = useState({ applicant_name: "", career_summary: "", contact: "" });

  const [sim, setSim] = useState({ job_category: "backend_engineer", region: "KANTO", years_exp: 2 });
  const [simResult, setSimResult] = useState<{ low: number; median: number; high: number } | null>(null);

  const { data: announcements = [] } = useQuery({
    queryKey: ["public-announcements"],
    queryFn: async () => {
      const u = new URL(`${apiOrigin}/public/announcements`);
      u.searchParams.set("channel", "public");
      const res = await fetch(u.toString());
      if (!res.ok) return [];
      const data = await res.json();
      return (data.announcements || []) as Announcement[];
    },
    staleTime: 60_000,
  });

  const { data: jobs = [], error: jobsErr, isPending, refetch } = useQuery({
    queryKey: ["public-jobs", q],
    queryFn: async () => {
      const u = new URL(`${apiOrigin}/public/jobs`);
      if (q) u.searchParams.set("q", q);
      const res = await fetch(u.toString());
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      return (data.jobs || []) as Job[];
    },
  });

  const { data: favorites = [], refetch: refetchFavs } = useQuery({
    queryKey: ["job-seeker-favorites", loggedIn],
    enabled: loggedIn,
    queryFn: async () => {
      const res = await fetch(`${apiOrigin}/job-seeker/favorites`, { headers: { ...authHeader() } });
      if (!res.ok) return [] as Favorite[];
      const data = await res.json();
      return (data.favorites || []) as Favorite[];
    },
  });

  const { data: history = [], refetch: refetchHistory } = useQuery({
    queryKey: ["job-seeker-history", loggedIn],
    enabled: loggedIn,
    queryFn: async () => {
      const res = await fetch(`${apiOrigin}/job-seeker/history?limit=30`, { headers: { ...authHeader() } });
      if (!res.ok) return [] as History[];
      const data = await res.json();
      return (data.history || []) as History[];
    },
  });

  const { data: companyProfile } = useQuery({
    queryKey: ["company-profile", selectedCompanyID],
    enabled: selectedCompanyID != null,
    queryFn: async () => {
      const res = await fetch(`${apiOrigin}/public/companies/${selectedCompanyID}`);
      if (!res.ok) return null;
      return (await res.json()) as CompanyProfile;
    },
  });
  const { data: companyReviews = [] } = useQuery({
    queryKey: ["company-reviews", selectedCompanyID],
    enabled: selectedCompanyID != null,
    queryFn: async () => {
      const res = await fetch(`${apiOrigin}/public/companies/${selectedCompanyID}/reviews`);
      if (!res.ok) return [] as CompanyReview[];
      const data = await res.json();
      return (data.reviews || []) as CompanyReview[];
    },
  });
  const { data: companyVideos = [] } = useQuery({
    queryKey: ["company-videos", selectedCompanyID],
    enabled: selectedCompanyID != null,
    queryFn: async () => {
      const res = await fetch(`${apiOrigin}/public/companies/${selectedCompanyID}/videos`);
      if (!res.ok) return [] as CompanyVideo[];
      const data = await res.json();
      return (data.videos || []) as CompanyVideo[];
    },
  });

  const applyMutation = useMutation({
    mutationFn: async (payload: { job: Job; body: typeof form }) => {
      const res = await fetch(`${apiOrigin}/public/applications`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ job_posting_id: payload.job.id, ...payload.body }),
      });
      if (!res.ok) throw new Error(await res.text());
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["public-jobs"] });
    },
  });

  async function openDetail(id: number) {
    setErr(null);
    const res = await fetch(`${apiOrigin}/public/jobs/${id}`);
    if (!res.ok) {
      setErr("案件を取得できませんでした");
      return;
    }
    const d = (await res.json()) as Job;
    setDetail(d);
    if (loggedIn) {
      await fetch(`${apiOrigin}/job-seeker/history`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ job_posting_id: id }),
      });
      void refetchHistory();
    }
  }

  async function toggleFavorite(jobID: number) {
    if (!loggedIn) {
      alert("ログイン後に利用できます");
      return;
    }
    const exists = favorites.some((f) => f.job_posting_id === jobID);
    if (exists) {
      await fetch(`${apiOrigin}/job-seeker/favorites/${jobID}`, { method: "DELETE", headers: { ...authHeader() } });
    } else {
      await fetch(`${apiOrigin}/job-seeker/favorites`, {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeader() },
        body: JSON.stringify({ job_posting_id: jobID }),
      });
    }
    void refetchFavs();
  }

  async function runSalarySimulation() {
    setErr(null);
    const res = await fetch(`${apiOrigin}/public/salary-simulations`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(sim),
    });
    if (!res.ok) {
      setErr("給与相場APIに接続できません。APIサーバーを再起動して再実行してください。");
      return;
    }
    setSimResult(await res.json());
  }

  async function submitApplication() {
    if (!applyJob) return;
    setErr(null);
    try {
      await applyMutation.mutateAsync({ job: applyJob, body: form });
      setApplyJob(null);
      setDetail(null);
      setForm({ applicant_name: "", career_summary: "", contact: "" });
      alert("応募を受け付けました");
    } catch (e) {
      setErr(e instanceof Error ? e.message : "応募に失敗しました");
    }
  }

  function logoutSeeker() {
    localStorage.removeItem(TOKEN_KEY);
    setLoggedIn(false);
    setPage("jobs");
  }

  return (
    <>
      <header>
        <button type="button" className="logo" style={{ background: "none", border: "none", cursor: "pointer" }} onClick={() => setPage("jobs")}>
          求人広告サイト
        </button>
        <nav className="header-nav">
          <button type="button" onClick={() => setPage("jobs")}>求人一覧</button>
          {!loggedIn ? (
            <>
              <button type="button" onClick={() => setPage("login")}>ログイン</button>
              <button type="button" className="primary-nav" onClick={() => setPage("register")}>会員登録</button>
            </>
          ) : (
            <>
              <button type="button" className="primary-nav" onClick={() => setPage("profile")}>マイページ</button>
              <button type="button" onClick={() => logoutSeeker()}>ログアウト</button>
            </>
          )}
        </nav>
      </header>

      {page === "jobs" && (
        <main>
          {announcements.length > 0 && (
            <div className="announcement-stack">
              {announcements.map((a) => (
                <div key={a.id} className="announcement-card">
                  <strong>{a.title}</strong>
                  <div className="announcement-body">{a.body}</div>
                </div>
              ))}
            </div>
          )}

          <div className="split-grid">
            <div>
              <div className="search">
                <input placeholder="キーワードで検索" value={q} onChange={(e) => setQ(e.target.value)} onKeyDown={(e) => e.key === "Enter" && void refetch()} />
                <button type="button" onClick={() => void refetch()}>検索</button>
              </div>
              {isPending && <p>読み込み中…</p>}
              {(() => {
                const qe = jobsErr instanceof Error ? jobsErr.message : jobsErr ? String(jobsErr) : null;
                const raw = err || qe;
                const msg = raw && raw.includes("404") ? "APIに接続できません。バックエンドを再起動して再読み込みしてください。" : raw;
                return msg ? <p className="error">{msg}</p> : null;
              })()}
              <div className="job-list">
                {jobs.map((j) => {
                  const fav = favorites.some((f) => f.job_posting_id === j.id);
                  return (
                    <div key={j.id} className="job-card">
                      <strong>{j.summary}</strong>
                      <div style={{ fontSize: "0.85rem", color: "#6b7280", marginTop: "0.35rem" }}>掲載: {j.publish_start} 〜 {j.publish_end}</div>
                      <div className="inline-buttons" style={{ marginTop: "0.5rem" }}>
                        <button type="button" onClick={() => void openDetail(j.id)}>詳細</button>
                        {loggedIn && <button type="button" onClick={() => void toggleFavorite(j.id)}>{fav ? "お気に入り解除" : "お気に入り"}</button>}
                        <button type="button" onClick={() => setSelectedCompanyID(1)}>企業詳細</button>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            <div className="card">
              <h3 style={{ marginTop: 0 }}>職種別給与相場シミュレータ</h3>
              <label>職種</label>
              <select className="field" value={sim.job_category} onChange={(e) => setSim({ ...sim, job_category: e.target.value })}>
                {JOB_CATEGORY_OPTIONS.map((x) => <option key={x.value} value={x.value}>{x.label}</option>)}
              </select>
              <label>地域</label>
              <select className="field" value={sim.region} onChange={(e) => setSim({ ...sim, region: e.target.value })}>
                {REGION_OPTIONS.map((x) => <option key={x.value} value={x.value}>{x.label}</option>)}
              </select>
              <label>経験年数</label>
              <input className="field" type="number" min={0} value={sim.years_exp} onChange={(e) => setSim({ ...sim, years_exp: Number(e.target.value) })} />
              <div className="modal-actions"><button type="button" onClick={() => void runSalarySimulation()}>試算</button></div>
              {simResult && (
                <div className="mini-list">
                  <div className="mini-row">下限: {simResult.low.toLocaleString()} 円/月</div>
                  <div className="mini-row">中央値: {simResult.median.toLocaleString()} 円/月</div>
                  <div className="mini-row">上限: {simResult.high.toLocaleString()} 円/月</div>
                </div>
              )}

              {loggedIn && (
                <>
                  <h4 style={{ marginBottom: "0.4rem" }}>お気に入り</h4>
                  <div className="mini-list">
                    {favorites.slice(0, 6).map((f) => <div key={f.job_posting_id} className="mini-row">求人ID #{f.job_posting_id}</div>)}
                    {favorites.length === 0 && <div style={{ color: "#64748b" }}>まだありません</div>}
                  </div>

                  <h4 style={{ marginBottom: "0.4rem", marginTop: "1rem" }}>閲覧履歴</h4>
                  <div className="mini-list">
                    {history.slice(0, 6).map((h) => <div key={h.id} className="mini-row">求人ID #{h.job_posting_id} / {new Date(h.viewed_at).toLocaleString("ja-JP")}</div>)}
                    {history.length === 0 && <div style={{ color: "#64748b" }}>まだありません</div>}
                  </div>
                </>
              )}
            </div>
          </div>
        </main>
      )}

      {page === "login" && <JobSeekerLogin onSuccess={() => { setLoggedIn(true); setPage("profile"); }} onBack={() => setPage("jobs")} />}
      {page === "register" && <JobSeekerRegister onDone={() => setPage("login")} onBack={() => setPage("jobs")} />}
      {page === "profile" && <JobSeekerProfile onUnauthorized={() => { setLoggedIn(false); setPage("login"); }} onBack={() => setPage("jobs")} />}

      {detail && (
        <div className="modal-backdrop" role="presentation" onClick={() => setDetail(null)}>
          <div className="modal" role="dialog" onClick={(e) => e.stopPropagation()}>
            <h2>{detail.summary}</h2>
            <p style={{ whiteSpace: "pre-wrap" }}>{detail.requirements}</p>
            <p style={{ fontSize: "0.9rem", color: "#6b7280" }}>掲載期間: {detail.publish_start} 〜 {detail.publish_end}</p>
            <div className="modal-actions">
              <button type="button" className="secondary" onClick={() => setDetail(null)}>閉じる</button>
              <button type="button" onClick={() => { setApplyJob(detail); setDetail(null); }}>応募する</button>
            </div>
          </div>
        </div>
      )}

      {applyJob && (
        <div className="modal-backdrop" role="presentation" onClick={() => setApplyJob(null)}>
          <div className="modal" role="dialog" onClick={(e) => e.stopPropagation()}>
            <h2>応募: {applyJob.summary}</h2>
            <label>お名前</label>
            <input className="field" value={form.applicant_name} onChange={(e) => setForm({ ...form, applicant_name: e.target.value })} />
            <label>職歴</label>
            <textarea rows={4} value={form.career_summary} onChange={(e) => setForm({ ...form, career_summary: e.target.value })} />
            <label>連絡先</label>
            <input className="field" value={form.contact} onChange={(e) => setForm({ ...form, contact: e.target.value })} />
            {err && <p className="error">{err}</p>}
            <div className="modal-actions">
              <button type="button" className="secondary" onClick={() => setApplyJob(null)}>戻る</button>
              <button type="button" onClick={() => void submitApplication()}>応募する</button>
            </div>
          </div>
        </div>
      )}

      {selectedCompanyID != null && (
        <div className="modal-backdrop" role="presentation" onClick={() => setSelectedCompanyID(null)}>
          <div className="modal" role="dialog" onClick={(e) => e.stopPropagation()}>
            <h2>企業詳細</h2>
            {companyProfile ? (
              <>
                <p><strong>{companyProfile.company_name}</strong></p>
                <p style={{ whiteSpace: "pre-wrap" }}>{companyProfile.description}</p>
                <p>住所: {companyProfile.address}</p>
                <p>言語: {companyProfile.languages || "未設定"} / 外国籍: {companyProfile.accept_foreigners ? "可" : "不可"}</p>
                {companyProfile.google_map_url && <p><a href={companyProfile.google_map_url} target="_blank" rel="noreferrer">Googleマップを開く</a></p>}
                {companyProfile.youtube_embed_url && (
                  <iframe title="company-video" src={companyProfile.youtube_embed_url} width="100%" height="260" style={{ border: 0, borderRadius: 8 }} allowFullScreen />
                )}
                {companyVideos.length > 0 && (
                  <section style={{ marginTop: "1rem", paddingTop: "0.75rem", borderTop: "1px solid #e5e7eb" }}>
                    <h3 style={{ margin: "0 0 0.5rem", fontSize: "1rem", fontWeight: 600 }}>掲載動画・リンク</h3>
                    <div className="mini-list">
                      {companyVideos.map((v) => <div key={v.id} className="mini-row"><a href={v.youtube_url} target="_blank" rel="noreferrer">{v.title}</a></div>)}
                    </div>
                  </section>
                )}
                {companyReviews.length > 0 && (
                  <section style={{ marginTop: "1rem", paddingTop: "0.75rem", borderTop: "1px solid #e5e7eb" }}>
                    <h3 style={{ margin: "0 0 0.5rem", fontSize: "1rem", fontWeight: 600 }}>口コミ</h3>
                    <p style={{ margin: "0 0 0.5rem", fontSize: "0.85rem", color: "#6b7280" }}>掲載者の体験に基づくコメントです。</p>
                    <div className="mini-list">
                      {companyReviews.map((r) => <div key={r.id} className="mini-row">★{r.rating} {r.reviewer}: {r.comment}</div>)}
                    </div>
                  </section>
                )}
              </>
            ) : <p>読み込み中…</p>}
            <div className="modal-actions"><button type="button" className="secondary" onClick={() => setSelectedCompanyID(null)}>閉じる</button></div>
          </div>
        </div>
      )}
    </>
  );
}

function JobSeekerLogin({ onSuccess, onBack }: { onSuccess: () => void; onBack: () => void }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [localErr, setLocalErr] = useState<string | null>(null);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setLocalErr(null);
    const res = await fetch(`${apiOrigin}/job-seeker/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (!res.ok) {
      setLocalErr("メールまたはパスワードが正しくありません");
      return;
    }
    const data = await res.json();
    localStorage.setItem(TOKEN_KEY, data.token);
    onSuccess();
  }

  return (
    <main>
      <div className="auth-card">
        <h1>求職者ログイン</h1>
        <p style={{ fontSize: "0.9rem", color: "#64748b" }}>ログイン後、マイページでプロフィールを編集できます。</p>
        <form onSubmit={(e) => void submit(e)}>
          <label>メールアドレス</label>
          <input className="field" type="email" autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          <label>パスワード</label>
          <input className="field" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} />
          {localErr && <p className="error">{localErr}</p>}
          <div className="modal-actions" style={{ marginTop: "1rem" }}>
            <button type="button" className="secondary" onClick={onBack}>戻る</button>
            <button type="submit">ログイン</button>
          </div>
        </form>
      </div>
    </main>
  );
}

function JobSeekerRegister({ onDone, onBack }: { onDone: () => void; onBack: () => void }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [localErr, setLocalErr] = useState<string | null>(null);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setLocalErr(null);
    const res = await fetch(`${apiOrigin}/job-seeker/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (res.status === 409) {
      setLocalErr("このメールアドレスは既に登録されています");
      return;
    }
    if (!res.ok) {
      setLocalErr(await res.text());
      return;
    }
    alert("登録しました。ログインしてください。");
    onDone();
  }

  return (
    <main>
      <div className="auth-card">
        <h1>求職者 会員登録</h1>
        <form onSubmit={(e) => void submit(e)}>
          <label>メールアドレス</label>
          <input className="field" type="email" autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          <label>パスワード</label>
          <input className="field" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} />
          {localErr && <p className="error">{localErr}</p>}
          <div className="modal-actions" style={{ marginTop: "1rem" }}>
            <button type="button" className="secondary" onClick={onBack}>戻る</button>
            <button type="submit">登録</button>
          </div>
        </form>
      </div>
    </main>
  );
}

function JobSeekerProfile({ onUnauthorized, onBack }: { onUnauthorized: () => void; onBack: () => void }) {
  const [profile, setProfile] = useState<JobSeekerProfile | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [localErr, setLocalErr] = useState<string | null>(null);
  const [edit, setEdit] = useState({ display_name: "", phone: "", career_summary: "", notes: "" });

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const res = await fetch(`${apiOrigin}/job-seeker/profile`, { headers: { ...authHeader() } });
      if (res.status === 401) {
        onUnauthorized();
        return;
      }
      if (!res.ok) {
        if (!cancelled) setLocalErr(await res.text());
        if (!cancelled) setLoading(false);
        return;
      }
      const p = (await res.json()) as JobSeekerProfile;
      if (!cancelled) {
        setProfile(p);
        setEdit({ display_name: p.display_name, phone: p.phone, career_summary: p.career_summary, notes: p.notes });
        setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [onUnauthorized]);

  async function save() {
    setSaving(true);
    setLocalErr(null);
    const res = await fetch(`${apiOrigin}/job-seeker/profile`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json", ...authHeader() },
      body: JSON.stringify(edit),
    });
    setSaving(false);
    if (res.status === 401) {
      onUnauthorized();
      return;
    }
    if (!res.ok) {
      setLocalErr(await res.text());
      return;
    }
    const r = await fetch(`${apiOrigin}/job-seeker/profile`, { headers: { ...authHeader() } });
    if (r.ok) setProfile(await r.json());
    alert("保存しました");
  }

  if (loading) return <main><p>読み込み中…</p></main>;

  return (
    <main>
      <div className="auth-card" style={{ maxWidth: 560 }}>
        <h1>マイページ（プロフィール）</h1>
        {profile && <p style={{ fontSize: "0.9rem", color: "#64748b" }}>登録メール: <strong>{profile.email}</strong></p>}
        <label>表示名</label>
        <input className="field" value={edit.display_name} onChange={(e) => setEdit({ ...edit, display_name: e.target.value })} />
        <label>電話番号</label>
        <input className="field" value={edit.phone} onChange={(e) => setEdit({ ...edit, phone: e.target.value })} />
        <label>職歴要約</label>
        <textarea rows={4} value={edit.career_summary} onChange={(e) => setEdit({ ...edit, career_summary: e.target.value })} />
        <label>特記・メモ</label>
        <textarea rows={3} value={edit.notes} onChange={(e) => setEdit({ ...edit, notes: e.target.value })} />
        {localErr && <p className="error">{localErr}</p>}
        <div className="modal-actions" style={{ marginTop: "1rem" }}>
          <button type="button" className="secondary" onClick={onBack}>求人一覧へ</button>
          <button type="button" onClick={() => void save()} disabled={saving}>{saving ? "保存中…" : "保存"}</button>
        </div>
      </div>
    </main>
  );
}
