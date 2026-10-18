package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/s4ndyp/backai/internal/agentclient"
	"github.com/s4ndyp/backai/internal/config"
	"github.com/s4ndyp/backai/internal/models"
	"github.com/s4ndyp/backai/internal/restic"
	"github.com/s4ndyp/backai/internal/smbmount"
	"github.com/s4ndyp/backai/internal/version"
)

func main() {
	_ = godotenv.Load()
	cfg := config.LoadAgent()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	token := cfg.Token
	if token == "" {
		if cfg.RegisterKey == "" || cfg.ServerURL == "" {
			log.Error("set BACKAI_AGENT_TOKEN or BACKAI_SERVER_URL + BACKAI_AGENT_REGISTER_KEY")
			os.Exit(1)
		}
		name := cfg.Name
		if name == "" {
			name, _ = os.Hostname()
		}
		res, err := agentclient.New("", "").Register(cfg.ServerURL, cfg.RegisterKey, name, version.Version)
		if err != nil {
			log.Error("register failed", "err", err)
			os.Exit(1)
		}
		token = res.Token
		log.Info("registered agent", "id", res.Agent.ID, "name", res.Agent.Name)
		log.Info("save this token securely", "BACKAI_AGENT_TOKEN", token)
	}

	client := agentclient.New(cfg.ServerURL, token)
	runner := &restic.Runner{Binary: cfg.ResticPath}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	ticker := time.NewTicker(cfg.PollEvery)
	defer ticker.Stop()

	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if err := client.Heartbeat(version.Version); err != nil {
				log.Warn("heartbeat", "err", err)
			}
		case <-ticker.C:
			work, err := client.ClaimWork()
			if err != nil {
				log.Warn("claim work", "err", err)
				continue
			}
			if work == nil {
				continue
			}
			runBackup(ctx, log, client, runner, work)
		}
	}
}

func runBackup(ctx context.Context, log *slog.Logger, client *agentclient.Client, runner *restic.Runner, work *models.AgentWorkItem) {
	started := time.Now()
	log.Info("backup started", "run", work.RunID, "job", work.JobName)

	report := func(rep models.AgentRunReport) {
		if err := client.Report(work.RunID, rep); err != nil {
			log.Warn("progress report", "err", err)
		}
	}

	mnt, err := smbmount.MountTemp(smbmount.Config{
		Host:     work.SMBHost,
		Share:    work.SMBShare,
		Username: work.SMBUsername,
		Password: work.SMBPassword,
	})
	if err != nil {
		rep := restic.FailedReport(err, started)
		_ = client.Report(work.RunID, rep)
		log.Error("smb mount", "err", err)
		return
	}
	defer func() { _ = mnt.Unmount() }()

	repoPath := restic.RepoPath(mnt.LocalPath, work.RepoSubpath)
	if err := runner.EnsureRepo(ctx, repoPath, work.ResticPassword); err != nil {
		rep := restic.FailedReport(err, started)
		_ = client.Report(work.RunID, rep)
		log.Error("restic init", "err", err)
		return
	}

	_ = client.Report(work.RunID, models.AgentRunReport{
		Status:      models.RunRunning,
		ProgressPct: 1,
	})

	res, err := runner.Backup(ctx, repoPath, work.ResticPassword, work.Sources, work.Excludes, func(pct float32, processed int64) {
		report(models.AgentRunReport{
			Status:         models.RunRunning,
			ProgressPct:    pct,
			BytesProcessed: processed,
		})
	})
	if err != nil {
		rep := restic.FailedReport(err, started)
		_ = client.Report(work.RunID, rep)
		log.Error("backup failed", "err", err)
		return
	}

	final := models.AgentRunReport{
		Status:          models.RunSucceeded,
		BytesAdded:      res.BytesAdded,
		BytesProcessed:  res.BytesProcessed,
		DurationMs:      res.DurationMs,
		FilesNew:        res.FilesNew,
		FilesChanged:    res.FilesChanged,
		FilesUnmodified: res.FilesUnmodified,
		SnapshotID:      res.SnapshotID,
		ProgressPct:     100,
	}
	_ = client.Report(work.RunID, final)
	log.Info("backup finished", "run", work.RunID, "snapshot", res.SnapshotID, "bytes_added", res.BytesAdded)
}
