# BackAI

Centrale **backup control plane** met **Restic**, **SMB/NAS** repositories, web-dashboard en **versioned agents** voor Windows en Linux (file-level).

De server draait als **één Docker-container** op **[PocketBase](https://pocketbase.io/)** (database + API + static UI), hetzelfde patroon als [tracknote](https://github.com/s4ndyp/tracknote).

## Architectuur

```
Browser → PocketBase (:8090) → pb_public (React UI)
              ↕ pb_data (SQLite)
              ↕ pb_hooks (agent API + scheduler)
         backai-agent → SMB NAS (restic repo)
```

## Deploy (Proxmox LXC + Docker)

```bash
git clone https://github.com/s4ndyp/backai
cd backai
cp .env.example .env
# Zet BACKAI_AGENT_REGISTER_KEY op een sterk geheim
docker compose pull
docker compose up -d
```

UI: **http://&lt;host&gt;:8090**  
PocketBase admin (optioneel): **http://&lt;host&gt;:8090/_/** — alleen nodig voor geavanceerd beheer.

### Docker image

```bash
docker pull ghcr.io/s4ndyp/backai:latest
```

Poort default **8090** (niet 8080). Pas aan via `BACKAI_HTTP_PORT` in `.env`.

## Agents

### Vereisten

- [Restic](https://restic.net/) in `PATH`
- **Linux**: `cifs-utils`, mount-rechten
- **Windows**: `net use` / SMB-toegang

### Registratie

```bash
export BACKAI_SERVER_URL=http://192.168.1.50:8090
export BACKAI_AGENT_REGISTER_KEY=<zelfde als in .env>
export BACKAI_AGENT_NAME=linux-pc-1
./backai-agent-<versie>-linux-amd64
```

Bewaar **`BACKAI_AGENT_TOKEN`** en gebruik die daarna permanent.

Downloads: **GitHub Releases** → tag `agents-latest` of semver `v*`.

## UI

1. **SMB repos** — NAS + restic-wachtwoord  
2. **Agents** — na registratie zichtbaar  
3. **Backup jobs** — agent, repo, paden, cron (`0 2 * * *`, `@daily`, `@hourly`)  
4. **Runs / Dashboard** — statistieken en voortgang  

## Ontwikkeling

```bash
# UI
cd web && npm install && npm run dev
# PocketBase lokaal (apart binary) of via Docker build

# Agent
go run ./cmd/agent
```

Web build output: `pb_public/` (wordt in de Docker image gebakken).

## Migratie vanaf Go + Postgres

Oude stack (`postgres` + `:8080`) is vervangen. Data moet opnieuw ingevoerd worden (homelab MVP). Agents moeten opnieuw registreren; zet `BACKAI_SERVER_URL` op poort **8090**.

## Licentie

MIT
