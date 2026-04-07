BEGIN;

CREATE INDEX IF NOT EXISTS idx_projects_user_id_created_at
ON projects(user_id, created_at DESC);

DO $$
DECLARE
  missing_count BIGINT;
BEGIN
  SELECT COUNT(*) INTO missing_count
  FROM deployments
  WHERE project_id IS NULL;

  IF missing_count > 0 THEN
    RAISE EXCEPTION
      'cannot enforce NOT NULL on deployments.project_id: % rows missing project_id',
      missing_count;
  END IF;
END
$$;

ALTER TABLE deployments
ALTER COLUMN project_id SET NOT NULL;

COMMIT;
