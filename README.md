# BackAI

Centrale **backup control plane** met **Restic**, **SMB/NAS** repositories, web-dashboard en **versioned agents** voor Windows en Linux (file-level).

## Architectuur

- **Server** (Docker): API + React UI + PostgreSQL, plant jobs en bewaart statistieken.
- **Agent** op elke client: mount SMB, `restic backup`, rapporteert voortgang naar de server.

```
Browser → backai-server → PostgreSQL
              ↑
         backai-agent (Windows/Linux) → SMB NAS (restic repo)
```

## Server deploy (Proxmox LXC + Docker)

1. LXC met Docker (of VM). Open poort **8080** (of kies andere via env).
2. Clone of kopieer `deploy/` naar de LXC.

```bash
cd deploy
cp .env.example .env
# Bewerk .env: sterke POSTGRES_PASSWORD, BACKAI_SESSION_SECRET, BACKAI_AGENT_REGISTER_KEY
docker compose pull
docker compose up -d
```

3. UI: `http://<lxc-ip>:8080`

### Docker image pull

Images worden gepubliceerd naar **GitHub Container Registry**:

```bash
docker pull ghcr.io/s4ndyp/backai:latest
```

Bij een **git tag** `v1.0.0` verschijnt ook `ghcr.io/s4ndyp/backai:1.0.0`.

Eerste keer GHCR (private repo): `docker login ghcr.io` met een GitHub PAT (read:packages).

## Agents installeren

### Vereisten

- [Restic](https://restic.net/) in `PATH`
- **Linux**: `cifs-utils` (`mount.cifs`) en root/CAP voor mount (agent draait typisch als root of met sudo voor mount)
- **Windows**: `net use` voor SMB (agent vaak als service account met rechten op shares)

### Eerste registratie

```bash
export BACKAI_SERVER_URL=http://192.168.1.50:8080
export BACKAI_AGENT_REGISTER_KEY=<zelfde als server BACKAI_AGENT_REGISTER_KEY>
export BACKAI_AGENT_NAME=linux-desktop-1
./backai-agent-<versie>-linux-amd64
```

De agent print een **`BACKAI_AGENT_TOKEN`** — bewaar die en zet daarna:

```bash
export BACKAI_AGENT_TOKEN=<token>
# BACKAI_AGENT_REGISTER_KEY niet meer nodig
```

### Downloads

Agent-clients staan op **GitHub → Releases**:

- **`agents-latest`** — bij elke merge naar `main` (meest recente build, aanbevolen voor homelab)
- **`v1.0.0`** e.d. — vaste semver-release na `git tag v1.0.0 && git push origin v1.0.0`

Bestandsnaam: `backai-agent-<versie>-linux-amd64`, `…-linux-arm64`, `…-windows-amd64.exe`

(CI bewaart ook **workflow artifacts** per run; Releases zijn de permanente downloadlocatie.)

Versie staat ingebakken (`backai-agent/<version>` User-Agent) en is zichtbaar in de UI onder Agents.

### Linux systemd (voorbeeld)

```ini
[Unit]
Description=BackAI Agent
After=network-online.target

[Service]
Environment=BACKAI_SERVER_URL=http://192.168.1.50:8080
Environment=BACKAI_AGENT_TOKEN=...
ExecStart=/usr/local/bin/backai-agent
Restart=always

[Install]
WantedBy=multi-user.target
```

## Gebruik in de UI

1. **SMB repos** — NAS host, share, submap, SMB-credentials, **restic repo-wachtwoord**.
2. **Agents** — verschijnen na registratie.
3. **Backup jobs** — kies agent + repo, bronmappen (één pad per regel), optioneel excludes, cron (`0 2 * * *` = 02:00, of `@daily` / `@hourly`).
4. **Runs** — live voortgang; dashboard toont 24u statistieken.

## Ontwikkeling lokaal

```bash
# Postgres
docker run -d --name backai-pg -e POSTGRES_PASSWORD=backai -e POSTGRES_USER=backai -e POSTGRES_DB=backai -p 5432:5432 postgres:16-alpine

export BACKAI_DATABASE_URL=postgres://backai:backai@localhost:5432/backai?sslmode=disable
export BACKAI_AGENT_REGISTER_KEY=dev-register-key

cd web && npm install && npm run build && cd ..
go run ./cmd/server

# apart terminal — agent
export BACKAI_SERVER_URL=http://localhost:8080
export BACKAI_AGENT_REGISTER_KEY=dev-register-key
go run ./cmd/agent
```

Frontend dev met proxy: `cd web && npm run dev`

## Roadmap (kort)

- Map-browser op agent (remote path picker)
- TLS / mTLS en UI-login
- Restore flows vanuit UI
- Agent auto-update kanaal

## Licentie

MIT (aan te passen naar wens)
