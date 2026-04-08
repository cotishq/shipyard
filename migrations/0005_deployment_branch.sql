ALTER TABLE deployments
ADD COLUMN branch TEXT NOT NULL DEFAULT 'main';

CREATE INDEX idx_deployments_project_id_branch_created_at
ON deployments(project_id, branch, created_at DESC);
