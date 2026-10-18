package models

import (
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Hostname      string     `json:"hostname"`
	OS            string     `json:"os"`
	AgentVersion  string     `json:"agentVersion"`
	CreatedAt     time.Time  `json:"createdAt"`
	LastSeenAt    *time.Time `json:"lastSeenAt,omitempty"`
}

type SMBRepository struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	SMBHost        string    `json:"smbHost"`
	SMBShare       string    `json:"smbShare"`
	RepoSubpath    string    `json:"repoSubpath"`
	Username       string    `json:"username"`
	Password       string    `json:"password,omitempty"`
	ResticPassword string    `json:"resticPassword,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type BackupJob struct {
	ID           uuid.UUID `json:"id"`
	AgentID      uuid.UUID `json:"agentId"`
	RepositoryID uuid.UUID `json:"repositoryId"`
	Name         string    `json:"name"`
	Sources      []string  `json:"sources"`
	Excludes     []string  `json:"excludes"`
	ScheduleCron string    `json:"scheduleCron"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"createdAt"`
	AgentName    string    `json:"agentName,omitempty"`
	RepoName     string    `json:"repoName,omitempty"`
}

type JobRunStatus string

const (
	RunPending   JobRunStatus = "pending"
	RunRunning   JobRunStatus = "running"
	RunSucceeded JobRunStatus = "succeeded"
	RunFailed    JobRunStatus = "failed"
	RunCancelled JobRunStatus = "cancelled"
)

type JobRun struct {
	ID              uuid.UUID    `json:"id"`
	JobID           uuid.UUID    `json:"jobId"`
	Status          JobRunStatus `json:"status"`
	RequestedAt     time.Time    `json:"requestedAt"`
	StartedAt       *time.Time   `json:"startedAt,omitempty"`
	FinishedAt      *time.Time   `json:"finishedAt,omitempty"`
	BytesAdded      int64        `json:"bytesAdded"`
	BytesProcessed  int64        `json:"bytesProcessed"`
	DurationMs      int64        `json:"durationMs"`
	FilesNew        int          `json:"filesNew"`
	FilesChanged    int          `json:"filesChanged"`
	FilesUnmodified int          `json:"filesUnmodified"`
	SnapshotID      string       `json:"snapshotId"`
	ErrorMessage    string       `json:"errorMessage,omitempty"`
	ProgressPct     float32      `json:"progressPct"`
	JobName         string       `json:"jobName,omitempty"`
	AgentName       string       `json:"agentName,omitempty"`
}

type DashboardStats struct {
	TotalAgents      int     `json:"totalAgents"`
	OnlineAgents     int     `json:"onlineAgents"`
	TotalJobs        int     `json:"totalJobs"`
	EnabledJobs      int     `json:"enabledJobs"`
	RunsLast24h      int     `json:"runsLast24h"`
	SuccessRate24h   float64 `json:"successRate24h"`
	BytesAdded24h    int64   `json:"bytesAdded24h"`
	AvgDurationMs24h int64   `json:"avgDurationMs24h"`
	AvgThroughputMBs float64 `json:"avgThroughputMBs"`
}

type AgentWorkItem struct {
	RunID        uuid.UUID `json:"runId"`
	JobID        uuid.UUID `json:"jobId"`
	JobName      string    `json:"jobName"`
	Sources      []string  `json:"sources"`
	Excludes     []string  `json:"excludes"`
	SMBHost      string    `json:"smbHost"`
	SMBShare     string    `json:"smbShare"`
	RepoSubpath  string    `json:"repoSubpath"`
	SMBUsername  string    `json:"smbUsername"`
	SMBPassword  string    `json:"smbPassword"`
	ResticPassword string  `json:"resticPassword"`
}

type AgentRunReport struct {
	Status          JobRunStatus `json:"status"`
	BytesAdded      int64        `json:"bytesAdded"`
	BytesProcessed  int64        `json:"bytesProcessed"`
	DurationMs      int64        `json:"durationMs"`
	FilesNew        int          `json:"filesNew"`
	FilesChanged    int          `json:"filesChanged"`
	FilesUnmodified int          `json:"filesUnmodified"`
	SnapshotID      string       `json:"snapshotId"`
	ErrorMessage    string       `json:"errorMessage"`
	ProgressPct     float32      `json:"progressPct"`
}
