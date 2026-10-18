/// <reference path="../pb_data/types.d.ts" />

migrate(
  (app) => {
    const open = "";

    const agents = new Collection({
      type: "base",
      name: "agents",
      listRule: open,
      viewRule: open,
      createRule: open,
      updateRule: open,
      deleteRule: open,
      fields: [
        { type: "text", name: "name", required: true, max: 120 },
        { type: "text", name: "hostname", required: false, max: 255 },
        { type: "text", name: "os", required: false, max: 32 },
        { type: "text", name: "agent_version", required: false, max: 64 },
        { type: "text", name: "token_hash", required: true, max: 128, hidden: true },
        { type: "date", name: "last_seen_at", required: false },
      ],
    });
    app.save(agents);

    const smbRepositories = new Collection({
      type: "base",
      name: "smb_repositories",
      listRule: open,
      viewRule: open,
      createRule: open,
      updateRule: open,
      deleteRule: open,
      fields: [
        { type: "text", name: "name", required: true, max: 120 },
        { type: "text", name: "smb_host", required: true, max: 255 },
        { type: "text", name: "smb_share", required: true, max: 255 },
        { type: "text", name: "repo_subpath", required: false, max: 255 },
        { type: "text", name: "username", required: false, max: 120 },
        { type: "text", name: "password", required: false, max: 255, hidden: true },
        { type: "text", name: "restic_password", required: true, max: 255, hidden: true },
      ],
    });
    app.save(smbRepositories);

    const backupJobs = new Collection({
      type: "base",
      name: "backup_jobs",
      listRule: open,
      viewRule: open,
      createRule: open,
      updateRule: open,
      deleteRule: open,
      fields: [
        { type: "text", name: "name", required: true, max: 120 },
        {
          type: "relation",
          name: "agent",
          required: true,
          maxSelect: 1,
          collectionId: agents.id,
          cascadeDelete: true,
        },
        {
          type: "relation",
          name: "repository",
          required: true,
          maxSelect: 1,
          collectionId: smbRepositories.id,
          cascadeDelete: false,
        },
        { type: "json", name: "sources", required: true },
        { type: "json", name: "excludes", required: false },
        { type: "text", name: "schedule_cron", required: false, max: 64 },
        { type: "bool", name: "enabled", required: true },
      ],
    });
    app.save(backupJobs);

    const jobRuns = new Collection({
      type: "base",
      name: "job_runs",
      listRule: open,
      viewRule: open,
      createRule: open,
      updateRule: open,
      deleteRule: open,
      indexes: [
        "CREATE INDEX idx_job_runs_status ON job_runs (status)",
        "CREATE INDEX idx_job_runs_agent ON job_runs (agent)",
      ],
      fields: [
        {
          type: "relation",
          name: "job",
          required: true,
          maxSelect: 1,
          collectionId: backupJobs.id,
          cascadeDelete: true,
        },
        {
          type: "relation",
          name: "agent",
          required: true,
          maxSelect: 1,
          collectionId: agents.id,
          cascadeDelete: true,
        },
        {
          type: "select",
          name: "status",
          required: true,
          maxSelect: 1,
          values: ["pending", "running", "succeeded", "failed", "cancelled"],
        },
        { type: "date", name: "requested_at", required: true },
        { type: "date", name: "started_at", required: false },
        { type: "date", name: "finished_at", required: false },
        { type: "number", name: "bytes_added", required: false, min: 0 },
        { type: "number", name: "bytes_processed", required: false, min: 0 },
        { type: "number", name: "duration_ms", required: false, min: 0 },
        { type: "number", name: "files_new", required: false, min: 0 },
        { type: "number", name: "files_changed", required: false, min: 0 },
        { type: "number", name: "files_unmodified", required: false, min: 0 },
        { type: "text", name: "snapshot_id", required: false, max: 128 },
        { type: "text", name: "error_message", required: false, max: 2000 },
        { type: "number", name: "progress_pct", required: false, min: 0, max: 100 },
      ],
    });
    app.save(jobRuns);
  },
  (app) => {
    for (const name of ["job_runs", "backup_jobs", "smb_repositories", "agents"]) {
      try {
        app.delete(app.findCollectionByNameOrId(name));
      } catch (_) {
        /* ignore */
      }
    }
  }
);
