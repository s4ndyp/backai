/// <reference path="../pb_data/types.d.ts" />

const REGISTER_KEY = () => $os.getenv("BACKAI_AGENT_REGISTER_KEY") || "change-agent-register-key";
const APP_VERSION = () => $os.getenv("BACKAI_VERSION") || "dev";

function sha256Hex(s) {
  return $security.sha256(s);
}

function randomToken() {
  return $security.randomString(48);
}

function agentFromRequest(c) {
  const tok = c.request.header.get("X-Backai-Agent-Token");
  if (!tok) return null;
  const hash = sha256Hex(tok);
  try {
    return $app.findFirstRecordByFilter("agents", "token_hash = {:h}", { h: hash });
  } catch (_) {
    return null;
  }
}

function readJson(c) {
  const raw = readerToString(c.request.body);
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch (_) {
    return {};
  }
}

function agentPublic(rec) {
  return {
    id: rec.id,
    name: rec.get("name"),
    hostname: rec.get("hostname"),
    os: rec.get("os"),
    agentVersion: rec.get("agent_version"),
    createdAt: rec.get("created"),
    lastSeenAt: rec.get("last_seen_at"),
  };
}

function cronDue(spec, now) {
  spec = (spec || "").trim();
  if (spec === "@hourly") return now.getMinutes() === 0;
  if (spec === "@daily") return now.getHours() === 2 && now.getMinutes() === 0;
  if (!spec) return false;
  const parts = spec.split(/\s+/);
  if (parts.length !== 5) return false;
  if (parts[2] !== "*" || parts[3] !== "*" || parts[4] !== "*") return false;
  const minOk = parts[0] === "*" || parts[0] === String(now.getMinutes());
  const hourOk = parts[1] === "*" || parts[1] === String(now.getHours());
  return minOk && hourOk;
}

function createRunForJob(jobRec) {
  const col = $app.findCollectionByNameOrId("job_runs");
  const run = new Record(col);
  run.set("job", jobRec.id);
  run.set("agent", jobRec.get("agent"));
  run.set("status", "pending");
  run.set("requested_at", new Date().toISOString());
  run.set("bytes_added", 0);
  run.set("bytes_processed", 0);
  run.set("duration_ms", 0);
  run.set("files_new", 0);
  run.set("files_changed", 0);
  run.set("files_unmodified", 0);
  run.set("progress_pct", 0);
  $app.save(run);
  return run;
}

routerAdd("GET", "/api/backai/health", (c) => {
  return c.json(200, { status: "ok" });
});

routerAdd("GET", "/api/backai/version", (c) => {
  return c.json(200, {
    version: APP_VERSION(),
    commit: $os.getenv("BACKAI_COMMIT") || "none",
    buildTime: $os.getenv("BACKAI_BUILD_TIME") || "unknown",
  });
});

routerAdd("GET", "/api/backai/dashboard", (c) => {
  const now = new Date();
  const fiveMinAgo = new Date(now.getTime() - 5 * 60 * 1000).toISOString();
  const dayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();

  const totalAgents = $app.countRecords("agents", "");
  let onlineAgents = 0;
  try {
    onlineAgents = $app.countRecords("agents", `last_seen_at >= "${fiveMinAgo}"`);
  } catch (_) {
    onlineAgents = 0;
  }

  const totalJobs = $app.countRecords("backup_jobs", "");
  const enabledJobs = $app.countRecords("backup_jobs", "enabled = true");

  const runs = $app.findRecordsByFilter("job_runs", `requested_at >= "${dayAgo}"`, "-requested_at", 500, 0);
  let success = 0;
  let bytesAdded = 0;
  let durationSum = 0;
  let durationCount = 0;
  for (const r of runs) {
    if (r.get("status") === "succeeded") {
      success++;
      bytesAdded += Number(r.get("bytes_added") || 0);
      const d = Number(r.get("duration_ms") || 0);
      if (d > 0) {
        durationSum += d;
        durationCount++;
      }
    }
  }
  const runsLast24h = runs.length;
  const avgDurationMs = durationCount ? Math.round(durationSum / durationCount) : 0;
  const avgThroughputMBs =
    avgDurationMs > 0 && bytesAdded > 0
      ? bytesAdded / (avgDurationMs / 1000) / (1024 * 1024)
      : 0;

  return c.json(200, {
    totalAgents,
    onlineAgents,
    totalJobs,
    enabledJobs,
    runsLast24h,
    successRate24h: runsLast24h ? (success / runsLast24h) * 100 : 0,
    bytesAdded24h: bytesAdded,
    avgDurationMs24h: avgDurationMs,
    avgThroughputMBs,
  });
});

