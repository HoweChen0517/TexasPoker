# TexasPoker

Multiplayer Texas Hold'em for web, with a Go game server and a React table UI.

## Features

- Real-time multiplayer room system (SSE + HTTP actions)
- Texas Hold'em hand flow: blinds, betting rounds, showdown
- Side-pot settlement and multi-player all-in support
- Classic + Short Deck mode (6+; Flush beats Full House)
- Host controls: start/restart hand, dissolve room
- Spectator flow: join next hand, seat switching
- Responsive web UI (desktop + mobile)

## Tech Stack

- Backend: Go (`server/`)
- Frontend: React + Vite + TypeScript (`web/`)
- Realtime transport: SSE (`/events`) + POST actions (`/action`)

## Project Structure

- `server/cmd/pokerd/main.go` - server entrypoint
- `server/internal/engine` - poker engine and hand evaluation
- `server/internal/room` - room/session lifecycle and host logic
- `server/internal/network` - SSE/event and action handlers
- `web/src/App.tsx` - main table page and controls

## Quick Start

### 1) Run server

```bash
cd server
go run ./cmd/pokerd
```

Server default: `http://localhost:8080`

### 2) Run web

```bash
cd web
npm install
npm run dev
```

Web default: `http://localhost:5173`

If needed:

```bash
VITE_API_URL=http://localhost:8080 npm run dev
```

## Join a Room

Open multiple tabs/devices with the same `room`:

- `http://localhost:5173/?room=alpha&user=u1&name=Alice`
- `http://localhost:5173/?room=alpha&user=u2&name=Bob`

## Deployment

Recommended split deployment:

- Frontend: Vercel (root: `web`)
- Backend: long-running host (Railway / Render / Fly.io / VPS)

Set frontend env:

- `VITE_API_URL=https://your-backend-domain`

## Realtime Protocol

Client -> server (`POST /action?room=...&user=...`):

- `{"type":"start_hand","payload":{"mode":"classic|short"}}`
- `{"type":"restart_hand"}`
- `{"type":"action","payload":{"action":"fold|check|call|bet|raise|all_in","amount":40}}`
- `{"type":"join_table","payload":{"seat":3}}`
- `{"type":"set_seat","payload":{"seat":5}}`
- `{"type":"leave_room"}`
- `{"type":"dissolve_room"}`

Server -> client (`GET /events?room=...&user=...&name=...`):

- `snapshot` (full room + hand state)
- `error` (message)

## Notes

- Rooms are keyed by `room` name.
- Empty rooms are auto-removed.
- Host can dissolve room manually.

## License

MIT (or your preferred license)
