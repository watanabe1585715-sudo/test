<script setup lang="ts">
import { useQuery, useQueryClient } from "@tanstack/vue-query";
import { storeToRefs } from "pinia";
import { computed, onMounted, ref } from "vue";
import type { ApplicationRow, CompanyProfile, JobRow } from "./apiTypes";
import { useJobSessionStore } from "./stores/jobSession";

const apiOrigin = import.meta.env.VITE_API_ORIGIN || "http://localhost:8080";
const JOB_CATEGORY_OPTIONS = [
  { value: "backend_engineer", label: "バックエンドエンジニア" },
  { value: "frontend_engineer", label: "フロントエンドエンジニア" },
  { value: "sales", label: "営業" },
] as const;
const REGION_OPTIONS = [
  { value: "KANTO", label: "関東" },
  { value: "KANSAI", label: "関西" },
  { value: "CHUBU", label: "中部" },
  { value: "KYUSHU", label: "九州" },
  { value: "TOHOKU", label: "東北" },
] as const;
const session = useJobSessionStore();
const { token } = storeToRefs(session);
const queryClient = useQueryClient();

const err = ref<string | null>(null);
const tab = ref<"jobs" | "company" | "analytics" | "scout">("jobs");
const loginEmail = ref("jobadmin@example.com");
const loginPassword = ref("password");

const q = ref("");
const edit = ref<JobRow | null>(null);
const createOpen = ref(false);
const appsJob = ref<JobRow | null>(null);
const applications = ref<ApplicationRow[]>([]);
const form = ref({ summary: "", requirements: "", publish_start: "", publish_end: "" });

const authHeader = computed(() => (token.value ? { Authorization: `Bearer ${token.value}` } : {}));

const { data: jobsData, isPending, refetch } = useQuery({
  queryKey: computed(() => ["admin-jobs", q.value, token.value ?? ""] as const),
  enabled: computed(() => !!token.value),
  queryFn: async () => {
    const u = new URL(`${apiOrigin}/admin/jobs/jobs`);
    if (q.value) u.searchParams.set("q", q.value);
    const res = await fetch(u.toString(), { headers: { ...authHeader.value } as HeadersInit });
    if (res.status === 401) {
      session.clear();
      throw new Error("セッションが切れました");
    }
    if (!res.ok) throw new Error(await res.text());
    const data = await res.json();
    return (data.jobs || []) as JobRow[];
  },
});
const jobs = computed(() => jobsData.value ?? []);

type JobAnnouncement = { id: number; title: string; body: string };
const { data: jobAnnouncementsData } = useQuery({
  queryKey: computed(() => ["job-admin-announcements", token.value ?? ""] as const),
  enabled: computed(() => !!token.value),
  queryFn: async () => {
    const res = await fetch(`${apiOrigin}/admin/jobs/announcements`, { headers: { ...authHeader.value } as HeadersInit });
    if (res.status === 401) {
      session.clear();
      return [] as JobAnnouncement[];
    }
    if (!res.ok) return [] as JobAnnouncement[];
    const data = await res.json();
    return (data.announcements || []) as JobAnnouncement[];
  },
  staleTime: 60_000,
});
const jobAnnouncements = computed(() => jobAnnouncementsData.value ?? []);

const companyForm = ref<CompanyProfile>({
  customer_id: 0,
  company_name: "",
  description: "",
  address: "",
  google_map_url: "",
  website_url: "",
  youtube_embed_url: "",
  accept_foreigners: false,
  languages: "ja",
});

const ai = ref({ job_title: "backend_engineer", prompt: "採用要件を整理したい" });
const aiSuggestion = ref("");

const mediaNew = ref({ media_name: "Indeed", external_account: "", status: "active", settings_json: "{}" });
const mediaEdit = ref<{ id: number; media_name: string; external_account: string; status: string; settings_json: string } | null>(null);

const interviewForm = ref({ job_posting_id: "", provider: "google_meet", meeting_url: "", scheduled_at: "" });
const scoutForm = ref({ job_posting_id: "", candidate_name: "", contact: "", message: "", status: "draft" });

const salaryForm = ref({ job_category: "backend_engineer", region: "KANTO", years_exp: 2 });
const salaryResult = ref<{ low: number; median: number; high: number } | null>(null);

