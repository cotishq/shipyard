export type Project = {
  id: string;
  user_id: string;
  name: string;
  repo_url: string;
  build_preset: string;
  output_dir: string;
  default_branch: string;
  is_active: boolean;
  created_at: string;
};

export type Deployment = {
  id: string;
  project_id?: string;
  branch?: string;
  status: string;
  attempt_count: string;
  max_attempts: string;
  created_at: string;
  started_at?: string;
  finished_at?: string;
  error_message?: string;
  build_duration_seconds?: number;
  url: string;
};

export type DeploymentLog = {
  message: string;
  time: string;
};

export type ApiError = {
  error: string;
};
