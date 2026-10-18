import { useEffect, useState } from "react";
import { api, DashboardStats, formatBytes, JobRun } from "../api";
import { Bar, BarChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from "recharts";

export default function Dashboard() {
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [runs, setRuns] = useState<JobRun[]>([]);

  useEffect(() => {
    const load = () => {
      api.dashboard().then(setStats).catch(console.error);
      api.runs().then(setRuns).catch(console.error);
    };
    load();
    const t = setInterval(load, 10000);
    return () => clearInterval(t);
  }, []);

  const chartData = [...runs].slice(0, 12).reverse().map((r) => ({
    name: (r.jobName || r.id?.slice(0, 8) || "run").slice(0, 12),
    mb: Math.round(r.bytesAdded / (1024 * 1024)),
    sec: Math.max(1, Math.round(r.durationMs / 1000)),
  }));

  return (
    <div>
      <header className="page-header">
        <h1>Dashboard</h1>
        <p>Overzicht van agents, jobs en backup-statistieken (laatste 24 uur).</p>
      </header>

      <div className="grid stats">
        <StatCard label="Agents online" value={`${stats?.onlineAgents ?? "–"} / ${stats?.totalAgents ?? "–"}`} />
        <StatCard label="Actieve jobs" value={`${stats?.enabledJobs ?? "–"} / ${stats?.totalJobs ?? "–"}`} />
        <StatCard label="Runs (24h)" value={stats?.runsLast24h ?? "–"} />
        <StatCard label="Succes (24h)" value={stats ? `${(stats.successRate24h ?? 0).toFixed(0)}%` : "–"} />
        <StatCard label="Data toegevoegd (24h)" value={stats ? formatBytes(stats.bytesAdded24h ?? 0) : "–"} />
        <StatCard label="Gem. throughput" value={stats ? `${(stats.avgThroughputMBs ?? 0).toFixed(1)} MiB/s` : "–"} />
      </div>

      <section className="panel">
        <h2>Recente runs — grootte (MiB)</h2>
        <div className="chart-wrap">
          <ResponsiveContainer width="100%" height={260}>
            <BarChart data={chartData}>
              <XAxis dataKey="name" stroke="#8b9cb3" fontSize={12} />
              <YAxis stroke="#8b9cb3" fontSize={12} />
              <Tooltip contentStyle={{ background: "#111827", border: "1px solid #334155" }} />
              <Bar dataKey="mb" fill="#38bdf8" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </section>
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="stat-card">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}
