import { useEffect, useState } from "react";
import { api, formatBytes, formatDuration, JobRun } from "../api";

export default function Runs() {
  const [runs, setRuns] = useState<JobRun[]>([]);

  useEffect(() => {
    const load = () => api.runs().then(setRuns).catch(console.error);
    load();
    const t = setInterval(load, 5000);
    return () => clearInterval(t);
  }, []);

  return (
    <div>
      <header className="page-header">
        <h1>Backup runs</h1>
        <p>Live voortgang, snelheid (via duur/grootte) en status per run.</p>
      </header>
      <div className="panel">
        <table>
          <thead>
            <tr>
              <th>Job</th>
              <th>Agent</th>
              <th>Status</th>
              <th>Voortgang</th>
              <th>Toegevoegd</th>
              <th>Duur</th>
              <th>Gevraagd</th>
            </tr>
          </thead>
          <tbody>
            {runs.map((r) => (
              <tr key={r.id}>
                <td>{r.jobName}</td>
                <td>{r.agentName}</td>
                <td><span className={`pill ${r.status}`}>{r.status}</span></td>
                <td>
                  <div className="progress">
                    <div style={{ width: `${r.progressPct}%` }} />
                  </div>
                  {r.progressPct.toFixed(0)}%
                </td>
                <td>{formatBytes(r.bytesAdded)}</td>
                <td>{r.durationMs ? formatDuration(r.durationMs) : "–"}</td>
                <td>{new Date(r.requestedAt).toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