const followUp = ref({ enabled: true, max_follow_up_days: 30, notes: "" });

const dashboard = ref<Record<string, number>>({});
const reports = ref<Array<{ id: number; report_kind: string; created_at: string }>>([]);
const inflow = ref<Array<{ media_name: string; views: number; clicks: number; applications: number; hires: number }>>([]);
const mediaConnections = ref<Array<{ id: number; media_name: string; external_account?: string; status: string; settings_json: string }>>([]);
const interviews = ref<Array<{ id: number; provider: string; meeting_url: string; scheduled_at?: string }>>([]);
const scouts = ref<Array<{ id: number; candidate_name: string; contact: string; status: string; message: string }>>([]);

function setFriendlyError(raw: string) {
  if (raw.includes("404")) {
    err.value = "APIに接続できません。バックエンドを再起動して再読み込みしてください。";
    return;
  }
  err.value = raw;
}

async function login() {
  err.value = null;
  const res = await fetch(`${apiOrigin}/admin/jobs/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: loginEmail.value, password: loginPassword.value }),
  });
  if (!res.ok) {
    err.value = "ログインに失敗しました";
    return;
  }
  const data = await res.json();
  session.setSession(data.token as string, data.customer_id as number);
  await Promise.all([loadCompanyProfile(), loadAnalytics(), loadScoutAndInterview(), loadMediaConnections(), loadFollowUpPolicy()]);
}

function logout() {
  session.clear();
  void queryClient.removeQueries({ queryKey: ["admin-jobs"] });
}

function resetForm() {
  form.value = { summary: "", requirements: "", publish_start: "", publish_end: "" };
}

function openCreate() {
  resetForm();
  createOpen.value = true;
}

function openEdit(j: JobRow) {
  form.value = { summary: j.summary, requirements: j.requirements, publish_start: String(j.publish_start).slice(0, 10), publish_end: String(j.publish_end).slice(0, 10) };
  edit.value = j;
}

async function saveJob(id?: number) {
  err.value = null;
  const url = id ? `${apiOrigin}/admin/jobs/jobs/${id}` : `${apiOrigin}/admin/jobs/jobs`;
  const res = await fetch(url, {
    method: id ? "PATCH" : "POST",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify(form.value),
  });
  if (!res.ok) {
    setFriendlyError(await res.text());
    return;
  }
  createOpen.value = false;
  edit.value = null;
  resetForm();
  await queryClient.invalidateQueries({ queryKey: ["admin-jobs"] });
}

async function deleteJob(id: number) {
  if (!confirm("削除しますか？")) return;
  const res = await fetch(`${apiOrigin}/admin/jobs/jobs/${id}`, { method: "DELETE", headers: { ...(authHeader.value as HeadersInit) } });
  if (!res.ok) {
    setFriendlyError(await res.text());
    return;
  }
  edit.value = null;
  await queryClient.invalidateQueries({ queryKey: ["admin-jobs"] });
}

async function updateGlobalOptions(jobID: number, accept_foreigners: boolean, languages: string) {
  const res = await fetch(`${apiOrigin}/admin/jobs/jobs/${jobID}/global-options`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify({ accept_foreigners, languages }),
  });
  if (!res.ok) setFriendlyError(await res.text());
}

async function openApps(j: JobRow) {
  appsJob.value = j;
  const res = await fetch(`${apiOrigin}/admin/jobs/jobs/${j.id}/applications`, { headers: { ...(authHeader.value as HeadersInit) } });
  if (!res.ok) {
    setFriendlyError(await res.text());
    return;
  }
  const data = await res.json();
  applications.value = data.applications || [];
}

async function loadCompanyProfile() {
  const res = await fetch(`${apiOrigin}/admin/jobs/company-profile`, { headers: { ...(authHeader.value as HeadersInit) } });
  if (!res.ok) return;
  const d = await res.json();
  companyForm.value = {
    customer_id: d.customer_id || 0,
    company_name: d.company_name || "",
    description: d.description || "",
    address: d.address || "",
    google_map_url: d.google_map_url || "",
    website_url: d.website_url || "",
    youtube_embed_url: d.youtube_embed_url || "",
    accept_foreigners: !!d.accept_foreigners,
    languages: d.languages || "ja",
  };
}

async function saveCompanyProfile() {
  const res = await fetch(`${apiOrigin}/admin/jobs/company-profile`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify(companyForm.value),
  });
  if (!res.ok) setFriendlyError(await res.text());
}

async function runAIAssist() {
  const res = await fetch(`${apiOrigin}/admin/jobs/ai/job-assist`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify(ai.value),
  });
  if (!res.ok) {
    setFriendlyError(await res.text());
    return;
  }
  const d = await res.json();
  aiSuggestion.value = d.suggestion || "";
}

async function loadMediaConnections() {
  const res = await fetch(`${apiOrigin}/admin/jobs/media-connections`, { headers: { ...(authHeader.value as HeadersInit) } });
  if (!res.ok) return;
  const d = await res.json();
  mediaConnections.value = d.connections || [];
}

async function createMediaConnection() {
  const res = await fetch(`${apiOrigin}/admin/jobs/media-connections`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify(mediaNew.value),
  });
  if (!res.ok) {
    setFriendlyError(await res.text());
    return;
  }
  mediaNew.value = { media_name: "Indeed", external_account: "", status: "active", settings_json: "{}" };
  await loadMediaConnections();
}

async function updateMediaConnection() {
  if (!mediaEdit.value) return;
  const res = await fetch(`${apiOrigin}/admin/jobs/media-connections/${mediaEdit.value.id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify(mediaEdit.value),
  });
  if (!res.ok) {
    setFriendlyError(await res.text());
    return;
  }
  mediaEdit.value = null;
  await loadMediaConnections();
}

