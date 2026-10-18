package restic

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/s4ndyp/backai/internal/models"
)

type Runner struct {
	Binary string
}

type BackupResult struct {
	SnapshotID      string
	BytesAdded      int64
	BytesProcessed  int64
	FilesNew        int
	FilesChanged    int
	FilesUnmodified int
	DurationMs      int64
}

func (r *Runner) EnsureRepo(ctx context.Context, repoPath, password string) error {
	cmd := exec.CommandContext(ctx, r.Binary, "-r", repoPath, "snapshots")
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+password)
	if out, err := cmd.CombinedOutput(); err == nil {
		_ = out
		return nil
	}
	initCmd := exec.CommandContext(ctx, r.Binary, "init", "-r", repoPath)
	initCmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+password)
	if out, err := initCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("restic init: %s: %w", string(out), err)
	}
	return nil
}

func (r *Runner) Backup(ctx context.Context, repoPath, password string, sources, excludes []string, onProgress func(pct float32, processed int64)) (BackupResult, error) {
	start := time.Now()
	args := []string{"-r", repoPath, "backup", "--json"}
	for _, ex := range excludes {
		if ex != "" {
			args = append(args, "--exclude", ex)
		}
	}
	args = append(args, sources...)

	cmd := exec.CommandContext(ctx, r.Binary, args...)
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+password)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return BackupResult{}, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return BackupResult{}, err
	}
	if err := cmd.Start(); err != nil {
		return BackupResult{}, err
	}

	var result BackupResult
	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		var msg struct {
			MessageType   string `json:"message_type"`
			PercentDone   float64 `json:"percent_done"`
			BytesProcessed int64  `json:"bytes_processed"`
			SnapshotID    string `json:"snapshot_id"`
			ID            string `json:"id"`
			Summary       *struct {
				FilesNew        int   `json:"files_new"`
				FilesChanged    int   `json:"files_changed"`
				FilesUnmodified int   `json:"files_unmodified"`
				DataAdded       int64 `json:"data_added"`
				TotalBytesProcessed int64 `json:"total_bytes_processed"`
			} `json:"summary"`
		}
		if json.Unmarshal([]byte(line), &msg) != nil {
			continue
		}
		switch msg.MessageType {
		case "status":
			if onProgress != nil && msg.PercentDone > 0 {
				onProgress(float32(msg.PercentDone*100), msg.BytesProcessed)
			}
		case "summary":
			if msg.Summary != nil {
				result.FilesNew = msg.Summary.FilesNew
				result.FilesChanged = msg.Summary.FilesChanged
				result.FilesUnmodified = msg.Summary.FilesUnmodified
				result.BytesAdded = msg.Summary.DataAdded
				result.BytesProcessed = msg.Summary.TotalBytesProcessed
			}
			if msg.SnapshotID != "" {
				result.SnapshotID = msg.SnapshotID
			}
		}
	}
	errBytes, _ := io.ReadAll(stderr)
	if err := cmd.Wait(); err != nil {
		return result, fmt.Errorf("restic backup: %s: %w", string(errBytes), err)
	}
	result.DurationMs = time.Since(start).Milliseconds()
	if result.SnapshotID == "" {
		result.SnapshotID = lookupLatestSnapshot(ctx, r.Binary, repoPath, password)
	}
	return result, nil
}

func lookupLatestSnapshot(ctx context.Context, binary, repoPath, password string) string {
	cmd := exec.CommandContext(ctx, binary, "-r", repoPath, "snapshots", "--latest", "1", "--json")
	cmd.Env = append(os.Environ(), "RESTIC_PASSWORD="+password)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	var snaps []struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(out, &snaps) == nil && len(snaps) > 0 {
		return snaps[0].ID
	}
	return ""
}

func RepoPath(mountRoot, subpath string) string {
	p := mountRoot
	if subpath != "" {
		p = filepath.Join(mountRoot, filepath.FromSlash(strings.ReplaceAll(subpath, "\\", "/")))
	}
	return filepath.Clean(p)
}

func FailedReport(err error, started time.Time) models.AgentRunReport {
	return models.AgentRunReport{
		Status:       models.RunFailed,
		ErrorMessage: err.Error(),
		DurationMs:   time.Since(started).Milliseconds(),
	}
}
