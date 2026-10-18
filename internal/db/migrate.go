package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
CREATE TABLE IF NOT EXISTS agents (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL,
  hostname TEXT NOT NULL DEFAULT '',
  os TEXT NOT NULL DEFAULT '',
  agent_version TEXT NOT NULL DEFAULT '',
  token_hash TEXT NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_seen_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS smb_repositories (
  id UUID PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  smb_host TEXT NOT NULL,
  smb_share TEXT NOT NULL,
  repo_subpath TEXT NOT NULL DEFAULT '',
  username TEXT NOT NULL DEFAULT '',
  password TEXT NOT NULL DEFAULT '',
  restic_password TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS backup_jobs (
  id UUID PRIMARY KEY,
  agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  repository_id UUID NOT NULL REFERENCES smb_repositories(id) ON DELETE RESTRICT,
  name TEXT NOT NULL,
  sources JSONB NOT NULL DEFAULT '[]',
  excludes JSONB NOT NULL DEFAULT '[]',
  schedule_cron TEXT NOT NULL DEFAULT '',
  enabled BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS job_runs (
  id UUID PRIMARY KEY,
  job_id UUID NOT NULL REFERENCES backup_jobs(id) ON DELETE CASCADE,
  status TEXT NOT NULL DEFAULT 'pending',
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at TIMESTAMPTZ,
  finished_at TIMESTAMPTZ,
  bytes_added BIGINT NOT NULL DEFAULT 0,
  bytes_processed BIGINT NOT NULL DEFAULT 0,
  duration_ms BIGINT NOT NULL DEFAULT 0,
  files_new INT NOT NULL DEFAULT 0,
  files_changed INT NOT NULL DEFAULT 0,
  files_unmodified INT NOT NULL DEFAULT 0,
  snapshot_id TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  progress_pct REAL NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_job_runs_job_id ON job_runs(job_id);
CREATE INDEX IF NOT EXISTS idx_job_runs_status ON job_runs(status);
CREATE INDEX IF NOT EXISTS idx_backup_jobs_agent ON backup_jobs(agent_id);
`

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, schema)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}