async function loadAnalytics() {
  const [dRes, rRes, iRes] = await Promise.all([
    fetch(`${apiOrigin}/admin/jobs/dashboard`, { headers: { ...(authHeader.value as HeadersInit) } }),
    fetch(`${apiOrigin}/admin/jobs/reports`, { headers: { ...(authHeader.value as HeadersInit) } }),
    fetch(`${apiOrigin}/admin/jobs/analytics/inflow`, { headers: { ...(authHeader.value as HeadersInit) } }),
  ]);
  if (dRes.ok) dashboard.value = (await dRes.json()).summary || {};
  if (rRes.ok) reports.value = (await rRes.json()).reports || [];
  if (iRes.ok) inflow.value = (await iRes.json()).inflow || [];
  if (reports.value.length === 0) {
    reports.value = [
      { id: 90001, report_kind: "monthly_inflow (demo)", created_at: new Date().toISOString() },
      { id: 90002, report_kind: "hiring_funnel (demo)", created_at: new Date().toISOString() },
    ];
  }
  if (inflow.value.length === 0) {
    inflow.value = [
      { media_name: "Indeed(デモ)", views: 1200, clicks: 190, applications: 32, hires: 2 },
      { media_name: "求人ボックス(デモ)", views: 640, clicks: 88, applications: 14, hires: 1 },
    ];
  }
}

async function loadScoutAndInterview() {
  const [sRes, ivRes] = await Promise.all([
    fetch(`${apiOrigin}/admin/jobs/scouts`, { headers: { ...(authHeader.value as HeadersInit) } }),
    fetch(`${apiOrigin}/admin/jobs/interviews`, { headers: { ...(authHeader.value as HeadersInit) } }),
  ]);
  if (sRes.ok) scouts.value = (await sRes.json()).scouts || [];
  if (ivRes.ok) interviews.value = (await ivRes.json()).interviews || [];
}

async function createInterview() {
  const payload = {
    provider: interviewForm.value.provider,
    meeting_url: interviewForm.value.meeting_url,
    job_posting_id: interviewForm.value.job_posting_id ? Number(interviewForm.value.job_posting_id) : undefined,
    scheduled_at: interviewForm.value.scheduled_at ? new Date(interviewForm.value.scheduled_at).toISOString() : undefined,
  };
  const res = await fetch(`${apiOrigin}/admin/jobs/interviews/links`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    setFriendlyError(await res.text());
    return;
  }
  interviewForm.value = { job_posting_id: "", provider: "google_meet", meeting_url: "", scheduled_at: "" };
  await loadScoutAndInterview();
}

