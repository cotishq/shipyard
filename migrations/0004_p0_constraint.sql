CREATE INDEX IF NOT EXISTS idx_projects_user_id_created_at
ON projects(user_id, created_at DESC);

-- Create a legacy owner user once (for old v1 deployments that had no project_id).
INSERT INTO users (id, email, name)
VALUES (
  '00000000-0000-0000-0000-000000000001',
  'legacy@shipyard.local',
  'Legacy Imports'
)
ON CONFLICT (email) DO NOTHING;

-- Create one legacy project to attach old deployments.
INSERT INTO projects (id, user_id, name, repo_url, build_preset, output_dir, default_branch, is_active)
VALUES (
  '00000000-0000-0000-0000-000000000002',
  '00000000-0000-0000-0000-000000000001',
  'legacy-import',
  'https://github.com/mdn/beginner-html-site',
  'static-copy',
  '',
  'main',
  TRUE
)
ON CONFLICT (user_id, name) DO NOTHING;

-- Backfill old rows.
UPDATE deployments
SET project_id = '00000000-0000-0000-0000-000000000002'
WHERE project_id IS NULL;

ALTER TABLE deployments
ALTER COLUMN project_id SET NOT NULL;