routerAdd("POST", "/api/backai/agent/register", (c) => {
  const body = readJson(c);
  if (body.registerKey !== REGISTER_KEY()) {
    return c.json(401, { error: "invalid register key" });
  }
  let name = body.name || "";
  if (!name) name = "agent";

  const plain = randomToken();
  const hash = sha256Hex(plain);

  const col = $app.findCollectionByNameOrId("agents");
  const rec = new Record(col);
  rec.set("name", name);
  rec.set("hostname", body.hostname || name);
  rec.set("os", body.os || body.osName || "");
  rec.set("agent_version", body.agentVersion || "");
  rec.set("token_hash", hash);
  $app.save(rec);

  return c.json(201, { agent: agentPublic(rec), token: plain });
});

routerAdd("POST", "/api/backai/agent/heartbeat", (c) => {
  const agent = agentFromRequest(c);
  if (!agent) return c.json(401, { error: "unauthorized" });
  const body = readJson(c);
  const ver = body.agentVersion || c.request.header.get("User-Agent") || "";
  agent.set("last_seen_at", new Date().toISOString());
  if (ver) agent.set("agent_version", ver);
  $app.save(agent);
  return c.json(200, { status: "ok" });
});

routerAdd("POST", "/api/backai/agent/work/claim", (c) => {
  const agent = agentFromRequest(c);
  if (!agent) return c.json(401, { error: "unauthorized" });

  let pending = [];
  try {
    pending = $app.findRecordsByFilter(
      "job_runs",
      `status = "pending" && agent = {:agentId}`,
      "requested_at",
      1,
      0,
      { agentId: agent.id }
    );
  } catch (_) {
    pending = [];
  }

  if (!pending.length) {
    return c.noContent(204);
  }

  const run = pending[0];
  const job = $app.findRecordById("backup_jobs", run.get("job"));
  if (!job.get("enabled")) {
    return c.noContent(204);
  }

  run.set("status", "running");
  run.set("started_at", new Date().toISOString());
  $app.save(run);

  const repo = $app.findRecordById("smb_repositories", job.get("repository"));
  const sources = job.get("sources") || [];
  const excludes = job.get("excludes") || [];

  return c.json(200, {
    runId: run.id,
    jobId: job.id,
    jobName: job.get("name"),
    sources: Array.isArray(sources) ? sources : [],
    excludes: Array.isArray(excludes) ? excludes : [],
    smbHost: repo.get("smb_host"),
    smbShare: repo.get("smb_share"),
    repoSubpath: repo.get("repo_subpath") || "",
    smbUsername: repo.get("username") || "",
    smbPassword: repo.get("password") || "",
    resticPassword: repo.get("restic_password") || "",
  });
});

routerAdd("POST", "/api/backai/agent/work/{runId}/report", (c) => {
  const agent = agentFromRequest(c);
  if (!agent) return c.json(401, { error: "unauthorized" });

  const runId = c.request.pathValue("runId");
  let run;
  try {
    run = $app.findRecordById("job_runs", runId);
  } catch (_) {
    return c.json(404, { error: "not found" });
  }
  if (run.get("agent") !== agent.id) {
    return c.json(403, { error: "forbidden" });
  }

  const body = readJson(c);
  const status = body.status || run.get("status");
  run.set("status", status);
  if (body.bytesAdded != null) run.set("bytes_added", body.bytesAdded);
  if (body.bytesProcessed != null) run.set("bytes_processed", body.bytesProcessed);
  if (body.durationMs != null) run.set("duration_ms", body.durationMs);
  if (body.filesNew != null) run.set("files_new", body.filesNew);
  if (body.filesChanged != null) run.set("files_changed", body.filesChanged);
  if (body.filesUnmodified != null) run.set("files_unmodified", body.filesUnmodified);
  if (body.snapshotId != null) run.set("snapshot_id", body.snapshotId);
  if (body.errorMessage != null) run.set("error_message", body.errorMessage);
  if (body.progressPct != null) run.set("progress_pct", body.progressPct);
  if (["succeeded", "failed", "cancelled"].includes(status)) {
    run.set("finished_at", new Date().toISOString());
  }
  $app.save(run);
  return c.json(200, { status: "ok" });
});

cronAdd("backai-schedule", "* * * * *", () => {
  const now = new Date();
  let jobs = [];
  try {
    jobs = $app.findRecordsByFilter("backup_jobs", "enabled = true && schedule_cron != ''", "name", 200, 0);
  } catch (_) {
    return;
  }
  const since = new Date(now.getTime() - 55 * 60 * 1000).toISOString();

  for (const job of jobs) {
    const spec = job.get("schedule_cron");
    if (!cronDue(spec, now)) continue;

    let recent = [];
    try {
      recent = $app.findRecordsByFilter(
        "job_runs",
        `job = {:jobId} && requested_at >= {:since}`,
        "-requested_at",
        1,
        0,
        { jobId: job.id, since }
      );
    } catch (_) {
      recent = [];
    }
    if (recent.length) continue;

    try {
      createRunForJob(job);
    } catch (e) {
      console.log("schedule run failed", job.id, e);
    }
  }
});
