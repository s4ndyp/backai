import { NavLink, Route, Routes } from "react-router-dom";
import Dashboard from "./pages/Dashboard";
import Agents from "./pages/Agents";
import Repositories from "./pages/Repositories";
import Jobs from "./pages/Jobs";
import Runs from "./pages/Runs";
import { useEffect, useState } from "react";
import { api } from "./api";

export default function App() {
  const [version, setVersion] = useState("");

  useEffect(() => {
    api.version().then((v) => setVersion(v.version)).catch(() => {});
  }, []);

  return (
    <div className="layout">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark" />
          <div>
            <strong>BackAI</strong>
            <small>Restic control plane</small>
          </div>
        </div>
        <nav>
          <NavLink to="/" end>Dashboard</NavLink>
          <NavLink to="/agents">Agents</NavLink>
          <NavLink to="/repositories">SMB repos</NavLink>
          <NavLink to="/jobs">Backup jobs</NavLink>
          <NavLink to="/runs">Runs</NavLink>
        </nav>
        {version && <footer className="sidebar-footer">v{version}</footer>}
      </aside>
      <main className="content">
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/agents" element={<Agents />} />
          <Route path="/repositories" element={<Repositories />} />
          <Route path="/jobs" element={<Jobs />} />
          <Route path="/runs" element={<Runs />} />
        </Routes>
      </main>
    </div>
  );
}
