import { useEffect, useState } from "react";
import { Agent, api } from "../api";

export default function Agents() {
  const [agents, setAgents] = useState<Agent[]>([]);

  useEffect(() => {
    api.agents().then(setAgents).catch(console.error);
  }, []);

  return (
    <div>
      <header className="page-header">
        <h1>Agents</h1>
        <p>Geregistreerde Windows- en Linux-clients. Installeer restic + backai-agent op elke machine.</p>
      </header>
      <div className="panel">
        <table>
          <thead>
            <tr>
              <th>Naam</th>
              <th>Host</th>
              <th>OS</th>
              <th>Versie</th>
              <th>Laatst gezien</th>
            </tr>
          </thead>
          <tbody>
            {agents.map((a) => (
              <tr key={a.id}>
                <td>{a.name}</td>
                <td className="mono">{a.hostname}</td>
                <td>{a.os}</td>
                <td className="mono">{a.agentVersion || "–"}</td>
                <td>{a.lastSeenAt ? new Date(a.lastSeenAt).toLocaleString() : "nooit"}</td>
              </tr>
            ))}
            {agents.length === 0 && (
              <tr><td colSpan={5} className="muted">Nog geen agents — start de agent met register key.</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
