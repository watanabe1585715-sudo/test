/** 案件管理 API が返す案件（JSON のフィールド名に合わせる） */
export interface JobRow {
  id: number;
  customer_id: number;
  summary: string;
  requirements: string;
  publish_start: string;
  publish_end: string;
  publication_status: string;
  created_at: string;
  updated_at: string;
}

export interface ApplicationRow {
  id: number;
  job_posting_id: number;
  applicant_name: string;
  career_summary: string;
  contact: string;
  created_at: string;
}

export interface CompanyProfile {
  customer_id: number;
  company_name: string;
  description: string;
  address: string;
  google_map_url?: string | null;
  website_url?: string | null;
  youtube_embed_url?: string | null;
  accept_foreigners: boolean;
  languages: string;
}
