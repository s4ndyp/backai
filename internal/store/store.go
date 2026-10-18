package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/s4ndyp/backai/internal/models"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateAgent(ctx context.Context, name, hostname, osName, version, tokenHash string) (models.Agent, error) {
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO agents (id, name, hostname, os, agent_version, token_hash)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id, name, hostname, osName, version, tokenHash)
	if err != nil {
		return models.Agent{}, err
	}
	return s.GetAgent(ctx, id)
}

func (s *Store) AgentByTokenHash(ctx context.Context, hash string) (models.Agent, error) {
	var a models.Agent
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, hostname, os, agent_version, created_at, last_seen_at
		FROM agents WHERE token_hash = $1`, hash).Scan(
		&a.ID, &a.Name, &a.Hostname, &a.OS, &a.AgentVersion, &a.CreatedAt, &a.LastSeenAt)
	return a, err
}

func (s *Store) GetAgent(ctx context.Context, id uuid.UUID) (models.Agent, error) {
	var a models.Agent
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, hostname, os, agent_version, created_at, last_seen_at
		FROM agents WHERE id = $1`, id).Scan(
		&a.ID, &a.Name, &a.Hostname, &a.OS, &a.AgentVersion, &a.CreatedAt, &a.LastSeenAt)
	return a, err
}

func (s *Store) ListAgents(ctx context.Context) ([]models.Agent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, hostname, os, agent_version, created_at, last_seen_at
		FROM agents ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Agent
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(&a.ID, &a.Name, &a.Hostname, &a.OS, &a.AgentVersion, &a.CreatedAt, &a.LastSeenAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) TouchAgent(ctx context.Context, id uuid.UUID, version string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE agents SET last_seen_at = now(), agent_version = $2 WHERE id = $1`, id, version)
	return err
}

func (s *Store) CreateRepository(ctx context.Context, r models.SMBRepository) (models.SMBRepository, error) {
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO smb_repositories (id, name, smb_host, smb_share, repo_subpath, username, password, restic_password)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, r.Name, r.SMBHost, r.SMBShare, r.RepoSubpath, r.Username, r.Password, r.ResticPassword)
	if err != nil {
		return models.SMBRepository{}, err
	}
	return s.GetRepository(ctx, id)
}

func (s *Store) GetRepository(ctx context.Context, id uuid.UUID) (models.SMBRepository, error) {
	var r models.SMBRepository
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, smb_host, smb_share, repo_subpath, username, password, restic_password, created_at
		FROM smb_repositories WHERE id = $1`, id).Scan(
		&r.ID, &r.Name, &r.SMBHost, &r.SMBShare, &r.RepoSubpath, &r.Username, &r.Password, &r.ResticPassword, &r.CreatedAt)
	return r, err
}

