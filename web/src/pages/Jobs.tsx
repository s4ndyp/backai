import { FormEvent, useEffect, useState } from "react";
import { Agent, api, BackupJob, Repository } from "../api";

export default function Jobs() {
  const [jobs, setJobs] = useState<BackupJob[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [repos, setRepos] = useState<Repository[]>([]);
  const [form, setForm] = useState({
    name: "",
    agentId: "",
    repositoryId: "",
    sources: "/home",
    excludes: "",
    scheduleCron: "0 2 * * *",
  });

  const reload = () => {
    api.jobs().then(setJobs).catch(console.error);
    api.agents().then(setAgents).catch(console.error);
    api.repos().then(setRepos).catch(console.error);
  };
  useEffect(() => { reload(); }, []);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    await api.createJob({
      name: form.name,
      agentId: form.agentId,
      repositoryId: form.repositoryId,
      sources: form.sources.split("\n").map((s) => s.trim()).filter(Boolean),
      excludes: form.excludes.split("\n").map((s) => s.trim()).filter(Boolean),
      scheduleCron: form.scheduleCron,
      enabled: true,
    });
    setForm({ ...form, name: "" });
    reload();
  }

  return (
    <div>
      <header className="page-header">
        <h1>Backup jobs</h1>
        <p>File-level backups via Restic naar SMB. Cron: minuut uur * * * of @daily / @hourly.</p>
      </header>

      <form className="panel form-grid" onSubmit={onSubmit}>
        <h2>Nieuwe job</h2>
        <label>Naam<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></label>
        <label>Agent
          <select value={form.agentId} onChange={(e) => setForm({ ...form, agentId: e.target.value })} required>
            <option value="">Kies agent</option>
            {agents.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
          </select>
        </label>
        <label>Repository
          <select value={form.repositoryId} onChange={(e) => setForm({ ...form, repositoryId: e.target.value })} required>
            <option value="">Kies repo</option>
            {repos.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
          </select>
        </label>
        <label>Bronmappen (één per regel)<textarea value={form.sources} onChange={(e) => setForm({ ...form, sources: e.target.value })} rows={3} /></label>
        <label>Excludes<textarea value={form.excludes} onChange={(e) => setForm({ ...form, excludes: e.target.value })} rows={2} /></label>
        <label>Schedule cron<input value={form.scheduleCron} onChange={(e) => setForm({ ...form, scheduleCron: e.target.value })} /></label>
        <button type="submit">Job aanmaken</button>
      </form>

      <div className="panel">
        <table>
          <thead>
            <tr><th>Naam</th><th>Agent</th><th>Repo</th><th>Schedule</th><th>Status</th><th></th></tr>
          </thead>
          <tbody>
            {jobs.map((j) => (
              <tr key={j.id}>
                <td>{j.name}</td>
                <td>{j.agentName}</td>
                <td>{j.repoName}</td>
                <td className="mono">{j.scheduleCron || "–"}</td>
                <td>{j.enabled ? "aan" : "uit"}</td>
                <td className="actions">
                  <button type="button" onClick={() => api.runJob(j.id).then(reload)}>Run now</button>
                  <button type="button" className="ghost" onClick={() => api.toggleJob(j.id, !j.enabled).then(reload)}>
                    {j.enabled ? "Uit" : "Aan"}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
