import { FormEvent, useEffect, useState } from "react";
import { api, Repository } from "../api";

export default function Repositories() {
  const [repos, setRepos] = useState<Repository[]>([]);
  const [form, setForm] = useState({
    name: "",
    smbHost: "",
    smbShare: "",
    repoSubpath: "backai",
    username: "",
    password: "",
    resticPassword: "",
  });

  const reload = () => api.repos().then(setRepos).catch(console.error);
  useEffect(() => { reload(); }, []);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    await api.createRepo(form);
    setForm({ ...form, name: "", password: "", resticPassword: "" });
    reload();
  }

  return (
    <div>
      <header className="page-header">
        <h1>SMB repositories</h1>
        <p>Restic-repositories op je NAS (SMB). Elke repo heeft een apart restic-wachtwoord.</p>
      </header>

      <form className="panel form-grid" onSubmit={onSubmit}>
        <h2>Nieuwe repo</h2>
        <label>Naam<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></label>
        <label>SMB host<input value={form.smbHost} onChange={(e) => setForm({ ...form, smbHost: e.target.value })} placeholder="192.168.1.10" required /></label>
        <label>Share<input value={form.smbShare} onChange={(e) => setForm({ ...form, smbShare: e.target.value })} placeholder="backups" required /></label>
        <label>Submap repo<input value={form.repoSubpath} onChange={(e) => setForm({ ...form, repoSubpath: e.target.value })} /></label>
        <label>Gebruiker<input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} /></label>
        <label>Wachtwoord<input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} /></label>
        <label>Restic repo-wachtwoord<input type="password" value={form.resticPassword} onChange={(e) => setForm({ ...form, resticPassword: e.target.value })} required /></label>
        <button type="submit">Opslaan</button>
      </form>

      <div className="panel">
        <table>
          <thead>
            <tr><th>Naam</th><th>Host</th><th>Share</th><th>Pad</th></tr>
          </thead>
          <tbody>
            {repos.map((r) => (
              <tr key={r.id}>
                <td>{r.name}</td>
                <td className="mono">{r.smbHost}</td>
                <td>{r.smbShare}</td>
                <td className="mono">{r.repoSubpath}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