func (s *Store) ListRepositories(ctx context.Context) ([]models.SMBRepository, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, smb_host, smb_share, repo_subpath, username, password, restic_password, created_at
		FROM smb_repositories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.SMBRepository
	for rows.Next() {
		var r models.SMBRepository
		if err := rows.Scan(&r.ID, &r.Name, &r.SMBHost, &r.SMBShare, &r.RepoSubpath, &r.Username, &r.Password, &r.ResticPassword, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) CreateJob(ctx context.Context, j models.BackupJob) (models.BackupJob, error) {
	id := uuid.New()
	src, _ := json.Marshal(j.Sources)
	ex, _ := json.Marshal(j.Excludes)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO backup_jobs (id, agent_id, repository_id, name, sources, excludes, schedule_cron, enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		id, j.AgentID, j.RepositoryID, j.Name, src, ex, j.ScheduleCron, j.Enabled)
	if err != nil {
		return models.BackupJob{}, err
	}
	return s.GetJob(ctx, id)
}

func (s *Store) GetJob(ctx context.Context, id uuid.UUID) (models.BackupJob, error) {
	var j models.BackupJob
	var src, ex []byte
	err := s.pool.QueryRow(ctx, `
		SELECT j.id, j.agent_id, j.repository_id, j.name, j.sources, j.excludes, j.schedule_cron, j.enabled, j.created_at,
		       COALESCE(a.name,''), COALESCE(r.name,'')
		FROM backup_jobs j
		LEFT JOIN agents a ON a.id = j.agent_id
		LEFT JOIN smb_repositories r ON r.id = j.repository_id
		WHERE j.id = $1`, id).Scan(
		&j.ID, &j.AgentID, &j.RepositoryID, &j.Name, &src, &ex, &j.ScheduleCron, &j.Enabled, &j.CreatedAt, &j.AgentName, &j.RepoName)
	if err != nil {
		return j, err
	}
	_ = json.Unmarshal(src, &j.Sources)
	_ = json.Unmarshal(ex, &j.Excludes)
	return j, nil
}

func (s *Store) ListJobs(ctx context.Context) ([]models.BackupJob, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT j.id, j.agent_id, j.repository_id, j.name, j.sources, j.excludes, j.schedule_cron, j.enabled, j.created_at,
		       COALESCE(a.name,''), COALESCE(r.name,'')
		FROM backup_jobs j
		LEFT JOIN agents a ON a.id = j.agent_id
		LEFT JOIN smb_repositories r ON r.id = j.repository_id
		ORDER BY j.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.BackupJob
	for rows.Next() {
		var j models.BackupJob
		var src, ex []byte
		if err := rows.Scan(&j.ID, &j.AgentID, &j.RepositoryID, &j.Name, &src, &ex, &j.ScheduleCron, &j.Enabled, &j.CreatedAt, &j.AgentName, &j.RepoName); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(src, &j.Sources)
		_ = json.Unmarshal(ex, &j.Excludes)
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) UpdateJobEnabled(ctx context.Context, id uuid.UUID, enabled bool) error {
	_, err := s.pool.Exec(ctx, `UPDATE backup_jobs SET enabled = $2 WHERE id = $1`, id, enabled)
	return err
}

func (s *Store) CreateRun(ctx context.Context, jobID uuid.UUID) (models.JobRun, error) {
	id := uuid.New()
	_, err := s.pool.Exec(ctx, `
		INSERT INTO job_runs (id, job_id, status) VALUES ($1, $2, 'pending')`, id, jobID)
	if err != nil {
		return models.JobRun{}, err
	}
	return s.GetRun(ctx, id)
}

func (s *Store) GetRun(ctx context.Context, id uuid.UUID) (models.JobRun, error) {
	var r models.JobRun
	err := s.pool.QueryRow(ctx, `
		SELECT jr.id, jr.job_id, jr.status, jr.requested_at, jr.started_at, jr.finished_at,
		       jr.bytes_added, jr.bytes_processed, jr.duration_ms, jr.files_new, jr.files_changed,
		       jr.files_unmodified, jr.snapshot_id, jr.error_message, jr.progress_pct,
		       COALESCE(j.name,''), COALESCE(a.name,'')
		FROM job_runs jr
		JOIN backup_jobs j ON j.id = jr.job_id
		LEFT JOIN agents a ON a.id = j.agent_id
		WHERE jr.id = $1`, id).Scan(
		&r.ID, &r.JobID, &r.Status, &r.RequestedAt, &r.StartedAt, &r.FinishedAt,
		&r.BytesAdded, &r.BytesProcessed, &r.DurationMs, &r.FilesNew, &r.FilesChanged,
		&r.FilesUnmodified, &r.SnapshotID, &r.ErrorMessage, &r.ProgressPct, &r.JobName, &r.AgentName)
	return r, err
}

func (s *Store) ListRuns(ctx context.Context, limit int) ([]models.JobRun, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT jr.id, jr.job_id, jr.status, jr.requested_at, jr.started_at, jr.finished_at,
		       jr.bytes_added, jr.bytes_processed, jr.duration_ms, jr.files_new, jr.files_changed,
		       jr.files_unmodified, jr.snapshot_id, jr.error_message, jr.progress_pct,
		       COALESCE(j.name,''), COALESCE(a.name,'')
		FROM job_runs jr
		JOIN backup_jobs j ON j.id = jr.job_id
		LEFT JOIN agents a ON a.id = j.agent_id
		ORDER BY jr.requested_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.JobRun
	for rows.Next() {
		var r models.JobRun
		if err := rows.Scan(&r.ID, &r.JobID, &r.Status, &r.RequestedAt, &r.StartedAt, &r.FinishedAt,
			&r.BytesAdded, &r.BytesProcessed, &r.DurationMs, &r.FilesNew, &r.FilesChanged,
			&r.FilesUnmodified, &r.SnapshotID, &r.ErrorMessage, &r.ProgressPct, &r.JobName, &r.AgentName); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) ClaimPendingRunForAgent(ctx context.Context, agentID uuid.UUID) (*models.AgentWorkItem, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var runID, jobID uuid.UUID
	var jobName string
	var src, ex []byte
	err = tx.QueryRow(ctx, `
		SELECT jr.id, j.id, j.name, j.sources, j.excludes
		FROM job_runs jr
		JOIN backup_jobs j ON j.id = jr.job_id
		WHERE j.agent_id = $1 AND jr.status = 'pending' AND j.enabled = true
		ORDER BY jr.requested_at
		FOR UPDATE SKIP LOCKED
		LIMIT 1`, agentID).Scan(&runID, &jobID, &jobName, &src, &ex)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}

	var sources, excludes []string
	_ = json.Unmarshal(src, &sources)
	_ = json.Unmarshal(ex, &excludes)

	var item models.AgentWorkItem
	err = tx.QueryRow(ctx, `
		SELECT r.smb_host, r.smb_share, r.repo_subpath, r.username, r.password, r.restic_password
		FROM backup_jobs j
		JOIN smb_repositories r ON r.id = j.repository_id
		WHERE j.id = $1`, jobID).Scan(
		&item.SMBHost, &item.SMBShare, &item.RepoSubpath, &item.SMBUsername, &item.SMBPassword, &item.ResticPassword)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		UPDATE job_runs SET status = 'running', started_at = now() WHERE id = $1`, runID)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	item.RunID = runID
	item.JobID = jobID
	item.JobName = jobName
	item.Sources = sources
	item.Excludes = excludes
	return &item, nil
}

func (s *Store) UpdateRunProgress(ctx context.Context, runID uuid.UUID, rep models.AgentRunReport) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE job_runs SET
		  status = $2,
		  bytes_added = $3,
		  bytes_processed = $4,
		  duration_ms = $5,
		  files_new = $6,
		  files_changed = $7,
		  files_unmodified = $8,
		  snapshot_id = $9,
		  error_message = $10,
		  progress_pct = $11,
		  finished_at = CASE WHEN $2 IN ('succeeded','failed','cancelled') THEN now() ELSE finished_at END
		WHERE id = $1`,
		runID, rep.Status, rep.BytesAdded, rep.BytesProcessed, rep.DurationMs,
		rep.FilesNew, rep.FilesChanged, rep.FilesUnmodified, rep.SnapshotID, rep.ErrorMessage, rep.ProgressPct)
	return err
}

func (s *Store) Dashboard(ctx context.Context) (models.DashboardStats, error) {
	var d models.DashboardStats
	err := s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents`).Scan(&d.TotalAgents)
	if err != nil {
		return d, err
	}
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*) FROM agents WHERE last_seen_at > now() - interval '5 minutes'`).Scan(&d.OnlineAgents)
	if err != nil {
		return d, err
	}
	err = s.pool.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE enabled) FROM backup_jobs`).Scan(&d.TotalJobs, &d.EnabledJobs)
	if err != nil {
		return d, err
	}
	var success, total int
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*),
		       COUNT(*) FILTER (WHERE status = 'succeeded'),
		       COALESCE(SUM(bytes_added) FILTER (WHERE status = 'succeeded'), 0),
		       COALESCE(AVG(duration_ms) FILTER (WHERE status = 'succeeded' AND duration_ms > 0), 0)
		FROM job_runs WHERE requested_at > now() - interval '24 hours'`).Scan(&d.RunsLast24h, &success, &d.BytesAdded24h, &d.AvgDurationMs24h)
	if err != nil {
		return d, err
	}
	total = d.RunsLast24h
	if total > 0 {
		d.SuccessRate24h = float64(success) / float64(total) * 100
	}
	if d.AvgDurationMs24h > 0 && d.BytesAdded24h > 0 {
		sec := float64(d.AvgDurationMs24h) / 1000
		d.AvgThroughputMBs = float64(d.BytesAdded24h) / sec / (1024 * 1024)
	}
	return d, nil
}

func (s *Store) ListEnabledJobsWithCron(ctx context.Context) ([]models.BackupJob, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT j.id, j.agent_id, j.repository_id, j.name, j.sources, j.excludes, j.schedule_cron, j.enabled, j.created_at, '', ''
		FROM backup_jobs j
		WHERE j.enabled = true AND j.schedule_cron <> ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.BackupJob
	for rows.Next() {
		var j models.BackupJob
		var src, ex []byte
		if err := rows.Scan(&j.ID, &j.AgentID, &j.RepositoryID, &j.Name, &src, &ex, &j.ScheduleCron, &j.Enabled, &j.CreatedAt, &j.AgentName, &j.RepoName); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(src, &j.Sources)
		_ = json.Unmarshal(ex, &j.Excludes)
		out = append(out, j)
	}
	return out, rows.Err()
}

func (s *Store) HasRecentRun(ctx context.Context, jobID uuid.UUID, since time.Time) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM job_runs WHERE job_id = $1 AND requested_at > $2`, jobID, since).Scan(&n)
	return n > 0, err
}

func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func (s *Store) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	return pool, nil
}
