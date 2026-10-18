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
  const data = await r.json();
  return data as T;
}

async function getList<T>(path: string): Promise<T[]> {
  const data = await get<T[] | null>(path);
  return Array.isArray(data) ? data : [];
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const r = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!r.ok) throw new Error(await r.text());
  return r.json();
}

export const api = {
  dashboard: () => get<DashboardStats>("/api/dashboard"),
  agents: () => getList<Agent>("/api/agents"),
  repos: () => getList<Repository>("/api/repositories"),
  createRepo: (body: Record<string, string>) => post<Repository>("/api/repositories", body),
  jobs: () => getList<BackupJob>("/api/jobs"),
  createJob: (body: Record<string, unknown>) => post<BackupJob>("/api/jobs", body),
  runJob: (id: string) => post<JobRun>(`/api/jobs/${id}/run`, {}),
  toggleJob: (id: string, enabled: boolean) =>
    fetch(`/api/jobs/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ enabled }),
    }).then((r) => r.json()),
  runs: () => getList<JobRun>("/api/runs"),
  version: () => get<{ version: string }>("/api/version"),
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