async function createScout() {
  const payload = {
    ...scoutForm.value,
    job_posting_id: scoutForm.value.job_posting_id ? Number(scoutForm.value.job_posting_id) : undefined,
  };
  const res = await fetch(`${apiOrigin}/admin/jobs/scouts`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    setFriendlyError(await res.text());
    return;
  }
  scoutForm.value = { job_posting_id: "", candidate_name: "", contact: "", message: "", status: "draft" };
  await loadScoutAndInterview();
}

async function updateScoutStatus(id: number, status: string, message: string) {
  const res = await fetch(`${apiOrigin}/admin/jobs/scouts/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify({ status, message }),
  });
  if (!res.ok) {
    setFriendlyError(await res.text());
    return;
  }
  await loadScoutAndInterview();
}

async function addVideo() {
  const title = prompt("動画タイトル");
  const youtube_url = prompt("YouTube URL");
  if (!title || !youtube_url) return;
  const res = await fetch(`${apiOrigin}/admin/jobs/company-videos`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify({ title, youtube_url }),
  });
  if (!res.ok) setFriendlyError(await res.text());
}

async function runSalarySimulation() {
  const res = await fetch(`${apiOrigin}/admin/jobs/salary-simulations`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify(salaryForm.value),
  });
  if (!res.ok) {
    setFriendlyError(await res.text());
    return;
  }
  salaryResult.value = await res.json();
}

async function loadFollowUpPolicy() {
  const res = await fetch(`${apiOrigin}/admin/jobs/follow-up-policy`, { headers: { ...(authHeader.value as HeadersInit) } });
  if (!res.ok) return;
  const d = await res.json();
  followUp.value = {
    enabled: !!d.enabled,
    max_follow_up_days: d.max_follow_up_days || 0,
    notes: d.notes || "",
  };
}

async function saveFollowUpPolicy() {
  const res = await fetch(`${apiOrigin}/admin/jobs/follow-up-policy`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json", ...(authHeader.value as HeadersInit) },
    body: JSON.stringify({ ...followUp.value, available_by_contract: true }),
  });
  if (!res.ok) setFriendlyError(await res.text());
}

onMounted(() => {
  session.hydrate();
  if (session.token) {
    void Promise.all([loadCompanyProfile(), loadAnalytics(), loadScoutAndInterview(), loadMediaConnections(), loadFollowUpPolicy()]);
  }
});
</script>

<template>
  <div class="layout">
    <div v-if="!session.token" class="card">
      <h2>案件管理ログイン</h2>
      <label>メール</label>
      <input v-model="loginEmail" style="width: 100%; margin-bottom: 0.5rem" />
      <label>パスワード</label>
      <input v-model="loginPassword" type="password" style="width: 100%; margin-bottom: 0.5rem" />
      <p v-if="err" class="err">{{ err }}</p>
      <button type="button" @click="login()">ログイン</button>
      <p style="font-size: 0.85rem; color: #71717a">初期: jobadmin@example.com / password</p>
    </div>

    <div v-else>
      <div class="row" style="justify-content: space-between; margin-bottom: 1rem">
        <h2 style="margin: 0">案件管理サイト</h2>
        <button class="ghost" type="button" @click="logout()">ログアウト</button>
      </div>

      <div class="row" style="margin-bottom: 0.75rem">
        <button type="button" @click="tab='jobs'">求人管理</button>
        <button type="button" @click="tab='company'">企業情報/媒体連携</button>
        <button type="button" @click="tab='analytics'">分析/レポート</button>
        <button type="button" @click="tab='scout'">面談/スカウト/フォロー</button>
      </div>

      <p v-if="err" class="err">{{ err }}</p>

      <template v-if="tab==='jobs'">
        <div v-if="jobAnnouncements.length" style="margin-bottom: 0.75rem">
          <div
            v-for="a in jobAnnouncements"
            :key="a.id"
            class="card"
            style="margin-bottom: 0.5rem; border-left: 4px solid #2563eb; padding: 0.65rem 1rem; background: #eff6ff"
          >
            <strong style="display: block; margin-bottom: 0.25rem">{{ a.title }}</strong>
            <div style="font-size: 0.9rem; white-space: pre-wrap; color: #334155">{{ a.body }}</div>
          </div>
        </div>
        <div class="card row">
          <input v-model="q" placeholder="検索" style="flex: 1; min-width: 200px" @keyup.enter="refetch()" />
          <button type="button" @click="refetch()">検索</button>
          <button type="button" @click="openCreate()">新規作成</button>
        </div>
        <p v-if="isPending" style="color: #71717a">読み込み中…</p>
        <button v-for="j in jobs" :key="j.id" type="button" class="job-item" @click="openEdit(j)">
          <strong>{{ j.summary }}</strong>
          <span style="color: #71717a; font-size: 0.85rem; margin-left: 0.5rem">({{ j.publication_status }})</span>
          <div style="font-size: 0.85rem; color: #71717a">{{ j.publish_start }} 〜 {{ j.publish_end }}</div>
        </button>
      </template>

      <template v-else-if="tab==='company'">
        <div class="card">
          <h3>企業プロフィール</h3>
          <label>企業名</label><input v-model="companyForm.company_name" style="width:100%" />
          <label>説明</label><textarea v-model="companyForm.description" rows="3" style="width:100%" />
          <label>住所</label><input v-model="companyForm.address" style="width:100%" />
          <label>GoogleMap URL</label><input v-model="companyForm.google_map_url" style="width:100%" />
          <label>YouTube埋め込みURL</label><input v-model="companyForm.youtube_embed_url" style="width:100%" />
          <label>言語</label><input v-model="companyForm.languages" style="width:100%" />
          <label><input type="checkbox" v-model="companyForm.accept_foreigners" /> 外国籍可</label>
          <div class="row" style="margin-top:.5rem"><button type="button" @click="saveCompanyProfile()">保存</button></div>
        </div>

        <div class="card" style="margin-top:.75rem">
          <h3>職種 AI アシスト</h3>
          <label>職種</label>
          <select v-model="ai.job_title" style="width:100%">
            <option v-for="x in JOB_CATEGORY_OPTIONS" :key="x.value" :value="x.value">{{ x.label }}</option>
          </select>
          <label>プロンプト</label><textarea v-model="ai.prompt" rows="2" style="width:100%" />
          <div class="row" style="margin-top:.5rem"><button type="button" @click="runAIAssist()">生成</button></div>
          <pre style="white-space:pre-wrap">{{ aiSuggestion }}</pre>
        </div>

        <div class="card" style="margin-top:.75rem">
          <h3>求人媒体連携</h3>
          <div v-for="m in mediaConnections" :key="m.id" class="row" style="border-bottom:1px solid #e4e4e7; padding:.35rem 0">
            <span>{{ m.media_name }} ({{ m.status }})</span>
            <button type="button" class="ghost" @click="mediaEdit={...m, external_account:m.external_account||''}">編集</button>
          </div>
          <h4 style="margin-top:0.75rem">新規連携</h4>
          <label style="display:block;margin-top:0.75rem">媒体名（例: Indeed、求人ボックス）</label>
          <input v-model="mediaNew.media_name" style="width:100%;margin-bottom:0.75rem" />
          <label style="display:block;margin-top:0.25rem">外部アカウント識別子（媒体側の企業ID・連携キーなど）</label>
          <input v-model="mediaNew.external_account" style="width:100%;margin-bottom:0.75rem" />
          <label style="display:block;margin-top:0.25rem">連携設定（JSON。APIキーやフィードURLなど）</label>
          <textarea v-model="mediaNew.settings_json" rows="3" style="width:100%;margin-bottom:0.75rem" />
          <button type="button" @click="createMediaConnection()">追加</button>
          <div v-if="mediaEdit" style="margin-top:.75rem">
            <h4>編集</h4>
            <label style="display:block;margin-top:0.5rem">媒体名</label>
            <input v-model="mediaEdit.media_name" style="width:100%;margin-bottom:0.75rem" />
            <label style="display:block">外部アカウント識別子</label>
            <input v-model="mediaEdit.external_account" style="width:100%;margin-bottom:0.75rem" />
            <label style="display:block">ステータス（active / inactive など）</label>
            <input v-model="mediaEdit.status" style="width:100%;margin-bottom:0.75rem" />
            <label style="display:block">連携設定（JSON）</label>
            <textarea v-model="mediaEdit.settings_json" rows="3" style="width:100%;margin-bottom:0.75rem" />
            <button type="button" @click="updateMediaConnection()">更新</button>
          </div>
        </div>
      </template>

      <template v-else-if="tab==='analytics'">
        <div class="card">
          <h3>ダッシュボード</h3>
          <div>求人数: {{ dashboard.jobs || 0 }}</div>
          <div>応募数: {{ dashboard.applications || 0 }}</div>
          <div>スカウト数: {{ dashboard.scouts || 0 }}</div>
          <div>面談数: {{ dashboard.interviews || 0 }}</div>
          <button type="button" @click="loadAnalytics()">再読み込み</button>
        </div>
        <div class="card" style="margin-top:.75rem">
          <h3>媒体別流入</h3>
          <div v-for="i in inflow" :key="i.media_name" class="row" style="border-bottom:1px solid #eee;padding:.25rem 0">
            <span>{{ i.media_name }}</span><span>閲覧 {{ i.views }} / 応募 {{ i.applications }} / 採用 {{ i.hires }}</span>
          </div>
        </div>
        <div class="card" style="margin-top:.75rem">
          <h3>レポート</h3>
          <div v-for="r in reports" :key="r.id">{{ r.report_kind }} - {{ r.created_at }}</div>
        </div>
      </template>

      <template v-else>
        <div class="card">
          <h3>Web面談</h3>
          <label style="display:block;margin-top:0.5rem">求人案件ID（任意）</label>
          <input v-model="interviewForm.job_posting_id" placeholder="例: 12" style="width:100%;margin-bottom:0.75rem" />
          <label style="display:block;margin-top:0.25rem">面談プロバイダ</label>
          <input v-model="interviewForm.provider" placeholder="google_meet / zoom など" style="width:100%;margin-bottom:0.75rem" />
          <label style="display:block;margin-top:0.25rem">会議URL</label>
          <input v-model="interviewForm.meeting_url" placeholder="https://…" style="width:100%;margin-bottom:0.75rem" />
          <label style="display:block;margin-top:0.25rem">予定日時</label>
          <input v-model="interviewForm.scheduled_at" type="datetime-local" style="width:100%;margin-bottom:0.75rem" />
          <button type="button" style="margin-top:0.25rem" @click="createInterview()">面談URL作成</button>
          <div v-for="iv in interviews" :key="iv.id" class="row" style="border-bottom:1px solid #eee;padding:.2rem 0;margin-top:0.5rem">{{ iv.provider }} - {{ iv.meeting_url }}</div>
        </div>

        <div class="card" style="margin-top:.75rem">
          <h3>スカウト</h3>
          <label style="display:block;margin-top:0.5rem">求人案件ID（任意）</label>
          <input v-model="scoutForm.job_posting_id" placeholder="例: 12" style="width:100%;margin-bottom:0.75rem" />
          <label style="display:block;margin-top:0.25rem">候補者名</label>
          <input v-model="scoutForm.candidate_name" placeholder="山田 太郎" style="width:100%;margin-bottom:0.75rem" />
          <label style="display:block;margin-top:0.25rem">連絡先</label>
          <input v-model="scoutForm.contact" placeholder="メールまたは電話" style="width:100%;margin-bottom:0.75rem" />
          <label style="display:block;margin-top:0.25rem">スカウトメッセージ</label>
          <textarea v-model="scoutForm.message" rows="4" placeholder="送付する文面" style="width:100%;margin-bottom:0.75rem" />
          <button type="button" style="margin-top:0.25rem" @click="createScout()">スカウト作成</button>
          <div v-for="s in scouts" :key="s.id" style="border-bottom:1px solid #eee;padding:.3rem 0">
            <div>{{ s.candidate_name }} ({{ s.status }})</div>
            <div class="row"><button type="button" class="ghost" @click="updateScoutStatus(s.id,'sent',s.message)">送信済みにする</button></div>
          </div>
        </div>

        <div class="card" style="margin-top:.75rem">
          <h3>YouTube動画登録</h3>
          <button type="button" @click="addVideo()">動画を追加</button>
        </div>

        <div class="card" style="margin-top:.75rem">
          <h3>フォローアップ設定</h3>
          <label><input type="checkbox" v-model="followUp.enabled" /> 有効</label>
          <label>最大日数</label><input v-model.number="followUp.max_follow_up_days" type="number" style="width:100%" />
          <label>メモ（契約内容・運用方針など）</label>
          <textarea v-model="followUp.notes" rows="2" style="width:100%" />
          <button type="button" @click="saveFollowUpPolicy()">保存</button>
        </div>

        <div class="card" style="margin-top:.75rem">
          <h3>給与相場シミュレータ</h3>
          <label>職種</label>
          <select v-model="salaryForm.job_category" style="width:100%">
            <option v-for="x in JOB_CATEGORY_OPTIONS" :key="x.value" :value="x.value">{{ x.label }}</option>
          </select>
          <label>地域</label>
          <select v-model="salaryForm.region" style="width:100%">
            <option v-for="x in REGION_OPTIONS" :key="x.value" :value="x.value">{{ x.label }}</option>
          </select>
          <input v-model.number="salaryForm.years_exp" type="number" style="width:100%" />
          <button type="button" @click="runSalarySimulation()">試算</button>
          <div v-if="salaryResult">下限 {{ salaryResult.low }} / 中央 {{ salaryResult.median }} / 上限 {{ salaryResult.high }}</div>
        </div>
      </template>

      <div v-if="createOpen" class="modal" @click.self="createOpen = false">
        <div class="modal-inner">
          <h3>案件作成</h3>
          <label>概要</label><textarea v-model="form.summary" rows="3" style="width: 100%"></textarea>
          <label>募集要望</label><textarea v-model="form.requirements" rows="4" style="width: 100%"></textarea>
          <label>掲載開始</label><input v-model="form.publish_start" type="date" style="width: 100%" />
          <label>掲載終了</label><input v-model="form.publish_end" type="date" style="width: 100%" />
          <div class="row" style="margin-top: 0.75rem; justify-content: flex-end; gap: 0.35rem">
            <button class="ghost" type="button" @click="createOpen = false">キャンセル</button>
            <button type="button" @click="saveJob()">作成</button>
          </div>
        </div>
      </div>

      <div v-if="edit" class="modal" @click.self="edit = null">
        <div class="modal-inner">
          <h3>案件更新</h3>
          <label>概要</label><textarea v-model="form.summary" rows="3" style="width: 100%"></textarea>
          <label>募集要望</label><textarea v-model="form.requirements" rows="4" style="width: 100%"></textarea>
          <label>掲載開始</label><input v-model="form.publish_start" type="date" style="width: 100%" />
          <label>掲載終了</label><input v-model="form.publish_end" type="date" style="width: 100%" />
          <h4>外国籍可否・使用言語</h4>
          <button type="button" class="ghost" @click="updateGlobalOptions(edit.id,true,'ja,en')">外国籍可（ja,en）を設定</button>
          <div class="row" style="margin-top: 0.75rem; justify-content: flex-end; gap: 0.35rem">
            <button class="ghost" type="button" @click="edit = null">キャンセル</button>
            <button class="ghost" type="button" @click="openApps(edit)">応募者一覧</button>
            <button class="danger" type="button" @click="deleteJob(edit.id)">削除</button>
            <button type="button" @click="saveJob(edit.id)">更新</button>
          </div>
        </div>
      </div>

      <div v-if="appsJob" class="modal" @click.self="appsJob = null">
        <div class="modal-inner">
          <h3>応募者: {{ appsJob.summary }}</h3>
          <div v-for="a in applications" :key="a.id" style="border-bottom: 1px solid #e4e4e7; padding: 0.5rem 0">
            <div><strong>{{ a.applicant_name }}</strong> — {{ a.contact }}</div>
            <div style="white-space: pre-wrap; font-size: 0.9rem">{{ a.career_summary }}</div>
          </div>
          <div class="row" style="margin-top: 0.75rem; justify-content: flex-end"><button class="ghost" type="button" @click="appsJob = null">閉じる</button></div>
        </div>
      </div>
    </div>
  </div>
</template>
