package api

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/s4ndyp/backai/internal/auth"
	"github.com/s4ndyp/backai/internal/config"
	"github.com/s4ndyp/backai/internal/models"
	"github.com/s4ndyp/backai/internal/store"
	"github.com/s4ndyp/backai/internal/version"
)

type Server struct {
	cfg   config.Server
	store *store.Store
	web   fs.FS
	log   *slog.Logger
}

func New(cfg config.Server, st *store.Store, web fs.FS, log *slog.Logger) *Server {
	return &Server{cfg: cfg, store: st, web: web, log: log}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.handleHealth)
	mux.HandleFunc("GET /api/version", s.handleVersion)
	mux.HandleFunc("GET /api/dashboard", s.handleDashboard)
	mux.HandleFunc("GET /api/agents", s.handleListAgents)
	mux.HandleFunc("GET /api/repositories", s.handleListRepos)
	mux.HandleFunc("POST /api/repositories", s.handleCreateRepo)
	mux.HandleFunc("GET /api/jobs", s.handleListJobs)
	mux.HandleFunc("POST /api/jobs", s.handleCreateJob)
	mux.HandleFunc("PATCH /api/jobs/{id}", s.handlePatchJob)
	mux.HandleFunc("POST /api/jobs/{id}/run", s.handleRunJob)
	mux.HandleFunc("GET /api/runs", s.handleListRuns)
	mux.HandleFunc("GET /api/runs/{id}", s.handleGetRun)

	mux.HandleFunc("POST /api/agent/register", s.handleAgentRegister)
	mux.HandleFunc("POST /api/agent/heartbeat", s.handleAgentHeartbeat)
	mux.HandleFunc("POST /api/agent/work/claim", s.handleAgentClaimWork)
	mux.HandleFunc("POST /api/agent/work/{id}/report", s.handleAgentReport)

	if s.web != nil {
		mux.Handle("/", spaHandler(s.web))
	}

	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if err := s.store.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "db": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version":   version.Version,
		"commit":    version.Commit,
		"buildTime": version.BuildTime,
	})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	d, err := s.store.Dashboard(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListAgents(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListRepositories(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	for i := range list {
		list[i].Password = ""
		list[i].ResticPassword = ""
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	var body models.SMBRepository
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if body.Name == "" || body.SMBHost == "" || body.SMBShare == "" || body.ResticPassword == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, smbHost, smbShare and resticPassword required"})
		return
	}
	created, err := s.store.CreateRepository(r.Context(), body)
	if err != nil {
		writeErr(w, err)
		return
	}
	created.Password = ""
	created.ResticPassword = ""
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListJobs(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var body models.BackupJob
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if body.Name == "" || body.AgentID == uuid.Nil || body.RepositoryID == uuid.Nil || len(body.Sources) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name, agentId, repositoryId and sources required"})
		return
	}
	created, err := s.store.CreateJob(r.Context(), body)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handlePatchJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if body.Enabled != nil {
		if err := s.store.UpdateJobEnabled(r.Context(), id, *body.Enabled); err != nil {
			writeErr(w, err)
			return
		}
	}
	j, err := s.store.GetJob(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (s *Server) handleRunJob(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	if _, err := s.store.GetJob(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	run, err := s.store.CreateRun(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, run)
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListRuns(r.Context(), 100)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	run, err := s.store.GetRun(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

type agentRegisterReq struct {
	Name         string `json:"name"`
	RegisterKey  string `json:"registerKey"`
	AgentVersion string `json:"agentVersion"`
}

func (s *Server) handleAgentRegister(w http.ResponseWriter, r *http.Request) {
	var body agentRegisterReq
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if body.RegisterKey != s.cfg.AgentRegisterKey {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid register key"})
		return
	}
	if body.Name == "" {
		body.Name, _ = os.Hostname()
	}
	plain, hash, err := auth.NewAgentToken()
	if err != nil {
		writeErr(w, err)
		return
	}
	hostname, _ := os.Hostname()
	agent, err := s.store.CreateAgent(r.Context(), body.Name, hostname, runtime.GOOS, body.AgentVersion, hash)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"agent": agent,
		"token": plain,
	})
}

func (s *Server) agentFromRequest(r *http.Request) (models.Agent, bool) {
	tok := r.Header.Get("X-Backai-Agent-Token")
	if tok == "" {
		return models.Agent{}, false
	}
	a, err := s.store.AgentByTokenHash(r.Context(), auth.HashToken(tok))
	if err != nil {
		return models.Agent{}, false
	}
	return a, true
}

func (s *Server) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	agent, ok := s.agentFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var body struct {
		AgentVersion string `json:"agentVersion"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	ver := body.AgentVersion
	if ver == "" {
		ver = r.Header.Get("User-Agent")
	}
	_ = s.store.TouchAgent(r.Context(), agent.ID, ver)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAgentClaimWork(w http.ResponseWriter, r *http.Request) {
	agent, ok := s.agentFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	item, err := s.store.ClaimPendingRunForAgent(r.Context(), agent.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if item == nil {
		writeJSON(w, http.StatusNoContent, nil)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleAgentReport(w http.ResponseWriter, r *http.Request) {
	agent, ok := s.agentFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	runID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	run, err := s.store.GetRun(r.Context(), runID)
	if err != nil {
		writeErr(w, err)
		return
	}
	job, err := s.store.GetJob(r.Context(), run.JobID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if job.AgentID != agent.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	var rep models.AgentRunReport
	if err := json.NewDecoder(r.Body).Decode(&rep); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := s.store.UpdateRunProgress(r.Context(), runID, rep); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil && status == http.StatusNoContent {
		return
	}
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Backai-Agent-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func spaHandler(web fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(web))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && !strings.Contains(r.URL.Path, ".") {
			if _, err := fs.Stat(web, strings.TrimPrefix(r.URL.Path, "/")); err != nil {
				data, _ := fs.ReadFile(web, "index.html")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	})
}

// Scheduler triggers cron jobs.
type Scheduler struct {
	store *store.Store
	log   *slog.Logger
}

func NewScheduler(st *store.Store, log *slog.Logger) *Scheduler {
	return &Scheduler{store: st, log: log}
}

func (sch *Scheduler) Start(ctx context.Context) {
	go sch.loop(ctx)
}

func (sch *Scheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	sch.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sch.tick(ctx)
		}
	}
}

func (sch *Scheduler) tick(ctx context.Context) {
	jobs, err := sch.store.ListEnabledJobsWithCron(ctx)
	if err != nil {
		sch.log.Error("scheduler list jobs", "err", err)
		return
	}
	for _, j := range jobs {
		if !cronDue(j.ScheduleCron, time.Now()) {
			continue
		}
		since := time.Now().Add(-55 * time.Minute)
		has, err := sch.store.HasRecentRun(ctx, j.ID, since)
		if err != nil || has {
			continue
		}
		if _, err := sch.store.CreateRun(ctx, j.ID); err != nil {
			sch.log.Error("scheduler create run", "job", j.Name, "err", err)
		} else {
			sch.log.Info("scheduled backup run", "job", j.Name)
		}
	}
}

// Minimal cron: only supports "M H * * *" and "@hourly" "@daily" for MVP.
func cronDue(spec string, now time.Time) bool {
	spec = strings.TrimSpace(spec)
	switch spec {
	case "@hourly":
		return now.Minute() == 0
	case "@daily":
		return now.Hour() == 2 && now.Minute() == 0
	case "":
		return false
	}
	parts := strings.Fields(spec)
	if len(parts) != 5 {
		return false
	}
	if parts[2] != "*" || parts[3] != "*" || parts[4] != "*" {
		return false
	}
	minOK := parts[0] == "*" || parts[0] == strconv.Itoa(now.Minute())
	hourOK := parts[1] == "*" || parts[1] == strconv.Itoa(now.Hour())
	return minOK && hourOK
}
