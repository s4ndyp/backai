import { pb } from "./pb";

export type DashboardStats = {
  totalAgents: number;
  onlineAgents: number;
  totalJobs: number;
  enabledJobs: number;
  runsLast24h: number;
  successRate24h: number;
  bytesAdded24h: number;
  avgDurationMs24h: number;
  avgThroughputMBs: number;
};

export type Agent = {
  id: string;
  name: string;
  hostname: string;
  os: string;
  agentVersion: string;
  createdAt: string;
  lastSeenAt?: string;
};

export type Repository = {
  id: string;
  name: string;
  smbHost: string;
  smbShare: string;
  repoSubpath: string;
  username: string;
  createdAt: string;
};

export type BackupJob = {
  id: string;
  agentId: string;
  repositoryId: string;
  name: string;
  sources: string[];
  excludes: string[];
  scheduleCron: string;
  enabled: boolean;
  agentName?: string;
  repoName?: string;
};

export type JobRun = {
  id: string;
  jobId: string;
  status: string;
  requestedAt: string;
  startedAt?: string;
  finishedAt?: string;
  bytesAdded: number;
  bytesProcessed: number;
  durationMs: number;
  progressPct: number;
  jobName?: string;
  agentName?: string;
  errorMessage?: string;
};

async function get<T>(path: string): Promise<T> {
  const r = await fetch(path);
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}

function mapAgent(r: Record<string, unknown>): Agent {
  return {
    id: String(r.id),
    name: String(r.name ?? ""),
    hostname: String(r.hostname ?? ""),
    os: String(r.os ?? ""),
    agentVersion: String(r.agent_version ?? ""),
    createdAt: String(r.created ?? ""),
    lastSeenAt: r.last_seen_at ? String(r.last_seen_at) : undefined,
  };
}

function mapRepo(r: Record<string, unknown>): Repository {
  return {
    id: String(r.id),
    name: String(r.name ?? ""),
    smbHost: String(r.smb_host ?? ""),
    smbShare: String(r.smb_share ?? ""),
    repoSubpath: String(r.repo_subpath ?? ""),
    username: String(r.username ?? ""),
    createdAt: String(r.created ?? ""),
  };
}

function mapJob(r: Record<string, unknown>): BackupJob {
  const expand = r.expand as Record<string, Record<string, unknown>> | undefined;
  const sources = r.sources;
  const excludes = r.excludes;
  return {
    id: String(r.id),
    agentId: String(r.agent ?? ""),
    repositoryId: String(r.repository ?? ""),
    name: String(r.name ?? ""),
    sources: Array.isArray(sources) ? sources.map(String) : [],
    excludes: Array.isArray(excludes) ? excludes.map(String) : [],
    scheduleCron: String(r.schedule_cron ?? ""),
    enabled: Boolean(r.enabled),
    agentName: expand?.agent ? String(expand.agent.name ?? "") : undefined,
    repoName: expand?.repository ? String(expand.repository.name ?? "") : undefined,
  };
}

function mapRun(r: Record<string, unknown>): JobRun {
  const expand = r.expand as Record<string, Record<string, unknown>> | undefined;
  return {
    id: String(r.id),
    jobId: String(r.job ?? ""),
    status: String(r.status ?? ""),
    requestedAt: String(r.requested_at ?? r.created ?? ""),
    startedAt: r.started_at ? String(r.started_at) : undefined,
    finishedAt: r.finished_at ? String(r.finished_at) : undefined,
    bytesAdded: Number(r.bytes_added ?? 0),
    bytesProcessed: Number(r.bytes_processed ?? 0),
    durationMs: Number(r.duration_ms ?? 0),
    progressPct: Number(r.progress_pct ?? 0),
    jobName: expand?.job ? String(expand.job.name ?? "") : undefined,
    agentName: expand?.agent ? String(expand.agent.name ?? "") : undefined,
    errorMessage: r.error_message ? String(r.error_message) : undefined,
  };
}

export const api = {
  dashboard: () => get<DashboardStats>("/api/backai/dashboard"),
  agents: async () => {
    const list = await pb.collection("agents").getFullList({ sort: "name" });
    return list.map((r) => mapAgent(r as unknown as Record<string, unknown>));
  },
  repos: async () => {
    const list = await pb.collection("smb_repositories").getFullList({ sort: "name" });
    return list.map((r) => mapRepo(r as unknown as Record<string, unknown>));
  },
  createRepo: async (body: Record<string, string>) => {
    const r = await pb.collection("smb_repositories").create({
      name: body.name,
      smb_host: body.smbHost,
      smb_share: body.smbShare,
      repo_subpath: body.repoSubpath || "",
      username: body.username || "",
      password: body.password || "",
      restic_password: body.resticPassword,
    });
    return mapRepo(r as unknown as Record<string, unknown>);
  },
  jobs: async () => {
    const list = await pb.collection("backup_jobs").getFullList({
      sort: "name",
      expand: "agent,repository",
    });
    return list.map((r) => mapJob(r as unknown as Record<string, unknown>));
  },
  createJob: async (body: Record<string, unknown>) => {
    const r = await pb.collection("backup_jobs").create({
      name: body.name,
      agent: body.agentId,
      repository: body.repositoryId,
      sources: body.sources,
      excludes: body.excludes ?? [],
      schedule_cron: body.scheduleCron ?? "",
      enabled: body.enabled ?? true,
    });
    return mapJob(r as unknown as Record<string, unknown>);
  },
  runJob: async (id: string) => {
    const job = await pb.collection("backup_jobs").getOne(id);
    const r = await pb.collection("job_runs").create({
      job: id,
      agent: job.agent,
      status: "pending",
      requested_at: new Date().toISOString(),
      bytes_added: 0,
      bytes_processed: 0,
      duration_ms: 0,
      progress_pct: 0,
    });
    return mapRun(r as unknown as Record<string, unknown>);
  },
  toggleJob: async (id: string, enabled: boolean) => {
    const r = await pb.collection("backup_jobs").update(id, { enabled });
    return mapJob(r as unknown as Record<string, unknown>);
  },
  runs: async () => {
    const list = await pb.collection("job_runs").getFullList({
      sort: "-requested_at",
      expand: "job,agent",
    });
    return list.map((r) => mapRun(r as unknown as Record<string, unknown>));
  },
  version: () => get<{ version: string }>("/api/backai/version"),
};

export function formatBytes(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1024 ** 2) return `${(n / 1024).toFixed(1)} KiB`;
  if (n < 1024 ** 3) return `${(n / 1024 ** 2).toFixed(1)} MiB`;
  return `${(n / 1024 ** 3).toFixed(2)} GiB`;
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms} ms`;
  const s = Math.floor(ms / 1000);
  const m = Math.floor(s / 60);
  if (m === 0) return `${s}s`;
  return `${m}m ${s % 60}s`;
}
